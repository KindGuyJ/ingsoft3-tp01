# Evidencias — TP1

## 1. Push directo a main rechazado
![push rechazado](img/push-rechazado.png)
Intenté hacer `git push` directo a `main` después de un commit local, y GitHub lo rechazó con un
error de tipo "protected branch hook declined". Esto confirma que la protección de rama está activa
y que alcanza incluso al administrador del repositorio (dueño de la cuenta), ya que tenía activada
la opción "Do not allow bypassing the above settings".

## 2. Aviso de conflicto en el Pull Request
![aviso de conflicto](img/aviso-conflicto.png)
Al crear dos ramas (`feature/titulo-a` y `feature/titulo-b`) partiendo ambas de `main` y modificando
la misma línea del `README.md`, mergeé primero el PR de la rama A. Al intentar mergear el PR de la
rama B, GitHub mostró el aviso de que no se puede mergear automáticamente porque hay conflictos
("This branch has conflicts that must be resolved").

## 3. Marcadores de conflicto en el archivo
![marcadores de conflicto](img/marcadores-conflicto.png)
Al abrir el editor de resolución de conflictos de GitHub (Resolve conflicts) sobre el PR de la rama
B, se veían los marcadores `<<<<<<<`, `=======` y `>>>>>>>` delimitando las dos versiones en
conflicto de la primera línea del `README.md`: una proveniente de `feature/titulo-b` (mi rama) y
otra ya integrada en `main` desde `feature/titulo-a`.

## 4. Release v1.0.0 publicada
![release publicada](img/release-v1.0.0.png)
Después de crear el tag `v1.0.0` sobre `main` y subirlo con `git push origin v1.0.0`, publiqué la
release correspondiente desde la sección "Releases" del repositorio, con el título `v1.0.0` y notas
describiendo qué incluye esta versión (protecciones de rama, flujo de Pull Requests y conflicto
resuelto).