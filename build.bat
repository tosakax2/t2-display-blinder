@echo off
setlocal enabledelayedexpansion

echo =======================================================
echo  T2 Display Blinder - Build Script
echo =======================================================

:: Check Go environment
where go >nul 2>nul
if %errorlevel% neq 0 (
    echo [ERROR] Go compiler is not found in PATH.
    echo Please install Go or add it to your system PATH.
    pause
    exit /b 1
)

:: Create output directory
if not exist "bin" (
    mkdir "bin"
)

echo [1/2] Running tests...
go test ./...
if %errorlevel% neq 0 (
    echo [ERROR] Tests failed.
    pause
    exit /b 1
)

echo [2/2] Building application binary...
:: -H windowsgui hides the command prompt window when launched
:: -s -w strips debug symbols to reduce binary size
go build -ldflags="-H windowsgui -s -w" -o bin\display-blinder.exe .\cmd\display-blinder
if %errorlevel% neq 0 (
    echo [ERROR] Build failed.
    pause
    exit /b 1
)

echo =======================================================
echo  Build Succeeded!
echo  - Output: bin\display-blinder.exe
echo =======================================================

if "%1"=="" (
    ping 127.0.0.1 -n 3 >nul
)
