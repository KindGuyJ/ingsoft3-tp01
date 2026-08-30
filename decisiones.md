# Decisiones — TP1

## 1. Por qué Git no pudo resolver el conflicto solo

Git fusiona automáticamente cuando los cambios de las dos ramas tocan **partes distintas** del
archivo: ahí no hay ambigüedad y puede aplicar los dos. Pero `feature/titulo-a` y
`feature/titulo-b` modificaron **la misma línea** del `README.md` (la línea 1, el título), cada una
con un texto distinto. Ahí Git se queda sin criterio: no entiende qué significa el contenido, así
que no puede saber si lo correcto es "versión A", "versión B" o una mezcla de las dos. No es una
limitación técnica ni una falla del algoritmo — es que **la decisión es de contenido, y esa la
tiene que tomar una persona**. Lo que hace Git es lo único razonable: frenar el merge, marcar el
archivo con `<<<<<<<`, `=======` y `>>>>>>>` para mostrar las dos versiones enfrentadas, y
delegarme la decisión a mí.

En mi caso resolví quedándome con la "versión A" (que era la que ya estaba integrada en `main`),
borré los marcadores y commiteé la resolución. Eso quedó en el commit de merge `300827b`.

**Qué habría tenido que pasar para que nunca apareciera:** que las dos ramas no vivieran en
paralelo sobre la misma línea. Concretamente, si después de mergear el PR de la rama A yo hubiera
actualizado `main` y recién ahí hubiera creado la rama B (o hubiera traído `main` a la rama B antes
de seguir editando), B habría partido de un `README.md` que ya decía "versión A" y no habría habido
nada que resolver. Esa es la razón práctica de por qué en la materia se usan **ramas cortas e
integración frecuente**: el conflicto no se elimina, pero mientras más corta es la rama, más chico
y más trivial es. Lo que sí sería una mala solución es volver al modelo de *lock* de los sistemas
centralizados (que un solo desarrollador pueda tocar un archivo a la vez): eso evita el conflicto
pero rompe el trabajo en paralelo, que es justamente lo que Git vino a habilitar.

## 2. Qué problemas encontré y cómo los solucioné

**El `.gitignore` estaba mal tipeado.** Lo creé como `.gigitnore` y no me di cuenta hasta que
revisé el repo terminado. Como el nombre estaba mal, Git nunca lo trató como archivo de exclusión:
el repositorio estaba, en los hechos, **sin `.gitignore`**, aunque el archivo se viera en la raíz.
Es un error silencioso — no da ningún error, simplemente no hace nada. Lo arreglé con
`git mv .gigitnore .gitignore`, y este mismo cambio entró por Pull Request como todo lo demás.

**Me olvidé de hacer `pull` después de mergear en la web.** El merge del PR ocurre en GitHub, no en
mi máquina, así que mi `main` local quedaba viejo y al crear la rama siguiente partía de un estado
desactualizado. Lo resolví incorporando `git switch main && git pull` como paso fijo antes de crear
cada rama nueva.

**No usé la misma estrategia de merge en todos los PRs.** Los primeros PRs los mergeé con el botón
por defecto (merge commit) en vez de *Squash and merge*, que es lo que indica la guía, así que el
historial de `main` quedó con commits de merge en vez de un commit por PR. No lo puedo cambiar
retroactivamente sin reescribir la historia (y eso sí sería peor), pero lo tengo identificado: se
ve en `git log --graph`, donde los primeros PRs forman "diamantes" y el último entra plano.

**Al armar el conflicto casi lo arruino.** Estuve a punto de crear la rama B estando parado en la
rama A. Si hubiera hecho eso, B habría heredado el cambio de A y no habría habido conflicto
ninguno. Volví a `main` antes de crear la segunda rama, que es exactamente la condición que
describo en el punto 1.

## 3. Declaración de uso de IA

- **Títulos y descripciones de los Pull Requests:** los generó automáticamente GitHub (Copilot) al
  mergear desde la web. Los leí antes de confirmar; en los PRs #2 y #3 quedaron con el título
  automático y sin descripción, lo cual reconozco que es una debilidad de la entrega.
- **Redacción de `evidencias.md`:** las descripciones de las cuatro capturas las escribí con
  asistencia de IA. Las verifiqué contrastando cada texto contra la captura correspondiente en
  [img/](img/) y contra lo que realmente había pasado en el repositorio.
- **Revisión final del repositorio y redacción de este archivo:** usé Claude para auditar el
  repositorio terminado contra el enunciado. Fue lo que detectó el `.gitignore` mal tipeado y la
  inconsistencia en la estrategia de merge. Verifiqué cada hallazgo por mi cuenta antes de
  aceptarlo: `git ls-files` me confirmó el nombre `.gigitnore`, y `git log --graph --oneline --all`
  me mostró los merge commits de los primeros PRs. El contenido de las respuestas es mío; la IA
  ayudó a ordenarlo y a encontrar los errores que yo no había visto.

---

# Decisiones — TP2

## 1. Qué es la aplicación y por qué elegí esta

Una tienda de indumentaria online: catálogo, carrito, checkout y panel de administración.
Es desarrollo propio — la estructura de carpetas y el patrón de capas los tomé de un
proyecto anterior de Arquitectura de Software II, pero el dominio, los modelos y las
reglas los escribí de cero para esta materia.

La elegí contra los criterios de la guía: la puedo ejecutar hoy (`docker compose up -d
--build` levanta los tres servicios desde cero), conozco sus comandos de build
(`go build ./cmd/api` y `npm run build`), la base se configura íntegramente por variables
de entorno, y el dominio de e-commerce da reglas de negocio con casos borde reales para
testear. Sobre todo: la entiendo lo suficiente para modificarla, que es la condición que
importa cuando el resto del semestre se construye encima.

**Las reglas de negocio que sostienen el sistema:**

| # | Regla |
|---|---|
| 1 | No se puede comprar más cantidad que el stock de la variante |
| 2 | El checkout descuenta el stock exactamente por lo comprado |
| 3 | Total = Σ(cantidad × precio); envío gratis desde un umbral configurable |
| 4 | `precio_unitario` es un snapshot: cambiar el precio no altera pedidos viejos |
| 5 | Transiciones válidas: `pendiente→pagado→enviado→entregado`; `cancelado` solo desde `pendiente`/`pagado` |
| 6 | Cancelar un pedido devuelve el stock |
| 7 | Un usuario solo ve y cancela sus propios pedidos |
| 8 | Email único; precio > 0; talle en lista permitida; carrito vacío rechazado |
| 9 | Un producto dado de baja no se vende: no se lista, su detalle público da 404 y el checkout lo rechaza |

## 2. La poda del stack: por qué no hay microservicios

El proyecto de referencia era una arquitectura de microservicios: tres APIs Go
independientes, MySQL, MongoDB, RabbitMQ, Solr y Memcached. Lo reduje a **un backend, una
base y un frontend**.

**Por qué:** cada servicio adicional se paga en todos los TPs siguientes — más pipelines
en el TP4, más entornos que aprobar en el TP6, más infraestructura que declarar en el TP8
y más superficie que asegurar en el TP9. La complejidad de microservicios no aporta nada
al objetivo de la materia, que es el sistema de entrega, no la arquitectura de la app.

**Qué perdí:** la búsqueda full-text de Solr y el caché de Memcached. Ninguno hace falta
para un catálogo de quince productos; un `WHERE categoria = ?` alcanza.

Tampoco hay gateway de pagos real: el checkout marca el pedido como pagado sin hablar con
nadie afuera. Es deliberado — el sistema tiene que levantar offline con un comando.

## 3. Las reglas de negocio viven en `services/`

Las dependencias apuntan siempre hacia adentro: los controllers parsean el request y
delegan, los services tienen las reglas, los repositories acceden a datos.

**La consecuencia práctica es la que importa:** los services no conocen ni Gin ni la base,
así que se testean con un repositorio falso en memoria, sin levantar MySQL. Eso es lo que
hace que los 63 tests del backend corran en 1,5 segundos, y lo que va a permitir que el
pipeline del TP4 los ejecute en cada push sin levantar infraestructura.

Un service nunca devuelve un `gin.H` ni un código HTTP: devuelve errores de dominio
tipados, y el controller los mapea a status codes.

## 4. Contenerización

- **Multi-stage:** `golang:1.24-alpine` compila, `alpine:3.20` ejecuta. El binario de Go
  es estático (`CGO_ENABLED=0`), así que la imagen final no necesita ni el toolchain ni
  librerías del sistema. El resultado pesa 37,5 MB contra los 395 MB del entorno de
  compilación.
- **Usuario no-root** en la imagen del backend: si alguien logra ejecutar algo dentro del
  contenedor, no lo hace como root. Cuesta dos líneas.
- **Los tests NO corren dentro del Dockerfile.** Van al pipeline del TP4. Meterlos en el
  build haría que un test roto rompa la construcción de la imagen, que es exactamente lo
  que el pipeline debería atrapar antes y con mejor diagnóstico.
- **Qué persiste:** dos volúmenes nombrados, `db_data` para MySQL y `uploads_data` para
  las imágenes que sube el admin.
- **Qué no persiste, a propósito:** el carrito (vive en el cliente) y todo lo escrito en
  la capa de escritura del contenedor.
- **`depends_on` con `condition: service_healthy`:** no alcanza con que el contenedor de
  MySQL haya arrancado; hay que esperar a que acepte conexiones. Sin eso el backend
  arranca antes que la base y falla de forma intermitente.

## 5. Cómo se hablan los servicios

El frontend llama a rutas relativas (`/api/...`), sin host ni puerto. Traduce el proxy de
Vite en desarrollo y nginx en el contenedor.

**Alternativa que descarté:** URLs absolutas inyectadas en build-time. Funciona, pero
obliga a reconstruir la imagen del frontend por cada entorno —lo que rompe la idea de
release inmutable— y agrega CORS al no ser same-origin.

Tres detalles del `nginx.conf` que costaron encontrar y que conviene no "mejorar":

- **El `proxy_pass` va sin barra final.** Con barra, nginx reescribe el prefijo y
  `/api/productos` llega al backend como `/productos` → 404 en todas las llamadas.
- **Un solo `resolver`: `127.0.0.11`**, el DNS de Docker. Agregar uno público "por las
  dudas" produce 502 intermitentes imposibles de diagnosticar.
- **`location /uploads/` además de `/api/`.** Si falta, las fotos subidas dan 404 mientras
  las estáticas andan perfecto — un síntoma que confunde muchísimo.

## 6. Imágenes de producto: dos vías que conviven

Las imágenes son una tabla propia colgada de `Producto`, con un campo `color` opcional.
**No cuelgan de `Variante`** porque las fotos varían por color, no por talle: colgarlas de
la variante duplicaría la misma foto en S/M/L/XL.

El campo `URL` acepta dos formas y por eso las dos vías conviven sin conflicto:

1. **Estáticas** (`/productos/xxx.jpg`): archivos commiteados en `frontend/public/`,
   servidos por nginx. Es lo que usa el seed. Funciona offline, cero infraestructura.
2. **Subidas** (`/uploads/xxx.jpg`): el admin sube, el backend guarda en `/app/uploads`,
   respaldado por el volumen `uploads_data`.

**Descarté:** guardarlas como BLOB en MySQL (infla la base, complica backups, no aporta
nada) y usar object storage externo tipo S3 o Cloudinary (es la dependencia de terceros
que la guía marca como riesgo).

## 7. La subida de imágenes: el service no conoce `multipart`

Es la decisión estructural de esta parte. El controller desarma el request y le pasa al
service datos planos más un `io.Reader`. Si el struct de entrada tuviera un
`*multipart.FileHeader`, `services/` pasaría a depender de `net/http` y dejaría de poder
testearse solo — que es exactamente lo que la arquitectura por capas trata de evitar.

El destino físico del archivo entra por una interfaz de una sola función,
`AlmacenImagenes.Guardar(nombre, contenido)`. El service no sabe si atrás hay un disco, un
volumen de Docker o un bucket. Hoy la implementa `repository.AlmacenDisco`; en los tests
se le pasa una en memoria, y por eso los 17 tests de imágenes no escriben un solo archivo.
**Esto es lo que deja preparado el TP6:** cuando haya que decidir qué pasa con las subidas
entre deploys, la respuesta es otra implementación de esta interfaz, no un refactor de las
reglas.

**Las reglas de la subida**, todas con su test: máximo 8 imágenes por producto; una sola
principal (`orden = 0`) por color; extensión en lista blanca; tamaño máximo configurable
por entorno; y **el contenido tiene que empezar con la firma del formato declarado**.

Esa última no estaba planeada. La agregué porque la lista de extensiones sola no alcanza:
la extensión la elige quien sube, así que un `.exe` renombrado a `.jpg` pasaba el filtro y
quedaba servido estáticamente. Los primeros bytes, en cambio, no se renombran — un JPEG
empieza con `FF D8 FF` y un PNG con `89 50 4E 47`. Se comparan contra la firma que
corresponde a la extensión declarada, así que un PNG subido como `.jpg` tampoco pasa.

**El nombre del archivo lo genera el backend**, nunca el cliente:
`p<productoID>-<16 hex aleatorios>.<ext>`. El nombre del cliente traería *path traversal*
(`../../etc/algo`) y colisiones entre dos admins que suben `foto.jpg`. Importa doble
porque el directorio se sirve entero de forma estática. Hay un test que sube
`../../etc/passwd.jpg` y verifica que el nombre guardado no tenga separadores.

**El orden de las dos escrituras:** primero el archivo, después la fila. Si falla la fila
queda un archivo huérfano, que es basura inofensiva; al revés quedaría una fila apuntando
a un 404, que el catálogo mostraría como imagen rota.

**`client_max_body_size` en nginx:** el default es 1 MB, así que sin tocarlo una foto de
celular de 3 MB muere en un `413` crudo *antes* de llegar al backend. Quedó en `10m`,
deliberadamente por encima del máximo del backend, para que el que corte sea el backend
con su mensaje explicado y no nginx.

## 8. Decisiones del backend que no se explican solas

**El service de usuarios no importa `middleware/`.** Necesita emitir un JWT en el login,
pero el generador vive en un paquete que importa Gin: si el service lo importara, el
paquete de reglas de negocio pasaría a depender del framework HTTP. En vez de eso recibe
una función, y `main.go` le pasa un closure que sí conoce el secreto y la duración. En los
tests se le pasa una que devuelve una cadena fija: los tests de usuarios corren sin JWT,
sin Gin y sin base.

**El login devuelve el mismo error para los tres rechazos.** Email inexistente, contraseña
equivocada y usuario dado de baja son todos `401 "email o contrasena incorrectos"`.
Distinguirlos convertiría la API en un verificador de qué emails están registrados. Cuando
el email no existe se hace igual una comparación bcrypt contra un hash descartable, para
que el tiempo de respuesta tampoco delate la diferencia. Hay un test que compara los dos
mensajes y falla si alguien los diferencia.

**El registro público nunca crea un admin.** El método fija `EsAdmin: false` y no acepta
ningún parámetro para cambiarlo. El admin lo crea el seed, que pasa por el mismo service
—mismo hash, mismas validaciones— y recién después hace un `UPDATE` explícito.

**El SKU se autogenera.** La columna tiene índice único: dos variantes sin SKU guardarían
la cadena vacía dos veces y la segunda chocaría. El síntoma era pésimo — el alta de un
producto fallaba recién en la segunda variante, con un error del driver.

**El stock se corrige por su propio endpoint**, no dentro del PUT del producto. Editar el
nombre o el precio no tiene por qué poder tocar el inventario de rebote. Además la
operación es distinta de la del checkout: el checkout **resta** lo comprado, ésta **fija**
un valor. Mezclarlas haría que un error de tipeo del admin se pareciera a una compra.

**La baja de productos es lógica** (`activo = false`), no un DELETE. Borrar la fila
rompería los `PedidoItem` que la referencian, y el historial de un pedido tiene que seguir
siendo legible aunque el producto ya no se venda. La consecuencia no obvia es que el panel
de admin **no puede usar el listado público**, que filtra los inactivos: un producto dado
de baja desaparecería también del panel y la baja sería irreversible desde la UI. Por eso
hay una ruta aparte que devuelve todo. Es ruta aparte y no un parámetro sobre la pública
para que el catálogo que ve cualquiera no tenga ninguna forma de devolver productos de
baja, ni por un error de tipeo.

**El detalle público es un método aparte, no un filtro adentro del detalle general.** Lo
intuitivo —filtrar los inactivos en un solo método— está mal, y el motivo no se ve hasta
que se rompe: la edición y el alta de variantes llaman a ese mismo método. Si filtrara
ahí, dar de baja un producto lo volvería ineditable e irreactivable desde el panel. Hay un
test escrito específicamente para que esa trampa no vuelva.

**La regla 9 se hace valer en el checkout, no solo escondiendo el producto.** Durante un
rato la baja fue nada más que un filtro del listado, y eso dejaba dos agujeros: con el
link guardado el producto se seguía viendo, y con un POST directo se seguía comprando.
**Es 409 y no 404 porque la variante existe**: lo que rechaza la compra es el estado del
producto, igual que en "stock insuficiente".

Un efecto lateral que vale la pena contar: el fixture de los tests de pedidos tenía el
producto sin `Activo`, o sea `false` por cero-value, y al agregar la regla fallaron todos
los tests de compra a la vez. Un cero-value que significaba "dado de baja" era una bomba
esperando.

**El carrito no está en la base.** Vive en el cliente y se valida entero contra el backend
recién en el checkout. Menos tablas, menos endpoints, y el checkout queda como una
operación transaccional limpia.

## 9. Decisiones del frontend

**No agregué ninguna dependencia.** El front es React + React Router y CSS escrito a mano.
Evalué y descarté un framework de estilos y una librería de estado: para seis pantallas no
compensan, y cada dependencia es algo más que instalar, auditar en el TP9 y explicar en la
defensa. El CSS entero son 220 líneas en un solo archivo.

**Al checkout se manda solo `variante_id` y `cantidad`.** El precio lo pone el backend. Si
el front mandara el precio, cualquiera con la consola abierta compraría a $1. Por eso el
total del carrito se muestra como *"total estimado"*: el que queda guardado es el que
calculó el backend con su propia configuración.

**El rol de admin en el front es cosmético, y eso es a propósito.** `es_admin` se guarda
en `localStorage` y solo decide si se dibuja el menú "Admin". Quien edite `localStorage`
ve el menú y no consigue nada: el rol real viaja firmado dentro del JWT y lo verifica el
middleware. Es la respuesta a "¿y si alguien se pone admin en el navegador?".

**Las reglas se importan de `utils.js`, no se reescriben en el componente.** Si el botón
"Agregar al carrito" tuviera su propia condición adentro del JSX, el test verde no
probaría nada sobre la pantalla.

**El filtro por categoría lo resuelve el backend, pero la lista de categorías se pide
aparte.** Si la lista saliera del resultado ya filtrado, al elegir "remeras" desaparecerían
todas las demás opciones y no habría cómo volver. La categoría elegida vive en la URL y no
en un `useState`, así el filtro se comparte por link y el botón "atrás" hace lo esperado.

**Los talles agotados se muestran deshabilitados, no ocultos.** Que el talle exista pero
esté sin stock es información para quien compra.

**Un 401 del backend limpia la sesión guardada.** Las pantallas autenticadas distinguen
"el token venció" (se borra la sesión y se va al login, recordando a dónde iba) de una
regla de negocio rechazada (409), que se muestra tal como la explicó el backend.

## 10. Publicación en el registry

Las imágenes se publicaron en `ghcr.io/kindguyj/tienda-backend:v2.0.0` y
`ghcr.io/kindguyj/tienda-frontend:v2.0.0`.

**`docker tag` no copia la imagen, le pone un segundo nombre.** El nombre completo es lo
que le dice a `docker push` a dónde subir: registry, dueño, paquete y versión. La imagen
local se llama `tienda-indumentaria-backend`, sin registry adentro, así que hay que darle
un segundo nombre que lleve el destino escrito. Las dos etiquetas quedan apuntando al
mismo ID y el disco no crece.

**El nombre del paquete en GHCR no tiene que coincidir con el del repositorio.** La ruta
es `ghcr.io/<dueño>/<paquete>`: el dueño es fijo (en minúsculas, que es lo que exige
Docker en los tags), el paquete lo elijo yo.

**`docker-compose.registry.yml` es un proyecto de Compose aparte**
(`name: tienda-indumentaria-registry`), distinto del de build. Si compartieran nombre
compartirían volúmenes, y el stack del registry arrancaría con la base ya sembrada por el
de build: la prueba dejaría de demostrar lo que tiene que demostrar. Con proyecto propio,
`up -d` baja las imágenes y arranca contra una base vacía, así que si algo faltara dentro
de la imagen publicada se vería ahí. El costo asumido es que los dos stacks publican los
mismos puertos y no pueden convivir: hay que bajar uno antes de levantar el otro.

**Un detalle que engaña:** `docker manifest inspect` responde OK con la sesión abierta
aunque el paquete sea privado. Para verificar de verdad que son públicos hay que pedir un
token anónimo al registry, que es como lo verifiqué.

## 11. Limitaciones conocidas

Las listo a propósito: son deuda técnica asumida, no cosas que se me pasaron.

1. **El checkout no es transaccional.** Si falla el descuento de stock a mitad de camino,
   el pedido queda creado con stock inconsistente. Lo mismo el alta de producto, que
   inserta el producto y después sus variantes (el SKU generado necesita el ID, que recién
   existe tras el INSERT).
2. **`AutoMigrate` no versiona el schema.** Sirve para desarrollo; para QA y PROD haría
   falta una herramienta de migraciones de verdad.
3. **El volumen de uploads no viaja a QA/PROD.** En el TP6 habrá que decidir entre object
   storage o aceptar que las imágenes subidas se pierden entre deploys. La interfaz
   `AlmacenImagenes` está puesta justamente para que esa decisión no toque las reglas.
4. **El umbral de envío gratis está escrito en dos lados:** el backend lo lee de una
   variable de entorno y el front lo tiene como valor por defecto. Si se cambia la
   variable, el carrito muestra un total distinto al que el backend termina cobrando. Hoy
   se banca porque el total del carrito está rotulado como *estimado* y el pedido
   confirmado muestra el del backend; la solución sería exponer la configuración en un
   endpoint público.

## 12. Problemas que encontré y cómo los resolví

**GORM reescribe las asociaciones precargadas.** Un `Save` sobre un producto leído con
`Preload("Variantes")` vuelve a escribir también las variantes. Como el producto se lee un
instante antes de guardarlo, editar el nombre podía pisar el stock con valores viejos. Lo
resolví con `Omit("Variantes", "Imagenes")`, y quedó comentado en el código porque es un
comportamiento que no se ve leyendo el `Save`.

**`binding:"required"` sobre un `float64` rechaza el 0 con un mensaje confuso.** El
validador de Gin dice "failed on the required tag" cuando el problema real es el valor. Lo
cambié a `binding:"gt=0"`, que dice lo que pasa.

**Un `<input type="number">` recortado al stock en cada tecla no se puede usar.** El
carrito tenía un campo de cantidad que en cada `onChange` hacía `min(valor, stock)`. Con
stock 5 y cantidad 1, borrar y escribir "3" daba 5: al borrar, el estado volvía a 1 —el
campo nunca quedaba vacío— y la tecla siguiente componía "13", que recortado da 5. Lo
encontró el test de "recalcula el total al cambiar la cantidad", que esperaba 35000 y
recibió 50000. Lo reemplacé por un selector de − / +, así el valor siempre está entre 1 y
el stock y no existe el estado intermedio inválido. **Vale como ejemplo de test que
encuentra un problema real de usabilidad, no un bug de cálculo:** el arreglo fue cambiar
el control, no aflojar el test.

**Git Bash traduce las rutas absolutas antes de pasárselas a Docker.**
`docker compose exec backend /bin/seed` falla con `no such file or directory` porque Git
Bash convierte `/bin/seed` en una ruta de Windows antes de que Docker la vea. Se resuelve
anteponiendo `MSYS_NO_PATHCONV=1`. Lo dejé anotado en el README porque es el tipo de error
que hace perder media hora buscando en el lugar equivocado.

**Con el image store de containerd, la imagen de la etapa de build no queda listada.** Al
sacar la evidencia de tamaños los comandos salían vacíos sin que nada fallara, con lo cual
la evidencia quedaba en blanco en silencio. Hay que traer las bases con `docker pull`
antes de medir.

**Medí el tamaño sobre la imagen equivocada.** Registré 26,3 MB para el backend, pero ese
número era de una imagen vieja, de antes de ponerle nombre fijo al proyecto de Compose y
de antes de la subida de imágenes. El número real es 37,5 MB. Lo detecté al completar las
evidencias y lo corregí ahí.

**Los fine-grained PAT no funcionan con GHCR.** El `docker login` dice `Succeeded` y el
push falla igual con `denied`. Hay que usar un PAT clásico con `write:packages`.

## 13. Declaración de uso de IA

Usé **Claude Code** como asistente durante todo el TP2. Las decisiones de arquitectura
—las capas, la poda del stack, qué persiste, las reglas de negocio— las fijé yo antes de
generar código, y el código generado las sigue.

**Qué generé con asistencia:** los services de usuarios, productos, pedidos e imágenes;
los controllers y el mapeo entre entidades y DTOs; el cableado de rutas; el seed; las
pantallas del frontend con sus componentes y contextos; los Dockerfiles y el compose; el
`smoke-test.sh`; y los tests de todo lo anterior.

**Cómo lo verifiqué** — no alcanza con que compile:

1. `go build ./...`, `go vet ./...` y `gofmt -l` limpios; `npm run build` limpio.
2. **63 tests de servicios en el backend y 35 en el frontend**, todos en verde. Varios
   están escritos para fallar si alguien "simplifica" la regla: por ejemplo el que compara
   los dos mensajes de error del login, o el que verifica que un producto dado de baja
   siga siendo editable.
3. **Prueba completa contra el stack en Docker**, no solo con tests unitarios: registro →
   login → checkout → listar pedidos → cancelar, más el ABM de admin, las transiciones de
   estado y la subida de imágenes. Quedó guionada en `app/smoke-test.sh` para que sea
   reproducible y no un comando suelto en el historial: **48 verificaciones**, tanto
   contra la API directa como a través de nginx, que es el camino real del navegador.
   Devuelve 0 o 1, así que en el TP4 se enchufa al pipeline sin reescribirlo.
4. Los caminos de error uno por uno: 401 sin token, 403 de cliente sobre endpoint de
   admin, 404 al cancelar el pedido de otro, 409 por stock insuficiente y por transición
   inválida, 400 por talle fuera de la lista.
5. Reconstrucción desde cero (`down -v` → `up --build` → seed) para confirmar que el
   camino que documenta el README funciona en una máquina limpia.
6. El sistema levantado **desde las imágenes publicadas**, sin construir y sin
   credenciales: las 48 verificaciones del smoke test pasan igual.

Un test falló la primera vez y destapó un problema real de usabilidad en el campo de
cantidad del carrito (ver punto 12). El arreglo fue cambiar el control, no aflojar el test.

**Las imágenes de terminal de `evidencias.md`** las generé corriendo los comandos de
verdad, guardando su salida literal a archivo y renderizándola como imagen. El texto no
está editado; está aclarado en una nota al pie de ese archivo.

**Qué tengo que poder explicar en la defensa:** por qué el service recibe una función en
vez de importar el paquete del middleware; por qué el login no distingue los motivos de
rechazo; qué hace el `Omit` de GORM y por qué; por qué el seed es un binario aparte que
pasa por el service; por qué al checkout se manda el id de la variante y no el precio; por
qué el `es_admin` del front no es un control de acceso; por qué la regla 9 devuelve 409 y
no 404; y por qué el nombre del proyecto de Compose determina qué volúmenes se usan.

---

# Decisiones — TP3

> En este práctico no hay `evidencias.md`: el proyecto es público
> (https://github.com/users/KindGuyJ/projects/1) y quien corrige ve en vivo la jerarquía,
> el sprint, el límite de trabajo en progreso y el issue cerrado por el pull request.
> Sacar capturas de lo mismo sería duplicar lo que ya se ve.

## 1. La duración del sprint: 14 días

Sprint 1 arranca el 30/08/2026 y cierra el 13/09/2026.

**Por qué dos semanas y no una.** La unidad de trabajo real de esta materia es el trabajo
práctico, y los prácticos no se entregan semanalmente. Un sprint de una semana me obligaría
a cerrar y replanificar en la mitad de un TP, con una ceremonia que no cambia ninguna
decisión: el trabajo pendiente es el mismo antes y después. Dos semanas es la ventana más
corta que contiene un arco de trabajo completo — en este caso la historia de CI, que empieza
en el TP3 y termina en el TP4.

**Por qué no un mes.** Un sprint largo esconde el atraso: si algo se traba, me entero recién
al final, cuando ya no queda margen. La gracia de la iteración corta es que el error se
descubre temprano y barato.

**Sobre la fecha de entrega (02/09/2026).** No hay ninguna duración de sprint que haga que
la entrega caiga justo en un cierre: entrego en el día 4 de 14. Y está bien que así sea —
un sprint no es un plazo de entrega, es una cadencia de planificación. La consecuencia
práctica es visible en el tablero: la historia y una de sus dos tareas se entregan
**abiertas**, porque el trabajo sigue en el TP4. Sería peor forzar un sprint de tres días
para que "cierre justo": tendría un cierre prolijo y ninguna utilidad.

## 2. El límite de trabajo en progreso: 2

**Por qué 2.** La regla de arranque es *cantidad de personas + 1*. Trabajando sola, eso da
dos. El «más uno» es la válvula para cuando algo queda esperando por afuera —una revisión,
una corrida de CI que tarda, una respuesta— y necesito avanzar en otra cosa sin dejar el día
muerto.

**Qué me haría subirlo.** Que el segundo espacio se llene de forma sistemática con trabajo
genuinamente bloqueado y no por elección mía. O sumar una persona al equipo: ahí el número
pasa a tres por la misma regla.

**Qué señal me diría que quedó demasiado alto.** Que nunca lo alcance. Un límite que no se
toca nunca no está limitando nada: es decoración. La señal contraria —chocarlo seguido— no
es mala, es el sistema funcionando: me está diciendo que termine algo antes de empezar otra
cosa.

**Qué pasa si lo subo a diez.** Deja de ser un límite. Trabajando sola nunca voy a tener diez
cosas en curso, así que la columna jamás se pondría en rojo y el mecanismo quedaría
desactivado sin que se note. Es la peor forma de romperlo: no falla, simplemente no hace
nada. Y el costo de no tenerlo es concreto — con muchas tareas empezadas y ninguna terminada,
el trabajo en progreso no entrega valor, solo acumula contexto que hay que recordar.

## 3. Diagnóstico de la historia mal escrita

La historia del ejercicio era:

> *Como desarrollador quiero crear la tabla usuarios para guardar los datos.*

**Por qué está mal escrita.** Es una **tarea disfrazada de historia**. Tiene la forma
—*Como… quiero… para…*— pero no la sustancia: el beneficiario es el desarrollador, no
alguien que recibe valor del producto; lo que pide es un paso técnico de implementación; y
el «para» no nombra ningún beneficio, apenas repite el mismo acto técnico con otras palabras
("crear la tabla… para guardar los datos"). Como consecuencia, no hay nada que un tercero
pueda verificar: no se puede escribir un criterio de aceptación sobre "la tabla existe" que
diga algo sobre el producto.

**Cómo la reescribiría**, sobre mi app:

> *Como clienta quiero registrarme con mi email y una contraseña para poder ver el historial
> de mis pedidos.*
>
> Criterios: el registro rechaza un email ya usado · el registro rechaza un email con formato
> inválido antes de enviar · después de registrarme quedo con sesión iniciada · desde mi
> cuenta veo únicamente mis pedidos.

Crear la tabla `usuarios` sigue existiendo, pero en el lugar que le corresponde: como
**tarea** colgando de esa historia. Ésa es la prueba de que el original era una tarea — al
reescribirlo bien, el enunciado viejo reaparece un nivel más abajo.

## 4. Problemas encontrados y cómo los resolví

**El scope `project` no viene con `gh auth login`.** La autenticación normal deja `repo`,
`workflow` y `gist`, pero cualquier comando `gh project` falla por permisos. Se agrega
aparte con `gh auth refresh -s project`.

**Escribir «Parte de #8» en el cuerpo de un issue NO crea jerarquía.** Éste es el problema
que más me habría costado si no lo verificaba: los cinco issues estaban creados, con sus
labels correctas, y el cuerpo de cada tarea decía "Parte de #8". Todo *parecía* bien. Pero
eso genera solamente una **mención** — un link en el historial del issue — y no la relación
padre-hijo. Consultando la API quedó a la vista: la épica devolvía `subIssues: ninguna`. Se
resuelve enlazando explícitamente con `gh issue edit 7 --add-sub-issue 8` (necesita `gh`
2.94 o superior). Es la misma trampa que las task-lists (`- [ ] #12`), que el enunciado
rechaza por lo mismo: se ven parecido y no son navegables.

**Crear el proyecto con `gh project create` deja el tablero vacío.** El comando no elige
ningún repositorio, así que no queda configurada la automatización *Auto-add to project* que
sí arma la creación desde la web. Los items se agregan con `gh project item-add`.

**Los Projects de usuario nacen privados, y el entregable es la URL.** Entregada así, quien
la abre recibe un 404 — ni siquiera un "no tenés permiso". Se corrige con
`gh project edit 1 --owner "@me" --visibility PUBLIC`, y se comprueba abriendo la propia URL
en una ventana de incógnito.

**Crear el campo Sprint no asigna a nadie al sprint.** El campo de tipo Iteration se
configura una vez y genera las iteraciones hacia adelante, pero los items quedan en blanco:
la historia y las dos tareas seguían sin sprint. Hay que asignarlas explícitamente, una por
una.

## 5. Declaración de uso de IA

Usé **Claude Code** para armar la secuencia de comandos `gh` de este práctico, para redactar
el cuerpo del bug sobre mi propia app, y para ordenar este documento.

**Las decisiones que se evalúan las tomé yo:** la duración del sprint, el número del límite
de trabajo en progreso y el diagnóstico de la historia mal escrita. La épica, la historia,
sus criterios y sus dos tareas están dictadas por el enunciado y se reprodujeron tal cual.

**Cómo lo verifiqué — y por qué importa acá en particular.** No alcanzaba con que los
comandos no dieran error: los cinco issues se habían creado bien y el resultado *parecía*
correcto, pero la jerarquía no existía. La verificación fue consultar la API de GitHub y
comparar contra lo que pide el enunciado, punto por punto:

- que la épica tenga a la historia como sub-issue, y la historia a sus dos tareas;
- que el bug **no** tenga padre;
- que la historia tenga sus cuatro criterios y la épica ninguno;
- que el proyecto sea público y tenga los cinco items;
- que el campo Sprint sea de tipo Iteration y que la historia y sus tareas estén en Sprint 1;
- que la protección de `main` siga exigiendo pull request.

Ese control encontró dos cosas que faltaban —la jerarquía sin enlazar y el sprint sin
asignar— y las dos están descritas arriba. El límite de trabajo en progreso y la
automatización *Item closed → Done* no se pueden consultar por API: ésos los verifiqué a
mano en el tablero.

**Qué tengo que poder explicar en la defensa:** por qué elegí catorce días y qué pasaría con
uno de tres; por qué el límite es dos y qué me haría cambiarlo; por qué cada criterio de
aceptación de mi historia es verificable y "que el CI funcione bien" no lo sería; por qué el
bug va al costado y no colgando de la historia; y por qué un `Closes #N` tiene que apuntar al
número de la tarea y no al de la historia.
