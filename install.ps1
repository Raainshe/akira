# Akira Installation Script for Windows PowerShell

param(
    [string]$Version = "latest"
)

# Configuration
$GITHUB_REPO = "raainshe/akira"
$BINARY_NAME = "akira.exe"
$INSTALL_DIR = "$env:LOCALAPPDATA\akira"

# Detect architecture
$Arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$OS = "windows"

# GitHub release URL
$RELEASE_URL = "https://github.com/$GITHUB_REPO/releases/download/v$Version/akira-$OS-$Arch.exe"

Write-Host "Installing Akira Torrent Management Bot" -ForegroundColor Blue
Write-Host "=========================================" -ForegroundColor Blue

# Create installation directory
if (!(Test-Path $INSTALL_DIR)) {
    New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
    Write-Host "Created installation directory: $INSTALL_DIR" -ForegroundColor Blue
}

# Download binary
Write-Host "Downloading Akira binary..." -ForegroundColor Blue
try {
    Invoke-WebRequest -Uri $RELEASE_URL -OutFile "$INSTALL_DIR\$BINARY_NAME" -UseBasicParsing
    Write-Host "Download completed successfully" -ForegroundColor Green
} catch {
    Write-Host "Failed to download binary: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Add to PATH if not already there
$CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($CurrentPath -notlike "*$INSTALL_DIR*") {
    Write-Host "Adding Akira to PATH..." -ForegroundColor Yellow
    $NewPath = "$CurrentPath;$INSTALL_DIR"
    [Environment]::SetEnvironmentVariable("PATH", $NewPath, "User")
    Write-Host "Added to PATH (requires restart of terminal)" -ForegroundColor Green
} else {
    Write-Host "Akira is already in PATH" -ForegroundColor Green
}

# Verify installation
Write-Host "Verifying installation..." -ForegroundColor Blue
try {
    $VersionOutput = & "$INSTALL_DIR\$BINARY_NAME" --version 2>&1
    Write-Host "Akira installed successfully!" -ForegroundColor Green
    Write-Host "Try running: akira --help" -ForegroundColor Blue
} catch {
    Write-Host "Installation complete, but verification failed" -ForegroundColor Yellow
    Write-Host "Try running: $INSTALL_DIR\$BINARY_NAME --help" -ForegroundColor Blue
}

Write-Host ""
Write-Host "Next steps:" -ForegroundColor Blue
Write-Host "1. Create a Discord application at https://discord.com/developers/applications"
Write-Host "2. Set up your .env file with Discord token and qBittorrent credentials"
Write-Host "3. Run 'akira daemon' to start the bot"
Write-Host ""
Write-Host "Enjoy using Akira!" -ForegroundColor Green