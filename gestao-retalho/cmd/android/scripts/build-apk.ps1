# Compila o APK no Windows (onde estão o Android SDK e o JDK).
#
# O Gradle não funciona diretamente em \\wsl.localhost\... (falha a bloquear
# ficheiros: "Incorrect function"), por isso o projeto é espelhado para uma
# pasta local e compilado lá. Correr antes, no WSL, scripts/build-go.sh e
# scripts/build-web.sh.
#
#   powershell -File cmd\android\scripts\build-apk.ps1 [-Release] [-Instalar]
#
#   (sem -Release)  APK de desenvolvimento, assinado com a chave de debug.
#   -Release        APK de distribuição, assinado com a chave da empresa
#                   (assinatura.properties, ver README) e copiado para
#                   gestao-retalho\dist\GestaoRetalho-<versão>.apk.
#   -Instalar       instala por adb no terminal ligado e abre a app.
param([switch]$Release, [switch]$Instalar)
$ErrorActionPreference = "Stop"

$origem  = Split-Path -Parent $PSScriptRoot          # cmd\android
$raiz    = Split-Path -Parent (Split-Path -Parent $origem)  # gestao-retalho
$destino = Join-Path $env:LOCALAPPDATA "gestao-retalho\android-build"

if (-not $env:ANDROID_HOME) { $env:ANDROID_HOME = Join-Path $env:LOCALAPPDATA "Android\Sdk" }
if (-not $env:JAVA_HOME) {
    $jdk = Get-ChildItem "C:\Program Files\Microsoft" -Directory -Filter "jdk-*" -ErrorAction SilentlyContinue | Sort-Object Name -Descending | Select-Object -First 1
    if ($jdk) { $env:JAVA_HOME = $jdk.FullName }
}

# Versão única para desktop e Android: productVersion do cmd\admin\wails.json.
$versao = (Get-Content (Join-Path $raiz "cmd\admin\wails.json") -Raw | ConvertFrom-Json).info.productVersion

robocopy $origem $destino /MIR /XD .gradle build /NFL /NDL /NJH /NJS /NP | Out-Null
if ($LASTEXITCODE -ge 8) { throw "robocopy falhou ($LASTEXITCODE)" }

$tarefa = if ($Release) { "assembleRelease" } else { "assembleDebug" }
Push-Location $destino
try {
    & .\gradlew.bat $tarefa "-PretalhoVersao=$versao" --console=plain
    if ($LASTEXITCODE -ne 0) { throw "gradle falhou" }
} finally { Pop-Location }

if ($Release) {
    $apk = Join-Path $destino "app\build\outputs\apk\release\app-release.apk"
    $dist = Join-Path $raiz "dist"
    New-Item -ItemType Directory -Force $dist | Out-Null
    $final = Join-Path $dist "GestaoRetalho-$versao.apk"
    Copy-Item $apk $final -Force
    $apk = $final
} else {
    $apk = Join-Path $destino "app\build\outputs\apk\debug\app-debug.apk"
}
Write-Host "APK ($versao): $apk"

if ($Instalar) {
    $adb = Join-Path $env:ANDROID_HOME "platform-tools\adb.exe"
    & $adb install -r $apk
    & $adb shell am start -n com.gestaoretalho.pos/.MainActivity
}
