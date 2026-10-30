# Prepara o ambiente de dev do Jarvis sem instalador e sem admin:
#   1. MySQL 8.4 portatil em %USERPROFILE%\jarvis-mysql, porta 3307 (nao mexe em outro MySQL do PC)
#   2. banco jarvis + usuario + migration
#   3. backend\.env (se ainda nao existir)
#   4. base sintetica para demo e eval (go run ./cmd/semear -orcamentos)
#
# Pode rodar de novo: cada passo pula o que ja estiver feito.
# Uso, da raiz do repo:  powershell -ExecutionPolicy Bypass -File scripts\setup-dev-portatil.ps1

param(
    [string]$Pasta = "$env:USERPROFILE\jarvis-mysql",
    [int]$Porta = 3307,
    [string]$Versao = "8.4.11",
    [string]$Senha = "jarvis_dev_2026",
    [switch]$ForcarDemo # grava a base sintetica mesmo com dado real no banco
)

$ErrorActionPreference = "Stop"
$Raiz = Split-Path -Parent $PSScriptRoot
$Ini = Join-Path $Pasta "my.ini"
$Dados = Join-Path $Pasta "data"

function Passo($texto) { Write-Host "`n==> $texto" -ForegroundColor Cyan }

# ---------------------------------------------------------------- 1. binarios
# usa o ZIP baixado nesta pasta; se nao tiver, aproveita um MySQL 8.4 ja instalado (MSI)
$Bin = Join-Path $Pasta "bin"
if (-not (Test-Path "$Bin\mysqld.exe") -and (Test-Path "C:\Program Files\MySQL\MySQL Server 8.4\bin\mysqld.exe")) {
    $Bin = "C:\Program Files\MySQL\MySQL Server 8.4\bin"
}

if (-not (Test-Path "$Bin\mysqld.exe")) {
    Passo "Baixando MySQL $Versao (ZIP, ~300 MB)"
    $Zip = Join-Path $env:TEMP "mysql-$Versao-winx64.zip"
    $Url = "https://cdn.mysql.com/Downloads/MySQL-8.4/mysql-$Versao-winx64.zip"
    $ProgressPreference = "SilentlyContinue" # a barra de progresso deixa o download 10x mais lento no PS 5.1
    try {
        Invoke-WebRequest $Url -OutFile $Zip -UseBasicParsing
    } catch {
        # versao antiga sai do CDN principal e vai para o arquivo
        Invoke-WebRequest "https://cdn.mysql.com/archives/mysql-8.4/mysql-$Versao-winx64.zip" -OutFile $Zip -UseBasicParsing
    }

    Passo "Extraindo em $Pasta"
    $Temp = Join-Path $env:TEMP "mysql-extrair"
    if (Test-Path $Temp) { Remove-Item $Temp -Recurse -Force }
    Expand-Archive $Zip -DestinationPath $Temp
    New-Item -ItemType Directory -Force $Pasta | Out-Null
    Move-Item "$Temp\mysql-$Versao-winx64\*" $Pasta -Force
    Remove-Item $Temp -Recurse -Force
    Remove-Item $Zip -Force
    $Bin = Join-Path $Pasta "bin"
}
Write-Host "MySQL: $Bin"

# ---------------------------------------------------------------- 2. my.ini e data dir
if (-not (Test-Path $Ini)) {
    Passo "Criando $Ini"
    New-Item -ItemType Directory -Force $Pasta | Out-Null
    $BaseDir = (Split-Path -Parent $Bin) -replace '\\', '/'
    $DataDir = $Dados -replace '\\', '/'
    @"
[mysqld]
basedir=$BaseDir
datadir=$DataDir
port=$Porta
bind-address=127.0.0.1
mysqlx=OFF
character-set-server=utf8mb4
"@ | Set-Content -Path $Ini -Encoding Ascii
}

if (-not (Test-Path $Dados)) {
    Passo "Inicializando o data dir (root sem senha, so escuta em localhost)"
    & "$Bin\mysqld.exe" --defaults-file="$Ini" --initialize-insecure --console
    if ($LASTEXITCODE -ne 0) { throw "mysqld --initialize falhou" }
}

# ---------------------------------------------------------------- 3. sobe o banco
function PortaAberta {
    $c = New-Object Net.Sockets.TcpClient
    try { $c.Connect("127.0.0.1", $Porta); return $true } catch { return $false } finally { $c.Close() }
}

if (-not (PortaAberta)) {
    Passo "Subindo o MySQL na porta $Porta (janela minimizada; fechar = parar o banco)"
    Start-Process -FilePath "$Bin\mysqld.exe" -ArgumentList "--defaults-file=`"$Ini`"", "--console" -WindowStyle Minimized
    $n = 0
    while (-not (PortaAberta)) {
        Start-Sleep -Seconds 1
        $n++
        if ($n -gt 40) { throw "MySQL nao respondeu na porta $Porta em 40s" }
    }
}
Write-Host "MySQL no ar na porta $Porta"

# ---------------------------------------------------------------- 4. banco, usuario, migration
Passo "Criando banco, usuario e tabelas"
$Sql = "CREATE DATABASE IF NOT EXISTS jarvis CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci; " +
       "CREATE USER IF NOT EXISTS 'jarvis'@'localhost' IDENTIFIED BY '$Senha'; " +
       "GRANT SELECT, INSERT, UPDATE, DELETE ON jarvis.* TO 'jarvis'@'localhost';"
& "$Bin\mysql.exe" -u root --host=127.0.0.1 --port=$Porta -e $Sql
if ($LASTEXITCODE -ne 0) { throw "falha ao criar banco/usuario (o root desta instancia tem senha?)" }

# Aplica cada migration uma vez so, registrando na tabela migracoes.
# A 001 e idempotente (IF NOT EXISTS / INSERT IGNORE), entao banco antigo sem registro nao quebra.
# "source" e nao pipe: o PS 5.1 converte o pipe para ASCII e o acento vira "?"
function MySql($sql) {
    $r = & "$Bin\mysql.exe" -u root --host=127.0.0.1 --port=$Porta --default-character-set=utf8mb4 -N jarvis -e $sql
    if ($LASTEXITCODE -ne 0) { throw "falha no SQL: $sql" }
    return $r
}
MySql "CREATE TABLE IF NOT EXISTS migracoes (nome VARCHAR(100) PRIMARY KEY, aplicada_em TIMESTAMP DEFAULT CURRENT_TIMESTAMP)" | Out-Null
$Feitas = @(MySql "SELECT nome FROM migracoes")
Get-ChildItem (Join-Path $Raiz "backend\migrations\*.sql") | Sort-Object Name | ForEach-Object {
    if ($Feitas -contains $_.Name) { return }
    Write-Host "aplicando $($_.Name)"
    $Migr = $_.FullName -replace '\\', '/'
    MySql "source $Migr" | Out-Null
    MySql "INSERT INTO migracoes (nome) VALUES ('$($_.Name)')" | Out-Null
}

# ---------------------------------------------------------------- 5. .env
$Env = Join-Path $Raiz "backend\.env"
if (-not (Test-Path $Env)) {
    Passo "Criando backend\.env"
    @"
PORTA=8080
MYSQL_DSN=jarvis:$Senha@tcp(127.0.0.1:$Porta)/jarvis?parseTime=true&charset=utf8mb4&loc=Local
API_TOKEN=jarvis_token_dev_troque_isso
# chave gratuita em https://console.groq.com/keys
GROQ_API_KEY=
GROQ_MODELO=openai/gpt-oss-120b
# seu e-mail: assume os dados de antes do login e ve o menu Administracao
DONO_EMAIL=
"@ | Set-Content -Path $Env -Encoding Ascii
}

$TemChave = Select-String -Path $Env -Pattern '^GROQ_API_KEY=\S+' -Quiet
if (-not $TemChave) {
    Write-Host "`nATENCAO: GROQ_API_KEY vazio em backend\.env. O sistema sobe, mas o chat fica desligado." -ForegroundColor Yellow
    Write-Host "Crie a chave em https://console.groq.com/keys e cole depois do '='." -ForegroundColor Yellow
}

# ---------------------------------------------------------------- 6. base sintetica
# so em banco sem dado real: dado ficticio misturado com fatura de verdade estraga os totais
$Reais = [int](MySql "SELECT COUNT(*) FROM transacoes WHERE fonte <> 'sintetico'")
if ($Reais -gt 0 -and -not $ForcarDemo) {
    Write-Host "`nO banco ja tem $Reais lancamentos reais: base sintetica NAO gravada." -ForegroundColor Yellow
    Write-Host "Para o video/eval use um banco separado (ver SETUP-DEV.md, 'Banco de demonstracao')."
} else {
    Passo "Gravando a base sintetica (julho a outubro de 2026)"
    Push-Location (Join-Path $Raiz "backend")
    try {
        go run ./cmd/semear -orcamentos
        if ($LASTEXITCODE -ne 0) { throw "semear falhou" }
    } finally {
        Pop-Location
    }
}

Write-Host "`nPronto. Para usar:" -ForegroundColor Green
Write-Host "  scripts\mysql-dev.bat   (se o banco nao estiver no ar, ex.: depois de reiniciar o PC)"
Write-Host "  dev.bat                 (API + front, abre http://localhost:5173)"
Write-Host "  cd backend; go run ./cmd/eval   (eval; precisa da GROQ_API_KEY)"
