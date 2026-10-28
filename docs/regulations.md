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

Propuestas actuales (D50): por petición posterior del usuario, limitar cada cuarteto no premáster a su categoría natural y la inmediatamente inferior de la lista. No modifica la elegibilidad histórica aclarada arriba ni la categoría de resultados importados. Ejemplos: 180 permite +160/+120, 200 permite +200/+160, 120 permite +120/+100. Los extremos de banda son inclusivos: 199 sigue permitiendo +160/+120; 200 ya no permite +120. +320 es abierto y mantiene +280 como alternativa inferior. Edad individual mínima 20 en +80 y 25 en las demás. Se conserva la excepción previa: un integrante de 20–24 restringe el equipo a +80, aunque su suma real supere 99 y la banda natural sea superior; no se elimina su única categoría permitida. Mixto exige dos hombres y dos mujeres. Se genera una lista de propuestas por categoría: cada propuesta es una alternativa independiente.

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

Se compara la mejor propuesta propia con la mejor marca guardada por club rival y categoría, con misma prueba, categoría M/F/X y piscina. Por petición expresa del usuario, Componer relevos toma los mejores tiempos registrados manuales o electrónicos sin conversión; la selección no afirma equivalencia oficial entre cronometrías. Para destacar una categoría se requieren al menos tres clubes rivales comparables. Ordena primero por puesto observado y después por margen respecto al tercer rival, normalizado por el tiempo de ese rival; empates completos conservan el orden de categorías.

Diferencia al tercero = estimación propia − tiempo del tercer rival. Es negativa si la estimación es más rápida. No es una probabilidad: faltan posibles inscritos y las marcas pueden pertenecer a otras fechas o alineaciones. Con menos referencias se muestran propuestas y puestos observados, pero no se inventa una recomendación de medalla.

## Oportunidades individuales

Desde D49 se utiliza la mejor referencia histórica guardada tras convertir con la tabla aportada por el usuario a 50 m electrónico. Las marcas de rivales pueden ser de 25/50 y E/M: primero se convierten, después se escoge la mejor por licencia. Solo los rivales se limitan a dos años móviles; el grupo es el actual según la temporada. Empates por tiempo comparable comparten puesto. El original E se prefiere en empate de selección, seguido de fecha reciente, ID/piscina. No se modifica ninguna marca guardada. Esta petición posterior sustituye el requisito previo de conservar el cronometraje propio y el uso de AQUA en D48.

Se excluye la licencia propia de los rivales y se muestran hasta diez licencias rivales distintas con datos compatibles disponibles. La consulta top diez de FEGAN puede incluir al propio nadador y no se solicita automáticamente un undécimo puesto para sustituirlo. La cobertura sigue limitada a los datos cargados. El listado de Marcas puede mostrar mejores registros de cualquier nadador por prueba/piscina/crono/categoría dentro de los filtros; esa consulta histórica no impone a sus filas la ventana reciente de las oportunidades.

## Cobertura insuficiente de un relevo

Las marcas de rivales se pueden consultar sin tener cuatro integrantes propios con referencias compatibles. En ese caso no se calcula puesto ni diferencia respecto al equipo propio. Para 4×50 libre hacen falta cuatro licencias distintas con 50 libre en la misma piscina y dentro de los filtros. La petición posterior del usuario permite combinar tiempos manuales y electrónicos para esta estimación; cada parcial conserva su cronometraje y fecha, sin ajuste. Los relevos conservan ese criterio sin ajuste; las oportunidades individuales se convierten según la petición posterior D49. Los recuentos por estilo no garantizan alineación válida: deben cumplirse también edad, suma y composición M/F/X. No se modifica ninguna regla deportiva al mostrar los rivales independientemente de la disponibilidad del equipo.

Las categorías visibles deben admitir cuatro miembros disponibles según edades, suma y composición M/F/X. La comprobación de edades es independiente de disponer de marcas: una categoría elegible puede mostrar rivales y explicar que faltan referencias propias. Se ocultan las categorías inviables por edad/composición, incluidos premáster fuera de +80.


## Conversión elegida por el usuario (D49, sustituye D48)

Fuente: imagen de la **Tabla de conversión de marcas**, FMN (Federación Madrileña de Natación), Normativa Natación 2023–2024, aportada expresamente por el usuario el 28/10/2026. Se conserva en [la imagen embebida](../internal/web/static/conversion-fmn-2023-2024.png). Su aplicación aquí es una decisión del usuario para comparar oportunidades; no se atribuye vigencia RFEN/FEGAN ni autorización para inscribir marcas convertidas. El documento dice que los tiempos convertidos no son marcas oficiales y los vincula a inscripción dentro de su ámbito; la aplicación no acredita esos requisitos.

- Piscina 25→50: sumar el ajuste de la celda por prueba y sexo. 50→25: restarlo. Tiempos enteros en centésimas, sin factor multiplicativo ni redondeo.
- Manual→electrónico: sumar 19 centésimas; en pruebas individuales de 50 m, 29. Se aplica además del ajuste de piscina cuando corresponda. Un relevo 4×50 no es una prueba de 50 m y no recibe la excepción de 29.
- Ejemplos: 50 libre masculino 30,00 M25 → 30,89 E50; femenino → 30,69 E50. 100 libre masculino 1:00,00 M25 → 1:01,79 E50. 100 libre femenino 1:00,00 M50 → 59,09 E25.
- Referencia común en Elegir pruebas: 50E. Tiempo convertido determina selección, puesto, diferencias y barras; sin puntos AQUA. Cada referencia muestra original, cronometraje, piscina, procedencia y símbolo ≈ si hubo conversión.
- Celdas vacías no significan ajuste cero. 100 estilos y 2000/3000/5000 libre no tienen ajuste de piscina; se conservan comparativas por piscina, con corrección manual→E y aviso, sin inventar equivalencias. Estas pruebas pueden ocupar dos filas.
- El criterio no se cambia por fecha/temporada. La edición histórica queda visible como procedencia del criterio seleccionado. Los relevos siguen comparándose por piscina, E/M sin ajuste, según D41; transcribir sus celdas no activa conversiones de relevos.

La investigación y la primera implementación AQUA de D48 quedan sustituidas por la petición expresa de esta imagen. Las referencias RFEN/AQUA previas se mantienen solo como evidencia histórica en decisions.md y verification.md, sin usar sus tablas ni puntos en el cálculo actual.
