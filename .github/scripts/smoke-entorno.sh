#!/usr/bin/env bash
#
# Smoke test post-deploy: ¿el entorno esta vivo? Lo usan deploy-qa y
# deploy-prod con las URLs de SU entorno, asi que QA y PROD se verifican con
# exactamente la misma prueba.
#
# USO: URL_API=https://... URL_FRONT=https://... smoke-entorno.sh
#
# Comprueba cuatro cosas, y si cualquiera falla vuelve a intentar:
#   1. API /healthz          el proceso esta vivo
#   2. API /api/productos    la api llega a SU base (un /healthz solo daria
#                            verde con la conexion a la base rota)
#   3. front /               nginx sirve la SPA
#   4. front /api/productos  el front llega a SU api: prueba el BACKEND_URL de
#                            la plantilla de nginx
#
# Lo que NO prueba: que la version que contesta sea la recien desplegada.
# Render sigue sirviendo la anterior mientras construye (y si el build falla),
# asi que este smoke puede dar verde contra la version vieja. Ver decisiones.md.

set -u

: "${URL_API:?falta URL_API}"
: "${URL_FRONT:?falta URL_FRONT}"

# 30 vueltas x 20 s = 10 minutos: el build de Render (2 a 5 min por servicio)
# mas el cold start del plan gratuito (hasta ~1 min). Un curl seco daria rojos
# falsos.
INTENTOS=30
ESPERA=20

for i in $(seq 1 "$INTENTOS"); do
  # --max-time en CADA curl: un servicio que se esta despertando acepta la
  # conexion y no contesta, y sin tope curl se quedaria esperando hasta que
  # el runner mate el job, con un error que no dice por que.
  if curl -fsS --max-time 10 "$URL_API/healthz" > /dev/null \
     && curl -fsS --max-time 10 "$URL_API/api/productos" > /dev/null \
     && curl -fsS --max-time 10 "$URL_FRONT/" > /dev/null \
     && curl -fsS --max-time 10 "$URL_FRONT/api/productos" > /dev/null; then
    echo "✅ El entorno responde (api + base + front + front→api), intento $i"
    exit 0
  fi
  echo "… intento $i/$INTENTOS sin respuesta completa, espero ${ESPERA}s"
  sleep "$ESPERA"
done

echo "❌ El entorno no respondio en $((INTENTOS * ESPERA / 60)) minutos"
exit 1
