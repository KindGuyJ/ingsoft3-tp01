# ---- Etapa 1: build (imagen con toolchain de Go) ----
FROM golang:1.24-alpine AS build
WORKDIR /src

# go.mod y go.sum primero: capa cacheable, solo se rehace si cambian las dependencias.
# El go.sum NO es opcional: sin el, go mod download no verifica checksums.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /api ./cmd/api
# El seed es un segundo binario, no un modo de la API: sembrar la base es una
# operacion manual de un solo uso (docker compose exec backend /bin/seed).
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /seed ./cmd/seed

# ---- Etapa 2: tests (parte del build: mismo codigo y mismas dependencias) ----
# La imagen final NO depende de esta etapa: un test roto no impide construirla.
# Los tests corren cuando el pipeline la ejecuta (docker run), no al construirla:
# por eso es ENTRYPOINT y no RUN.
FROM build AS test
# Sin cgo, igual que el build: alpine no trae gcc, y gobco lo necesitaria para
# analizar el paquete net (falla con "error processing cgo for package net").
ENV CGO_ENABLED=0
# gobco mide cobertura de ramas, que Go no trae. Version fija: el build es reproducible.
RUN go install github.com/rillig/gobco@v1.3.4
ENTRYPOINT ["sh", "./cobertura.sh"]

# ---- Etapa 3: runtime (imagen minima, sin toolchain) ----
FROM alpine:3.20 AS final
RUN adduser -D -u 10001 app
WORKDIR /app
RUN mkdir -p /app/uploads && chown app:app /app/uploads
COPY --from=build /api /bin/api
COPY --from=build /seed /bin/seed
USER app
ENV GIN_MODE=release
EXPOSE 8080
ENTRYPOINT ["/bin/api"]
