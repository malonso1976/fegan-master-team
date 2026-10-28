# Preparar una release

El workflow **Prepare release**, en `.github/workflows/release.yml`, genera un borrador para revisión. Se activa al subir una etiqueta `vX.Y.Z` y también desde **Actions → Prepare release → Run workflow**, indicando una etiqueta existente. No publica automáticamente ni crea etiquetas. Solo admite versiones estables con tres números, sin ceros iniciales ni sufijos (por ejemplo `v1.0.0`). La versión inicial se elige al preparar la primera release.

## Preparación por etiqueta

El commit etiquetado debe incluir el workflow y `scripts/package-release.py`. Revisa que los cambios estén integrados y subidos antes de etiquetar. Ejemplo, sustituyendo `v1.0.0` por la versión que corresponda:

```sh
git switch main
git pull --ff-only
git tag -a v1.0.0 -m 'FEGAN Master Team v1.0.0'
git push origin v1.0.0
```

Subir la etiqueta inicia el workflow. Una etiqueta que no tenga formato válido falla antes de compilar. Para la ejecución manual, el workflow debe estar disponible en la rama predeterminada; selecciona `main` en Run workflow y escribe la etiqueta existente. Se compila el commit de esa etiqueta, no la última revisión de `main`.

## Comprobaciones y archivos

1. Resolver la etiqueta a un SHA completo y validar la versión.
2. Ejecutar `go test ./...`, `go vet ./...` y `go test -race ./...` en Linux, macOS y Windows. Go procede de `.tool-versions` mediante setup-go.
3. Cuando pasan los tres sistemas, `scripts/build.sh` compila seis ejecutables sin CGO en Ubuntu. `scripts/package-release.py` prepara seis ZIP y `SHA256SUMS`, calculado sobre los ZIP.
4. Descargar los paquetes del mismo workflow, verificar sus SHA-256 y comprobar que la etiqueta sigue apuntando al commit validado. Crear un borrador con notas generadas por GitHub.

Cada ZIP se llama `fegan-master-team-vX.Y.Z-SISTEMA-ARQUITECTURA.zip`, con `darwin`, `linux` o `windows` y `amd64` o `arm64`. Contiene una carpeta del mismo nombre con el ejecutable (con `.exe` en Windows) y `LEEME.txt`. La guía incluye versión, commit, arquitectura, arranque, copia de seguridad y verificación de la descarga. Los ZIP conservan el permiso de ejecución para macOS/Linux. Se incluyen exclusivamente los seis ejecutables y las guías, sin bases ni datos de usuario.

`amd64` corresponde a Intel/AMD de 64 bits; `arm64` a ARM de 64 bits, incluido Apple Silicon. Los tres runners prueban su arquitectura nativa disponible; no se ejecutan los seis binarios en todas las arquitecturas. La compilación cruzada no sustituye pruebas de ejecución en el dispositivo de destino.

## Revisar y publicar

Al terminar, abre **Releases**, localiza el borrador y revisa notas, versión y siete adjuntos (seis ZIP y SHA256SUMS). Prueba los ZIP que necesites y publica manualmente cuando proceda. Los binarios no se firman ni notarizan; esta tarea no añade instaladores, actualizaciones automáticas ni certificados.

Una release existente no se sobrescribe. Si una ejecución falla durante la creación o subida de archivos, puede quedar un borrador incompleto: revísalo y resuélvelo manualmente antes de repetir, o usa una versión nueva. Las ejecuciones de una misma etiqueta se serializan sin cancelar otra en curso. No muevas etiquetas usadas para releases.

Los paquetes del workflow se conservan siete días como artefactos de Actions; los adjuntos de la release no dependen de ese plazo. No hace falta un PAT: solo el trabajo `draft` recibe `contents: write` mediante GITHUB_TOKEN. Los demás trabajos tienen `contents: read`; el permiso global está vacío. Las acciones están fijadas por SHA y utilizan Node.js 24. No se ejecuta código del repositorio en el trabajo con escritura, que solo verifica y adjunta archivos.

## Empaquetado local sin publicar

Python 3 (solo biblioteca estándar) se usa para empaquetar y probar los ZIP, no para compilar ni ejecutar la aplicación. Tras `sh scripts/build.sh`:

```sh
python3 -m unittest discover -s scripts -p 'test_*.py'
python3 scripts/package-release.py v1.0.0 --commit "$(git rev-parse HEAD)" --output /tmp/fegan-release-v1.0.0
```

El directorio de salida debe estar vacío o no existir; no se sobrescribe. Comprobar los ZIP desde esa carpeta:

```sh
shasum -a 256 -c SHA256SUMS
```

La ejecución local no crea etiquetas ni contacta con GitHub. Para preparar una distribución oficial, usa el workflow con la etiqueta del código validado. La primera ejecución remota debe comprobarse en Actions: la validación local no acredita permisos, disponibilidad de runners ni creación efectiva del borrador.
