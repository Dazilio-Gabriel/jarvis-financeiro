# Configura o MySQL 8.4 recém-instalado: cria o data dir, registra o serviço
# do Windows, sobe o serviço e define a senha do root.
#
# O instalador MSI do MySQL só copia os binários — nada disso vem pronto.
#
# COMO RODAR: abra o PowerShell COMO ADMINISTRADOR e execute:
#
#     powershell -ExecutionPolicy Bypass -File C:\jarvis_financeiro\scripts\setup-mysql.ps1
#
# Roda uma vez só. Se rodar de novo com o serviço já criado, ele avisa e sai.

#Requires -RunAsAdministrator
$ErrorActionPreference = 'Stop'

$base    = 'C:\Program Files\MySQL\MySQL Server 8.4'
$dataRoot= 'C:\ProgramData\MySQL\MySQL Server 8.4'
$data    = Join-Path $dataRoot 'Data'
$cfg     = Join-Path $dataRoot 'my.ini'
$mysqld  = Join-Path $base 'bin\mysqld.exe'
$mysql   = Join-Path $base 'bin\mysql.exe'
$servico = 'MySQL84'

if (-not (Test-Path $mysqld)) {
    throw "mysqld.exe nao encontrado em $mysqld. O MySQL 8.4 esta instalado?"
}

if (Get-Service -Name $servico -ErrorAction SilentlyContinue) {
    Write-Host "O servico '$servico' ja existe. Nada a fazer." -ForegroundColor Yellow
    Get-Service -Name $servico | Format-Table Name, Status, StartType -AutoSize
    exit 0
}

# --- 1. Arquivo de configuracao -------------------------------------------------
Write-Host "[1/5] Criando my.ini..." -ForegroundColor Cyan
New-Item -ItemType Directory -Path $dataRoot -Force | Out-Null

@"
[mysqld]
basedir=$base
datadir=$data
port=3306
character-set-server=utf8mb4
collation-server=utf8mb4_0900_ai_ci
default-storage-engine=InnoDB

[client]
port=3306
default-character-set=utf8mb4
"@ | Out-File -FilePath $cfg -Encoding ascii -Force

# --- 2. Inicializar o data dir --------------------------------------------------
# --initialize-insecure cria o root SEM senha. Definimos a senha no passo 5,
# antes de qualquer coisa conseguir se conectar de fora do localhost.
if (Test-Path $data) {
    Write-Host "[2/5] Data dir ja existe, pulando initialize." -ForegroundColor Yellow
} else {
    Write-Host "[2/5] Inicializando o data dir (demora ~30s)..." -ForegroundColor Cyan
    & $mysqld --defaults-file="$cfg" --initialize-insecure
    if ($LASTEXITCODE -ne 0) { throw "mysqld --initialize-insecure falhou (codigo $LASTEXITCODE)" }
}

# --- 3. Registrar o servico -----------------------------------------------------
Write-Host "[3/5] Registrando o servico '$servico'..." -ForegroundColor Cyan
& $mysqld --install $servico --defaults-file="$cfg"
if ($LASTEXITCODE -ne 0) { throw "mysqld --install falhou (codigo $LASTEXITCODE)" }

# --- 4. Subir o servico ---------------------------------------------------------
Write-Host "[4/5] Iniciando o servico..." -ForegroundColor Cyan
Start-Service -Name $servico
Set-Service -Name $servico -StartupType Automatic

# --- 5. Senha do root -----------------------------------------------------------
Write-Host "[5/5] Defina a senha do root do MySQL." -ForegroundColor Cyan
$senha = Read-Host -AsSecureString "Senha para root@localhost"
$senhaTexto = [Runtime.InteropServices.Marshal]::PtrToStringAuto(
    [Runtime.InteropServices.Marshal]::SecureStringToBSTR($senha)
)

$sql = "ALTER USER 'root'@'localhost' IDENTIFIED BY '$senhaTexto';"
$sql | & $mysql --defaults-file="$cfg" -u root
if ($LASTEXITCODE -ne 0) { throw "Nao consegui definir a senha do root (codigo $LASTEXITCODE)" }

Write-Host ""
Write-Host "MySQL pronto." -ForegroundColor Green
Get-Service -Name $servico | Format-Table Name, Status, StartType -AutoSize
Write-Host "Proximo passo — criar o banco do projeto:" -ForegroundColor Green
Write-Host "  & '$mysql' -u root -p < C:\jarvis_financeiro\backend\migrations\001_inicial.sql"
Write-Host ""
Write-Host "AVISO: antes de rodar a migration, troque 'TROQUE_ESTA_SENHA' no arquivo" -ForegroundColor Yellow
Write-Host "001_inicial.sql e no .env pela senha que voce quer para o usuario 'jarvis'." -ForegroundColor Yellow
