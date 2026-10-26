# Guía técnica

## Requisitos y desarrollo

Go 1.26 o posterior. Con asdf estándar: `asdf plugin add golang https://github.com/asdf-community/asdf-golang.git` y `asdf install`. No se usa el complemento ivm-golang. No hacen falta Node, Java, Python, Poppler, SQLite de sistema ni CGO para ejecutar o compilar la aplicación. Las dependencias se descargan en la primera compilación.

```sh
go mod download
go run ./cmd/fegan-master-team -no-browser -data ./local-data/team.sqlite
```

`-port 0` (predeterminado) solicita un puerto disponible al sistema. `-port 8080` solicita uno concreto; si está ocupado se informa del error y no se cambia de puerto silenciosamente. El servidor siempre escucha en `127.0.0.1`.

La base predeterminada usa `os.UserConfigDir()/fegan-master-team/team.sqlite` (la ruta exacta se imprime al arrancar). El directorio se crea con permisos 0700 y el archivo con 0600 donde el sistema admita permisos POSIX. En Windows se aplican los controles de acceso del perfil.

## API

Recursos locales, JSON, mismo origen. `GET /api/session` devuelve token de instancia, catálogo de pruebas y auto_close (seguimiento del navegador activado). Los POST requieren cabecera `X-CSRF-Token`. No se habilita CORS.

| Método/ruta | Contrato |
|---|---|
| POST /api/browser/presence | Respuesta persistente text/event-stream con token local. Se mantiene mientras la pestaña está abierta y se cancela con pagehide; sin seguimiento devuelve 204. |
| GET /api/setup | Club del equipo, catálogo guardado y estado de actualización. |
| POST /api/catalog | Actualiza catálogo de la temporada actual desde PDF. |
| POST /api/setup | {club_id}; guarda selección e inicia la carga automática. |
| GET /api/sync | Progreso y advertencias del trabajo actual. |
| POST /api/sync | {scope: all/rivals/relays, force: boolean}; inicia trabajo en segundo plano. |
| GET /api/data | Instantánea: clubs, swimmers y marks. |
| POST /api/clubs | Club {id, name}; alta/actualización por id. |
| POST /api/swimmers | Nadador {license, name, birth_year, gender, club_id, member, available}. |
| POST /api/marks | Marca completa; id ausente/vacío crea, id existente corrige. Tiempos en centiseconds. |
| POST /api/reset | {confirmation: "REINICIAR"}; confirmación exacta y token actual, cancela trabajos/peticiones antiguos, vacía la base y renueva token. Recargar tras éxito; confirmación ausente/incorrecta devuelve 400 sin cambios. |
| POST /api/delete | {kind: clubs/swimmers/marks, id}; borrado definitivo en cascada, recuento y home_cleared. |
| POST /api/relay-pdf/preview | multipart: file, championship winter/summer, date ISO de respaldo, chrono E/M, verified booleano textual, aliases objeto JSON alias:código. Devuelve id y preview sin guardar. |
| POST /api/import/start | Query con gender/pool/chrono opcionales; inicia preparación asíncrona y devuelve id. |
| GET /api/import/status/{id} | ready, error o id de confirmación y preview. Solo la consulta más reciente disponible; al cerrar se cancela. |
| GET /api/history | Hasta 500 registros recientes, versiones previas y resúmenes de importación. |
| GET /api/backup | Descarga binaria SQLite consistente. |
| POST /api/import/preview | Query FEGAN; devuelve id opaco y preview con datos, filtros, fecha y advertencias. |
| POST /api/import/confirm | {id}; confirma la instantánea del servidor, caduca a los 15 minutos. |
| POST /api/recommend | Filter + license; season_start omitido = temporada actual; pool/chrono omitidos se expanden sin mezclar; lista con pool, chrono, own_mark y details[{swimmer,mark}]. |
| POST /api/relays | Filter + club_id, event, gender, category (0 = todas), competition opcional (winter2627/league2627), limit. Con pool y chrono explícitos devuelve RelayAdvice compatible; si se omite alguno devuelve [{pool,chrono,advice}], calculados independientemente. Sin competition aplica criterios comunes de edad, sin corte/verificación forzada de invierno. |

Filter: from/to ISO inclusivos, season_start (año inicial; edad = season_start + 1 − nacimiento), pool 25/50, chrono E/M, verified_only booleano. Query: club_id y/o license y/o birth_from/birth_to; event código o `all`; gender M/F, pool, chrono, from/to y member en la consulta síncrona. En import/start, gender vacío expande M/F, pool 0 expande 25/50 y chrono vacío expande E/M. La categoría inscrita relay_category es obligatoria para nuevos relevos; age_total e integrantes son opcionales. Los campos desconocidos y cuerpos JSON superiores a 1 MiB se rechazan. Multipart local está limitado a 21 MiB y el archivo PDF a 20 MiB/400 páginas. La consulta amplia usa 30 minutos con polling, sin ampliar el plazo de respuesta HTTP de 180 segundos. Conserva la consulta síncrona compatible con filtros explícitos y plazo de 150 segundos. Un conflicto o validación devuelve JSON `error`; la interfaz conserva el formulario para corregirlo.

Mark: id, license (vacío en relevos), club_id histórico, event, pool, chrono, date ISO, location, centiseconds, source (manual/fegan/fegan-pdf/local-pdf), source_url, retrieved_at, verified, notes, relay_licenses, age_total y relay_category para relevos, gender. Para nuevas marcas manuales el servidor fija el origen. En ediciones conserva los metadatos de procedencia originales.

## Base y evolución

Esquema 3: suppressed(kind,key) guarda hashes de exclusión de entidades/marcas borradas; settings(key,value) guarda configuración y catálogo; se conservan las tablas del esquema anterior: clubs(id,payload), swimmers(license,club_id,payload), marks(id,fingerprint,club_id,license,payload), history(id,entity,entity_id,before,at) y schema_version. FK activas, índices por club y licencia. Historial y entidad se actualizan en la misma transacción. Importaciones guardan también filtros y resumen de inserciones/duplicados.

La licencia de un relevo es NULL en SQL; sus integrantes opcionales se validan desde Go. Las relaciones de club de una marca son históricas y no necesitan coincidir con el club actual del nadador. Los registros anteriores sin relay_category no se asignan por suma automáticamente y quedan fuera de las comparativas hasta revisión. No existe tabla completa de afiliaciones por temporada: el historial de edición y las marcas conservan contexto.

La migración 1/2→3 crea una copia consistente `.vN-UUID.bak` si ya existe base y añade settings/suppressed dentro de una transacción. La tabla de exclusiones usa SHA-256 del identificador o de la huella deportiva, sin guardar una copia de la entidad borrada. Los borrados purgan sus versiones y todos los resúmenes de importación (pueden contener licencias). Borrar club elimina nadadores actuales y marcas históricas propias, además de dependencias de esos nadadores en otros clubes. Se conserva el catálogo de referencia y se limpia home_club si corresponde. Altas manuales pueden recrear identidades; las exclusiones de marcas permanecen. Las copias anteriores al borrado permanecen bajo control del usuario. Rechaza versiones futuras; hay pruebas de preservación de entidades y respaldo. Para nuevos cambios, añadir migraciones versionadas equivalentes. No alterar el JSON de entidades sin compatibilidad explícita.

## Pruebas y distribución

```sh
go test ./...
go vet ./...
go test -race ./...
sh scripts/build.sh
```

Las pruebas cubren migración 1/2→3 con respaldo, borrado en cascada y exclusiones de reimportación de marcas originales/corregidas, carga local PDF con revisión/confirmación/deduplicación, consulta asíncrona de ambos géneros/piscinas/cronos y detalle de rivales sin mezclar grupos; tiempos exactos; fronteras septiembre/octubre, edades de la circular, grupo 100+ y deduplicación por nadador; optimización frente a un oráculo exhaustivo independiente; elegibilidad, premáster, categorías inferiores, recomendación por referencia de podio, estilos y rivales; contrato HTML FEGAN con fixture sintético; filtros HTTP, caché y división de fechas; importación transaccional, conflictos de identidad, preservación de correcciones, restauración e historial; protección de Host/Origin/token y recursos integrados.

Las pruebas ordinarias FEGAN levantan servidores temporales locales y utilizan PDF sintéticos; no necesitan Internet. Las comprobaciones reales optativas requieren `FEGAN_LIVE_DOCUMENT_CHECK=1` y las de documentos descargados `FEGAN_LOCAL_PDF_CHECK=1` con los archivos temporales indicados en las pruebas. Se deben distinguir de una comprobación manual de conectividad y formato real. Ninguna prueba unitara certifica la validez de normativa deportiva.

El script genera ejecutables Windows, macOS y Linux en amd64 y arm64, con CGO desactivado, y un fichero SHA256SUMS. Compilar para un sistema no acredita haber ejecutado allí. El flujo CI ejecuta pruebas en sistemas nativos y crea artefactos por plataforma; no publica releases automáticamente. Binarios sin firma/notarización: la distribución firmada es un trabajo posterior.

## Mantenimiento de FEGAN

Contrato observado en https://www.fegan.org/marks/ el 26/10/2026. Parámetros y reglas en ARCHITECTURE.md. Si cambia la página, revisar el adaptador y actualizar fixtures sintéticos conservando sus casos de error. No incorporar datos personales reales a pruebas ni registrar cuerpos de respuesta en logs.

Un trabajo automático dura como máximo 30 minutos y comunica fallos por consulta/fuente, conservando los lotes correctos. El usuario puede repetir o acotar una consulta. La caché de marcas dura 15 minutos y la de documentos una hora, ambas en memoria; una actualización manual forzada invalida la caché correspondiente. las consultas de grupo correctas del día, el catálogo, la configuración, los datos y el último estado se guardan en SQLite para poder seguir trabajando sin conexión. El catálogo de pruebas está versionado en domain.Events, no se considera inmutable ante cambios futuros.

## Entorno de este repositorio

`.tool-versions` selecciona Go 1.26.2 con el complemento estándar `golang` de asdf. Se instaló y verificó en el equipo; el complemento ivm-golang existente no se desinstaló ni se utiliza en este proyecto. El proyecto también funciona con una instalación estándar de Go 1.26 o posterior, sin asdf; el archivo no es necesario para ejecutar los binarios distribuidos. En equipos con otro gestor/complemento, selecciona allí la versión correspondiente.

## Cierre ligado al navegador

El arranque normal activa `TrackBrowser` antes de servir HTTP. Cada pestaña mantiene una petición POST autenticada con respuesta abierta; se desactiva exclusivamente el plazo de escritura de esa respuesta. Se emiten comentarios cada 15 segundos y se detecta la desconexión con el contexto de la petición. No depende de temporizadores periódicos de JavaScript para decidir si una pestaña sigue abierta. `pagehide` cancela la conexión; una restauración de caché de navegación la restablece. Los errores de transporte se reintentan cada 500 ms.

Al perder la última conexión se concede un margen de cinco segundos. Una conexión nueva anula el cierre pendiente; una generación invalida callbacks anteriores de temporizadores durante recargas sucesivas. Nunca se inicia un plazo de salida antes de la primera conexión, para permitir abrir manualmente la URL si falla el lanzador del navegador. `-no-browser` desactiva este seguimiento y mantiene el cierre manual.

Al terminar el margen, main cierra HTTP de forma ordenada (hasta cinco segundos para peticiones activas, después cierre de conexiones), cancela trabajos y sincronización, espera su terminación y cierra SQLite. Los lotes ya confirmados se conservan y las consultas pendientes sin confirmar se descartan. Una pestaña navegada a otra página cuenta como cerrada. Un navegador que descarte/suspenda completamente la página o su conexión durante más de cinco segundos también puede activar la salida: no se inspeccionan procesos ni ventanas del sistema operativo.

## Reinicio de datos

`operations` separa las operaciones API del reinicio exclusivo; `resetMu` serializa reinicios y cierre del servidor. El contexto `workCtx` identifica la generación vigente. Las peticiones API normales se cancelan con él y mantienen una barrera de lectura; la presencia y el propio reinicio quedan fuera. Se comprueba el contexto y el token después de adquirir la barrera, impidiendo escrituras que estaban esperando antes del reinicio.

Tras validar confirmación, el servidor cancela workCtx y llama a Suspend del sincronizador antes de esperar la barrera exclusiva: ambos productores deben poder liberar el mutex del cliente FEGAN incluso con peticiones bloqueadas. Espera los trabajos de preparación, ejecuta Store.Reset en transacción con plazo de 30 segundos y descarta cachés/vistas previas. Sustituye el sincronizador y el contexto, y renueva el token solo si el borrado se confirma. La transacción no depende de que la pestaña permanezca conectada una vez confirmado el reinicio. Un error revierte los datos; los trabajos previos permanecen cancelados y las consultas preparadas se descartan. El esquema 3 y las copias externas se conservan.
