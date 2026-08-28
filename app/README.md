# Tienda de Indumentaria

Aplicación full-stack de e-commerce de indumentaria. App del semestre de
**Ingeniería del Software 3** (UCC, 2026).

| Capa | Tecnología |
|---|---|
| Backend | Go 1.24 + Gin + GORM |
| Base de datos | MySQL 8.4 |
| Frontend | React + Vite, servido por nginx |
| Orquestación | Docker Compose |

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
| `PATCH` | `/api/pedidos/:id/estado` | admin |

El token va en `Authorization: Bearer <token>` y sale del login.

## Levantar desde el registry (sin construir)

```bash
cp .env.example .env
docker compose -f docker-compose.registry.yml up -d
```

Descarga las imágenes publicadas en ghcr.io en vez de construirlas.

## Desarrollo local (sin contenedores)

```bash
# Base de datos como contenedor suelto
docker run -d --name mysql-tienda -e MYSQL_ROOT_PASSWORD=root \
  -e MYSQL_DATABASE=tienda -p 3306:3306 mysql:8.4

# Backend → http://localhost:8080
cd backend && DB_HOST=localhost JWT_SECRET=dev go run ./cmd/api

# Frontend → http://localhost:5173 (Vite proxea /api al backend)
cd frontend && npm install && npm run dev
```

## Tests

```bash
cd backend && go test ./internal/services/... -v
cd frontend && npm run test
```

Los tests del backend no necesitan base de datos: los services solo conocen
interfaces, así que corren contra un repositorio fake en memoria.

## Estructura

```
backend/
  cmd/api/            punto de entrada
  cmd/seed/           carga inicial del catálogo (binario aparte)
  internal/
    controllers/      handlers HTTP (sin lógica de negocio)
    services/         ← las reglas de negocio, y sus tests
    repository/       acceso a datos (GORM)
    dao/              entidades
    dto/              contratos de la API
    middleware/       JWT y rol admin
    config/           variables de entorno
    errors/           errores de dominio tipados
frontend/
  src/
    services/api.js   cliente HTTP (rutas relativas)
    utils.js          lógica pura testeable
    pages/            vistas
  nginx.conf          proxy /api y /uploads hacia el backend
```

La app vive en `app/`; los documentos del trabajo práctico están **un nivel más
arriba**, en la raíz del repositorio, porque son de todos los TPs y no solo de esta
aplicación.

## Documentación del proyecto

- [`../decisiones.md`](../decisiones.md) — por qué se decidió cada cosa
- [`../evidencias.md`](../evidencias.md) — evidencias de funcionamiento
