# Campus-AutoLogin 部署指南

## 前置条件

1. **CMake** >= 3.16  
   下载: https://cmake.org/download/

2. **Qt 6.x** (Core, Network, Widgets, Test 模块)  
   下载: https://www.qt.io/download-open-source  
   推荐 Qt 6.5+ MSVC 2022 64-bit

3. **编译器** (选一):
   - MSVC 2022 (推荐): 安装 Visual Studio 2022 Community 并勾选 "C++桌面开发"
   - Ninja + MSVC (更快): `winget install Ninja-build.Ninja`

## 一键构建 & 打包

```powershell
# 步骤 1: 构建
.\scripts\build.ps1 -QtDir "C:\Qt\6.7.0\msvc2022_64"

# 步骤 2: 打包为可分发的 zip
.\scripts\package.ps1 -QtDir "C:\Qt\6.7.0\msvc2022_64"
```

输出文件: `build\Campus-AutoLogin-v1.0.0.zip`

## 手动构建

### 1. 配置

```powershell
$env:CMAKE_PREFIX_PATH = "C:\Qt\6.7.0\msvc2022_64"
cmake -S . -B build/Release -G "Ninja" -DCMAKE_BUILD_TYPE=Release
```

或使用 MSVC:
```powershell
cmake -S . -B build/Release -G "Visual Studio 17 2022" -A x64
```

### 2. 编译

```powershell
cmake --build build/Release --config Release --parallel
```

### 3. 运行测试

```powershell
cd build/Release
ctest --output-on-failure
```

### 4. 部署依赖 (windeployqt)

```powershell
C:\Qt\6.7.0\msvc2022_64\bin\windeployqt.exe `
  --release --no-translations --no-system-d3d-compiler `
  build/Release/CampusAutoLogin.exe
```

## 打包为单一 exe

### 方案 A: 静态编译 (推荐，真正单体 exe)

需要自行编译 Qt 静态库。步骤：

```powershell
# 1. 下载 Qt 源码
git clone https://code.qt.io/qt/qt5.git -b 6.7.0

# 2. 配置静态编译
configure.bat -static -release -prefix C:\Qt\6.7.0-static `
  -nomake examples -nomake tests `
  -qt-libpng -qt-libjpeg -openssl-linked

# 3. 编译 (需要数小时)
cmake --build . --parallel
cmake --install .

# 4. 使用静态 Qt 构建项目
.\scripts\build.ps1 -QtDir "C:\Qt\6.7.0-static"
```

静态编译后生成的 exe 约 15-25 MB，无需任何外部 DLL。

### 方案 B: Enigma Virtual Box (快速)

将 windeployqt 部署后的文件打包为单体 exe:

1. 下载 Enigma Virtual Box: https://enigmaprotector.com/en/aboutvb.html
2. 选择 `CampusAutoLogin.exe` 作为主程序
3. 添加所有同级目录的 DLL 和子文件夹
4. 选择输出路径，点击 Process

### 方案 C: 直接分发 zip (最简单)

`package.ps1` 生成的 zip 包含所有必要文件，解压即可运行。

## 分发清单

将以下内容发送给用户:

```
Campus-AutoLogin-v1.0.0.zip
├── CampusAutoLogin.exe       (主程序)
├── Qt6Core.dll               (Qt 运行时)
├── Qt6Network.dll
├── Qt6Widgets.dll
├── ...                       (其他 Qt DLL)
├── platforms/                (Qt 平台插件)
│   └── qwindows.dll
└── styles/                   (Qt 样式插件)
    └── qwindowsvistastyle.dll
```

## 项目结构

```
Auto_login/
├── CMakeLists.txt        (唯一构建配置)
├── README.md             (用户手册)
├── DEPLOY.md             (本文件)
├── scripts/
│   ├── build.ps1         (构建脚本)
│   └── package.ps1       (打包脚本)
├── src/
│   ├── main.cpp
│   ├── core/             (核心模块)
│   │   ├── logger        (日志系统)
│   │   ├── crypto_utils  (DPAPI加密)
│   │   ├── ip_detector   (IP/网卡检测)
│   │   ├── config_manager(配置管理)
│   │   ├── login_manager (登录/登出)
│   │   ├── heartbeat     (心跳监测)
│   │   └── network_manager(网关探测)
│   └── ui/               (界面模块)
│       ├── main_window   (主窗口)
│       ├── tray_icon     (系统托盘)
│       ├── auto_start    (开机自启)
│       └── advanced_settings_dialog(高级设置)
└── tests/                (测试)
    ├── test_ip_detector.cpp
    └── test_login_manager.cpp
```