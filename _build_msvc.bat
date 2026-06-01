@echo off
setlocal EnableDelayedExpansion

call "C:\Program Files (x86)\Microsoft Visual Studio\18\BuildTools\VC\Auxiliary\Build\vcvars64.bat"
if %ERRORLEVEL% neq 0 (
    echo [FAIL] vcvars64.bat failed — is Visual Studio 2026 Build Tools installed?
    pause
    exit /b %ERRORLEVEL%
)

cd /d "D:\My Application\Auto_login"

echo [1/2] Configuring CMake...
cmake -S . -B build\Release -G Ninja -DCMAKE_BUILD_TYPE=Release -DCMAKE_PREFIX_PATH=C:/Qt/6.7.0/msvc2022_64
if %ERRORLEVEL% neq 0 (
    echo [FAIL] CMake configure failed
    pause
    exit /b %ERRORLEVEL%
)

echo [2/2] Building...
cmake --build build\Release --config Release
if %ERRORLEVEL% neq 0 (
    echo [FAIL] Build failed
    pause
    exit /b %ERRORLEVEL%
)

echo.
echo ============================================
echo BUILD OK
echo ============================================
dir build\Release\CampusAutoLogin.exe
pause
