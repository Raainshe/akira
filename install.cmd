@echo off
setlocal enabledelayedexpansion

REM Akira Installation Script for Windows CMD

REM Configuration
set "GITHUB_REPO=raainshe/akira"
set "BINARY_NAME=akira.exe"
set "INSTALL_DIR=%LOCALAPPDATA%\akira"

REM Detect architecture
if "%PROCESSOR_ARCHITECTURE%"=="ARM64" (
    set "ARCH=arm64"
) else if "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
    set "ARCH=amd64"
) else (
    set "ARCH=386"
)
set "OS=windows"

echo Installing Akira Torrent Management Bot
echo =========================================

REM Create installation directory
if not exist "%INSTALL_DIR%" (
    mkdir "%INSTALL_DIR%"
    echo Created installation directory: %INSTALL_DIR%
)

REM Get latest release version
echo Finding latest release...
powershell -Command "try { $response = Invoke-RestMethod -Uri 'https://api.github.com/repos/%GITHUB_REPO%/releases/latest' -UseBasicParsing; $response.tag_name } catch { exit 1 }" > "%TEMP%\version.txt"
if errorlevel 1 (
    echo Failed to get latest release info
    echo Please check if releases are available at: https://github.com/%GITHUB_REPO%/releases
    pause
    exit /b 1
)

set /p LATEST_VERSION=<"%TEMP%\version.txt"
del "%TEMP%\version.txt"

echo Latest version: %LATEST_VERSION%

REM GitHub release URL for ZIP file
set "RELEASE_URL=https://github.com/%GITHUB_REPO%/releases/download/%LATEST_VERSION%/akira-%LATEST_VERSION%-%OS%-%ARCH%.zip"
set "ZIP_FILE=%INSTALL_DIR%\akira-%LATEST_VERSION%-%OS%-%ARCH%.zip"

echo Download URL: %RELEASE_URL%

REM Download binary with retry logic
echo Downloading Akira binary...
set "MAX_RETRIES=3"
set "RETRY_COUNT=0"

:download_loop
set /a RETRY_COUNT+=1
echo Attempt %RETRY_COUNT% of %MAX_RETRIES%...

powershell -Command "try { Invoke-WebRequest -Uri '%RELEASE_URL%' -OutFile '%ZIP_FILE%' -UseBasicParsing; exit 0 } catch { exit 1 }"
if errorlevel 1 (
    echo Download attempt %RETRY_COUNT% failed
    if %RETRY_COUNT% equ %MAX_RETRIES% (
        echo All download attempts failed. Please manually download from:
        echo https://github.com/%GITHUB_REPO%/releases
        echo Direct link: %RELEASE_URL%
        pause
        exit /b 1
    )
    echo Retrying in 3 seconds...
    timeout /t 3 /nobreak >nul
    goto download_loop
)

echo Download completed successfully

REM Extract ZIP file
echo Extracting binary...
powershell -Command "try { Expand-Archive -Path '%ZIP_FILE%' -DestinationPath '%INSTALL_DIR%' -Force; exit 0 } catch { exit 1 }"
if errorlevel 1 (
    echo Failed to extract binary
    pause
    exit /b 1
)

echo Extraction completed successfully

REM Clean up ZIP file
del "%ZIP_FILE%" 2>nul

REM Find and rename the extracted binary
echo Setting up binary...

REM List all files in the directory for debugging
echo Files in installation directory:
dir "%INSTALL_DIR%\*.exe" /b

REM Find the .exe file
for %%f in ("%INSTALL_DIR%\*.exe") do (
    set "ORIGINAL_BINARY=%%f"
    goto :found_binary
)

echo No .exe files found in extracted files
echo Available files:
dir "%INSTALL_DIR%" /b
pause
exit /b 1

:found_binary
set "TARGET_BINARY=%INSTALL_DIR%\%BINARY_NAME%"

echo Found binary: %ORIGINAL_BINARY%
echo Original path: %ORIGINAL_BINARY%
echo Target path: %TARGET_BINARY%

REM Verify original file exists
if not exist "%ORIGINAL_BINARY%" (
    echo ERROR: Original binary not found at: %ORIGINAL_BINARY%
    pause
    exit /b 1
)

REM Remove existing akira.exe if it exists
if exist "%TARGET_BINARY%" (
    del "%TARGET_BINARY%"
    echo Removed existing %BINARY_NAME%
)

REM Rename the binary to akira.exe
echo Renaming binary...
ren "%ORIGINAL_BINARY%" "%BINARY_NAME%"
if errorlevel 1 (
    echo Failed to rename binary
    echo Original: %ORIGINAL_BINARY%
    echo Target: %TARGET_BINARY%
    pause
    exit /b 1
)

echo Binary renamed to: %BINARY_NAME%

REM Verify the renamed file exists
if not exist "%TARGET_BINARY%" (
    echo ERROR: Renamed binary not found at: %TARGET_BINARY%
    pause
    exit /b 1
)

REM Add to PATH
echo Adding Akira to PATH...

REM Get current PATH
for /f "tokens=2*" %%a in ('reg query "HKCU\Environment" /v PATH 2^>nul') do set "CURRENT_PATH=%%b"

REM Check if already in PATH
echo %CURRENT_PATH% | findstr /i "%INSTALL_DIR%" >nul
if errorlevel 1 (
    REM Add to user PATH
    set "NEW_PATH=%CURRENT_PATH%;%INSTALL_DIR%"
    reg add "HKCU\Environment" /v PATH /t REG_EXPAND_SZ /d "%NEW_PATH%" /f >nul
    echo Added to user PATH permanently
) else (
    echo Akira is already in user PATH
)

REM Also add to current session PATH
set "PATH=%PATH%;%INSTALL_DIR%"
echo Added to current session PATH

REM Verify installation
echo Verifying installation...
"%TARGET_BINARY%" --version >nul 2>&1
if errorlevel 1 (
    echo Installation complete, but verification failed
    echo Try running: %TARGET_BINARY% --help
) else (
    echo Akira installed successfully!
    echo Binary location: %TARGET_BINARY%
    echo Try running: akira --help
)

echo.
echo Next steps:
echo 1. Create a Discord application at https://discord.com/developers/applications
echo 2. Set up your .env file with Discord token and qBittorrent credentials
echo 3. Run 'akira daemon' to start the bot
echo.
echo Enjoy using Akira!

pause
