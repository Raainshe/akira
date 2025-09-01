# 🌟 Akira - Torrent Management Discord Bot

A powerful Discord bot for managing qBittorrent downloads and uploads with an intuitive CLI and beautiful TUI interface.

## Features

- 🔍 **Torrent Search & Management**: Add, pause, resume, and delete torrents
- 📊 **Real-time Monitoring**: Track download progress and seeding status
- 🎮 **Interactive TUI**: Beautiful terminal interface for visual management
- 🤖 **Discord Integration**: Manage torrents directly from Discord
- 🔄 **Automated Seeding**: Smart seeding management with configurable rules
- 📁 **File Management**: Organize and manage downloaded files

## Quick Install

### Linux/macOS
```bash
curl -fsSL https://raw.githubusercontent.com/raainshe/akira/main/install.sh | bash
```

### Manual Installation
1. Download the latest release for your platform from [GitHub Releases](https://github.com/raainshe/akira/releases)
2. Extract the binary and make it executable:
   ```bash
   chmod +x akira
   ```
3. Move to your PATH:
   ```bash
   # System-wide (requires sudo)
   sudo mv akira /usr/local/bin/
   
   # User directory
   mkdir -p ~/.local/bin && mv akira ~/.local/bin/
   ```

### From Source
```bash
git clone https://github.com/raainshe/akira.git
cd akira
make build
make install-user  # or make install for system-wide
```

## Setup

1. **Create Discord Application**
   - Go to [Discord Developer Portal](https://discord.com/developers/applications)
   - Create a new application and bot
   - Copy your bot token

2. **Configure Environment**
   ```bash
   cp .env.example .env
   # Edit .env with your Discord token and qBittorrent credentials
   ```

3. **Start the Bot**
   ```bash
   akira daemon
   ```

## Usage

### CLI Commands
```bash
# Start the Discord bot daemon
akira daemon

# Interactive TUI
akira tui

# Check daemon status
akira status

# Stop daemon
akira stop

# Restart daemon
akira restart
```

### Discord Commands
- `/torrent add <magnet>` - Add a new torrent
- `/torrent list` - List all torrents
- `/torrent pause <id>` - Pause a torrent
- `/torrent resume <id>` - Resume a torrent
- `/torrent delete <id>` - Delete a torrent
- `/status` - Show system status

## Configuration

The bot uses environment variables for configuration. See `.env.example` for all available options.

### Key Settings
- `DISCORD_TOKEN` - Your Discord bot token
- `QBITTORRENT_URL` - qBittorrent Web UI URL
- `QBITTORRENT_USERNAME` - qBittorrent username
- `QBITTORRENT_PASSWORD` - qBittorrent password

## Development

### Prerequisites
- Go 1.21+
- qBittorrent with Web UI enabled
- Discord Bot Token

### Build
```bash
make build        # Build for current platform
make build-all    # Build for all platforms
make install-user # Install to user directory
```

### Testing
```bash
make test
make test-race
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- 📖 [Documentation](https://github.com/raainshe/akira/wiki)
- 🐛 [Issue Tracker](https://github.com/raainshe/akira/issues)
- 💬 [Discussions](https://github.com/raainshe/akira/discussions)

---

Made with ❤️ by the Akira community