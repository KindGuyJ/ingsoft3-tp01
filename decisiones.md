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
