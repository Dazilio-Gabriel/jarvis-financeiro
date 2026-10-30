# Jarvis Financeiro: instala o que faltar, prepara o banco e sobe o sistema. Um comando so.
# Uso (PowerShell, na raiz do repo):
#   powershell -ExecutionPolicy Bypass -File scripts\rodar-tudo.ps1
# Pode rodar quantas vezes quiser: cada passo pula o que ja estiver pronto.

$ErrorActionPreference = "Stop"
$Raiz = Split-Path -Parent $PSScriptRoot
Set-Location $Raiz

function Passo($texto) { Write-Host "`n==> $texto" -ForegroundColor Cyan }

function AtualizarPath {
    $env:Path = [Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
                [Environment]::GetEnvironmentVariable("Path", "User")
}

# ---------------------------------------------------------------- 1. ferramentas
Passo "Conferindo Git, Go e Node"
$Pacotes = @{ git = "Git.Git"; go = "GoLang.Go"; node = "OpenJS.NodeJS.LTS" }
foreach ($cmd in $Pacotes.Keys) {
    if (-not (Get-Command $cmd -ErrorAction SilentlyContinue)) {
        Write-Host "$cmd nao encontrado: instalando $($Pacotes[$cmd]) pelo winget..."
        winget install --id $Pacotes[$cmd] -e --accept-source-agreements --accept-package-agreements
        AtualizarPath
        if (-not (Get-Command $cmd -ErrorAction SilentlyContinue)) {
            throw "$cmd instalado, mas o terminal ainda nao enxerga. Feche e abra o PowerShell e rode de novo."
        }
    }
    Write-Host "ok $cmd"
}

# ---------------------------------------------------------------- 2. codigo atualizado
Passo "Baixando a versao mais nova do repositorio"
git pull --ff-only
if ($LASTEXITCODE -ne 0) { Write-Host "git pull falhou (alteracao local?). Seguindo com o codigo atual." -ForegroundColor Yellow }

# ---------------------------------------------------------------- 3. chave da Groq
$Env = Join-Path $Raiz "backend\.env"
$TemChave = (Test-Path $Env) -and (Select-String -Path $Env -Pattern '^GROQ_API_KEY=gsk_\S+' -Quiet)
if (-not $TemChave) {
    Passo "Chave da Groq"
    Write-Host "Cole a chave (comeca com gsk_). Crie em https://console.groq.com/keys se nao tiver."
    $Chave = (Read-Host "GROQ_API_KEY").Trim()
    if ($Chave -notmatch '^gsk_\S+$') { throw "isso nao parece uma chave da Groq (tem que comecar com gsk_)" }
    if (Test-Path $Env) {
        $Linhas = Get-Content $Env | Where-Object { $_ -notmatch '^GROQ_API_KEY=' }
        $Linhas + "GROQ_API_KEY=$Chave" | Set-Content $Env -Encoding Ascii
    } else {
        $env:JARVIS_GROQ_INICIAL = $Chave # o setup cria o .env; a chave entra logo depois
    }
}

# ---------------------------------------------------------------- 4. banco + dados de demo
Passo "Banco MySQL 8.4 na porta 3307 e base sintetica"
& powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $Raiz "scripts\setup-dev-portatil.ps1")
if ($LASTEXITCODE -ne 0) { throw "setup do banco falhou" }

if ($env:JARVIS_GROQ_INICIAL) {
    $Linhas = Get-Content $Env | Where-Object { $_ -notmatch '^GROQ_API_KEY=' }
    $Linhas + "GROQ_API_KEY=$($env:JARVIS_GROQ_INICIAL)" | Set-Content $Env -Encoding Ascii
}

# ---------------------------------------------------------------- 4b. dono
# so o DONO_EMAIL assume os dados de antes do login e ve o menu Administracao
if (-not (Select-String -Path $Env -Pattern '^DONO_EMAIL=\S+' -Quiet)) {
    Passo "E-mail do dono"
    $Dono = (Read-Host "Seu e-mail (o mesmo que voce vai usar para entrar no Jarvis)").Trim()
    if ($Dono -match '^[^@\s]+@[^@\s]+\.[^@\s]+$') {
        $Linhas = Get-Content $Env | Where-Object { $_ -notmatch '^DONO_EMAIL=' }
        $Linhas + "DONO_EMAIL=$Dono" | Set-Content $Env -Encoding Ascii
    } else {
        Write-Host "E-mail invalido; sem DONO_EMAIL ninguem ve o menu Administracao. Edite backend\.env depois." -ForegroundColor Yellow
    }
}

# ---------------------------------------------------------------- 5. frontend
if (-not (Test-Path (Join-Path $Raiz "frontend\node_modules"))) {
    Passo "Instalando dependencias do frontend (npm install)"
    Push-Location (Join-Path $Raiz "frontend")
    npm install --no-audit --no-fund
    Pop-Location
}

# ---------------------------------------------------------------- 6. sobe tudo
Passo "Subindo API + frontend (abre o navegador em http://localhost:5173)"
Start-Process -FilePath (Join-Path $Raiz "dev.bat") -WorkingDirectory $Raiz

Write-Host "`nPronto. Menu 'Assistente' = chat com a IA." -ForegroundColor Green
Write-Host "Testes: cd backend; go test ./...      Eval: cd backend; go run ./cmd/eval  (~6 min)"
