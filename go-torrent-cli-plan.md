# Go Torrent CLI + Discord Bot Implementation Plan

## Project Overview
Build a unified Go application that provides both CLI commands and Discord bot functionality for managing qBittorrent. The application will run in two modes:
- **CLI Mode**: Execute one-time commands (list, add, delete torrents, etc.)
- **Daemon Mode**: Run Discord bot + background services continuously

## Phase 1: Project Setup & Foundation

### Step 1: Initialize Go Module
- Create project directory: `go-torrent-cli`
- Initialize Go module: `go mod init go-torrent-cli`
- Create `.gitignore` file (exclude `.env`, `bot_activity.log`, binaries)

### Step 2: Create Project Structure
```
akira/
├── main.go                     # Entry point
├── .env                        # Configuration
├── .env.example               # Example configuration
├── cmd/                       # CLI command implementations
│   ├── daemon.go              # Daemon mode (Discord bot + services)
│   ├── list.go                # List torrents command
│   ├── add.go                 # Add magnet command
│   ├── delete.go              # Delete torrents command
│   ├── diskspace.go           # Disk space command
│   ├── logs.go                # View logs command
│   ├── seed_status.go         # Seeding status command
│   └── stop_seeds.go          # Stop seeding command
└── internal/                  # Internal packages
    ├── config/                # Configuration management
    │   └── config.go
    ├── logging/               # Logging system
    │   └── logger.go
    ├── cache/                 # Caching layer
    │   └── cache.go
    ├── qbittorrent/           # qBittorrent API client
    │   ├── client.go
    │   └── types.go
    ├── core/                  # Shared business logic
    │   ├── torrent_service.go
    │   ├── disk_service.go
    │   └── seeding_service.go
    └── bot/                   # Discord bot
        ├── bot.go
        └── commands/          # Discord slash commands
            ├── torrents.go
            ├── add.go
            ├── delete.go
            ├── diskspace.go
            ├── logs.go
            ├── seed.go
            └── help.go
```

### Step 3: Install Dependencies
```bash
go get github.com/spf13/cobra@latest           # CLI framework
go get github.com/joho/godotenv@latest         # Environment variables
go get github.com/sirupsen/logrus@latest       # Logging
go get github.com/patrickmn/go-cache@latest    # In-memory caching
go get github.com/bwmarrin/discordgo@latest    # Discord bot
go get github.com/olekukonko/tablewriter@latest # Pretty CLI tables
```

### Step 4: Create .env.example File
Include all configuration variables:
- Discord bot token, client ID, guild ID
- qBittorrent URL, username, password
- Save paths for different categories
- Seeding configuration
- Logging configuration
- Cache TTL settings

## Phase 2: Core Systems Implementation

### Step 5: Configuration System (`internal/config/`)


### Step 6: Logging System (`internal/logging/`)
- Implement structured logging with `logrus`
- Support multiple output destinations (file + stdout)
- Configure log levels, formatting, timestamps
- Create logging contexts for different components
- Implement log rotation if needed

### Step 7: Cache System (`internal/cache/`)
- Wrap `go-cache` with custom interface
- Implement TTL-based caching for:
  - qBittorrent session authentication
  - Torrent list data
  - Disk space information
- Add cache invalidation methods
- Create cache statistics/monitoring

## Phase 3: qBittorrent Integration

### Step 8: qBittorrent Types (`internal/qbittorrent/types.go`)
Define structs for:
- `Torrent` with all necessary fields
- `TorrentState` enum/constants
- API request/response types
- Error types specific to qBittorrent

### Step 9: qBittorrent Client (`internal/qbittorrent/client.go`)
Implement HTTP client with:
- Session management (login/logout)
- Cookie jar for session persistence
- Request timeout configuration
- Error handling and retries
- API methods:
  - `Login()` - authenticate with qBittorrent
  - `GetTorrents()` - list all torrents
  - `AddMagnet()` - add magnet links
  - `DeleteTorrents()` - remove torrents
  - `PauseTorrents()` - pause/stop torrents
  - `GetDiskSpace()` - check disk usage

### Step 10: Test qBittorrent Integration
- Create simple test program to verify API connectivity
- Test authentication, listing torrents, adding magnets
- Verify caching works correctly
- Handle common error scenarios

## Phase 4: Core Business Logic

### Step 11: Torrent Service (`internal/core/torrent_service.go`)
Implement business logic layer:
- Wrap qBittorrent client with caching
- Add filtering logic (by category, state, etc.)
- Implement torrent progress tracking
- Add validation for magnet links, categories
- Handle path mapping for different categories

### Step 12: Disk Service (`internal/core/disk_service.go`)
Implement disk space functionality:
- Cross-platform disk space checking
- Path validation and normalization
- Format bytes to human-readable units
- Cache disk space results

### Step 13: Seeding Service (`internal/core/seeding_service.go`)
Implement automatic seeding management:
- Track torrent download start/completion times
- Calculate seeding stop times based on multiplier
- Periodic background checking (every 5 minutes)
- Persistent storage of tracking data
- Automatic torrent pausing when seeding time reached

## Phase 5: CLI Implementation

### Step 14: Main Entry Point (`main.go`)
- Initialize Cobra CLI application
- Set up global flags (config file, log level, etc.)
- Add all subcommands
- Handle graceful shutdown signals
- Add version information

### Step 15: CLI Commands (`cmd/*.go`)
Implement each command:

**List Command (`cmd/list.go`)**:
- `--seeding` flag to show only seeding torrents
- `--category` flag to filter by category
- Pretty table output with progress bars
- Color coding for different states

**Add Command (`cmd/add.go`)**:
- Validate magnet link format
- `--category` flag with validation
- `--path` flag to override save path
- Progress indication for add operation

**Delete Command (`cmd/delete.go`)**:
- Interactive selection of torrents to delete
- `--category` filter
- `--delete-files` flag
- Confirmation prompts

**Disk Space Command (`cmd/diskspace.go`)**:
- `--path` flag to specify custom path
- Formatted output (GB, TB, percentages)
- Optional JSON output format

**Logs Command (`cmd/logs.go`)**:
- `--tail` flag to limit number of entries
- `--follow` flag for real-time log streaming
- `--level` filter by log level

**Seeding Commands**:
- Seed status overview
- Stop all seeds with confirmation
- Stop specific seeds with interactive selection

### Step 16: CLI Testing
- Test each command thoroughly
- Verify error handling and user feedback
- Test edge cases (empty results, network errors)
- Ensure proper exit codes

## Phase 6: Discord Bot Implementation

### Step 17: Discord Bot Core (`internal/bot/bot.go`)
- Initialize Discord session
- Register slash commands
- Handle bot events (ready, interaction)
- Implement command routing
- Add error handling and logging

### Step 18: Discord Slash Commands (`internal/bot/commands/*.go`)
Implement Discord equivalents of CLI commands:

**Torrents Command**:
- Paginated embed messages for torrent lists
- Real-time progress updates
- Filter buttons (all, downloading, seeding)

**Add Magnet Command**:
- Magnet link validation
- Category selection dropdown
- Progress tracking with message updates
- Success/error feedback

**Delete Command**:
- Interactive torrent selection
- Confirmation buttons
- Category filtering

**Disk Space Command**:
- Formatted embed with usage statistics
- Optional chart generation (using ASCII or image)

**Logs Command**:
- Paginated log display
- Filter by level buttons

**Seeding Management**:
- Status overview embeds
- Stop buttons with confirmations

### Step 19: Discord Bot Features
- Command permission checking
- Rate limiting protection
- Help command with usage examples
- Error message formatting
- Activity status updates

## Phase 7: Daemon Mode Implementation

### Step 20: Daemon Command (`cmd/daemon.go`)
Implement daemon mode that runs:
- Discord bot in background goroutine
- Seeding manager in background goroutine
- Graceful shutdown handling (SIGINT, SIGTERM)
- Health checking and restart logic
- PID file management

### Step 21: Background Services Coordination
- Shared context for shutdown coordination
- Wait groups for proper cleanup
- Error handling and recovery
- Service health monitoring
- Graceful degradation if services fail

### Step 22: Integration Testing
- Test daemon startup/shutdown
- Verify Discord bot responds while daemon runs
- Test CLI commands work while daemon runs
- Check seeding manager operates correctly
- Test recovery from failures

## Phase 8: Advanced Features & Polish

### Step 23: Enhanced Caching Strategy
- Implement cache warming on startup
- Add cache statistics and monitoring
- Implement intelligent cache invalidation
- Add cache debugging commands

### Step 24: Error Handling & Recovery
- Implement retry logic for network operations
- Add circuit breaker pattern for qBittorrent connectivity
- Graceful degradation when services unavailable
- User-friendly error messages

### Step 25: Configuration Validation
- Validate all config values on startup
- Check qBittorrent connectivity
- Verify Discord token validity
- Test save paths accessibility
- Provide helpful error messages for misconfigurations

### Step 26: Documentation & Help
- Add comprehensive help text for all commands
- Create usage examples for common scenarios
- Add troubleshooting guide
- Document configuration options
- Create quick start guide

## Phase 9: Testing & Quality Assurance

### Step 27: Unit Testing
- Write tests for core business logic
- Mock qBittorrent API for testing
- Test configuration loading
- Test caching behavior
- Test error conditions

### Step 28: Integration Testing
- Test with real qBittorrent instance
- Test Discord bot interactions
- Test daemon mode stability
- Test concurrent CLI usage
- Performance testing with large torrent lists

### Step 29: User Acceptance Testing
- Test with different operating systems
- Test various qBittorrent configurations
- Verify user workflows end-to-end
- Test edge cases and error scenarios

## Phase 10: Deployment & Distribution

### Step 30: Build System
- Create Makefile for building
- Set up cross-compilation for different platforms
- Create release scripts
- Add version information to builds

### Step 31: Distribution
- Create GitHub releases
- Provide pre-built binaries
- Create installation instructions
- Set up automatic release pipeline

### Step 32: Monitoring & Maintenance
- Add application metrics
- Monitor resource usage
- Set up log analysis
- Plan for updates and maintenance

## Key Implementation Notes

### Command Mapping (Discord ↔ CLI)
- `/torrents` ↔ `list`
- `/seed` ↔ `list --seeding`
- `/addmagnet` ↔ `add <magnet>`
- `/delete` ↔ `delete`
- `/diskspace` ↔ `diskspace`
- `/seedstatus` ↔ `seed-status`
- `/stopallseeds` ↔ `stop-seeds --all`
- `/logs` ↔ `logs`

### Shared Components
- Configuration system used by both CLI and Discord bot
- qBittorrent client shared between all components
- Logging system captures activity from all sources
- Caching improves performance for both interfaces
- Core business logic prevents code duplication

### Daemon vs CLI Mode
- **Daemon Mode**: Long-running process with Discord bot + seeding manager
- **CLI Mode**: One-shot commands that exit after completion
- Both modes can run simultaneously
- Shared configuration and logging
- Cache can be shared between modes

### Error Handling Strategy
- Network errors: Retry with exponential backoff
- Authentication errors: Re-login automatically
- Configuration errors: Fail fast with helpful messages
- User errors: Provide clear feedback and suggestions
- Unexpected errors: Log details, show user-friendly message

This plan provides a comprehensive roadmap for building your Go torrent CLI + Discord bot application. Each phase builds upon the previous ones, allowing you to test and validate functionality incrementally.