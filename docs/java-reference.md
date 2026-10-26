# Revisión de la referencia Java

Fuente leída: `/Users/marianoao/Projects/personal/marianoao/fegan-relay-calculator` el 26/10/2026. Revisión estática; no se ejecutó el programa Java. No se han copiado resultados ni identidades personales de sus recursos.

| Función Java | Aplicación Go |
|---|---|
| Marks.getSwimmerMarks: mejores resultados por licencia | Consulta general acotada por licencia, preserva cada resultado y su contexto; opción de todas las pruebas individuales. |
| Marks.getAgeGroupMark: top 10 por prueba y género | Importación por nacimiento/categoría y mejor marca local por licencia, sin limitar el cálculo a diez rivales ni activar comparación ponderada. |
| EventMark.isCandidateMarkText: puntuación respecto a primera/última marca con tolerancia | Puesto observado, cobertura y diferencia al primero. No se presenta ese porcentaje como probabilidad. |
| Relays.calculateFree / calculateIM | Combinaciones de cuatro distintos, asignaciones por estilo, disponibilidad, temporada explícita y comparación entre categorías elegibles. |
| JSON de marcas de nadador y referencias de relevos | Alta manual de marcas completas; resultados de relevos rivales por categoría inscrita y suma real opcional, con integrantes opcionales. |

## Diferencias deliberadas

- Java utiliza `float` para tiempos y trunca centésimas al formatear. Go conserva centésimas enteras.
- `SwimmerBestMarks.getBestFreeRelayMark` busca la mejor marca entre libre, espalda, braza y mariposa. Go exige libre para un relevo libre.
- `Relay.getAgeGroup` clasifica por suma en una única categoría. Go separa suma real y categoría inscrita, y compara todas las categorías inferiores elegibles conforme a la aclaración del usuario y las circulares 26-30/26-32.
- Java aumenta la edad desde septiembre. Go inicia temporada en octubre y usa la edad a 31/12 de su segundo año, confirmada por el usuario y las circulares (D06).
- Los extremos de años de nacimiento de Java pueden incluir seis años para un grupo de cinco. Go compara edades con límites no solapados. En 2026/27, 40–44 son nacimientos 1983–1987, incluidos.
- Java activa `input_equate` para competidores sin preservar la base de comparación de piscina/crono. Go separa esas dimensiones sin conversiones.
- Java infiere categoría mediante consultas adicionales; Go exige la categoría en las consultas y conserva el contexto seleccionado.
- Los fallos de búsqueda en Go no activan silenciosamente datos locales de ejemplo.
