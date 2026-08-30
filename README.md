# Tienda de Indumentaria — Ingeniería del Software 3

[![CI](https://github.com/KindGuyJ/ingsoft3-tp01/actions/workflows/ci.yml/badge.svg)](https://github.com/KindGuyJ/ingsoft3-tp01/actions/workflows/ci.yml)

Aplicación full-stack de e-commerce de indumentaria, y repositorio de **todos los trabajos
prácticos** de la materia (UCC, 2026). La aplicación vive en [`app/`](app/); los documentos
del trabajo práctico están acá, en la raíz, porque son de toda la cursada y no de una sola
entrega.

| Capa | Tecnología |
|---|---|
| Backend | Go 1.24 + Gin + GORM |
| Base de datos | MySQL 8.4 |
| Frontend | React + Vite, servido por nginx |
| Orquestación | Docker Compose |
| CI | GitHub Actions |

## Levantar el sistema desde cero

Requisitos: Docker y Docker Compose (`docker compose version`).

```bash
git clone https://github.com/KindGuyJ/ingsoft3-tp01.git
cd ingsoft3-tp01/app

# 1. El secreto NO viaja en el repo: hay que crearlo.
cp .env.example .env
#    Editá .env y poné una contraseña de base y un JWT_SECRET reales.
#    Para el secreto: openssl rand -hex 64

# 2. Levantar todo.
docker compose up -d --build

# 3. Cargar el catálogo de ejemplo (si no, la tienda arranca vacía).
docker compose exec backend /bin/seed
```

> Desde **Git Bash** en Windows, el tercer comando necesita `MSYS_NO_PATHCONV=1`
> adelante: si no, `/bin/seed` se traduce a una ruta de Windows y el exec falla.

El seed carga 15 productos con sus variantes y crea el usuario administrador
`admin@tienda.local` / `admin12345` (cambialo fuera de desarrollo). Es idempotente:
si ya hay productos, no hace nada. Con `-reset` borra el catálogo y lo recarga.

El `cp .env.example .env` no es opcional ni se hace solo: el `.env` está en
`.gitignore` a propósito, es lo único que no puede viajar en el repositorio.

Esperá a que la base pase a `healthy`:

```bash
docker compose ps
```

| Servicio | URL |
|---|---|
| Frontend | http://localhost:3000 |
| API | http://localhost:8080 |
| Healthcheck | http://localhost:8080/healthz |

## API

| Método | Ruta | Quién |
|---|---|---|
| `GET` | `/healthz` | público |
| `GET` | `/api/productos?categoria=` | público |
| `GET` | `/api/productos/:id` | público |
| `POST` | `/api/usuarios/registro` | público |
| `POST` | `/api/usuarios/login` | público |
| `POST` | `/api/pedidos` | autenticado — el checkout |
| `GET` | `/api/pedidos` | autenticado — solo los propios |
| `POST` | `/api/pedidos/:id/cancelar` | autenticado — devuelve el stock |
| `POST` | `/api/productos` | admin |
| `PUT` | `/api/productos/:id` | admin |
| `POST` | `/api/productos/:id/variantes` | admin |
| `POST` | `/api/productos/:id/imagenes` | admin — **multipart**, ver abajo |
| `PATCH` | `/api/pedidos/:id/estado` | admin |

El token va en `Authorization: Bearer <token>` y sale del login.

### Subida de imágenes

Es el único endpoint que no recibe JSON: llega un `multipart/form-data` con el
archivo en el campo `archivo` y el resto como campos de texto (`color`,
`alt_text`, `orden`).

```bash
curl -X POST http://localhost:3000/api/productos/1/imagenes \
  -H "Authorization: Bearer $TOKEN" \
  -F "archivo=@foto.jpg" -F "color=Negro" -F "orden=0"
```

Devuelve `201` con la URL relativa (`/uploads/p1-a91f….jpg`), que es la que hay
que usar en el `<img>`. El archivo queda en el volumen `uploads_data`.

Reglas que aplica: `.jpg`/`.jpeg`/`.png` únicamente, el contenido tiene que ser
realmente una imagen de esa extensión, máximo `MAX_IMAGEN_MB` (5 por defecto),
hasta 8 imágenes por producto y una sola principal (`orden = 0`) por color.

> Si subís `MAX_IMAGEN_MB`, subí también `client_max_body_size` en
> [`app/frontend/nginx.conf`](app/frontend/nginx.conf): tiene que quedar por encima, o
> nginx corta la subida con un `413` crudo antes de que el backend pueda explicar el error.

## Levantar desde el registry (sin construir)

```bash
cd app
cp .env.example .env
docker compose down                                   # los puertos son los mismos
docker compose -f docker-compose.registry.yml up -d
```

Descarga las imágenes publicadas en ghcr.io en vez de construirlas.

Es un **proyecto de Compose aparte** (`tienda-indumentaria-registry`), con sus
propios volúmenes: arranca con la base vacía, sin los datos del stack de build.
Para sembrarla: `docker compose -f docker-compose.registry.yml exec backend /bin/seed`.

## Desarrollo local (sin contenedores)

```bash
# Base de datos como contenedor suelto
docker run -d --name mysql-tienda -e MYSQL_ROOT_PASSWORD=root \
  -e MYSQL_DATABASE=tienda -p 3306:3306 mysql:8.4

# Backend → http://localhost:8080
cd app/backend && DB_HOST=localhost JWT_SECRET=dev go run ./cmd/api

# Frontend → http://localhost:5173 (Vite proxea /api al backend)
cd app/frontend && npm install && npm run dev
```

## Tests

```bash
cd app/backend && go test ./internal/services/... -v
cd app/frontend && npm run test
```

Los tests del backend no necesitan base de datos: los services solo conocen
interfaces, así que corren contra un repositorio fake en memoria.

### Prueba de humo contra el sistema levantado

[`app/smoke-test.sh`](app/smoke-test.sh) recorre la API entera con `curl` — registro,
login, checkout, cancelación y el ABM del admin — verificando el código HTTP de cada paso
y el efecto sobre el stock. Prueba lo que los tests unitarios no pueden: que el ruteo, el
JWT, GORM y MySQL funcionan **juntos**.

```bash
cd app
docker compose up -d
MSYS_NO_PATHCONV=1 docker compose exec backend /bin/seed   # sin esto no hay catálogo
./smoke-test.sh
```

```
OK: 36 verificaciones, todas en verde.
```

Devuelve 0 si pasa todo y 1 si algo falla, así que sirve tal cual como gate de
un pipeline. Es repetible: no consume stock del seed y usa un email distinto en
cada corrida.

Para probar el camino real del navegador —el que pasa por el proxy de nginx en
vez de pegarle a la API directo— apuntalo al 3000:

```bash
BASE=http://localhost:3000 ./smoke-test.sh
```

Solo necesita `curl`, `sed` y `grep`; no usa `jq`.

## Integración continua

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) construye las dos imágenes en jobs
paralelos —`build-backend` y `build-frontend`— en cada Pull Request y en cada push a
`main`, usando **los mismos Dockerfiles** que se despliegan. No compila por su cuenta: no
hay una sola línea de Go ni de Node en el workflow.

`main` no acepta un merge si no se cumplen las dos condiciones: que el cambio venga por
Pull Request y que los dos checks estén en verde. El badge de arriba muestra el estado del
último build de `main` y lleva al historial de corridas.

## Estructura del repositorio

```
.github/workflows/ci.yml   el pipeline
decisiones.md              por qué se decidió cada cosa, TP por TP
evidencias.md              evidencias de funcionamiento
img/                       capturas
app/                       la aplicación
  docker-compose.yml
  docker-compose.registry.yml
  smoke-test.sh            prueba de humo de la API con curl
  backend/
    cmd/api/               punto de entrada
    cmd/seed/              carga inicial del catálogo (binario aparte)
    internal/
      controllers/         handlers HTTP (sin lógica de negocio)
      services/            ← las reglas de negocio, y sus tests
      repository/          acceso a datos (GORM)
      dao/                 entidades
      dto/                 contratos de la API
      middleware/          JWT y rol admin
      config/              variables de entorno
      errors/              errores de dominio tipados
  frontend/
    src/
      services/api.js      cliente HTTP (rutas relativas)
      utils.js             lógica pura testeable
      pages/               vistas
    nginx.conf             proxy /api y /uploads hacia el backend
```

## Documentación del proyecto

- [`decisiones.md`](decisiones.md) — por qué se decidió cada cosa, acumulado TP por TP
- [`evidencias.md`](evidencias.md) — evidencias de funcionamiento

## Entregas

Cada trabajo práctico queda congelado en su tag, así que se puede volver al estado exacto
en que se entregó:

| Tag | Trabajo práctico |
|---|---|
| [`v1.0.0`](../../releases/tag/v1.0.0) | TP1 — Git colaborativo, protecciones de rama y conflicto resuelto |
| [`v2.0.0`](../../releases/tag/v2.0.0) | TP2 — Contenerización con Docker Compose y publicación en ghcr.io |
| [`v3.0.0`](../../releases/tag/v3.0.0) | TP3 — Planificación DevOps |
| [`v4.0.0`](../../releases/tag/v4.0.0) | TP4 — CI: pipelines as code |

> El `README.md` de `v1.0.0` conserva el título `# Proyecto IngSoft3 - versión A`: era el
> resultado del ejercicio de conflicto del TP1, y sigue ahí, en ese tag y en el commit de
> merge `300827b`.
