@echo off
setlocal
cd /d "%~dp0"

rem Mesma porta default do main.go, e respeita PORTA se voce exportar.
if "%PORTA%"=="" set "PORTA=8080"

rem O PATH desta maquina nem sempre traz o node.
if exist "C:\Program Files\nodejs\npm.cmd" set "PATH=C:\Program Files\nodejs;%PATH%"

start "Jarvis API (Go)" cmd /k "cd /d %~dp0backend && go run ./cmd/servidor"

rem O Vite so pode subir depois que a API responder. Se ele subir antes, o proxy
rem do /api devolve 502 ate o Go terminar de compilar.
echo Compilando a API e esperando a porta %PORTA%...
set /a lnTent=0

:esperar
set /a lnTent+=1
if %lnTent% gtr 30 (
  echo.
  echo A API nao subiu em 30s. Veja o erro na janela "Jarvis API (Go)".
  pause
  exit /b 1
)
ping -n 2 127.0.0.1 >nul
curl -s -o nul http://localhost:%PORTA%/api/saude || goto esperar

start "Jarvis Front (Vite)" cmd /k "cd /d %~dp0frontend && npm run dev"

rem O Vite leva ~3s para servir; abrir antes disso mostra tela de erro.
ping -n 5 127.0.0.1 >nul
start "" http://localhost:5173

echo.
echo API   http://localhost:%PORTA%
echo Front http://localhost:5173
echo.
echo Para parar: feche as duas janelas que abriram.
