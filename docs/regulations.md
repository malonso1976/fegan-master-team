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

## Tipos de competición, independientes de la temporada

La aclaración del usuario de 27/10/2026 distingue `ChampionshipRules` (`championship`) y `LeagueRules` (`league`). Las URL anteriores quedan como fuentes históricas de consulta en esta documentación, sin incorporarse al código ni a las respuestas como identificadores de reglas. Tampoco se fija la temporada del cálculo a la de esas circulares.

| Regla implementada | Campeonato | Liga |
|---|---|---|
| Relevos implementados | 4x50 libre/estilos, M/F/X | 4x50 libre/estilos, M/F/X |
| Referencias | Marca verificada desde 01/09/2023 | Acreditada o aproximada, según filtro local |
| Mixtos | Dos mujeres y dos hombres | Dos mujeres y dos hombres |
| Alternativas | Propuestas independientes, sin planificación conjunta | Revisar el límite de un relevo por nadador/jornada |

El 01/09/2023 es la fecha histórica de implantación de la exigencia de marca para el campeonato, confirmada por el usuario; no representa el inicio de la temporada seleccionada ni se desplaza cada año. El filtro de fechas puede acotar más las referencias. La verificación local no acredita por sí sola una inscripción federativa.

La interfaz conserva el cálculo con criterios comunes de edad y su filtro voluntario de verificadas; no pide un tipo de competición. La API permite aplicar explícitamente campeonato o liga. Los identificadores antiguos que contenían una temporada se han sustituido por los genéricos. No se automatizan inscripciones, límites conjuntos ni acreditación oficial; hay que revisar la normativa de la competición correspondiente. Mantener tipos genéricos no presume que todas las reglas futuras permanecerán sin cambios.

## Recomendación competitiva

Se compara la mejor propuesta propia con la mejor marca guardada por club rival y categoría, con misma prueba, categoría M/F/X, piscina y cronometraje. Para destacar una categoría se requieren al menos tres clubes rivales comparables. Ordena primero por puesto observado y después por margen respecto al tercer rival, normalizado por el tiempo de ese rival; empates completos conservan el orden de categorías.

Diferencia al tercero = estimación propia − tiempo del tercer rival. Es negativa si la estimación es más rápida. No es una probabilidad: faltan posibles inscritos y las marcas pueden pertenecer a otras fechas o alineaciones. Con menos referencias se muestran propuestas y puestos observados, pero no se inventa una recomendación de medalla.

## Oportunidades individuales

Se utiliza la mejor marca histórica guardada del nadador en la piscina elegida, hasta la fecha actual, incluso si pertenece a otro grupo de edad en la fecha de la prueba. Se compara con el grupo de edad actual, usando las mejores marcas de cada rival en la ventana móvil de los dos últimos años. Una fila por prueba: el cronometraje lo determina la mejor marca propia, como confirmó el usuario el 27/10/2026. No se convierten ni mezclan E/M. Si los mejores tiempos propios empatan, se prefiere electrónico; después, la fecha más reciente y un identificador estable.

Se excluye la licencia propia de los rivales y se muestran hasta diez licencias rivales distintas con datos compatibles disponibles. La consulta top diez de FEGAN puede incluir al propio nadador y no se solicita automáticamente un undécimo puesto para sustituirlo. La cobertura sigue limitada a los datos cargados. El listado de Marcas puede mostrar mejores registros de cualquier nadador por prueba/piscina/crono/categoría dentro de los filtros; esa consulta histórica no impone a sus filas la ventana reciente de las oportunidades.
