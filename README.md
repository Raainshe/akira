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

### Windows
#### Option 1: Using PowerShell (Recommended)
```powershell
# Run PowerShell as Administrator (or regular PowerShell for user install)
Set-ExecutionPolicy Bypass -Scope Process -Force
iwr -useb https://raw.githubusercontent.com/raainshe/akira/main/install.ps1 | iex
```

#### Option 2: Using Command Prompt
```cmd
# Run Command Prompt as Administrator (or regular CMD for user install)
curl -fsSL https://raw.githubusercontent.com/raainshe/akira/main/install.cmd | cmd
```

#### Option 3: Manual Installation
1. Download the latest Windows release from [GitHub Releases](https://github.com/raainshe/akira/releases)
2. Extract the ZIP file
3. Rename `akira-windows-amd64.exe` to `akira.exe`
4. Add the folder to your PATH or run from the extracted directory

### Manual Installation
1. Download the latest release for your platform from [GitHub Releases](https://github.com/raainshe/akira/releases)
2. Extract the binary and make it executable:
   ```bash
   # Linux/macOS
   chmod +x akira
   
   # Windows
   # The .exe file is already executable
   ```
3. Move to your PATH:
   ```bash
   # Linux/macOS - System-wide (requires sudo)
   sudo mv akira /usr/local/bin/
   
   # Linux/macOS - User directory
   mkdir -p ~/.local/bin && mv akira ~/.local/bin/
   
   # Windows - Add to PATH
   # Copy akira.exe to a folder in your PATH (e.g., C:\Windows\System32)
   # Or add the folder containing akira.exe to your PATH environment variable
   ```

### From Source
```bash
git clone https://github.com/raainshe/akira.git
cd akira
make build
make install-user  # or make install for system-wide
```

### Docker

Run Akira in a container while connecting to an existing qBittorrent instance on your host or LAN.

```bash
cp .env.example .env   # set DISCORD_BOT_TOKEN, qBittorrent URL and credentials
docker compose up -d --build
docker compose logs -f akira
```

Or use Makefile shortcuts:

```bash
make docker-up
make docker-down
```

**Configuration notes:**

- Set `QBITTORRENT_URL` to your qBittorrent Web UI (e.g. `http://192.168.0.101:8080` or `http://host.docker.internal:8080` if qBittorrent runs on the Docker host).
- `QBITTORRENT_*_SAVE_PATH` values must be **qBittorrent's paths** (e.g. `E:\Series` on Windows), not paths inside the container.
- Logs and seeding state are persisted in `./data`.
- For Discord disk-space commands to reflect real storage, mount the host download drive into the container and set `DISK_SPACE_CHECK_PATH` to the mount point (see `.env.example`).

## Setup

1. **Create Discord Application**
   - Go to [Discord Developer Portal](https://discord.com/developers/applications)
   - Create a new application and bot
   - Copy your bot token

2. **Configure Environment**
   ```bash
   # Linux/macOS
   cp .env.example .env
   
   # Windows
   copy .env.example .env
   ```
   Edit `.env` with your Discord token and qBittorrent credentials

3. **Start the Bot**
   ```bash
   # Linux/macOS
   akira daemon
   
   # Windows
   akira.exe daemon
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