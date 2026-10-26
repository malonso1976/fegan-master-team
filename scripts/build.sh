#!/bin/sh
set -eu
mkdir -p dist
for target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
  target_os=${target%/*}
  target_arch=${target#*/}
  extension=
  if [ "$target_os" = windows ]; then extension=.exe; fi
  output="dist/fegan-master-team-$target_os-$target_arch$extension"
  CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" go build -trimpath -ldflags='-s -w' -o "$output" ./cmd/fegan-master-team
  echo "Generado: $output"
done
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum dist/fegan-master-team-* > dist/SHA256SUMS
else
  shasum -a 256 dist/fegan-master-team-* > dist/SHA256SUMS
fi
