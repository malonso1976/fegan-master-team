# Temporadas, edades y categorías

Fuentes revisadas el 26/10/2026:

- [Circular 26-32: Campeonato Gallego Máster de Invierno 2026/27](https://www.fegan.org/fegan/tprd/2627/xeral/circulares/cir_2632_cto_gal_master_inv.pdf), apartado 3 (página 2); inscripción y puntuación (página 3).
- [Circular 26-30: Liga Gallega Máster 2026/27](https://www.fegan.org/fegan/tprd/2627/xeral/circulares/cir_2630_liga_gal_master.pdf), apartado 3 (página 2); puntuación (página 3).

Se extrajo el texto y se inspeccionaron visualmente ambas páginas 2. Los PDF fuente no se modificaron.

## Calendario confirmado por el usuario

Temporada desde 1 de octubre de su año inicial hasta 30 de septiembre del siguiente. Edad = segundo año − año de nacimiento, cumplida a 31 de diciembre de ese segundo año, como indican ambas circulares.

Ejemplo: 2026/27 usa 31/12/2027 tanto para una prueba en noviembre de 2026 como para una en junio de 2027. El cambio de temporada se produce entre 30 de septiembre y 1 de octubre. Las ventanas de marcas de referencia son independientes y pueden abarcar temporadas anteriores.

Grupos individuales: 20–24, 25–29, etc., hasta 95–99; 100+ es abierto. En 2026/27, 40–44 corresponde a 1983–1987.

## Categoría natural e inscripción en relevos

| Categoría | Banda natural indicada en las circulares |
|---|---|
| +80 premáster | 80–99 |
| +100 | 100–119 |
| +120 | 120–159 |
| +160 | 160–199 |
| +200 | 200–239 |
| +240 | 240–279 |
| +280 | 280–319 |
| +320 | 320 o más |

La tabla no presenta bandas numéricamente solapadas. Sin embargo, el apartado 3.3 restringe los premáster a +80 y permite al resto intervenir en todas las categorías, incluida premáster. Se consultó esta combinación de reglas al usuario con un ejemplo concreto y confirmó que un equipo no premáster de suma 180 puede escoger categorías inferiores (+120, +100 y demás elegibles). Esta interpretación de elegibilidad es una aclaración explícita del usuario, distinguida de los intervalos impresos.

Implementación: para una categoría seleccionada, suma real al menos igual al umbral (+80, +100, etc.), sin límite superior de la banda natural. Edad individual mínima 20 en +80 y 25 en las demás. Un integrante de 20–24 restringe el equipo a +80, aunque su suma real supere 99. Mixto exige dos hombres y dos mujeres. Se genera una lista de propuestas por categoría: cada propuesta es una alternativa independiente.

Guardar siempre la categoría inscrita de un resultado rival. No deducirla de su suma real, que puede no determinar la categoría elegida. Los integrantes y la suma real pueden ser desconocidos.

## Diferencias entre perfiles

| Regla | Invierno 26-32 | Liga 26-30 |
|---|---|---|
| Relevos del programa | 4x50 libre/estilos, M/F/X | 4x50 libre/estilos, M/F/X, en sus jornadas |
| Referencias de inscripción | Acreditadas desde 01/09/2023 | Acreditadas o aproximadas; no sin marca |
| Participación individual | Hasta 4 pruebas, 2 por jornada | 2 por jornada |
| Participación en relevos | Todas las pruebas de relevos | Un relevo por nadador y jornada |
| Equipos por club | Un equipo por categoría | Un equipo por categoría |
| Mixtos | Dos hombres y dos mujeres | Dos hombres y dos mujeres |

La interfaz no pide un perfil: aplica los criterios comunes de edad y elegibilidad, conserva el filtro voluntario de verificadas y agrupa resultados por piscina y cronometraje. No impone el corte de fechas del campeonato de invierno ni acredita una inscripción. Los perfiles explícitos de la API se mantienen para compatibilidad: el perfil de invierno excluye referencias anteriores a la fecha indicada y fuerza marcas verificadas; esta marca de verificación local no certifica acreditación federativa. El perfil de liga permite aproximadas salvo filtro voluntario de verificadas. No se automatiza una inscripción ni una planificación conjunta que aplique los límites por jornada: la interfaz advierte de la revisión necesaria. Tampoco se calculan puntos WAQ, récords o clasificación conjunta del club.

Los perfiles están versionados por la temporada de las fuentes. Si se usa otra temporada, los resultados avisan de que es una simulación con reglas 2026/27, sin presuponer la normativa histórica o futura.

## Recomendación competitiva

Se compara la mejor propuesta propia con la mejor marca guardada por club rival y categoría, con misma prueba, categoría M/F/X, piscina y cronometraje. Para destacar una categoría se requieren al menos tres clubes rivales comparables. Ordena primero por puesto observado y después por margen respecto al tercer rival, normalizado por el tiempo de ese rival; empates completos conservan el orden de categorías.

Diferencia al tercero = estimación propia − tiempo del tercer rival. Es negativa si la estimación es más rápida. No es una probabilidad: faltan posibles inscritos y las marcas pueden pertenecer a otras fechas o alineaciones. Con menos referencias se muestran propuestas y puestos observados, pero no se inventa una recomendación de medalla.
