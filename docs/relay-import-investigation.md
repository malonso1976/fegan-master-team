# Investigación de las marcas de relevos

Fecha: 27/10/2026. Petición: averiguar el motivo y revisar conjuntamente la solución antes de aplicarla.

## Hallazgo reproducido

El listado de Marcas, los resultados recientes y la vista previa de importación omiten el sexo del relevo. Muestran «Relevo +160», por ejemplo, tanto para masculino como femenino y mixto. La búsqueda tampoco incorpora ese dato. Así, resultados distintos pueden parecer el mismo relevo con tiempos contradictorios.

Se ejecutó `markRows`, de `internal/web/static/app.js`, con tres registros ficticios M/F/X: produjo exactamente la misma etiqueta para los tres. `showPreview` tiene la misma omisión. El formulario de edición sí muestra el sexo guardado. Las comparativas del dominio filtran explícitamente por `Gender`.

Además, «Solo mejores individuales por nadador» excluye todos los relevos. Es el comportamiento previsto del filtro, pero puede contribuir a la impresión de que faltan resultados. No existe un filtro específico de sexo en el listado.

## Contraste con las fuentes

Se volvieron a consultar los documentos oficiales descubiertos desde [resultados máster](https://www.fegan.org/post/57) y su archivo de temporadas, sin importar datos:

| Campeonato | Masculinos | Femeninos | Mixtos | Total |
|---|---:|---:|---:|---:|
| Verano 2026 | 62 | 49 | 75 | 186 |
| Invierno 2026 | 56 | 47 | 75 | 178 |
| Verano 2025 | 56 | 31 | 66 | 153 |
| Invierno 2025 | 59 | 45 | 80 | 184 |
| Total | 233 | 172 | 296 | 701 |

La base del perfil se abrió exclusivamente mediante SQLite `mode=ro` y `query_only=ON`, sin usar el inicializador de la aplicación. Sus 701 marcas `fegan-pdf` coinciden con la nueva extracción en club, prueba, sexo, categoría inscrita, piscina, cronometraje, fecha y tiempo. No se realizaron escrituras.

Una segunda extracción mediante pypdf en modo de disposición visual, independiente del lector PDF Go, reproduce para los cuatro campeonatos el mismo multiconjunto de sexo, prueba, categoría y tiempo: 186, 178, 153 y 184 resultados. Este contraste agregado no certifica por sí solo todas las correspondencias de club, fechas o piscinas.

Ejemplo contrastado también mediante imágenes renderizadas de las páginas originales: CIDSANTIAGO, 4×50 libre, +160, 13/06/2026, piscina de 25 m electrónica:

| Sexo | Tiempo del PDF y de la base |
|---|---|
| Masculino | 1:43.94 |
| Femenino | 2:14.18 |

Fuentes: [prueba masculina 17](https://www.fegan.org/fegan/mm/tprd/2526/20260613_cto_gal_mast_ver/ResultList_17.pdf) y [prueba femenina 16](https://www.fegan.org/fegan/mm/tprd/2526/20260613_cto_gal_mast_ver/ResultList_16.pdf). Ambos aparecen en el listado como «Relevo +160».

## Propuesta revisada con el usuario

1. Mostrar masculino/femenino/mixto junto a la categoría de edad en listados, resultados recientes y vistas previas.
2. Permitir filtrar por sexo y tipo de resultado (individual/relevo), manteniendo separados piscina y cronometraje.
3. Ampliar fixtures anónimos para M/F/X y continuaciones entre páginas, y verificar la información visible.

El usuario confirmó que la omisión visual explica el caso y pidió preparar la corrección. Se ha aplicado una etiqueta compartida que distingue M/F/X y categoría inscrita, filtros combinables de tipo y categoría, búsqueda por la etiqueta y desactivación de mejores individuales al elegir solo relevos. Se verificó con datos ficticios el renderizado M/F/X, la vista previa y las combinaciones de filtros; también se comprobó en el navegador. No se cambió el lector PDF, por lo que no se añadieron nuevos fixtures del contrato externo en esta corrección visual. La ampliación de fixtures para formatos y continuaciones sigue siendo una mejora futura.

No se modificó la base del usuario ni se requiere borrar, reiniciar o reimportar. No se afirma compatibilidad con todos los formatos de PDF ni que repetir el mismo importador certifique por sí solo la corrección de los datos.
