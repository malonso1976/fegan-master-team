# Registro de decisiones y dudas

Fecha de inicio: 26/10/2026. Toda decisión nueva debe añadirse aquí, y cualquier duda de alcance o interpretación deportiva debe consultarse con el usuario.

| ID | Estado | Decisión y razón |
|---|---|---|
| D01 | Confirmada por usuario | Destino: marianoao/fegan-master-team, carpeta actual. El repositorio marianoao/marianoao es la referencia Java; su copia local fue facilitada posteriormente. |
| D02 | Confirmada por usuario | Uso local para una persona, sin acceso compartido ni cuentas. |
| D03 | Confirmada por usuario | Licencia única por nadador. Pertenencia a club identificable por código. Búsqueda local e importación por esos identificadores. |
| D04 | Implementada | Ejecutable Go con recursos web embebidos y SQLite sin CGO: distribución simple y portabilidad. |
| D05 | Implementada | Los registros manuales no se sobrescriben al importar. Fichas locales existentes prevalecen; conflicto en nacimiento/categoría aborta toda la importación. |
| D06 | Confirmada por usuario y contrastada | Temporada octubre–septiembre; edad a 31/12 del segundo año. Grupos individuales quinquenales desde 20 y abierto 100+. Sustituye la propuesta inicial de año aislado. |
| D07 | Implementada | Libre utiliza exclusivamente marcas de libre. Las categorías de relevos se comparan explícitamente, separadas de la suma real de edades, conforme a D15. |
| D08 | Implementada | Recomendación explicable mediante puesto y diferencia observados, sin porcentajes que parezcan probabilidades de victoria. No extrapolar marcas entre piscinas o cronometrías. |
| D09 | Implementada | Relevos rivales requieren categoría inscrita y admiten suma real opcional sin conocer integrantes; si se proporcionan, deben ser cuatro licencias distintas y categoría consistente. |
| D10 | Implementada | FEGAN: previa/confirmación, separación por fechas al alcanzar 100 filas, error ante truncamiento irrecuperable, límites de carga y caché temporal. |
| D11 | Implementada | No se incluyen datos personales de la referencia Java como datos de ejemplo. Fixtures sintéticos en pruebas. |
| D12 | Confirmada por usuario | Calendario de temporada octubre–septiembre. Se conservan ventanas de marcas por fechas para analizar una o varias temporadas; la temporada objetivo determina las edades. |
| D13 | Sustituida por D27/D28 | No hay eliminación permanente en esta versión. Baja del equipo y disponibilidad se representan como indicadores editables; las correcciones conservan historial. |
| D14 | Implementada | Mejor marca por club rival y categoría inscrita explícita; puesto observado y margen al tercer rival. Se exige al menos tres clubes rivales para recomendar categoría. |
| D15 | Confirmada por usuario | Un equipo sin premáster de suma 180 puede entrar en +160, +120, +100 y categorías inferiores elegibles. Los de 20–24 solo en +80. Se utiliza la participación en todas las categorías de los apartados 3.3 de ambas circulares, interpretada con la confirmación explícita del usuario, en lugar de imponer máximos por banda natural. |
| D16 | Implementada | Perfiles de invierno y liga 2026/27 con referencia a las circulares suministradas. Otra temporada es simulación con esas reglas, advertida en resultados. |
| D17 | Implementada | El tercer tiempo rival es una referencia competitiva; no representa una probabilidad ni confirma medalla. La normativa de inscripciones y los límites conjuntos requieren revisión humana. |
| D18 | Sustituida por petición del usuario | Go 1.26.2 con el complemento estándar golang de asdf, instalado y verificado. No se usa ivm-golang; no se elimina su instalación previa. |
| D19 | Confirmada por usuario | El club propio se elige primero desde catálogo oficial, conservando código y nombres corto/medio. Se deriva la URL de la temporada actual y se verifica su encabezado; se conserva el origen. |
| D20 | Confirmada por usuario / concretada | Rivales automáticos: ventana móvil de dos años y top diez por licencia, agrupando consultas por grupo, categoría, prueba, piscina y crono. Pruebas relevantes = aquellas con marcas recientes del miembro; no se inventan pruebas sin historial. |
| D21 | Implementada | Trabajo en segundo plano con estado y errores visibles, una ejecución simultánea y lotes transaccionales. Actualiza rivales tras cambios, y permite repetición explícita. |
| D22 | Confirmada por usuario / concretada | Relevos PDF de invierno/verano: dos temporadas más recientes con resultados publicados, descubiertas desde post/57 y memoria deportiva. PDF completo en ZIP, PDF directo o por prueba, siempre desde enlaces oficiales. |
| D23 | Implementada | Lenex oficial del campeonato se usa solo para verificar alias/código de clubes; las marcas importadas proceden de los PDF. Alias desconocidos se comunican sin asignaciones inventadas. |
| D24 | Implementada | Esquema SQLite 2 para configuración/catálogo/estado, con respaldo automático antes de migrar desde 1. Lector PDF en Go sin dependencias nativas, versión fijada compatible con Go 1.26. |
| D25 | Implementada | Si la consulta top diez devuelve filas fuera de los filtros, se intenta el historial completo con los mismos filtros y se obtiene el top diez local. Si sigue siendo inconsistente, se informa del fallo sin importar ese grupo. Las consultas de grupo correctas del día se reutilizan en SQLite para evitar repetirlas tras cada edición; actualizar manualmente permite forzar nueva consulta. |

| D26 | Confirmada por usuario / implementada | Clubes por nombre de base local + código, con orden alfabético español. Pantallas separadas de Clubes y Nadadores; formularios separados de marcas individuales y relevos. |
| D27 | Confirmada por usuario / concretada | Borrado definitivo con confirmación y recuento; cascada de marcas individuales y relevos con licencia explícita. Borrar club elimina sus nadadores actuales y dependencias, además de resultados de ese club; conserva otros nadadores. Se purgan versiones relacionadas y resúmenes de importación. No se purgan copias anteriores. |
| D28 | Implementada | Exclusiones SHA-256 persistentes en esquema 3 impiden restauración por importaciones, incluso concurrentes o de vistas previas anteriores. Se guardan huellas original/corregida de marcas eliminadas. Una alta manual permite recrear identidad, sin restaurar las marcas. Respaldo `.vN-UUID.bak` antes de migrar desde 1/2. Sustituye D13 en cuanto a ausencia de borrado. |
| D29 | Confirmada por usuario / concretada | PDF locales como alternativa a la consulta web: no requieren conexión, tienen vista previa y confirmación, origen local, hash de contenido y nombre de archivo. Verificación local inicialmente desactivada; cronometraje y campeonato se declaran al cargar, fecha del documento si reconocible o fecha de respaldo. Alias desconocidos requieren correspondencia explícita con club conocido. Se mantiene la consulta web opcional y la carga completa existente. |
| D30 | Confirmada por usuario / concretada | Marcas individuales: club obligatorio e inicialmente vacío; nombre/licencia conocidos completan género, grupo de edad según fecha y club actual. Editar conserva el club del resultado histórico hasta cambiar de nadador. Importación FEGAN sin selectores de piscina/crono y categoría vacía = M/F; consulta ambas piscinas y cronometrajes en segundo plano, sin convertir resultados. |
| D31 | Confirmada por usuario / concretada | Elegir pruebas sin temporada/crono: temporada actual del servidor, piscina seleccionable y grupos E/M separados, detalle de rivales y barras de tiempo con diferencias. Componer relevos sin normativa/piscina/crono: criterios comunes de edad y grupos 25/50 × E/M independientes. No se impone por defecto el filtro de invierno; la acreditación de inscripciones sigue requiriendo revisión de la circular concreta. |

| D32 | Confirmada por usuario / concretada | Al cerrar la última pestaña del arranque normal termina el proceso; conexión HTTP persistente autenticada, margen de cinco segundos para recargas y soporte de varias pestañas. Sin conexión inicial no se cierra; -no-browser mantiene servidor independiente. Navegar fuera o perder completamente la conexión también activa el margen. Cierre ordenado de HTTP, trabajos y SQLite; los datos confirmados se conservan. |

| D33 | Confirmada por usuario / concretada | Ajustes permite reiniciar toda la base y volver al club inicial. Confirmación destructiva escribiendo REINICIAR, también exigida por la API; elimina clubes, nadadores, marcas, historial, settings y exclusiones en una transacción. Cancela consultas, suspende productores y espera su finalización antes de vaciar; descarta vistas previas/cachés y renueva token para rechazar escrituras de otras pestañas antiguas. Conserva esquema, archivo SQLite y copias externas. Tras recargar vuelve a consultar catálogo, sin cargar equipo hasta elegir club. |

## Dudas para una siguiente validación funcional

- Confirmar si en el futuro se desea reproducir como indicador adicional la tolerancia heurística por prueba del Java. Se dispone ya de puesto y diferencia sin esta heurística.
- Definir si interesa importar archivos JSON históricos de Java. No son un requisito inicial confirmado y carecen de varios campos necesarios (piscina/crono, fechas de referencia y club).
- Acordar si hace falta planificar varias inscripciones/relevos conjuntamente con límites por participante.

## Incidencias de investigación resueltas

El plugin GitHub devolvió 404 para la referencia Java y Git por HTTPS no pudo autenticar. El usuario facilitó una copia local legible y se completó la revisión desde esa copia. No se modificó ese proyecto.
