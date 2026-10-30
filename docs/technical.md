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
| POST /api/swimmers/deactivate | {licenses: [string]}; de 1 a 1000 entradas, deduplicadas. Actualiza solo member=false y available=false de fichas existentes en una transacción con historial. Un identificador inválido/ausente revierte el lote. Devuelve {updated: número de fichas modificadas}. Conserva marcas y resto de campos; requiere los controles locales de POST. |
| POST /api/swimmers/activate | {licenses: [string]}; mismas garantías y límites de deactivate, actualiza member=true y available=true. Conserva ficha, club, marcas y versiones; repetir no añade historial si ya estaba activo. |
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
| POST /api/recommend | Filter + license; season_start omitido = temporada actual. Pool omitido/0 convierte a 50E con tabla FMN aportada; sin ajuste de piscina, una comparación por piscina normalizada a E. Pool 25/50 explícito conserva la comparación exacta anterior. Histórica propia completa y rivales de dos años móviles. El servidor determina from/to/chrono; los antiguos valores recibidos se aceptan pero no cambian ese criterio. Respuesta con pool, chrono, time comparable, own_mark intacto, conversion {edition,source_url}, warnings y details[{swimmer,mark,time}], incluido birth_year. |
| POST /api/relays | Filter + club_id, event, gender, category (0 = todas), competition opcional (championship/league), limit. Con pool y chrono explícitos devuelve RelayAdvice compatible; si se omite alguno devuelve [{pool,chrono,advice}], calculados por piscina. chrono vacío combina referencias E/M sin convertir tiempos; con pool omitido produce dos grupos 25/50, con chrono vacío. E/M explícito conserva una consulta restringida. Cada Leg añade chrono y date; cada CategoryAdvice añade age_eligible independiente de las marcas. La interfaz oculta categorías inviables por edades y piscinas sin propuestas ni rivales. Sin competition aplica criterios comunes de edad; campeonato exige verificación desde la fecha histórica 2023-09-01. Devuelve rules en vez de source y no incluye URL ni temporada fija de circular. Cada groups[] añade rivals: [{club,mark}] y coverage: [{event,swimmers}], incluso si results está vacío. Rivals contiene la mejor referencia por club rival con los mismos filtros que el puesto; coverage cuenta licencias candidatas por estilo, sin garantizar una composición válida. |

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

## CI con acciones inmutables y Node.js 24

Verificado el 27/10/2026 mediante metadatos y manifiestos del repositorio oficial en el SHA concreto:

| Acción | Versión comentada | SHA fijado | Manifiesto |
|---|---|---|---|
| checkout | v7.0.1 | 3d3c42e5aac5ba805825da76410c181273ba90b1 | [node24](https://github.com/actions/checkout/blob/3d3c42e5aac5ba805825da76410c181273ba90b1/action.yml) |
| setup-go | v7.0.0 | b7ad1dad31e06c5925ef5d2fc7ad053ef454303e | [node24](https://github.com/actions/setup-go/blob/b7ad1dad31e06c5925ef5d2fc7ad053ef454303e/action.yml) |
| upload-artifact | v7.0.2 | cf430e030ddbb5b0abf93d22962f4752f3646cd9 | [node24](https://github.com/actions/upload-artifact/blob/cf430e030ddbb5b0abf93d22962f4752f3646cd9/action.yml) |

El comentario de versión facilita revisar actualizaciones, pero la ejecución usa el SHA completo. No es necesario añadir setup-node ni una variable que fuerce otro runtime: las propias acciones declaran Node.js 24. Se conservan permisos vacíos en el workflow y solo contents:read en cada trabajo; no se concede actions:write ni id-token:write para subir artefactos, que usan el token de ejecución de Actions. Checkout usa persist-credentials:false. Cada trabajo tiene plazo de 20 minutos. La validación local de sintaxis con actionlint no sustituye ejecutar los jobs en los runners de GitHub.

## Consulta de mejores marcas y oportunidades

La tabla de Marcas filtra y ordena localmente la instantánea (incluida ordenación numérica por centésimas/piscina). Su estado de orden se conserva al refrescar y usa aria-sort y botones de cabecera. Mejores individuales agrupa por licencia, prueba, piscina, crono y género; no por club, para conservar referencias históricas tras traslados. Se escoge menor tiempo, luego fecha más reciente e ID estable. El filtrado se aplica antes de agrupar.

Recommend admite pool=0 para el modo convertido de D49; la API fija From/To a hoy menos dos años/hoy y deja crono vacío para considerar E/M. Selecciona la menor referencia histórica convertida por prueba, hasta To. El límite From solo afecta a rivales. En empate convertido se prefiere original E; después fecha más reciente, ID y piscina. Rivales agrupados por licencia tras convertir, excluyendo al objetivo, ordenados por tiempo/licencia y limitados a diez del grupo de edad/género. La consulta automática FEGAN considera también pruebas propias antiguas y solicita ambas cronometrías y piscinas con ajuste, en ventana de dos años. La selección inicial del club consulta cinco años; referencias anteriores solo se usan si ya están guardadas.

Historial por miembro: la interfaz prepara el contrato POST /api/import/start ya existente con license exacta, event=all, gender M/F del miembro, from=hoy−5 años, to=hoy, club_id vacío y member=false. Pool/chrono omitidos consultan 25/50 × E/M con sus filtros originales; luego GET status y POST confirm. No hay una escritura directa nueva ni un cambio de esquema. La pertenencia y el club actual existentes se conservan; resultados antiguos mantienen su club histórico.

La sincronización scope=all (selección inicial y actualización completa) genera M/F × 25/50 × E/M × todas las pruebas individuales con club_id exacto, member=true y from=hoy−5 años. Los resultados conservan fecha y club histórico; las identidades locales existentes prevalecen. settings.full_sync_done contiene ClubID, Date e HistoryYears=5; la caché diaria exige coincidencia de los tres. No hay migración de esquema ni reinicio. Los trabajos de rivales y PDF mantienen sus ventanas previas. El acceso por licencia continúa permitiendo resultados de otros clubes; el inicial por club solo recupera los publicados con el código seleccionado.


## Suplemento de mejores marcas personales (D45/D46)

`internal/fegan/personal.go` consulta best_results_person para una licencia y género conocidos, en cuatro grupos 25/50 × E/M y con los filtros temporales originales. El HTML no publica género: se toma de la ficha o petición explícita, nunca del nombre. Se validan licencia, nombre, nacimiento, encabezados, fechas, piscina/crono, ausencia de parciales y códigos de club acreditados por el catálogo del formulario general. Los resultados conservan source=fegan y la URL personal, y se deduplican con el historial general; este formulario devuelve mejores marcas, no todas las participaciones.

La sincronización completa consulta este suplemento después del historial del club y antes de rivales, solo para miembros conocidos del club elegido. Incluye resultados en otros clubes sin cambiar fichas actuales. Fallos/identidades incompatibles generan avisos y no acreditan una carga completa. WidePreview complementa una consulta por licencia con género conocido; la vista previa se confirma por el flujo existente. Filas generales con nacimiento vacío se omiten con aviso; el recuento bruto se conserva para dividir respuestas de cien filas sin truncamiento. No se inventa una fecha de nacimiento.

D45 reduce las ventanas del club y la preparación por miembro a cinco años. El sello diario exige HistoryYears=5; sellos antiguos de dos/diez años no impiden completar la consulta nueva. Marcas anteriores guardadas se conservan. Rivales mantienen dos años y los PDF dos temporadas publicadas.

## Conversión de referencias (D49, sustituye D48)

`internal/domain/tables/conversion.json` transcribe la imagen FMN Natación 2023–2024 elegida por el usuario: ajustes en centésimas para M/F (17 pruebas individuales y 5 relevos) y X (5 relevos), +19 centésimas manual→electrónico y +29 para pruebas individuales de 50 m. `conversion.go` suma/resta el ajuste de piscina y suma la corrección manual; no usa flotantes ni puntos. Rechaza conversiones sin celda publicada o tiempo resultante no positivo. La tabla también transcribe relevos por fidelidad a la fuente, pero Elegir pruebas los excluye y Componer relevos no utiliza este conversor.

No hay temporadas fijas en el cálculo: la edición es procedencia del criterio elegido, no vigencia anual automática. La imagen `internal/web/static/conversion-fmn-2023-2024.png` está embebida y el enlace funciona sin conexión. `ComparablePools` amplía solo pruebas con ajuste; el adaptador consulta ambas cronometrías, agrupadas con piscina/grupo/prueba, sin equate FEGAN. La caché permite cargar los nuevos grupos M junto a los E existentes.

Pool=0 usa referencia común 50E; sin ajuste de piscina, conserva origen y convierte M→E con aviso. Selección por menor tiempo comparable; empate propio prefiere original E, fecha reciente, ID/piscina. Originales, versiones e importaciones no cambian; no hay migración. Pool=25/50 explícito conserva el contrato exacto anterior sin conversiones. Se retiran los campos points y vigencias de D48. Los tiempos convertidos no son marcas oficiales ni acreditan inscripción.

## Límite de categorías propuestas en relevos (D50)

Para cada cuarteto: categoría natural e inmediatamente inferior de RelayCategories; con premáster, solo +80 como excepción aunque la suma quede en una banda mayor. La API conserva las categorías en groups con age_eligible=false/results=[] si no hay cuarteto que cumpla el límite, y la interfaz las oculta. El optimizador no se ejecuta en esos grupos, pero sí se valida la petición y se respeta cancelación. El filtro se aplica a cada composición antes del top K, no a la edad media/global del club ni al resultado ya truncado. Elegibilidad independiente de marcas y composición mixta 2M/2F usan el mismo predicado. No altera la categoría inscrita de resultados históricos ni aplica esta política a rivales importados.


## Releases en borrador (D51)

Workflow independiente `.github/workflows/release.yml`: push de etiquetas v* con validación estricta vX.Y.Z, o workflow_dispatch con etiqueta existente. Prepare fija el SHA de la etiqueta; validate prueba ese SHA en los tres sistemas y package compila/empaca el mismo SHA. `.tool-versions` selecciona Go, sin ivm-golang. Solo draft tiene contents:write; no hace checkout ni ejecuta scripts del tag. Descarga artefactos de la misma ejecución, comprueba SHA256SUMS y el destino actual de la etiqueta, y usa gh release create --verify-tag --draft --generate-notes. No sobrescribe releases ni crea etiquetas. Concurrencia por etiqueta, sin cancelación de la ejecución activa.

`scripts/package-release.py` usa Python 3 estándar, disponible en el runner Ubuntu, solo para distribución: seis ZIP con ejecutable/LEEME, permisos Unix, versión/SHA en la guía y SHA256SUMS de los ZIP. Archivos seleccionados explícitamente, sin bases de usuario. Directorios de salida no vacíos se rechazan; los errores de versión/commit/ejecutable ausente fallan antes de crear archivos. Pruebas de contrato en scripts/test_package_release.py, también ejecutadas por el trabajo portable del CI.

Se reutilizan checkout/setup-go/upload-artifact fijados a SHA con Node.js 24. Se añade download-artifact v8.0.2, SHA 9000827ccba6bdab643e8b6fd33ac0654aef8333, cuyo manifiesto declara node24 y admite digest-mismatch:error. Referencias verificadas: [manifiesto del SHA](https://github.com/actions/download-artifact/blob/9000827ccba6bdab643e8b6fd33ac0654aef8333/action.yml), [GitHub CLI release create](https://cli.github.com/manual/gh_release_create). El ejecutable gh procede del runner alojado por GitHub; no hay acción externa de publicación ni PAT. Véase docs/releases.md para uso, recuperación de borradores parciales y límites de la verificación local.


## Identidad del módulo Go (D52, 30/10/2026)

El módulo principal y sus imports internos utilizan `github.com/malonso1976/fegan-master-team`; la guía incluida en los ZIP apunta a ese repositorio. El cambio de espacio de nombres no añade dependencias ni modifica go.sum, API, esquema o datos. Las rutas locales y referencias históricas a la implementación Java conservan su procedencia original. Los workflows usan github.repository y no requieren fijar un propietario. Cambiar el módulo no transfiere repositorios ni modifica el remoto Git local.
