# ---- Etapa 1: build ----
FROM node:26-alpine AS build
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
# Asocia el paquete de ghcr.io con el repo (y le da permiso al pipeline de publicarlo).
LABEL org.opencontainers.image.source=https://github.com/KindGuyJ/ingsoft3-tp01
# En templates/ y no en conf.d/: nginx leeria conf.d/ tal cual, con los ${...}
# sin reemplazar, y no arrancaria.
COPY default.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /app/dist /usr/share/nginx/html
# Defaults = los valores de compose. Render (QA/PROD) los pisa con las
# variables de cada servicio: la direccion del backend no va en la imagen.
ENV BACKEND_URL=http://backend:8080 \
    DNS_RESOLVER=127.0.0.11
# TP7: el commit del que salio la imagen, en /version.txt. Es la contraparte del
# commit de /healthz en la api: con esto el smoke comprueba que el FRONT tambien
# es la imagen de la corrida, no la anterior. Va al final, como en el backend.
ARG COMMIT=""
RUN echo "$COMMIT" > /usr/share/nginx/html/version.txt
EXPOSE 80
