# Instrucciones para agentes

- Producto local para una persona, escrito en Go, distribuible en Windows, macOS y Linux. Interfaz web embebida, servidor solo en loopback.
- Lee ARCHITECTURE.md, docs/regulations.md y docs/decisions.md antes de cambiar contratos o reglas deportivas. Documenta toda decisión, duda y limitación en esos documentos; consulta al usuario las dudas que afecten al alcance o a la interpretación de normativa.
- Licencia de nadador e identificador de club son cadenas: conserva ceros iniciales. La licencia identifica de forma única a una persona. No fusiones por nombre.
- Conserva la procedencia, los datos históricos y las correcciones manuales. Importaciones transaccionales e idempotentes. Nunca inventes resultados, reglas oficiales, probabilidades o equivalencias de piscina/cronometría.
- Separa dominio, persistencia, adaptador FEGAN e interfaz. No añadas servicios externos obligatorios ni herramientas de compilación de interfaz sin justificarlo.
- Usa tiempos enteros en centésimas. No mezcles piscinas, categorías, parciales ni cronometrajes en comparativas. Identifica las estimaciones y la cobertura del conjunto observado.
- No alteres archivos personales ni bases de datos del usuario durante pruebas. Usa directorios temporales. No registres licencias o nombres en logs.
- Verificación: `go test ./...`, `go vet ./...`, `go test -race ./...` y compilación mediante `scripts/build.sh`. Prueba con fixtures anónimos el contrato FEGAN; una prueba unitaria no acredita disponibilidad externa.
- Antes de entregar, actualiza README.md y docs, registra comprobaciones y limitaciones reales. No publiques ni envíes datos a GitHub sin petición del usuario.

- Temporada octubre–septiembre; edad a 31/12 del segundo año. No vuelvas al incremento en septiembre del Java. La categoría inscrita de relevo es independiente de la suma real; el usuario confirmó categorías inferiores elegibles y premáster solo +80. No atribuyas a la circular una interpretación que provenga de esta aclaración sin distinguirla.

- Usa Go mediante el complemento estándar golang de asdf, fijado en .tool-versions; no ivm-golang. Inicio por club obligatorio; preserva nombres corto/medio/código. Rivales automáticos top diez por licencia en ventana móvil de dos años, con consultas compartidas por grupo/prueba/piscina/crono. PDF de relevos: dos últimas temporadas publicadas desde enlaces oficiales, sin inventar alias de clubes ni marcas de documentos inaccesibles.

- La interfaz tiene entradas independientes para Nadadores y Clubes, y formularios separados para marcas individuales y relevos. Clubes siempre por nombre guardado + código y orden alfabético. En marcas individuales el club empieza vacío y solo se propone el actual al elegir un nadador; al editar se respeta el club histórico.
- Borrado definitivo transaccional: nadador y sus marcas, incluidos relevos con su licencia; club y sus nadadores actuales más marcas dependientes. Borra versiones relacionadas y evita resurrecciones por importaciones usando exclusiones hash persistentes. No pruebes sobre datos personales. Las copias antiguas no se purgan.
- Admitir PDF locales sin conexión con vista previa, procedencia local y correspondencias explícitas para clubes desconocidos. No atribuir verificación federativa a un archivo subido. Al quitar selectores de piscina/crono, mantener grupos de resultados separados; no imponer silenciosamente 25 m electrónico ni un perfil de invierno.

- En arranque normal, cerrar la última pestaña termina el proceso tras cinco segundos de margen; recargas y otras pestañas deben mantenerlo. No ligar el cierre al proceso del lanzador ni usar solo latidos JavaScript sujetos a suspensión. `-no-browser` conserva el servidor independiente. Cerrar ordenadamente HTTP, trabajos y SQLite.

- Reinicio global solo tras confirmación explícita REINICIAR en interfaz y API. Detén productores y cancela peticiones de la generación antigua antes de vaciar transaccionalmente datos, historial, configuración y exclusiones; invalida vistas previas/cachés y tokens antiguos. Conserva copias externas y esquema. Prueba solo con bases temporales; nunca reinicies la base del perfil del usuario durante verificaciones.

- Reglas de relevos por tipo ChampionshipRules/LeagueRules, sin URLs ni temporadas fijas en el dominio. El corte 01/09/2023 de campeonato es histórico, no cambia con el año de la temporada.
- Oportunidades: una fila por prueba en la piscina elegida, mejor marca histórica guardada propia y rivales de dos años móviles; usar el cronometraje de la mejor propia (confirmado por usuario). No mezclar E/M; empate propio E/M prefiere E. Marcas: mejores por licencia/prueba/piscina/crono/categoría dentro de filtros, sin borrar lo oculto.
