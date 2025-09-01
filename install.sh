#!/bin/bash
# Akira Installation Script for Linux/macOS

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
VERSION="latest"
INSTALL_DIR="/usr/local/bin"
USER_INSTALL_DIR="$HOME/.local/bin"
BINARY_NAME="akira"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}❌ Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

# GitHub release URL
GITHUB_REPO="raainshe/akira"
RELEASE_URL="https://github.com/$GITHUB_REPO/releases/download/v$VERSION/akira-$OS-$ARCH"

echo -e "${BLUE}🌟 Installing Akira Torrent Management Bot${NC}"
echo "=========================================="

# Check if running as root
if [[ $EUID -eq 0 ]]; then
    echo -e "${YELLOW}⚠️  Running as root. Installing to system directory.${NC}"
    TARGET_DIR="$INSTALL_DIR"
else
    echo -e "${BLUE}📁 Installing to user directory: $USER_INSTALL_DIR${NC}"
    TARGET_DIR="$USER_INSTALL_DIR"
    mkdir -p "$TARGET_DIR"
fi

# Download binary
echo -e "${BLUE}📥 Downloading Akira binary...${NC}"
if command -v curl >/dev/null 2>&1; then
    curl -L -o "$TARGET_DIR/$BINARY_NAME" "$RELEASE_URL"
elif command -v wget >/dev/null 2>&1; then
    wget -O "$TARGET_DIR/$BINARY_NAME" "$RELEASE_URL"
else
    echo -e "${RED}❌ Neither curl nor wget found. Please install one and try again.${NC}"
    exit 1
fi

# Make executable
chmod +x "$TARGET_DIR/$BINARY_NAME"

# Add to PATH if needed
if [[ "$TARGET_DIR" == "$USER_INSTALL_DIR" ]]; then
    if [[ ":$PATH:" != *":$USER_INSTALL_DIR:"* ]]; then
        echo -e "${YELLOW}⚠️  Adding $USER_INSTALL_DIR to PATH...${NC}"
        echo "export PATH=\"\$PATH:$USER_INSTALL_DIR\"" >> "$HOME/.bashrc"
        echo "export PATH=\"\$PATH:$USER_INSTALL_DIR\"" >> "$HOME/.zshrc" 2>/dev/null || true
        echo -e "${GREEN}✅ Added to shell configuration files${NC}"
        echo -e "${YELLOW}🔄 Please restart your terminal or run: source ~/.bashrc${NC}"
    fi
fi

# Verify installation
if command -v akira >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Akira installed successfully!${NC}"
    echo -e "${BLUE}🚀 Try running: akira --help${NC}"
else
    echo -e "${YELLOW}⚠️  Installation complete, but 'akira' command not found in PATH${NC}"
    echo -e "${BLUE}💡 Try running: $TARGET_DIR/akira --help${NC}"
fi

echo ""
echo -e "${BLUE}📖 Next steps:${NC}"
echo "1. Create a Discord application at https://discord.com/developers/applications"
echo "2. Set up your .env file with Discord token and qBittorrent credentials"
echo "3. Run 'akira daemon' to start the bot"
echo ""
echo -e "${GREEN}🎉 Enjoy using Akira!${NC}"
