#!/usr/bin/env bash
#
# Prueba de humo de la API: recorre el flujo completo de un cliente y el ABM del
# admin, verificando el codigo HTTP de cada paso y el efecto sobre el stock.
#
# No reemplaza a los tests de `go test ./internal/services/...`, que corren sin
# base y prueban las reglas aisladas. Esto prueba lo que aquellos no pueden: que
# el cableado de rutas, el middleware de JWT, GORM y MySQL funcionan JUNTOS.
#
# USO
#   ./smoke-test.sh                        contra localhost:8080 (la API directo)
#   BASE=http://localhost:3000 ./smoke-test.sh   a traves de nginx (prueba el proxy)
#
# REQUISITOS
#   - El stack levantado:  docker compose up -d
#   - La base sembrada:    MSYS_NO_PATHCONV=1 docker compose exec backend /bin/seed
#   - Solo curl, sed y grep. Sin jq: no esta instalado y en el runner de CI del
#     TP4 tampoco lo estaria.
#
# SALIDA
#   0 si pasa todo, 1 si algo falla. Por eso sirve como gate de un pipeline.
#
# EFECTO SOBRE LOS DATOS
#   Es repetible: no consume stock del seed. Lo que compra lo cancela, y lo que
#   no puede cancelar (el pedido que queda en "enviado") lo compra sobre un
#   producto descartable que crea y da de baja en la misma corrida. Cada corrida
#   deja en la base dos usuarios de prueba (el cliente y el "intruso") y un
#   producto inactivo, mas las dos imagenes que sube en el volumen uploads_data.
#   Todo eso es invisible en el catalogo; para empezar de cero, docker compose
#   down -v.

set -u

BASE="${BASE:-http://localhost:8080}"
# El seed deja la variante 2 = "Remera basica de algodon" talle M / color Negro.
# Se pueden pisar por entorno si se prueba contra otros datos.
PRODUCTO="${PRODUCTO:-1}"
VARIANTE="${VARIANTE:-2}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@tienda.local}"
ADMIN_PASS="${ADMIN_PASS:-admin12345}"
# Los mismos valores que .env.example. Si los cambias en el .env, cambialos aca
# o pasalos por entorno: son configuracion, no constantes del dominio.
UMBRAL="${UMBRAL_ENVIO_GRATIS:-50000}"
ENVIO="${COSTO_ENVIO:-5000}"

# Email distinto en cada corrida: si no, el registro del segundo intento choca
# contra la regla de email unico y el script no seria repetible.
CLIENTE="cliente-$(date +%s)@ejemplo.com"
CLAVE="claveSegura1"

fallos=0
paso=0

# Archivos para probar la subida de imagenes. Rutas RELATIVAS a proposito: una
# ruta absoluta estilo /tmp/x.jpg la reescribe MSYS en Git Bash antes de que
# curl la vea. Se borran al salir, pase lo que pase.
IMAGEN_OK="smoke-imagen-$$.jpg"
IMAGEN_FALSA="smoke-falsa-$$.jpg"
IMAGEN_TEXTO="smoke-texto-$$.txt"
trap 'rm -f "$IMAGEN_OK" "$IMAGEN_FALSA" "$IMAGEN_TEXTO"' EXIT

# Un JPEG minimo: del contenido, el backend solo exige que empiece con la firma
# FF D8 FF. La falsa tiene extension .jpg pero contenido de texto, que es
# justamente el caso que la lista de extensiones sola no atrapa.
printf '\377\330\377 smoke-test' > "$IMAGEN_OK"
printf 'MZ esto no es una imagen' > "$IMAGEN_FALSA"
printf 'texto plano' > "$IMAGEN_TEXTO"

# --- helpers ----------------------------------------------------------------

# verificar <esperado> <descripcion> <comando curl...>
# Ejecuta el comando, compara el codigo HTTP y acumula fallos.
verificar() {
  local esperado="$1" descripcion="$2"; shift 2
  paso=$((paso + 1))
  local codigo
  codigo=$("$@" -s -o /dev/null -w '%{http_code}')
  if [ "$codigo" = "$esperado" ]; then
    printf '  ok   %2d. %-52s HTTP %s\n' "$paso" "$descripcion" "$codigo"
  else
    printf '  FALLA %2d. %-52s HTTP %s (se esperaba %s)\n' "$paso" "$descripcion" "$codigo" "$esperado"
    fallos=$((fallos + 1))
  fi
}

# comparar <esperado> <obtenido> <descripcion>
# Para lo que no es un codigo HTTP: stock, totales.
comparar() {
  local esperado="$1" obtenido="$2" descripcion="$3"
  paso=$((paso + 1))
  if [ "$esperado" = "$obtenido" ]; then
    printf '  ok   %2d. %-52s %s\n' "$paso" "$descripcion" "$obtenido"
  else
    printf '  FALLA %2d. %-52s %s (se esperaba %s)\n' "$paso" "$descripcion" "$obtenido" "$esperado"
    fallos=$((fallos + 1))
  fi
}

# Extrae la PRIMERA aparicion de un campo de una respuesta JSON.
#
# El `tr` es lo que lo hace correcto: parte el JSON en una linea por campo. Sin
# eso, el `.*` de sed es codicioso y se queda con la ULTIMA aparicion — en la
# respuesta de crear un producto, el "id" de la variante anidada en vez del "id"
# del producto. No es un parser de JSON y no pretende serlo: alcanza para "id" y
# "token", que son campos planos.
campo() { tr ',{' '\n\n' | sed -n "s/.*\"$1\":\"\{0,1\}\([^,\"}]*\).*/\1/p" | head -1; }

# Stock actual de la variante. El `tr` parte el JSON en una linea por objeto,
# que es lo que permite aislar la variante buscada sin un parser.
stock() {
  curl -s "$BASE/api/productos/$PRODUCTO" \
    | tr '{' '\n' \
    | grep "\"id\":$VARIANTE," \
    | sed -n 's/.*"stock":\([0-9]*\).*/\1/p'
}


echo "Prueba de humo contra $BASE"
echo "cliente de esta corrida: $CLIENTE"
echo

# --- 1. Publico -------------------------------------------------------------

echo "-- endpoints publicos --"
verificar 200 "healthz responde"                curl "$BASE/healthz"
verificar 200 "el catalogo lista"               curl "$BASE/api/productos"
verificar 200 "filtro por categoria"            curl "$BASE/api/productos?categoria=camisas"
verificar 200 "detalle de un producto"          curl "$BASE/api/productos/$PRODUCTO"
verificar 400 "un id no numerico da 400"        curl "$BASE/api/productos/abc"
verificar 404 "un producto inexistente da 404"  curl "$BASE/api/productos/999999"

# --- 2. Registro y login ----------------------------------------------------

echo
echo "-- registro y login --"
verificar 201 "registro de un cliente nuevo" \
  curl -X POST "$BASE/api/usuarios/registro" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$CLIENTE\",\"password\":\"$CLAVE\",\"nombre\":\"Ana\",\"apellido\":\"Perez\"}"

# El mismo email en MAYUSCULAS: el service lo normaliza, asi que tiene que
# chocar igual contra la regla de email unico.
verificar 409 "email repetido (aun en mayusculas) da 409" \
  curl -X POST "$BASE/api/usuarios/registro" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$(echo "$CLIENTE" | tr 'a-z' 'A-Z')\",\"password\":\"$CLAVE\",\"nombre\":\"Ana\",\"apellido\":\"Perez\"}"

verificar 400 "password corta rechazada" \
  curl -X POST "$BASE/api/usuarios/registro" -H 'Content-Type: application/json' \
  -d '{"email":"corta@ejemplo.com","password":"1234","nombre":"Ana","apellido":"Perez"}'

verificar 400 "email invalido rechazado" \
  curl -X POST "$BASE/api/usuarios/registro" -H 'Content-Type: application/json' \
  -d '{"email":"no-es-un-email","password":"claveSegura1","nombre":"Ana","apellido":"Perez"}'

verificar 401 "login con password incorrecta" \
  curl -X POST "$BASE/api/usuarios/login" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$CLIENTE\",\"password\":\"noEsLaClave\"}"

verificar 200 "login correcto" \
  curl -X POST "$BASE/api/usuarios/login" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$CLIENTE\",\"password\":\"$CLAVE\"}"

TOKEN=$(curl -s -X POST "$BASE/api/usuarios/login" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$CLIENTE\",\"password\":\"$CLAVE\"}" | campo token)
if [ -z "$TOKEN" ]; then
  echo "  FALLA: el login no devolvio token; sin eso no se puede seguir"
  exit 1
fi

# --- 3. Autorizacion --------------------------------------------------------

echo
echo "-- autorizacion --"
verificar 401 "sin token no se ven pedidos"     curl "$BASE/api/pedidos"
verificar 401 "un token inventado no sirve"     curl "$BASE/api/pedidos" -H "Authorization: Bearer no.es.un.token"
verificar 200 "con token si"                    curl "$BASE/api/pedidos" -H "Authorization: Bearer $TOKEN"

# --- 4. Checkout: el corazon del dominio ------------------------------------

echo
echo "-- checkout --"
STOCK_INICIAL=$(stock)
echo "  stock de la variante $VARIANTE antes: $STOCK_INICIAL"

RESP=$(curl -s -X POST "$BASE/api/pedidos" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"items\":[{\"variante_id\":$VARIANTE,\"cantidad\":2}]}")
PEDIDO=$(echo "$RESP" | campo id)
TOTAL=$(echo "$RESP" | sed -n 's/.*"total":\([0-9.]*\).*/\1/p')
PRECIO=$(echo "$RESP" | sed -n 's/.*"precio_unitario":\([0-9.]*\).*/\1/p')

comparar "$((STOCK_INICIAL - 2))" "$(stock)" "el checkout descuenta el stock (regla 2)"
# Regla 3: subtotal 2 x precio, mas envio si no llega al umbral. Se calcula con
# los valores que devolvio la API, no con numeros fijos: el umbral y el costo de
# envio entran por configuracion y cambian por entorno.
ESPERADO=$(awk -v p="$PRECIO" -v u="$UMBRAL" -v e="$ENVIO" \
  'BEGIN{ s = 2*p; printf "%g", (s >= u) ? s : s + e }')
comparar "$ESPERADO" "$(awk -v t="$TOTAL" 'BEGIN{printf "%g", t}')" \
  "total = subtotal + envio (regla 3)"

verificar 409 "no se puede comprar mas que el stock (regla 1)" \
  curl -X POST "$BASE/api/pedidos" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"items\":[{\"variante_id\":$VARIANTE,\"cantidad\":9999}]}"

verificar 400 "carrito vacio rechazado (regla 8)" \
  curl -X POST "$BASE/api/pedidos" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"items":[]}'

verificar 404 "una variante inexistente da 404" \
  curl -X POST "$BASE/api/pedidos" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"items":[{"variante_id":999999,"cantidad":1}]}'

# --- 5. Regla 7: los pedidos son de su dueno --------------------------------

echo
echo "-- pedidos ajenos (regla 7) --"
INTRUSO="intruso-$(date +%s)@ejemplo.com"
curl -s -o /dev/null -X POST "$BASE/api/usuarios/registro" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$INTRUSO\",\"password\":\"$CLAVE\",\"nombre\":\"Juan\",\"apellido\":\"Gomez\"}"
TOKEN_INTRUSO=$(curl -s -X POST "$BASE/api/usuarios/login" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$INTRUSO\",\"password\":\"$CLAVE\"}" | campo token)

# 404 y no 403: contestar "prohibido" le confirmaria que ese pedido existe.
verificar 404 "otro usuario no puede cancelar mi pedido" \
  curl -X POST "$BASE/api/pedidos/$PEDIDO/cancelar" -H "Authorization: Bearer $TOKEN_INTRUSO"
verificar 403 "un cliente no puede cambiar estados" \
  curl -X PATCH "$BASE/api/pedidos/$PEDIDO/estado" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"estado":"pagado"}'

# --- 6. Cancelacion: el stock vuelve ----------------------------------------

echo
echo "-- cancelacion --"
verificar 204 "cancelar el pedido propio" \
  curl -X POST "$BASE/api/pedidos/$PEDIDO/cancelar" -H "Authorization: Bearer $TOKEN"
comparar "$STOCK_INICIAL" "$(stock)" "cancelar devuelve el stock (regla 6)"
verificar 409 "un pedido cancelado no se cancela de nuevo (regla 5)" \
  curl -X POST "$BASE/api/pedidos/$PEDIDO/cancelar" -H "Authorization: Bearer $TOKEN"

# --- 7. Admin ---------------------------------------------------------------

echo
echo "-- admin --"
TOKEN_ADMIN=$(curl -s -X POST "$BASE/api/usuarios/login" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASS\"}" | campo token)
if [ -z "$TOKEN_ADMIN" ]; then
  echo "  AVISO: no se pudo loguear como admin ($ADMIN_EMAIL); se saltean sus pruebas."
  echo "         Corre el seed o pasa ADMIN_EMAIL y ADMIN_PASS por entorno."
else
  NUEVO=$(curl -s -X POST "$BASE/api/productos" \
    -H "Authorization: Bearer $TOKEN_ADMIN" -H 'Content-Type: application/json' \
    -d '{"nombre":"Producto de prueba (smoke-test)","precio":45000,"categoria":"pruebas",
         "variantes":[{"talle":"m","color":"Negro","stock":3}]}')
  NUEVO_ID=$(echo "$NUEVO" | campo id)
  comparar "1" "$([ -n "$NUEVO_ID" ] && echo 1 || echo 0)" "el admin puede crear un producto"

  verificar 400 "talle fuera de la lista rechazado (regla 8)" \
    curl -X POST "$BASE/api/productos" -H "Authorization: Bearer $TOKEN_ADMIN" -H 'Content-Type: application/json' \
    -d '{"nombre":"Malo","precio":1000,"variantes":[{"talle":"XXXL","color":"Negro","stock":1}]}'

  verificar 400 "precio cero rechazado (regla 8)" \
    curl -X POST "$BASE/api/productos" -H "Authorization: Bearer $TOKEN_ADMIN" -H 'Content-Type: application/json' \
    -d '{"nombre":"Malo","precio":0,"variantes":[{"talle":"M","color":"Negro","stock":1}]}'

  if [ -n "$NUEVO_ID" ]; then
    # El talle se guardo normalizado a "M", asi que mandar "m" tiene que chocar.
    verificar 409 "variante duplicada rechazada (regla 8)" \
      curl -X POST "$BASE/api/productos/$NUEVO_ID/variantes" -H "Authorization: Bearer $TOKEN_ADMIN" \
      -H 'Content-Type: application/json' -d '{"talle":"m","color":"negro","stock":5}'

    VAR_NUEVA=$(curl -s -X POST "$BASE/api/productos/$NUEVO_ID/variantes" \
      -H "Authorization: Bearer $TOKEN_ADMIN" -H 'Content-Type: application/json' \
      -d '{"talle":"L","color":"Beige","stock":5}' | campo id)
    comparar "1" "$([ -n "$VAR_NUEVA" ] && echo 1 || echo 0)" "una variante nueva si entra"

    # --- Subida de imagenes ----------------------------------------------
    #
    # Se prueba sobre el producto descartable: las fotos quedan colgadas de un
    # producto que se da de baja al final, asi que no ensucian el catalogo.

    verificar 401 "sin token no se puede subir una imagen" \
      curl -X POST "$BASE/api/productos/$NUEVO_ID/imagenes" -F "archivo=@$IMAGEN_OK"

    verificar 403 "un cliente no puede subir imagenes" \
      curl -X POST "$BASE/api/productos/$NUEVO_ID/imagenes" -H "Authorization: Bearer $TOKEN" \
      -F "archivo=@$IMAGEN_OK"

    verificar 201 "el admin sube la imagen principal" \
      curl -X POST "$BASE/api/productos/$NUEVO_ID/imagenes" -H "Authorization: Bearer $TOKEN_ADMIN" \
      -F "archivo=@$IMAGEN_OK" -F "color=Negro" -F "alt_text=Foto de prueba"

    verificar 409 "una segunda principal del mismo color da 409" \
      curl -X POST "$BASE/api/productos/$NUEVO_ID/imagenes" -H "Authorization: Bearer $TOKEN_ADMIN" \
      -F "archivo=@$IMAGEN_OK" -F "color=Negro"

    verificar 400 "una extension no permitida se rechaza" \
      curl -X POST "$BASE/api/productos/$NUEVO_ID/imagenes" -H "Authorization: Bearer $TOKEN_ADMIN" \
      -F "archivo=@$IMAGEN_TEXTO"

    # El .jpg renombrado: pasa el filtro de extension y lo frena la firma.
    verificar 400 "un archivo que no es imagen se rechaza por su contenido" \
      curl -X POST "$BASE/api/productos/$NUEVO_ID/imagenes" -H "Authorization: Bearer $TOKEN_ADMIN" \
      -F "archivo=@$IMAGEN_FALSA"

    verificar 404 "subir a un producto inexistente da 404" \
      curl -X POST "$BASE/api/productos/999999/imagenes" -H "Authorization: Bearer $TOKEN_ADMIN" \
      -F "archivo=@$IMAGEN_OK"

    # La prueba de punta a punta: la imagen que se subio tiene que poder
    # DESCARGARSE. Contra el 3000 esto ademas verifica el location /uploads/ de
    # nginx, que es donde se rompe si falta ese bloque.
    IMG_URL=$(curl -s -X POST "$BASE/api/productos/$NUEVO_ID/imagenes" \
      -H "Authorization: Bearer $TOKEN_ADMIN" \
      -F "archivo=@$IMAGEN_OK" -F "color=Beige" | campo url)
    comparar "1" "$([ -n "$IMG_URL" ] && echo 1 || echo 0)" "la subida devuelve la URL de la imagen"
    if [ -n "$IMG_URL" ]; then
      verificar 200 "la imagen subida se sirve" curl "$BASE$IMG_URL"
    fi

    # --- Transiciones de estado (regla 5) --------------------------------
    #
    # El pedido se hace sobre la variante RECIEN creada y no sobre la del
    # seed. No es un detalle: el ultimo paso deja el pedido en "enviado", que
    # ya no se puede cancelar, asi que ese stock no vuelve nunca. Sobre el
    # seed, cada corrida del script consumiria una unidad para siempre y a la
    # decima el script empezaria a fallar solo. Sobre un producto descartable
    # que se da de baja al final, no le cuesta nada a nadie.
    if [ -n "$VAR_NUEVA" ]; then
      OTRO=$(curl -s -X POST "$BASE/api/pedidos" -H "Authorization: Bearer $TOKEN" \
        -H 'Content-Type: application/json' \
        -d "{\"items\":[{\"variante_id\":$VAR_NUEVA,\"cantidad\":1}]}" | campo id)
      verificar 204 "pendiente -> pagado" \
        curl -X PATCH "$BASE/api/pedidos/$OTRO/estado" -H "Authorization: Bearer $TOKEN_ADMIN" \
        -H 'Content-Type: application/json' -d '{"estado":"pagado"}'
      verificar 409 "pagado -> entregado NO (no se saltea la cadena)" \
        curl -X PATCH "$BASE/api/pedidos/$OTRO/estado" -H "Authorization: Bearer $TOKEN_ADMIN" \
        -H 'Content-Type: application/json' -d '{"estado":"entregado"}'
      verificar 204 "pagado -> enviado" \
        curl -X PATCH "$BASE/api/pedidos/$OTRO/estado" -H "Authorization: Bearer $TOKEN_ADMIN" \
        -H 'Content-Type: application/json' -d '{"estado":"enviado"}'
      verificar 409 "un pedido enviado ya no se cancela (regla 5)" \
        curl -X POST "$BASE/api/pedidos/$OTRO/cancelar" -H "Authorization: Bearer $TOKEN"
      verificar 409 "entregado es terminal" \
        curl -X PATCH "$BASE/api/pedidos/$OTRO/estado" -H "Authorization: Bearer $TOKEN_ADMIN" \
        -H 'Content-Type: application/json' -d '{"estado":"pendiente"}'
    fi

    # Se da de baja para no dejar basura VISIBLE en el catalogo. La fila queda
    # en la base con activo = false, que es justo lo que hay que poder mostrar:
    # dar de baja un producto no borra los pedidos que lo incluyeron.
    verificar 200 "editar y dar de baja el producto de prueba" \
      curl -X PUT "$BASE/api/productos/$NUEVO_ID" -H "Authorization: Bearer $TOKEN_ADMIN" \
      -H 'Content-Type: application/json' \
      -d '{"nombre":"Producto de prueba (dado de baja)","precio":45000,"categoria":"pruebas","activo":false}'

    # Regla 9: dar de baja saca de la VENTA, no solo del listado. Las dos
    # verificaciones de abajo son el motivo por el que la regla vive en el
    # service: esconder el producto del catalogo no impide comprarlo con el id
    # de la variante, que es lo que hace el segundo curl.
    verificar 404 "el detalle publico de un producto de baja no existe"       curl "$BASE/api/productos/$NUEVO_ID"
    verificar 409 "no se puede comprar un producto dado de baja (regla 9)"       curl -X POST "$BASE/api/pedidos" -H "Authorization: Bearer $TOKEN"       -H 'Content-Type: application/json'       -d "{\"items\":[{\"variante_id\":$VAR_NUEVA,\"cantidad\":1}]}"

    # ...pero el ABM lo sigue viendo: si el filtro de "activo" estuviera dentro
    # de VerDetalle en vez de en VerDetallePublico, esto daria 404 y no habria
    # forma de reactivar ni corregir un producto dado de baja.
    verificar 201 "el admin todavia puede tocar un producto de baja"       curl -X POST "$BASE/api/productos/$NUEVO_ID/variantes"       -H "Authorization: Bearer $TOKEN_ADMIN" -H 'Content-Type: application/json'       -d '{"talle":"XXL","color":"Prueba","stock":1}'
  fi
fi

# --- Resultado --------------------------------------------------------------

echo
if [ "$fallos" -eq 0 ]; then
  echo "OK: $paso verificaciones, todas en verde."
  exit 0
fi
echo "FALLARON $fallos de $paso verificaciones."
exit 1
