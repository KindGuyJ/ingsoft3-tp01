# ---- Etapa 1: build ----
FROM node:22-alpine AS build
WORKDIR /app

# Lockfile primero: capa cacheable.
COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

# ---- Etapa 2: nginx sirve los estáticos ----
FROM nginx:1.27-alpine AS final
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
