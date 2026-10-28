#!/usr/bin/env bash
# Compila os frontends do admin (cmd/admin) e do PDV (cmd/app) para a
# WebView do terminal (vite --mode android) e copia-os para
# app/src/main/assets/web/{admin,pdv}, com a ponte JS injetada antes do
# bundle. Correr dentro do WSL/Linux (precisa de node).
#
#   scripts/build-web.sh              → ponte real (backend Go no terminal)
#   PONTE=demo scripts/build-web.sh   → ponte de demonstração (4 lojas de exemplo)
set -euo pipefail

AQUI="$(cd "$(dirname "$0")/.." && pwd)"
CMD="$(cd "$AQUI/.." && pwd)"
ASSETS="$AQUI/app/src/main/assets/web"

rm -rf "$ASSETS"
mkdir -p "$ASSETS"
if [ "${PONTE:-real}" = "demo" ]; then
  cp "$AQUI/web/ponte-demo.js" "$ASSETS/ponte.js"
else
  cp "$AQUI/web/ponte.js" "$ASSETS/ponte.js"
fi

# compilar <pasta em cmd/> <destino em assets/web/>
compilar() {
  local origem="$CMD/$1/frontend" destino="$ASSETS/$2"
  cd "$origem"
  [ -d node_modules ] || npm ci --no-audit --no-fund
  if [ ! -d wailsjs ]; then
    echo "Falta cmd/$1/frontend/wailsjs: correr 'wails generate module' em cmd/$1" >&2
    exit 1
  fi
  npx vite build --mode android
  mkdir -p "$destino"
  cp -r dist-android/. "$destino/"
  # A ponte tem de existir antes de qualquer script da app.
  sed -i 's#<head>#<head>\n  <script src="../ponte.js"></script>#' "$destino/index.html"
}

compilar admin admin
compilar app pdv
echo "OK: $ASSETS"
