#!/bin/sh
# Corre los tests del backend, mide la cobertura y FALLA si no llega al umbral.
#
# Es el ENTRYPOINT de la etapa `test` del Dockerfile: el pipeline la ejecuta con
# `docker run -v ...:/out` y lo que se escribe en /out queda en el runner.
# En tu maquina tambien anda: OUT=./cobertura sh cobertura.sh
set -eu

# El umbral es sobre SENTENCIAS, la unica metrica que Go mide de forma nativa.
# Por que este numero: ver decisiones.md (TP5).
UMBRAL=80

# Lo que NO entra en la cuenta. Es una lista de exclusion y no de inclusion a
# proposito: un paquete nuevo entra solo, y si no tiene tests baja el numero.
#   cmd, config        -> el arranque: cablear la app y leer variables de entorno
#   dao, dto           -> structs de datos, sin comportamiento
#   controllers, repository -> adaptadores HTTP y GORM: los verifica la
#                         integracion contra la base de verdad (TP7)
EXCLUIDOS='/internal/(config|dao|dto|controllers|repository)/|/cmd/'

OUT=${OUT:-/out}
mkdir -p "$OUT"

# 1) Tests + cobertura. -coverpkg=./... hace que un test de services tambien
#    cuente lo que ejecuta en otros paquetes (por ejemplo, errors/).
go test -coverpkg=./... -coverprofile="$OUT/todo.out" ./...

# 2) Recorte: se sacan del perfil los paquetes excluidos.
grep -v -E "$EXCLUIDOS" "$OUT/todo.out" > "$OUT/cobertura.out"
go tool cover -func="$OUT/cobertura.out" > "$OUT/cobertura.txt"
go tool cover -html="$OUT/cobertura.out" -o "$OUT/cobertura.html"

# 3) Ramas. Go no las mide: lo hace gobco, que corre los tests de cada paquete
#    con los `if` instrumentados. Solo informa, no frena. Su salida lista cada
#    condicion que ningun test recorrio en las dos direcciones.
: > "$OUT/ramas.txt"
for p in ./internal/services/ ./internal/middleware/; do
  gobco -branch "$p" >> "$OUT/ramas.txt" 2>&1
done

# 4) El freno.
TOTAL=$(awk '/^total:/ { sub("%", "", $3); print $3 }' "$OUT/cobertura.txt")
RAMAS=$(awk '/^Branch coverage:/ { split($3, f, "/"); c += f[1]; t += f[2] }
             END { printf "%d/%d (%.1f%%)", c, t, 100 * c / t }' "$OUT/ramas.txt")
echo "Cobertura de sentencias: ${TOTAL}% (umbral ${UMBRAL}%)" | tee "$OUT/resumen.txt"
echo "Cobertura de ramas:      ${RAMAS} (informativa, sin umbral)" | tee -a "$OUT/resumen.txt"

if ! awk -v t="$TOTAL" -v u="$UMBRAL" 'BEGIN { exit !(t + 0 >= u + 0) }'; then
  echo "ERROR: la cobertura de sentencias (${TOTAL}%) no llega al umbral (${UMBRAL}%)" | tee -a "$OUT/resumen.txt"
  exit 1
fi
