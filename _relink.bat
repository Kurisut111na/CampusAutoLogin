@echo off
call "C:\Program Files (x86)\Microsoft Visual Studio\18\BuildTools\VC\Auxiliary\Build\vcvars64.bat"
cd /d "D:\My Application\Auto_login\build\Release"
cmake --build . --target CampusAutoLogin
