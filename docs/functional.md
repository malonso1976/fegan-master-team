# Guía funcional

## Arranque y club del equipo

Ejecuta el archivo de tu sistema. Abre una dirección local en el navegador; el proceso termina automáticamente cinco segundos después de cerrar su última pestaña. Recargar o mantener otra pestaña abierta no lo cierra. Con `-no-browser`, usa Ctrl+C para salir. Los datos permanecen en el ordenador. Sin conexión puedes gestionar, cargar PDF locales y calcular sobre las marcas guardadas.

La primera vez elige tu club: busca por código o nombres largo, medio y corto del catálogo de la temporada actual. Puedes usar el catálogo guardado o crear un club manualmente si no hay conexión. Los desplegables muestran nombre y código, ordenados alfabéticamente. Las correcciones de nombres guardadas en tu base prevalecen sobre el catálogo.

La selección inicia la actualización del club, rivales y fuentes de relevos con progreso y advertencias. Puede tardar varios minutos; puedes seguir trabajando. Revisa traslados de club y disponibilidad tras la primera carga. No se incluyen datos de demostración.

## Clubes y Nadadores

Son dos entradas distintas del menú. En Clubes puedes crear y editar código, nombre largo, medio y corto. En Nadadores puedes crear y editar licencia, nombre, nacimiento, categoría y club actual. Licencia y código conservan ceros iniciales; no se identifica a personas solo por su nombre.

Busca nadadores por nombre, licencia, nombre o código de club. Los indicadores miembro y disponible determinan quién puede entrar en las propuestas de relevos. Para una baja temporal basta con desmarcarlos. Un traslado cambia su club actual, conservando el club histórico de cada resultado.

Borrar un nadador elimina sus marcas individuales en cualquier club y los relevos guardados que lo incluyan expresamente entre sus licencias. Borrar un club elimina sus nadadores actuales con sus dependencias y todas las marcas registradas con ese club; conserva a los nadadores de otros clubes aunque tengan resultados históricos en él. Los resultados de relevos sin integrantes conocidos no pueden atribuirse a una persona concreta.

Antes del borrado se muestra el número de nadadores y marcas afectados y se solicita confirmación. Se eliminan sus versiones anteriores y los resúmenes de importación; no hay papelera ni deshacer. Las actualizaciones e importaciones omiten los registros eliminados para que no reaparezcan. Puedes crear manualmente de nuevo un club o licencia, pero no se restauran sus marcas. Si borras el club del equipo, tendrás que elegir uno al continuar. El catálogo de referencia y las copias de seguridad anteriores no se purgan.

## Marcas

Hay dos formularios desplegables: Marca individual y Marca de relevos. Indica prueba, tiempo, piscina, cronometraje, fecha, lugar y notas. Ejemplos de tiempo: `28.45`, `1:02.34`, `18:29.35`. Verificada por mí es una revisión local, sin certificar homologación federativa.

En la marca individual busca un nadador existente por nombre o licencia. Si el nombre no es único, elige la licencia. Se completa su categoría masculina/femenina y grupo de edad usando la temporada de la fecha del resultado. Se propone su club actual; puedes cambiarlo si representa a otro club en ese resultado. El club es obligatorio y queda vacío cuando no has elegido nadador. Al editar se conserva el club histórico registrado.

Para relevos indica categoría masculina/femenina/mixta y la categoría de edad inscrita (+80, +100, etc.). La suma real de edades y las cuatro licencias son opcionales si no están publicadas. Si indicas integrantes deben existir, ser diferentes y cumplir edad, categoría y composición; el orden corresponde a los parciales. Si además indicas la suma, debe coincidir. No se inventan parciales de un relevo.

Los relevos aparecen como «Relevo masculino · +160», «Relevo femenino · +160» o «Relevo mixto · +160», también en los resultados recientes y las vistas previas de importación. El listado permite buscar por nadador, club o etiqueta de relevo (por ejemplo, `femenino` o `+160`), filtrar por prueba, fechas, categoría masculina/femenina/mixta y tipo individual/relevo, editar o borrar. Los filtros se combinan. Al seleccionar solo relevos se desmarca y deshabilita Solo mejores individuales por nadador para evitar ocultarlos; al volver a individuales o ambos tipos se habilita de nuevo, sin activarse automáticamente. Pulsa las cabeceras Fecha, Nadador/relevo, Prueba, Tiempo, Piscina/crono, Club u Origen para ordenar; vuelve a pulsar para invertir el orden. La flecha indica la dirección. Solo mejores individuales por nadador muestra el mejor tiempo por licencia, prueba, piscina, cronometraje y categoría dentro de los filtros activos; omite relevos y no agrupa por club histórico. Los empates conservan el registro más reciente. No se modifican ni borran las marcas que quedan ocultas. Editar conserva la procedencia y versión anterior; borrar elimina ambas y evita la reimportación de esa marca. Los registros borrados no se consultan en el historial. «Historial de correcciones» muestra hasta las 500 versiones o resúmenes recientes restantes.

## Actualización e importación de FEGAN

La actualización automática consulta dos años móviles de resultados del club, M/F, piscinas 25/50 y cronometrajes E/M. Después carga rivales para las pruebas con marcas históricas guardadas de los miembros (excluyendo fechas futuras), compartiendo consultas por grupo de edad/prueba/piscina/crono. Obtiene hasta diez referencias distintas por licencia, sin equivalencias y sin repetir consultas correctas del día, salvo actualización forzada.

Las altas y ediciones de nadadores y marcas actualizan rivales. Los botones Actualizar todo el club, Actualizar rivales y Consultar relevos en FEGAN permiten repetir la carga. Solo hay un trabajo automático simultáneo; se muestran incidencias y cobertura incompleta. El borrado no inicia una nueva consulta.

En Importar de FEGAN puedes consultar otros periodos, clubes o licencias:

1. Acota por club, licencia o intervalo de nacimientos. Marca incorporar como miembros solo para el club correspondiente.
2. Selecciona prueba o todas las individuales, fechas y categoría. Categoría vacía consulta masculina y femenina. No necesitas escoger piscina ni cronometraje: se consultan ambos tipos por separado.
3. La consulta se prepara en segundo plano; una búsqueda amplia puede tardar varios minutos. No se guardan lotes parciales si falla esta consulta manual. Mantén la página abierta para recibir su vista previa.
4. Revisa los resultados con nombre/código de club, piscina y cronometraje; confirma para guardar. Los duplicados, registros borrados y correcciones manuales se respetan; un conflicto de identidad revierte la transacción.

La vista previa caduca 15 minutos después de estar preparada. Los errores externos se muestran como incidencias, sin presentarlos como una consulta vacía.

## Relevos desde PDF locales

En Importar de FEGAN está el formulario Relevos desde un PDF local. No se necesita conexión ni se envía el documento a FEGAN. Selecciona un archivo por carga (hasta 20 MiB y 400 páginas), invierno/verano, fecha de respaldo y cronometraje del documento. La fecha reconocida en los encabezados prevalece sobre la fecha de respaldo. He verificado el documento empieza desmarcado.

Pulsa Leer PDF y revisar resultados. Se reconocen pruebas, categorías y tiempos; bajas, descalificaciones y parciales quedan fuera. La vista previa muestra los clubes desconocidos. Para vincular uno, añade una línea `nombre que aparece en el PDF = código` en Correspondencias y vuelve a leer el archivo. El código debe existir en la base o catálogo guardado. Revisa y confirma las marcas; las filas no vinculadas no se guardan. Puedes cargar sucesivamente varios PDF completos o por prueba. No hay OCR para PDF escaneados sin texto.

El origen queda identificado como local-pdf con nombre de archivo y huella de contenido; no se guarda el archivo ni su ruta. Los duplicados se omiten, también entre web y fichero si coincide el contenido deportivo. La consulta web de campeonatos sigue disponible como alternativa: obtiene invierno/verano de las dos últimas temporadas con resultados publicados desde fuentes oficiales. Los PDF locales también permiten completar documentos web inaccesibles.

## Elegir pruebas

Busca por nombre, licencia o club actual (código o nombres largo, medio y corto). La búsqueda ignora acentos y mayúsculas y acepta varias palabras, aunque aparezcan en distinto orden, por ejemplo `costa alvaro`. Si solo hay una coincidencia se selecciona automáticamente; si hay varias elige la licencia correcta. El desplegable muestra nombre, licencia y club. No se fusionan personas por nombre.

Elige piscina y, opcionalmente, solo marcas verificadas. Se toma la mejor marca histórica guardada de cada prueba, sin limitar la referencia propia a los dos últimos años. Para comparar un historial más amplio, incorpora previamente esas temporadas: no se afirma disponer de toda la trayectoria FEGAN de cada nadador.

Cada prueba se muestra una sola vez. Su cronometraje es el de la mejor marca propia; los rivales se comparan únicamente con el mismo cronometraje y piscina. Un empate de tiempo propio entre electrónico y manual elige electrónico; otros empates conservan la referencia más reciente. La fecha propia queda visible en la tabla. Los rivales aportan su mejor marca compatible dentro de los dos últimos años móviles, hasta diez licencias rivales distintas disponibles, excluyendo al nadador seleccionado.

El grupo de edad se calcula con la temporada actual de octubre a septiembre y la edad a 31 de diciembre del segundo año. Las fechas de las marcas históricas no cambian ese grupo objetivo. Se muestran puesto observado, rivales y diferencia al mejor; los empates comparten puesto. No son probabilidades ni clasificaciones oficiales.

Abre Reconocer a los rivales y comparar marcas para ver nombre, licencia, año de nacimiento, club del resultado, tiempo, fecha y lugar. Tu nadador está destacado en las barras. La diferencia es tiempo rival menos propio: negativa indica un rival más rápido. Otros integrantes del mismo club también pueden ser rivales del grupo. Una marca antigua del rival queda fuera aunque siga siendo su mejor registro histórico; la mejor propia puede ser anterior.

## Componer relevos

Selecciona club, prueba, masculina/femenina/mixta, temporada, categoría de edad o todas, fechas y número de propuestas. No se solicita normativa, piscina o cronometraje. Se calculan independientemente los grupos 25/50 m × electrónico/manual, sin mezclar ni convertir referencias. Se aplican criterios comunes de edad de las circulares; revisa acreditación y límites de participación de la competición antes de inscribir.

Solo participan miembros disponibles del club con marca en distancia y estilo requeridos. Libre usa cuatro parciales de libre; estilos asigna espalda, braza, mariposa y libre. Mixto requiere dos mujeres y dos hombres. La suma estimada no incluye bonificación por salida. Las propuestas son alternativas independientes y pueden compartir integrantes.

Se comparan +80, +100, +120, +160, +200, +240, +280 y +320. Los no premáster pueden escoger categorías inferiores cuya suma mínima alcancen; un equipo de 180 años puede entrar en +160, +120, +100 y +80. Si incluye a alguien de 20–24 años, solo +80, incluso si suma más de 99. Véase [normativa](regulations.md).

Para cada grupo comparable se muestra mejor estimación, puesto frente a clubes rivales y diferencia al tercer rival. Con al menos tres clubes rivales se destaca la categoría más favorable por puesto y margen relativo sobre el tercero. Sin suficientes referencias se muestran propuestas sin inventar posibilidades de medalla. Las referencias rivales usan su categoría inscrita explícita, no una categoría deducida de la suma real.

## Reiniciar desde cero

En Ajustes pulsa Reiniciar base de datos. El diálogo muestra el alcance y los recuentos actuales; escribe exactamente `REINICIAR` para habilitar Borrar todos los datos y empezar de nuevo. Cancelar o cerrar el diálogo no modifica nada. Puedes descargar una copia de seguridad antes desde la misma pantalla.

Se eliminan todos los clubes, nadadores, marcas, historial, configuración, catálogo guardado y exclusiones de registros eliminados. Se detienen las actualizaciones y consultas pendientes y se descartan las vistas previas de importación. No hay deshacer en la aplicación; las copias de seguridad que ya hayas guardado se conservan.

Al confirmar se recarga la pantalla inicial y debes elegir un club de nuevo. El catálogo se vuelve a consultar; no se carga ningún equipo hasta elegirlo. El archivo de la base conserva su ruta. Las otras pestañas deben recargarse para poder guardar de nuevo, evitando restauraciones accidentales desde formularios antiguos.

## Copias y restauración

Descarga regularmente una copia SQLite desde la barra lateral. Contiene datos, historial, configuración y exclusiones de borrado. Las migraciones crean también un respaldo anterior; las copias previas a un borrado pueden contener lo eliminado y quedan bajo tu control.

Para restaurar cierra la aplicación, conserva la base actual y sustitúyela por la copia, o abre la copia con `-data /ruta/a/copia.sqlite`. No sobrescribas una base activa. No hay cifrado ni sincronización automática de copias.
