# FEGAN Master Team

Aplicación local en Go para gestionar un equipo de natación máster desde el navegador del propio ordenador. Clubes y nadadores por licencia, historial de marcas, importación de FEGAN, comparativas individuales y composiciones de relevos.

## Ejecutar

Con Go 1.26 o posterior (asdf estándar: `asdf install golang 1.26.2`; `.tool-versions` usa `golang`):

```sh
go run ./cmd/fegan-master-team
```

Se abrirá una dirección `http://127.0.0.1:PUERTO`. La aplicación elige un puerto disponible y guarda los datos en el perfil de usuario. Al cerrar la última pestaña de la aplicación, el proceso termina automáticamente tras cinco segundos de margen para permitir recargas. Con `-no-browser`, el servidor permanece abierto hasta pulsar Ctrl+C. Opciones:

```sh
go run ./cmd/fegan-master-team -port 8080 -no-browser -data ./local-data/team.sqlite
```

Una vez compilada no requiere instalar Go ni una base de datos. Funciona sin conexión salvo para las consultas a FEGAN. No está diseñada para acceso compartido ni escucha en la red local.

## Uso inicial

1. Selecciona primero tu club del catálogo FEGAN de la temporada actual, con código y nombres corto/medio.
2. La aplicación carga en segundo plano sus marcas de los dos últimos años, los diez mejores rivales por grupo y prueba, y los relevos de invierno/verano de las dos últimas temporadas con resultados. El progreso y cualquier incidencia permanecen visibles.
3. Añade manualmente resultados ausentes, incluidos relevos de clubes rivales por categoría de edad. También puedes subir PDF locales de campeonatos y revisar sus marcas antes de guardarlas, sin conexión.
4. Compara pruebas y calcula relevos con integrantes disponibles, detalle de rivales y grupos comparables separados.
5. Para empezar de nuevo, usa Ajustes → Reiniciar base de datos y confirma escribiendo REINICIAR.
6. Gestiona por separado clubes y nadadores; los borrados incluyen marcas dependientes y requieren confirmación. Descarga una copia SQLite desde la barra lateral.

Los cálculos son estimaciones sobre los datos guardados. La temporada va de octubre a septiembre y usa la edad a 31/12 del segundo año. Se comparan categorías de relevos elegibles según los criterios comunes de edad de invierno/liga 2026/27 y la aclaración del usuario. No presenta probabilidades de medalla ni clasificaciones oficiales.

## Desarrollo y distribución

```sh
go test ./...
go vet ./...
go test -race ./...
sh scripts/build.sh
```

El script genera ejecutables para Windows, macOS y Linux, amd64/arm64, en `dist/`. La compilación cruzada no sustituye pruebas de ejecución en cada sistema. Véase [verificación](docs/verification.md).

- [Arquitectura](ARCHITECTURE.md)
- [Instrucciones de trabajo](AGENTS.md)
- [Guía funcional](docs/functional.md)
- [Guía técnica y API](docs/technical.md)
- [Decisiones y dudas](docs/decisions.md)
- [Reglas de temporada y relevos](docs/regulations.md)
- [Referencia Java y diferencias](docs/java-reference.md)

Aplicación independiente; FEGAN es la fuente externa de los resultados importados. No se incluyen datos personales de muestra.
