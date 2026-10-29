@echo off
setlocal

rem Sobe a instancia de desenvolvimento do MySQL na porta 3307.
rem Ela nao e servico do Windows: existe enquanto esta janela estiver aberta.
rem Criada por scripts\setup-dev-portatil.ps1; detalhes em SETUP-DEV.md.

set "CFG=%USERPROFILE%\jarvis-mysql\my.ini"

rem ZIP portatil do setup tem prioridade; senao usa o MySQL 8.4 instalado pelo MSI
set "MYSQLD=%USERPROFILE%\jarvis-mysql\bin\mysqld.exe"
if not exist "%MYSQLD%" set "MYSQLD=C:\Program Files\MySQL\MySQL Server 8.4\bin\mysqld.exe"

if not exist "%MYSQLD%" (
  echo mysqld.exe nao encontrado. Rode antes:
  echo   powershell -ExecutionPolicy Bypass -File scripts\setup-dev-portatil.ps1
  pause
  exit /b 1
)

if not exist "%CFG%" (
  echo my.ini nao encontrado em:
  echo   %CFG%
  echo Rode scripts\setup-dev-portatil.ps1 para criar a instancia.
  pause
  exit /b 1
)

rem Ja esta no ar? Entao nao sobe outro: o segundo aborta com erro de porta ocupada.
curl -s -o nul telnet://127.0.0.1:3307 2>nul && (
  echo A porta 3307 ja esta ocupada. O banco provavelmente ja esta rodando.
  pause
  exit /b 0
)

echo Subindo MySQL de desenvolvimento na porta 3307...
echo Deixe esta janela aberta. Ctrl+C para parar.
echo.

"%MYSQLD%" --defaults-file="%CFG%" --console
