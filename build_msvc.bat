@echo off
call "C:\Program Files (x86)\Microsoft Visual Studio\18\BuildTools\VC\Auxiliary\Build\vcvars64.bat" >nul 2>&1
cd /d "D:\My Application\Auto_login"
powershell -ExecutionPolicy Bypass -File ".\scripts\build.ps1" -QtDir "C:\Qt\6.7.0\msvc2022_64"
exit /b %ERRORLEVEL%
