#!/usr/bin/env bash
# Compila o backend Go (cmd/android/gobridge) para Android com gomobile e
# grava app/libs/gobridge.aar. Correr no WSL/Linux depois de instalar o
# ambiente (JDK 17, Android SDK + NDK, gomobile — ver README).
#
# Duas arquiteturas no mesmo APK: android/arm (armeabi-v7a, o Sunmi V2 Pro
# é só 32 bits) e android/arm64 (telemóveis e terminais modernos, que já não
# correm 32 bits). Só um alvo: ALVOS=android/arm scripts/build-go.sh
set -euo pipefail

AQUI="$(cd "$(dirname "$0")/.." && pwd)"
RAIZ="$(cd "$AQUI/../.." && pwd)" # gestao-retalho/ (onde está o go.mod)

export JAVA_HOME="${JAVA_HOME:-$HOME/.local/jdk}"
export ANDROID_HOME="${ANDROID_HOME:-$HOME/Android/Sdk}"
if [ -z "${ANDROID_NDK_HOME:-}" ]; then
  ANDROID_NDK_HOME="$(ls -d "$ANDROID_HOME"/ndk/* | sort -V | tail -1)"
fi
export ANDROID_NDK_HOME
# Bibliotecas alinhadas a 16 KB: obrigatório nos Android 15+ com páginas
# de 16 KB (e na Play Store); sem efeito nos outros.
export CGO_LDFLAGS="${CGO_LDFLAGS:-} -Wl,-z,max-page-size=16384"
export PATH="$JAVA_HOME/bin:$HOME/.local/go/bin:$HOME/go/bin:$PATH"

mkdir -p "$AQUI/app/libs"
cd "$RAIZ"
gomobile bind \
  -target="${ALVOS:-android/arm,android/arm64}" \
  -androidapi 21 \
  -javapkg com.gestaoretalho \
  -ldflags "-s -w" \
  -o "$AQUI/app/libs/gobridge.aar" \
  ./cmd/android/gobridge
rm -f "$AQUI/app/libs/gobridge-sources.jar"
ls -la "$AQUI/app/libs/gobridge.aar"
