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

Write-Host "Installing Akira Torrent Management Bot" -ForegroundColor Blue
Write-Host "=========================================" -ForegroundColor Blue

# Create installation directory
if (!(Test-Path $INSTALL_DIR)) {
    New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
    Write-Host "Created installation directory: $INSTALL_DIR" -ForegroundColor Blue
}

# Get latest release version
Write-Host "Finding latest release..." -ForegroundColor Blue
try {
    $ReleasesResponse = Invoke-RestMethod -Uri "https://api.github.com/repos/$GITHUB_REPO/releases/latest" -UseBasicParsing -TimeoutSec 30
    $LatestVersion = $ReleasesResponse.tag_name
    Write-Host "Latest version: $LatestVersion" -ForegroundColor Green
} catch {
    Write-Host "Failed to get latest release info: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Please check if releases are available at: https://github.com/$GITHUB_REPO/releases" -ForegroundColor Yellow
    exit 1
}

# GitHub release URL for ZIP file
$RELEASE_URL = "https://github.com/$GITHUB_REPO/releases/download/$LatestVersion/akira-$LatestVersion-$OS-$Arch.zip"
$ZIP_FILE = "$INSTALL_DIR\akira-$LatestVersion-$OS-$Arch.zip"

Write-Host "Download URL: $RELEASE_URL" -ForegroundColor Cyan

# Download binary with retry logic
Write-Host "Downloading Akira binary..." -ForegroundColor Blue
$MaxRetries = 3
$RetryCount = 0

do {
    $RetryCount++
    try {
        Write-Host "Attempt $RetryCount of $MaxRetries..." -ForegroundColor Yellow
        Invoke-WebRequest -Uri $RELEASE_URL -OutFile $ZIP_FILE -UseBasicParsing -TimeoutSec 60
        Write-Host "Download completed successfully" -ForegroundColor Green
        break
    } catch {
        Write-Host "Download attempt $RetryCount failed: $($_.Exception.Message)" -ForegroundColor Red
        if ($RetryCount -eq $MaxRetries) {
            Write-Host "All download attempts failed. Please manually download from:" -ForegroundColor Red
            Write-Host "https://github.com/$GITHUB_REPO/releases" -ForegroundColor Yellow
            Write-Host "Direct link: $RELEASE_URL" -ForegroundColor Yellow
            exit 1
        }
        Write-Host "Retrying in 3 seconds..." -ForegroundColor Yellow
        Start-Sleep -Seconds 3
    }
} while ($RetryCount -lt $MaxRetries)

# Extract ZIP file
Write-Host "Extracting binary..." -ForegroundColor Blue
try {
    Expand-Archive -Path $ZIP_FILE -DestinationPath $INSTALL_DIR -Force
    Write-Host "Extraction completed successfully" -ForegroundColor Green
} catch {
    Write-Host "Failed to extract binary: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Clean up ZIP file
Remove-Item $ZIP_FILE -Force -ErrorAction SilentlyContinue

# Find and rename the extracted binary
Write-Host "Setting up binary..." -ForegroundColor Blue

# List all files in the directory for debugging
Write-Host "Files in installation directory:" -ForegroundColor Cyan
Get-ChildItem -Path $INSTALL_DIR | ForEach-Object { Write-Host "  $($_.Name)" -ForegroundColor Yellow }

# Look for the Windows binary with more flexible pattern matching
$ExtractedFiles = Get-ChildItem -Path $INSTALL_DIR -Name "*.exe" | Where-Object { $_ -like "*windows*" }
if ($ExtractedFiles.Count -eq 0) {
    Write-Host "No Windows binary found in extracted files" -ForegroundColor Red
    Write-Host "Available files:" -ForegroundColor Yellow
    Get-ChildItem -Path $INSTALL_DIR | ForEach-Object { Write-Host "  $($_.Name)" -ForegroundColor Yellow }
    exit 1
}

$OriginalBinary = Join-Path $INSTALL_DIR $ExtractedFiles[0]
$TargetBinary = Join-Path $INSTALL_DIR $BINARY_NAME

Write-Host "Found binary: $($ExtractedFiles[0])" -ForegroundColor Green
Write-Host "Will rename to: $BINARY_NAME" -ForegroundColor Green

# Remove existing akira.exe if it exists
if (Test-Path $TargetBinary) {
    Remove-Item $TargetBinary -Force
    Write-Host "Removed existing $BINARY_NAME" -ForegroundColor Yellow
}

# Rename the binary to akira.exe
Move-Item -Path $OriginalBinary -Destination $TargetBinary -Force
Write-Host "Binary renamed to: $BINARY_NAME" -ForegroundColor Green

# Add to PATH with immediate effect for current session
Write-Host "Adding Akira to PATH..." -ForegroundColor Yellow
$CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($CurrentPath -notlike "*$INSTALL_DIR*") {
    $NewPath = "$CurrentPath;$INSTALL_DIR"
    [Environment]::SetEnvironmentVariable("PATH", $NewPath, "User")
    Write-Host "Added to user PATH permanently" -ForegroundColor Green
} else {
    Write-Host "Akira is already in user PATH" -ForegroundColor Green
}

# Also add to current session PATH
$env:PATH = "$env:PATH;$INSTALL_DIR"
Write-Host "Added to current session PATH" -ForegroundColor Green

# Verify installation
Write-Host "Verifying installation..." -ForegroundColor Blue
try {
    $VersionOutput = & $TargetBinary --version 2>&1
    Write-Host "Akira installed successfully!" -ForegroundColor Green
    Write-Host "Binary location: $TargetBinary" -ForegroundColor Cyan
    Write-Host "Try running: akira --help" -ForegroundColor Blue
} catch {
    Write-Host "Installation complete, but verification failed" -ForegroundColor Yellow
    Write-Host "Try running: $TargetBinary --help" -ForegroundColor Blue
}

Write-Host ""
Write-Host "Next steps:" -ForegroundColor Blue
Write-Host "1. Create a Discord application at https://discord.com/developers/applications"
Write-Host "2. Set up your .env file with Discord token and qBittorrent credentials"
Write-Host "3. Run 'akira daemon' to start the bot"
Write-Host ""
Write-Host "Enjoy using Akira!" -ForegroundColor Green