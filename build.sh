#!/usr/bin/env bash
# Compila OverdriveTuner y su instalador para Windows 64 bits. Requiere Go 1.22+.
set -euo pipefail
cd "$(dirname "$0")"
OUT=dist
mkdir -p "$OUT"

echo "==> Pruebas"
go vet ./...
go test ./internal/...

echo "==> Recursos de Windows (icono + manifiesto)"
go run ./tools/mkres cmd/fpsforge/rsrc_windows_amd64.syso admin fpsforge "$OUT/overdrivetuner.ico"
go run ./tools/mkres cmd/setup/rsrc_windows_amd64.syso admin setup

export GOOS=windows GOARCH=amd64 CGO_ENABLED=0
FLAGS=(-trimpath -ldflags "-H=windowsgui -s -w")

echo "==> Programa"
go build "${FLAGS[@]}" -o cmd/setup/payload/OverdriveTuner.exe ./cmd/fpsforge
cp cmd/setup/payload/OverdriveTuner.exe "$OUT/OverdriveTuner-Portable.exe"

echo "==> Instalador"
go build "${FLAGS[@]}" -o "$OUT/OverdriveTuner-Setup.exe" ./cmd/setup

echo "==> Parche (actualiza una instalación existente)"
go build -trimpath -ldflags "-H=windowsgui -s -w -X main.patchMode=1" -o "$OUT/OverdriveTuner-Patch.exe" ./cmd/setup
rm -f cmd/setup/payload/OverdriveTuner.exe

(cd "$OUT" && sha256sum OverdriveTuner-Setup.exe OverdriveTuner-Patch.exe OverdriveTuner-Portable.exe > SHA256.txt)
ls -la "$OUT"
