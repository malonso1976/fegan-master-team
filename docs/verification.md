# Verificación de la primera versión

Fecha: 26/10/2026. Entorno ejecutado: macOS arm64, Go 1.26.2 mediante el complemento estándar golang de asdf. La ruta comprobada es ~/.asdf/installs/golang/1.26.2/go/bin/go; el proyecto ya no utiliza ivm-golang. Datos de pruebas en directorios temporales; no se cargaron datos de prueba en el perfil predeterminado del usuario.

## Comprobaciones completadas

- `go test ./...`: pruebas del dominio, SQLite, adaptador FEGAN y API local.
- `go test -race ./...`: superado tras incorporar temporadas y elegibilidad de categorías inferiores.
- `go vet ./...`: análisis estático sin incidencias.
- Validación de sintaxis del JavaScript integrado.
- Optimización de relevos comparada con un oráculo exhaustivo independiente en 20 equipos sintéticos, libre/estilos y mixtos.
- Fronteras 30/09–01/10, ejemplos de nacimientos de la circular, grupo 100+ abierto, premáster solo +80, categorías inferiores, categoría más favorable según rivales y restricciones de referencias de invierno.
- Idempotencia de importación, conflicto de identidad con reversión completa, preservación de correcciones y recuperación desde copia SQLite, incluido historial.
- Contrato HTML FEGAN con datos sintéticos, errores de formato/HTTP, caché y división de intervalos al alcanzar 100 resultados.
- PDF sintéticos de catálogo y relevos, columnas de nombre corto/medio/código, alias oficiales Lenex, exclusión de bajas/descalificaciones e identificación de clubes no vinculables.
- Ventana móvil de dos años, una consulta compartida por grupo/prueba/piscina/crono, top diez por licencia y recuperación mediante historial cuando la consulta top diez devuelve filtros incompatibles.
- Trabajo en segundo plano, estado persistido y reutilización de consultas de grupo correctas del día; migración 1/2→3 con copia y conservación de los datos.
- Protección del servidor local frente a Host/Origin externos y POST sin token, y rechazo de JSON adicional.
- Contratos de las API de recomendación individual y de categorías con temporada 2026/27.

## FEGAN real

Se ejecutó una consulta pública de 50 libre masculino, piscina 25 m y cronometraje electrónico, para un club y fecha concretos: se recibieron 11 marcas. Se confirmó en la base temporal; al repetir, se conservaron los resultados como duplicados, usando la caché. La descarga de copia devolvió una base SQLite válida.

Se comprobó además «todas las pruebas» para una licencia y fecha: 21 consultas y dos marcas compatibles, incluidos los casos sin resultados. Esto acredita esa observación concreta, no disponibilidad futura ni exhaustividad de toda FEGAN.

## Catálogo, campeonatos y sincronización automática

El catálogo PDF real 2026/27 devuelve 58 clubes, con código, nombre largo, medio y corto verificados, incluidos los ceros iniciales. La prueba de cuatro campeonatos descubiertos desde post/57 y la memoria deportiva devolvió 186 marcas (verano 2026), 178 (invierno 2026), 153 (verano 2025) y 184 (invierno 2025): 701 relevos, sin alias de club desconocidos. Cada campeonato se importó dos veces en una base temporal: la segunda importación no añadió filas y reconoció todos sus resultados como duplicados. Se utilizaron los PDF oficiales accesibles mediante ZIP y páginas de resultados ante los enlaces directos rotos.

La carga automática completa desde la selección de un club se ejecutó en una base temporal: 4.451 marcas nuevas y 570 duplicados omitidos. Esta ejecución detectó y comunicó una respuesta top diez fuera de filtros; posteriormente se incorporó recuperación mediante historial y se verificaron con éxito las siete consultas reales del evento/piscina/cronometraje afectado. Los registros incorrectos no se incorporaron. La cobertura se informa por fuentes y no se afirma exhaustividad universal de FEGAN.

## Navegador

Comprobado el nuevo inicio por selección de club, búsqueda en el catálogo y arranque automático con progreso visible y navegación disponible. También comprobados el alta de club y nadador con ceros iniciales, marca manual, recomendación individual y composición de un relevo con cuatro licencias distintas. Después de los ajustes de normativa, se verificó el resumen de todas las categorías: un equipo sintético de suma real 164 puede entrar en +80/+100/+120/+160 y la aplicación destaca +160 frente a +120 según tres clubes rivales sintéticos por categoría, con edades a 31/12/2027.

## Distribución

Compilaciones sin CGO para macOS, Linux y Windows en amd64 y arm64 mediante `scripts/build.sh`, con SHA256SUMS. Se ha arrancado y utilizado la versión de macOS arm64. Windows, Linux y macOS amd64 se han verificado por compilación; su ejecución nativa queda pendiente de CI o de equipos de esos sistemas. El flujo CI está preparado pero no se ha ejecutado en GitHub ni se han publicado releases.

## Límites de esta validación

No es una certificación federativa. Se aplica la interpretación de categorías inferiores confirmada expresamente por el usuario y registrada en decisions.md y regulations.md. No se verificó una inscripción completa de campeonato, ni una planificación conjunta de varias pruebas/equipos. Las recomendaciones dependen de las referencias disponibles y no representan probabilidades de medalla.

## Ajustes de gestión e interfaz del 26/10/2026

- Go utilizado directamente desde el complemento estándar golang de asdf, versión 1.26.2.
- Pruebas con detector de carreras y análisis estático correctos, más comprobación de sintaxis JavaScript. Fuentes HTTP simuladas; sin alterar la base del perfil del usuario.
- Borrado de nadador con marcas de otro club y relevos que incluyen su licencia; purga de versiones; reimportación sin resurrección. Borrado de marca corregida protege tanto su huella original como la corregida. Borrado del club propio limpia su selección; borrar un club histórico conserva a nadadores cuyo club actual es otro.
- Migración desde esquema 2 a 3 con respaldo v2, preservación de club y nueva tabla de exclusiones. La migración desde 1 también continúa cubierta.
- PDF local multipart sintético: rechazo de contenido inválido y alias con código inexistente, vista previa sin persistencia, procedencia local y verificación desactivada, confirmación e importación repetida sin nuevas filas. El flujo completo de selección de archivo del sistema no se automatizó en el navegador; se verificaron su formulario y el contrato multipart del servidor.
- Importación amplia asíncrona contra fuente simulada: ocho combinaciones M/F × 25/50 × E/M, respuesta inicial inmediata, consulta de estado y vista previa retenida para confirmación sin guardar datos.
- Comparativas automáticas sin crono/temporada explícitos: detalles de rivales y marca propia, sin mezclar piscina o cronometraje.
- Navegador macOS, base temporal con nombres ficticios: menús separados, nombres/códigos de clubes y orden Ártabro / Club Brisa / Zeta, club inicialmente vacío, autocompletado por nombre con categoría 45–49 y club actual, edición histórica con categoría 40–44 y conservación del resultado sin duplicarlo, importación sin los selectores retirados y relevos calculados en grupos 25 m electrónico / 50 m manual sin normativa seleccionada.
- Detalle de rivales comprobado en pantalla con licencia, club, fecha, lugar, diferencia y barras, destacando al nadador propio. Las marcas ficticias no se distribuyen con la aplicación.
- Regenerados los seis ejecutables Windows/macOS/Linux, amd64/arm64, con sumas SHA-256 verificadas. La ejecución nativa de Windows/Linux sigue pendiente; la compilación cruzada no la acredita.

## Cierre automático al cerrar el navegador

- Pruebas de ciclo de vida: sin conexión inicial no se programa salida; otra pestaña mantiene el proceso; recargar anula el cierre; la última desconexión termina el seguimiento; limpieza repetida y rechazo de nuevas conexiones tras finalizar.
- Conexión HTTP real en servidor temporal: presencia requiere token local, permanece abierta sin actividad del usuario y termina al desconectar el cliente. Pruebas ordinarias, detector de carreras, análisis estático y sintaxis JavaScript correctos.
- Comprobación con el navegador integrado sobre macOS y base temporal vacía: apertura de pestaña, recarga, segunda pestaña, cierre de la primera y, finalmente, cierre de la última. El proceso terminó por sí solo con código 0, sin enviar Ctrl+C. Se sustituyó solo el lanzador automático por un comando de prueba para no abrir el navegador habitual del usuario; la pestaña real se abrió con la herramienta de navegador.
- Compilación de los seis ejecutables actualizada y sumas SHA-256 verificadas. No se ha probado este cierre con navegadores nativos de Windows/Linux ni el descarte agresivo de pestañas por ahorro de recursos. Ese descarte puede equivaler a una desconexión y activar el margen de salida.

## Reinicio completo desde Ajustes

- Store.Reset vacía en una transacción clubes, nadadores, marcas, historial, configuración y exclusiones; conserva esquema 3 y permite volver a importar marcas previamente excluidas. Un fallo simulado al borrar clubes revierte también las marcas y nadadores eliminados antes en esa transacción.
- API: ausencia de token, confirmación vacía/incorrecta o campos desconocidos no modifican datos ni cancelan trabajos. Un reinicio confirmado vacía el club propio y catálogo guardado, elimina vistas previas y rechaza formularios de pestañas con token anterior. Una sesión nueva puede volver a guardar clubes.
- Concurrencia con fuente HTTP simulada: reinicio durante consulta de catálogo, sincronización automática y preparación asíncrona de importación. Cancela y espera todas ellas sin bloqueo, elimina el estado del trabajo y evita que el catálogo o sus datos reaparezcan después.
- Navegador macOS con base temporal ficticia: Ajustes, diálogo con recuentos y alcance, botón final deshabilitado antes de escribir REINICIAR y habilitado después. Se pulsó Cancelar y se comprobó que el club ficticio seguía guardado. El borrado se verificó en pruebas API y de persistencia con bases temporales; no se ejecutó la acción destructiva sobre la base del perfil del usuario.
- Pruebas ordinarias, detector de carreras, análisis estático y sintaxis JavaScript correctos. Regenerados seis ejecutables y comprobadas sus sumas SHA-256; Windows/Linux siguen pendientes de ejecución nativa.
