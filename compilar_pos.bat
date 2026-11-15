@echo off
title Compilar Ponto de Venda (PDV)
set "PATH=C:\Users\MR BETO\go\bin;C:\Program Files\Go\bin;%PATH%"
cd /d "%~dp0gestao-retalho\cmd\app"
echo ========================================================
echo  Compilando Ponto de Venda / PDV (gestao-retalho.exe)...
echo ========================================================
wails build
echo ========================================================
echo  Concluido com sucesso!
echo ========================================================
pause
