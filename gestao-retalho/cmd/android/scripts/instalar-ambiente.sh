#!/bin/bash
# Instala (sem sudo, tudo na pasta do utilizador) o necessário para compilar
# o backend Go para Android com gomobile: JDK 17, Android SDK command-line
# tools, plataforma 34, NDK r27c e gomobile/gobind.
#
#   bash cmd/android/scripts/instalar-ambiente.sh
#
# Downloads forçados por IPv4: em redes sem IPv6 funcional (como no WSL de
# desenvolvimento) o curl e o Java tentavam IPv6 primeiro e ficavam a ~7 KB/s.
set -euo pipefail
export PATH=$HOME/.local/go/bin:$HOME/go/bin:/usr/local/bin:/usr/bin:/bin
LOG=$HOME/instalar-ambiente-android.log
exec > >(tee -a "$LOG") 2>&1
echo "== início $(date)"

NDK_VERSAO=27.2.12479018 # = r27c
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
baixar() { curl -4 -fSL --retry 3 -o "$2" "$1"; }

mkdir -p ~/.local ~/Android/Sdk

# 1. JDK 17 (Temurin)
if [ ! -x ~/.local/jdk/bin/javac ]; then
  echo "== JDK 17"
  baixar "https://api.adoptium.net/v3/binary/latest/17/ga/linux/x64/jdk/hotspot/normal/eclipse" "$TMP/jdk.tar.gz"
  rm -rf ~/.local/jdk && mkdir -p ~/.local/jdk
  tar -xzf "$TMP/jdk.tar.gz" -C ~/.local/jdk --strip-components=1
fi
export JAVA_HOME=$HOME/.local/jdk
export PATH=$JAVA_HOME/bin:$PATH
java -version 2>&1 | head -1

# 2. Android command-line tools
export ANDROID_HOME=$HOME/Android/Sdk
if [ ! -x "$ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager" ]; then
  echo "== cmdline-tools"
  baixar "https://dl.google.com/android/repository/commandlinetools-linux-11076708_latest.zip" "$TMP/cmdline-tools.zip"
  unzip -q "$TMP/cmdline-tools.zip" -d "$TMP"
  mkdir -p "$ANDROID_HOME/cmdline-tools" && rm -rf "$ANDROID_HOME/cmdline-tools/latest"
  mv "$TMP/cmdline-tools" "$ANDROID_HOME/cmdline-tools/latest"
fi
SDKM=$ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager
export SDKMANAGER_OPTS="-Djava.net.preferIPv4Stack=true"

# 3. Plataforma 34 (android.jar, usado pelo gomobile para compilar o AAR)
if [ ! -f "$ANDROID_HOME/platforms/android-34/android.jar" ]; then
  echo "== platforms;android-34"
  yes | "$SDKM" --licenses > /dev/null || true
  "$SDKM" --install "platforms;android-34" | grep -v '^\[' || true
fi

# 4. NDK (direto, com progresso visível — são ~650 MB)
if [ ! -d "$ANDROID_HOME/ndk/$NDK_VERSAO" ]; then
  echo "== NDK r27c"
  baixar "https://dl.google.com/android/repository/android-ndk-r27c-linux.zip" "$TMP/ndk.zip"
  unzip -q "$TMP/ndk.zip" -d "$TMP"
  mkdir -p "$ANDROID_HOME/ndk"
  mv "$TMP/android-ndk-r27c" "$ANDROID_HOME/ndk/$NDK_VERSAO"
fi
export ANDROID_NDK_HOME=$ANDROID_HOME/ndk/$NDK_VERSAO

# 5. gomobile/gobind na mesma versão do golang.org/x/mobile do go.mod (as
#    mais recentes exigem Go 1.26; GOTOOLCHAIN=local não deixa trocar o Go).
echo "== gomobile"
export GOTOOLCHAIN=local
X_MOBILE=v0.0.0-20260813181013-1960c775504c
go install "golang.org/x/mobile/cmd/gomobile@$X_MOBILE"
go install "golang.org/x/mobile/cmd/gobind@$X_MOBILE"
gomobile init
gomobile version || true

echo "== fim $(date) OK"
