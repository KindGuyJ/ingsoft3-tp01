# ---- Etapa 1: build ----
FROM node:22-alpine AS build
WORKDIR /app

# Lockfile primero: capa cacheable.
COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

# ---- Etapa 2: tests (parte del build: ya tiene Node y las dependencias) ----
# npm ci de la etapa 1 instala tambien las devDependencies (vitest), y la imagen
# final no se lleva nada de esto: copia solo dist/. Los tests corren cuando el
# pipeline ejecuta la etapa (docker run), no al construirla.
FROM build AS test
ENTRYPOINT ["npm", "run", "test:ci"]

# ---- Etapa 3: nginx sirve los estáticos (queda ULTIMA: es la imagen final) ----
FROM nginx:1.27-alpine AS final
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
