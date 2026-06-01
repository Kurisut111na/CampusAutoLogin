param(
    [string]$BuildType = "Release",
    [string]$QtDir = "",
    [string]$Generator = "Ninja"
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$BuildDir = Join-Path $ProjectRoot "build\$BuildType"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host " Campus-AutoLogin Build Script" -ForegroundColor Cyan
Write-Host " Build Type: $BuildType" -ForegroundColor Cyan
Write-Host " Generator : $Generator" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

if (-not (Get-Command cmake -ErrorAction SilentlyContinue)) {
    Write-Error "CMake not found in PATH. Please install CMake >= 3.16."
    exit 1
}

$cmakeVersion = (cmake --version | Select-Object -First 1)
Write-Host "[OK] $cmakeVersion" -ForegroundColor Green

if ($QtDir) {
    $env:CMAKE_PREFIX_PATH = $QtDir
    Write-Host "[INFO] Using Qt from: $QtDir" -ForegroundColor Yellow
} else {
    if ($env:CMAKE_PREFIX_PATH) {
        Write-Host "[INFO] CMAKE_PREFIX_PATH = $env:CMAKE_PREFIX_PATH" -ForegroundColor Yellow
    } else {
        Write-Warning "Qt directory not specified. Set -QtDir or CMAKE_PREFIX_PATH env var."
        Write-Host "  Example: .\build.ps1 -QtDir 'C:\Qt\6.7.0\msvc2022_64'" -ForegroundColor Gray
    }
}

Write-Host ""
Write-Host ">>> Configuring CMake..." -ForegroundColor Cyan

$cmakeArgs = @(
    "-S", $ProjectRoot,
    "-B", $BuildDir,
    "-G", $Generator,
    "-DCMAKE_BUILD_TYPE=$BuildType"
)

if ($Generator -eq "Ninja") {
    $cmakeArgs += "-DCMAKE_MAKE_PROGRAM=ninja"
}

& cmake @cmakeArgs
if ($LASTEXITCODE -ne 0) {
    Write-Error "CMake configure failed!"
    exit $LASTEXITCODE
}

Write-Host ""
Write-Host ">>> Building..." -ForegroundColor Cyan

& cmake --build $BuildDir --config $BuildType --parallel
if ($LASTEXITCODE -ne 0) {
    Write-Error "Build failed!"
    exit $LASTEXITCODE
}

$ExePath = Join-Path $BuildDir "CampusAutoLogin.exe"
if (Test-Path $ExePath) {
    Write-Host ""
    Write-Host "[SUCCESS] Build complete: $ExePath" -ForegroundColor Green
    $fileInfo = Get-Item $ExePath
    Write-Host "  Size: $([math]::Round($fileInfo.Length / 1KB, 1)) KB" -ForegroundColor Gray
} else {
    Write-Warning "Build succeeded but .exe not found at expected path."
    Get-ChildItem $BuildDir -Recurse -Filter "*.exe" | ForEach-Object {
        Write-Host "  Found: $($_.FullName)" -ForegroundColor Gray
    }
}