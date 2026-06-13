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

#### Local development

Run Akira in a container while connecting to an existing qBittorrent instance on your host or LAN.

```bash
cp .env.example .env   # set DISCORD_BOT_TOKEN, qBittorrent URL and credentials
docker compose up -d --build
docker compose logs -f akira
```

Or use Makefile shortcuts:

```bash
make docker-up      # build and run locally
make docker-down
make docker-dev     # Air live reload for development
make docker-dev-down
```

**Configuration notes:**

- Set `QBITTORRENT_URL` to your qBittorrent Web UI (e.g. `http://192.168.0.101:8080` or `http://host.docker.internal:8080` if qBittorrent runs on the Docker host).
- `QBITTORRENT_*_SAVE_PATH` values must be **qBittorrent's paths** (e.g. `E:\Series` on Windows), not paths inside the container.
- Logs and seeding state are persisted in `./data`.
- For Discord disk-space commands to reflect real storage, mount the host download drive into the container and set `DISK_SPACE_CHECK_PATH` to the mount point (see `.env.example`).

#### Production deployment (Windows Server auto-update)

Push to `main` builds and publishes `ghcr.io/raainshe/akira:main`. A Watchtower sidecar on the server polls every 5 minutes, pulls when the image digest changes, and recreates the `akira` container. Your `.env` and `./data` on the host are unchanged across updates.

**One-time server setup:**

1. Install Docker (Engine or Desktop) with **Linux containers** enabled.
2. Create a deploy directory, e.g. `C:\akira\`.
3. Copy to the server:
   - `docker-compose.prod.yml`
   - `.env` (from `.env.example`, with real secrets — never baked into the image)
4. Create the data directory: `mkdir data`
5. Start the stack:

   ```powershell
   docker compose -f docker-compose.prod.yml pull
   docker compose -f docker-compose.prod.yml up -d
   ```

   Or with Make: `make docker-prod-up`

6. Verify both containers are running:

   ```powershell
   docker compose -f docker-compose.prod.yml ps
   ```

   You should see `akira` and `watchtower`.

7. After the first GitHub Actions publish, set the GHCR package to **Public**: GitHub → Packages → akira → Package settings → Change visibility.

**Update flow (automatic):**

```
Push to main → GitHub Actions publishes image → Watchtower detects new digest → akira restarts
```

No manual binary download required. Updates apply within the Watchtower poll interval (5 minutes).

**Useful commands:**

```bash
make docker-prod-up    # pull latest image and start/restart stack
make docker-prod-down  # stop stack
make docker-prod-logs  # follow akira logs
```

**Windows Server note:** If Watchtower cannot connect to Docker, try swapping the socket volume in `docker-compose.prod.yml` to the Windows named pipe form for your install, e.g. `//./pipe/docker_engine://./pipe/docker_engine`.

**What persists across updates:**

- `.env` on the host
- `./data/` (`bot_activity.log`, `seeding_tracking.json`, pid file)

## Setup

1. **Create Discord Application**
   - Go to [Discord Developer Portal](https://discord.com/developers/applications)
   - Create a new application and bot
   - Copy your bot token

2. **Configure Environment**
   ```bash
   # Linux/macOS
   cp .env.example akira.env
   # or
   cp .env.example .env
   
   # Windows (recommended: use akira.env - not hidden)
   copy .env.example akira.env
   # or
   copy .env.example .env
   ```
   Edit `akira.env` (or `.env`) with your Discord token and qBittorrent credentials
   
   **Note:** On Windows Server, `akira.env` is recommended as it's not a hidden file and easier to locate.

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