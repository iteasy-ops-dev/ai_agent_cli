@echo off
setlocal enabledelayedexpansion

:: ItEasy AI Agent Cross-Platform Build Script for Windows
:: Supports: Windows, macOS, Linux for multiple architectures

:: Build configuration
set APP_NAME=iteasy-ai-agent
set BUILD_DIR=dist
if "%VERSION%"=="" set VERSION=dev
for /f "delims=" %%i in ('powershell -command "Get-Date -UFormat '%%Y-%%m-%%dT%%H:%%M:%%SZ'"') do set BUILD_TIME=%%i
if "%GITHUB_SHA%"=="" (
    for /f "delims=" %%i in ('git rev-parse --short HEAD 2^>nul') do set GIT_COMMIT=%%i
    if "!GIT_COMMIT!"=="" set GIT_COMMIT=unknown
) else (
    set GIT_COMMIT=%GITHUB_SHA%
)

:: Colors (limited support in cmd)
set "ESC="
set "RESET=%ESC%[0m"
set "RED=%ESC%[31m"
set "GREEN=%ESC%[32m"
set "YELLOW=%ESC%[33m"
set "BLUE=%ESC%[34m"

goto main

:print_header
echo.
echo ================================================================
echo                ItEasy AI Agent Build Script
echo                   Cross-Platform Builder
echo ================================================================
echo.
goto :eof

:print_info
echo [INFO] %~1
goto :eof

:print_warning
echo [WARN] %~1
goto :eof

:print_error
echo [ERROR] %~1
goto :eof

:print_build_info
echo Build Information:
echo   Version: %VERSION%
echo   Build Time: %BUILD_TIME%
echo   Git Commit: %GIT_COMMIT%
for /f "delims=" %%i in ('go version 2^>nul') do echo   Go Version: %%i
echo.
goto :eof

:clean_build_dir
call :print_info "Cleaning build directory..."
if exist "%BUILD_DIR%" rmdir /s /q "%BUILD_DIR%"
mkdir "%BUILD_DIR%\windows" "%BUILD_DIR%\macos" "%BUILD_DIR%\linux" 2>nul
goto :eof

:build_platform
set goos=%~1
set goarch=%~2
set output_name=%~3
set platform_name=%goos%/%goarch%

call :print_info "Building for %platform_name%..."

:: Determine output directory
if "%goos%"=="windows" set output_dir=%BUILD_DIR%\windows
if "%goos%"=="darwin" set output_dir=%BUILD_DIR%\macos
if "%goos%"=="linux" set output_dir=%BUILD_DIR%\linux

set output_path=%output_dir%\%APP_NAME%-%output_name%

:: Build command with ldflags for version info
set ldflags=-X main.version=%VERSION% -X main.buildTime=%BUILD_TIME% -X main.gitCommit=%GIT_COMMIT%

set GOOS=%goos%
set GOARCH=%goarch%
set CGO_ENABLED=0

go build -ldflags="%ldflags%" -o "%output_path%" .

if !errorlevel! equ 0 (
    for %%i in ("%output_path%") do set file_size=%%~zi
    call :format_size !file_size! formatted_size
    call :print_info "✓ Built %platform_name% (!formatted_size!)"
    call :create_archive "%goos%" "%output_path%" "%output_name%"
) else (
    call :print_error "✗ Failed to build %platform_name%"
    exit /b 1
)
goto :eof

:format_size
set size=%~1
if %size% lss 1024 (
    set %~2=%size%B
) else if %size% lss 1048576 (
    set /a kb_size=%size%/1024
    set %~2=!kb_size!KB
) else (
    set /a mb_size=%size%/1048576
    set %~2=!mb_size!MB
)
goto :eof

:create_archive
set goos=%~1
set binary_path=%~2
set output_name=%~3

for %%i in ("%binary_path%") do (
    set archive_dir=%%~dpi
    set binary_name=%%~nxi
)

pushd "!archive_dir!"

if "%goos%"=="windows" (
    :: Create ZIP for Windows using PowerShell
    set zip_name=%APP_NAME%-%output_name%.zip
    powershell -command "Compress-Archive -Path '!binary_name!' -DestinationPath '!zip_name!' -Force" 2>nul
    if !errorlevel! equ 0 (
        call :print_info "  → Created !zip_name!"
    )
) else (
    :: Create tar.gz for Unix-like systems (if tar is available)
    set tar_name=%APP_NAME%-%output_name%.tar.gz
    tar -czf "!tar_name!" "!binary_name!" 2>nul
    if !errorlevel! equ 0 (
        call :print_info "  → Created !tar_name!"
    ) else (
        :: Fallback to ZIP if tar is not available
        set zip_name=%APP_NAME%-%output_name%.zip
        powershell -command "Compress-Archive -Path '!binary_name!' -DestinationPath '!zip_name!' -Force" 2>nul
        if !errorlevel! equ 0 (
            call :print_info "  → Created !zip_name! (tar not available)"
        )
    )
)

popd
goto :eof

:generate_checksums
call :print_info "Generating checksums..."

set checksum_file=%BUILD_DIR%\checksums.txt
if exist "%checksum_file%" del "%checksum_file%"

for /d %%d in ("%BUILD_DIR%\*") do (
    for %%f in ("%%d\*.zip" "%%d\*.tar.gz") do (
        if exist "%%f" (
            for %%i in ("%%f") do (
                set filename=%%~nxi
                set dirname=%%~nxd
                for /f "delims=" %%h in ('powershell -command "Get-FileHash -Algorithm SHA256 '%%f' | Select-Object -ExpandProperty Hash"') do (
                    echo %%h  !dirname!/!filename! >> "%checksum_file%"
                )
            )
        )
    )
)

call :print_info "✓ Checksums saved to checksums.txt"
goto :eof

:show_build_summary
echo.
echo ================================================================
echo                     Build Summary
echo ================================================================

echo Built binaries:
for /r "%BUILD_DIR%" %%f in (%APP_NAME%-*) do (
    echo %%f | findstr /v ".zip .tar.gz" >nul
    if !errorlevel! equ 0 (
        for %%i in ("%%f") do (
            set rel_path=%%f
            set rel_path=!rel_path:%CD%\%BUILD_DIR%\=!
            call :format_size %%~zi formatted_size
            echo   !rel_path! ^(!formatted_size!^)
        )
    )
)

echo.
echo Archives:
for /r "%BUILD_DIR%" %%f in (*.zip *.tar.gz) do (
    for %%i in ("%%f") do (
        set rel_path=%%f
        set rel_path=!rel_path:%CD%\%BUILD_DIR%\=!
        call :format_size %%~zi formatted_size
        echo   !rel_path! ^(!formatted_size!^)
    )
)

echo.
echo Build completed successfully!
echo Output directory: %BUILD_DIR%
goto :eof

:main
if "%~1"=="clean" goto clean
if "%~1"=="help" goto help
if "%~1"=="-h" goto help
if "%~1"=="--help" goto help
if "%~1"=="" goto build_all

call :print_error "Unknown command: %~1"
echo Use '%~0 help' for usage information
exit /b 1

:clean
call :print_info "Cleaning build directory..."
if exist "%BUILD_DIR%" rmdir /s /q "%BUILD_DIR%"
call :print_info "✓ Cleaned"
goto :eof

:help
echo Usage: %~0 [clean^|help]
echo.
echo Commands:
echo   ^(no args^)  Build for all platforms
echo   clean      Clean build directory
echo   help       Show this help
echo.
echo Environment variables:
echo   VERSION    Set build version ^(default: dev^)
goto :eof

:build_all
call :print_header
call :print_build_info

:: Check if Go is installed
go version >nul 2>&1
if !errorlevel! neq 0 (
    call :print_error "Go is not installed or not in PATH"
    exit /b 1
)

:: Check if we're in a Go module
if not exist "go.mod" (
    call :print_error "go.mod not found. Please run this script from the project root."
    exit /b 1
)

call :clean_build_dir

:: Build for all platforms
set failed_builds=0

:: Windows builds
call :build_platform "windows" "amd64" "windows-amd64.exe"
if !errorlevel! neq 0 set /a failed_builds+=1

call :build_platform "windows" "386" "windows-386.exe"
if !errorlevel! neq 0 set /a failed_builds+=1

:: macOS builds
call :build_platform "darwin" "amd64" "macos-amd64"
if !errorlevel! neq 0 set /a failed_builds+=1

call :build_platform "darwin" "arm64" "macos-arm64"
if !errorlevel! neq 0 set /a failed_builds+=1

:: Linux builds
call :build_platform "linux" "amd64" "linux-amd64"
if !errorlevel! neq 0 set /a failed_builds+=1

call :build_platform "linux" "arm64" "linux-arm64"
if !errorlevel! neq 0 set /a failed_builds+=1

call :build_platform "linux" "386" "linux-386"
if !errorlevel! neq 0 set /a failed_builds+=1

if !failed_builds! equ 0 (
    call :generate_checksums
    call :show_build_summary
) else (
    call :print_error "!failed_builds! builds failed"
    exit /b 1
)

goto :eof