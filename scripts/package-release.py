#!/usr/bin/env python3
"""Package build.sh outputs without including any local profile or database."""
import argparse
import hashlib
from pathlib import Path
import re
import zipfile

TARGETS = [(os_name, arch) for os_name in ("darwin", "linux", "windows")
           for arch in ("amd64", "arm64")]


def instructions(version, commit, os_name, arch):
    executable = "fegan-master-team.exe" if os_name == "windows" else "fegan-master-team"
    launch = {
        "windows": f"Abre {executable}, o ejecuta .\\{executable} desde PowerShell.",
        "darwin": f"Desde Terminal, entra en esta carpeta y ejecuta ./{executable}.",
        "linux": f"Desde una terminal, entra en esta carpeta y ejecuta ./{executable}.",
    }[os_name]
    return f"""FEGAN Master Team {version}
Commit: {commit}
Plataforma: {os_name}/{arch}
amd64 = Intel/AMD de 64 bits; arm64 = ARM de 64 bits (Apple Silicon en macOS).

Extrae todo el ZIP antes de ejecutar. No hace falta instalar Go ni SQLite.
{launch}
En macOS/Linux, si se perdió el permiso de ejecución: chmod +x {executable}
Los ejecutables no están firmados ni notarizados; el sistema puede bloquearlos.

La aplicación abre tu navegador en una dirección de 127.0.0.1 y un puerto libre.
Selecciona el club al iniciar por primera vez. Las consultas FEGAN requieren red;
los datos ya guardados se pueden consultar sin conexión. No es un servidor compartido.
Cerrar la última pestaña termina el proceso tras cinco segundos.

Opciones: -port 8080 -no-browser -data RUTA/team.sqlite
Con -no-browser, abre la URL impresa y pulsa Ctrl+C para terminar el servidor.
La ruta de la base de datos del perfil se imprime al iniciar. Este ZIP no incluye
nadadores, marcas ni bases de datos. Antes de actualizar, descarga una copia desde
la aplicación y detén el proceso anterior. Sustituye solo el ejecutable, conserva
la base; algunas migraciones pueden impedir volver a una versión anterior.

SHA256SUMS, adjunto a la release, permite verificar el ZIP descargado.
macOS: shasum -a 256 NOMBRE.zip
Linux: sha256sum NOMBRE.zip
Windows PowerShell: Get-FileHash NOMBRE.zip -Algorithm SHA256
Compara el valor con la entrada del ZIP en SHA256SUMS.

Documentación y código: https://github.com/malonso1976/fegan-master-team
"""


def add_entry(archive, name, content, mode):
    entry = zipfile.ZipInfo(name)  # Stable timestamp; no local paths in the ZIP.
    entry.create_system = 3
    entry.external_attr = mode << 16
    entry.compress_type = zipfile.ZIP_DEFLATED
    archive.writestr(entry, content)


def package(version, commit, binaries, output):
    if not re.fullmatch(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", version):
        raise ValueError("la versión debe tener formato vX.Y.Z, sin sufijos")
    if not re.fullmatch(r"[0-9a-f]{40}", commit):
        raise ValueError("se requiere el SHA completo del commit")
    sources = []
    for os_name, arch in TARGETS:
        extension = ".exe" if os_name == "windows" else ""
        source = binaries / f"fegan-master-team-{os_name}-{arch}{extension}"
        if not source.is_file() or source.stat().st_size == 0:
            raise ValueError(f"falta un ejecutable no vacío: {source.name}")
        sources.append((os_name, arch, extension, source))
    if output.exists() and any(output.iterdir()):
        raise ValueError("el directorio de salida debe estar vacío; no se sobrescribe")
    output.mkdir(parents=True, exist_ok=True)
    checksums = []
    for os_name, arch, extension, source in sources:
        name = f"fegan-master-team-{version}-{os_name}-{arch}"
        destination = output / f"{name}.zip"
        with zipfile.ZipFile(destination, "x") as archive:
            add_entry(archive, f"{name}/fegan-master-team{extension}", source.read_bytes(), 0o100755)
            add_entry(archive, f"{name}/LEEME.txt",
                      instructions(version, commit, os_name, arch).encode("utf-8"), 0o100644)
        digest = hashlib.sha256(destination.read_bytes()).hexdigest()
        checksums.append(f"{digest}  {destination.name}\n")
        print(destination)
    (output / "SHA256SUMS").write_text("".join(checksums), encoding="utf-8")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("version")
    parser.add_argument("--commit", required=True)
    parser.add_argument("--binaries", type=Path, default=Path("dist"))
    parser.add_argument("--output", type=Path, default=Path("dist/release"))
    args = parser.parse_args()
    try:
        package(args.version, args.commit, args.binaries, args.output)
    except (ValueError, OSError) as error:
        parser.exit(1, f"Error: {error}\n")
