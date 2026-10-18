#!/usr/bin/env bash
#
# Smoke test post-deploy: ¿el entorno esta vivo? Lo usan deploy-qa y
# deploy-prod con las URLs de SU entorno, asi que QA y PROD se verifican con
# exactamente la misma prueba.
#
# USO: URL_API=https://... URL_FRONT=https://... [COMMIT_ESPERADO=<sha>] smoke-entorno.sh
#
# Comprueba cuatro cosas, y si cualquiera falla vuelve a intentar:
#   1. API /healthz          el proceso esta vivo
#   2. API /api/productos    la api llega a SU base (un /healthz solo daria
#                            verde con la conexion a la base rota)
#   3. front /               nginx sirve la SPA
#   4. front /api/productos  el front llega a SU api: prueba el BACKEND_URL de
#                            la plantilla de nginx
#
# Y si viene COMMIT_ESPERADO, que los DOS servicios sean la imagen de ese
# commit: la api lo informa en /healthz y el front en /version.txt (los dos lo
# traen horneado desde el build: ARG COMMIT en cada Dockerfile). Sin esto, el
# smoke podia dar verde contra la version VIEJA: Render la sigue sirviendo
# mientras levanta la nueva (y si el deploy falla). Con esto, el verde llega
# recien cuando contesta lo que se acaba de desplegar.

set -u

: "${URL_API:?falta URL_API}"
: "${URL_FRONT:?falta URL_FRONT}"
COMMIT_ESPERADO="${COMMIT_ESPERADO:-}"

# 30 vueltas x 20 s = 10 minutos: bajar la imagen y levantarla (desde el TP7
# Render ya no construye) mas el cold start del plan gratuito (hasta ~1 min).
# Un curl seco daria rojos falsos.
INTENTOS=30
ESPERA=20

# ¿La api y el front son la imagen del commit esperado? Si no se pidio
# ninguno, no se compara.
commit_ok() {
  [ -z "$COMMIT_ESPERADO" ] && return 0
  curl -fsS --max-time 10 "$URL_API/healthz" | grep -q "\"commit\":\"$COMMIT_ESPERADO\"" \
    && [ "$(curl -fsS --max-time 10 "$URL_FRONT/version.txt")" = "$COMMIT_ESPERADO" ]
}

for i in $(seq 1 "$INTENTOS"); do
  # --max-time en CADA curl: un servicio que se esta despertando acepta la
  # conexion y no contesta, y sin tope curl se quedaria esperando hasta que
  # el runner mate el job, con un error que no dice por que.
  if curl -fsS --max-time 10 "$URL_API/healthz" > /dev/null \
     && curl -fsS --max-time 10 "$URL_API/api/productos" > /dev/null \
     && curl -fsS --max-time 10 "$URL_FRONT/" > /dev/null \
     && curl -fsS --max-time 10 "$URL_FRONT/api/productos" > /dev/null \
     && commit_ok; then
    echo "✅ El entorno responde (api + base + front + front→api${COMMIT_ESPERADO:+ + api y front en $COMMIT_ESPERADO}), intento $i"
    exit 0
  fi
  echo "… intento $i/$INTENTOS sin respuesta completa, espero ${ESPERA}s"
  # Mientras Render levanta la imagen nueva, esto muestra la VIEJA contestando.
  if [ -n "$COMMIT_ESPERADO" ]; then
    echo "  api /healthz contesta:     $(curl -fsS --max-time 10 "$URL_API/healthz" 2>&1 || true)"
    # head -c: un front viejo no tiene version.txt, y nginx contesta el
    # index.html de la SPA (try_files); alcanza con ver que no es un commit.
    echo "  front /version.txt contesta: $(curl -fsS --max-time 10 "$URL_FRONT/version.txt" 2>&1 | head -c 60 || true)"
  fi
  sleep "$ESPERA"
done

echo "❌ El entorno no respondio en $((INTENTOS * ESPERA / 60)) minutos"
exit 1
