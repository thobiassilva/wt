#Requires -Version 5.1
<#
.SYNOPSIS
    Instala o wt (git worktree manager) no Windows.
.DESCRIPTION
    Detecta a arquitetura, baixa o binario correto do GitHub Releases,
    verifica o checksum SHA256 e instala em $HOME\.local\bin\wt.exe.
.EXAMPLE
    irm https://raw.githubusercontent.com/thobiassilva/wt/main/install.ps1 | iex
#>
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$Repo    = 'thobiassilva/wt'
$BinDir  = if ($env:BIN_DIR) { $env:BIN_DIR } else { "$HOME\.local\bin" }

function Write-Info  { param($msg) Write-Host ">>> $msg" -ForegroundColor Green }
function Write-Warn  { param($msg) Write-Host "aviso: $msg" -ForegroundColor Yellow }
function Write-Fatal { param($msg) Write-Host "erro: $msg" -ForegroundColor Red; exit 1 }

# --- Detect arch ---
$arch = if ([System.Environment]::Is64BitOperatingSystem) {
    $cpu = (Get-CimInstance Win32_Processor).Architecture
    # 12 = ARM64
    if ($cpu -eq 12) { 'arm64' } else { 'amd64' }
} else {
    Write-Fatal "Arquitetura 32-bit nao suportada."
}

# --- Fetch latest version ---
Write-Info "Buscando ultima versao..."
try {
    $release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
    $version = $release.tag_name
} catch {
    Write-Fatal "Nao foi possivel buscar a versao mais recente: $_"
}
Write-Info "Versao: $version"

# --- Build URLs ---
$versionNum = $version.TrimStart('v')
$archive    = "wt_${versionNum}_windows_${arch}.zip"
$baseUrl    = "https://github.com/$Repo/releases/download/$version"
$archiveUrl = "$baseUrl/$archive"
$checksumUrl = "$baseUrl/checksums.txt"

# --- Download to temp dir ---
$tmp = New-TemporaryFile | ForEach-Object { $_.DirectoryName }
$tmpArchive  = Join-Path $tmp $archive
$tmpChecksum = Join-Path $tmp 'checksums.txt'

Write-Info "Baixando $archive..."
try {
    Invoke-WebRequest -Uri $archiveUrl  -OutFile $tmpArchive  -UseBasicParsing
    Invoke-WebRequest -Uri $checksumUrl -OutFile $tmpChecksum -UseBasicParsing
} catch {
    Write-Fatal "Falha no download: $_"
}

# --- Verify checksum ---
Write-Info "Verificando checksum..."
$expected = (Get-Content $tmpChecksum | Where-Object { $_ -match [regex]::Escape($archive) }) -split '\s+' | Select-Object -First 1
$actual   = (Get-FileHash -Algorithm SHA256 $tmpArchive).Hash.ToLower()
if ($expected -ne $actual) {
    Write-Fatal "Checksum invalido — download corrompido. Esperado: $expected  Obtido: $actual"
}
Write-Info "Checksum OK"

# --- Extract ---
$tmpExtract = Join-Path $tmp 'wt-extract'
Expand-Archive -Path $tmpArchive -DestinationPath $tmpExtract -Force

# --- Install ---
if (-not (Test-Path $BinDir)) {
    New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
}
Copy-Item (Join-Path $tmpExtract 'wt.exe') (Join-Path $BinDir 'wt.exe') -Force
Write-Info "Instalado em: $BinDir\wt.exe"

# --- Cleanup ---
Remove-Item $tmpArchive, $tmpChecksum, $tmpExtract -Recurse -Force -ErrorAction SilentlyContinue

# --- Verify PATH ---
$userPath = [System.Environment]::GetEnvironmentVariable('PATH', 'User')
if ($userPath -notlike "*$BinDir*") {
    Write-Warn "$BinDir nao esta no PATH."
    Write-Host ""
    Write-Host "  Execute o comando abaixo para adicionar permanentemente:" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "    [System.Environment]::SetEnvironmentVariable('PATH', `$env:PATH + ';$BinDir', 'User')" -ForegroundColor White
    Write-Host ""
    Write-Host "  Depois reabra o terminal." -ForegroundColor Cyan
} else {
    Write-Info "wt $version instalado. Execute 'wt --help' para comecar."
}
