@echo off
setlocal

rem Sobe a instancia de desenvolvimento do MySQL na porta 3307.
rem Ela nao e servico do Windows: existe enquanto esta janela estiver aberta.
rem Detalhes do porque em SETUP-DEV.md.

set "MYSQLD=C:\Program Files\MySQL\MySQL Server 8.4\bin\mysqld.exe"
set "CFG=C:\Users\Dazilio\jarvis-mysql\my.ini"

if not exist "%MYSQLD%" (
  echo mysqld.exe nao encontrado em:
  echo   %MYSQLD%
  pause
  exit /b 1
)

if not exist "%CFG%" (
  echo my.ini nao encontrado em:
  echo   %CFG%
  echo Veja SETUP-DEV.md para recriar a instancia.
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
