param(
    [string]$BuildType = "Release",
    [string]$QtDir = "",
    [switch]$NoCompress
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$BuildDir = Join-Path $ProjectRoot "build\$BuildType"
$ExePath = Join-Path $BuildDir "CampusAutoLogin.exe"
$PackageDir = Join-Path $ProjectRoot "build\Campus-AutoLogin-v1.0.0"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host " Campus-AutoLogin Package Script" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

if (-not (Test-Path $ExePath)) {
    Write-Error "Executable not found: $ExePath"
    Write-Host "  Run .\scripts\build.ps1 first." -ForegroundColor Yellow
    exit 1
}

if ($QtDir) {
    $env:CMAKE_PREFIX_PATH = $QtDir
}

$windeployqt = ""
$possiblePaths = @(
    "$QtDir\bin\windeployqt.exe",
    "$env:CMAKE_PREFIX_PATH\bin\windeployqt.exe",
    (Get-Command windeployqt -ErrorAction SilentlyContinue).Source
)

foreach ($path in $possiblePaths) {
    if ($path -and (Test-Path $path)) {
        $windeployqt = $path
        break
    }
}

if (-not $windeployqt) {
    $found = Get-ChildItem -Path "C:\Qt" -Recurse -Filter "windeployqt.exe" -ErrorAction SilentlyContinue |
        Select-Object -First 1
    if ($found) {
        $windeployqt = $found.FullName
    }
}

if (-not $windeployqt) {
    Write-Error "windeployqt.exe not found!"
    Write-Host "  Specify -QtDir or ensure Qt bin is in PATH." -ForegroundColor Yellow
    Write-Host "  Example: .\package.ps1 -QtDir 'C:\Qt\6.7.0\msvc2022_64'" -ForegroundColor Gray
    exit 1
}

Write-Host "[OK] windeployqt: $windeployqt" -ForegroundColor Green

if (Test-Path $PackageDir) {
    Write-Host "[INFO] Removing old package directory..." -ForegroundColor Yellow
    Remove-Item -Recurse -Force $PackageDir
}

New-Item -ItemType Directory -Force -Path $PackageDir | Out-Null

Write-Host ""
Write-Host ">>> Copying executable..." -ForegroundColor Cyan
Copy-Item $ExePath $PackageDir

Write-Host ">>> Running windeployqt..." -ForegroundColor Cyan
$targetExe = Join-Path $PackageDir "CampusAutoLogin.exe"
& $windeployqt --release --no-translations --no-system-d3d-compiler $targetExe
if ($LASTEXITCODE -ne 0) {
    Write-Warning "windeployqt reported an error (may be non-critical)"
}

Write-Host ""
Write-Host ">>> Cleaning up unnecessary files..." -ForegroundColor Cyan
$unused = @("*.pdb", "imageformats\*.pdb", "iconengines\*.pdb")
foreach ($pattern in $unused) {
    Get-ChildItem -Path $PackageDir -Filter $pattern -Recurse -ErrorAction SilentlyContinue |
        Remove-Item -Force
}

Write-Host ""
Write-Host "[SUCCESS] Package ready: $PackageDir" -ForegroundColor Green

$totalSize = (Get-ChildItem $PackageDir -Recurse | Measure-Object -Property Length -Sum).Sum
Write-Host "  Total size: $([math]::Round($totalSize / 1MB, 1)) MB" -ForegroundColor Gray

if (-not $NoCompress) {
    $zipPath = Join-Path $ProjectRoot "build\Campus-AutoLogin-v1.0.0.zip"
    Write-Host ""
    Write-Host ">>> Creating zip archive..." -ForegroundColor Cyan
    if (Test-Path $zipPath) { Remove-Item $zipPath -Force }
    Compress-Archive -Path $PackageDir -DestinationPath $zipPath
    $zipSize = (Get-Item $zipPath).Length
    Write-Host "[SUCCESS] Zip created: $zipPath" -ForegroundColor Green
    Write-Host "  Size: $([math]::Round($zipSize / 1MB, 1)) MB" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  Send this zip file to users. They just need to:" -ForegroundColor Cyan
    Write-Host "    1. Extract the zip" -ForegroundColor White
    Write-Host "    2. Double-click CampusAutoLogin.exe" -ForegroundColor White
}