@echo off
title Gestao de Retalho - Ponto de Venda (PDV)
set "WAILS_DEV_ALLOWED_HOSTS=wails.localhost,wails.localhost:34115,localhost,localhost:34115,127.0.0.1,127.0.0.1:34115"
set "PATH=C:\Users\MR BETO\go\bin;C:\Program Files\Go\bin;%PATH%"
cd /d "%~dp0gestao-retalho\cmd\app"
echo ========================================================
echo  Iniciando Ponto de Venda / PDV (cmd/app)...
echo ========================================================
wails dev
pause

