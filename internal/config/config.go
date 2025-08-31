package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Discord     DiscordConfig     `json:"discord"`
	QBittorrent QBittorrentConfig `json:"qbittorrent"`
	Cache       CacheConfig       `json:"cache"`
	Logging     LoggingConfig     `json:"logging"`
	Seeding     SeedingConfig     `json:"seeding"`
	Proxy       ProxyConfig       `json:"proxy"`
}

// DiscordConfig holds Discord bot configuration
type DiscordConfig struct {
	BotToken string   `json:"bot_token"`
	GuildIDs []string `json:"guild_ids"`
}

// QBittorrentConfig holds qBittorrent client configuration
type QBittorrentConfig struct {
	URL                string          `json:"url"`
	Username           string          `json:"username"`
	Password           string          `json:"password"`
	SavePaths          SavePathsConfig `json:"save_paths"`
	DiskSpaceCheckPath string          `json:"disk_space_check_path"`
	RequestTimeout     time.Duration   `json:"request_timeout"`
}

// SavePathsConfig holds different category save paths
type SavePathsConfig struct {
	Default string `json:"default"`
	Series  string `json:"series"`
	Movies  string `json:"movies"`
	Anime   string `json:"anime"`
}

// CacheConfig holds caching configuration
type CacheConfig struct {
	TorrentListTTL    time.Duration `json:"torrent_list_ttl"`
	TorrentDetailsTTL time.Duration `json:"torrent_details_ttl"`
	AuthSessionTTL    time.Duration `json:"auth_session_ttl"`
	DiskSpaceTTL      time.Duration `json:"disk_space_ttl"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`
	MaxItems          int           `json:"max_items"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `json:"level"`
	File       string `json:"file"`
	MaxSize    int    `json:"max_size"`    // megabytes
	MaxBackups int    `json:"max_backups"` // number of backup files
	MaxAge     int    `json:"max_age"`     // days
	Compress   bool   `json:"compress"`    // compress rotated files
	ToStdout   bool   `json:"to_stdout"`   // also log to stdout
}

// SeedingConfig holds automatic seeding management configuration
type SeedingConfig struct {
	TimeMultiplier   float64       `json:"time_multiplier"`    // multiplier for seeding time (e.g., 10 means seed for 10x download time)
	CheckInterval    time.Duration `json:"check_interval"`     // how often to check for torrents to stop seeding
	TrackingDataFile string        `json:"tracking_data_file"` // file to store seeding tracking data
}

// ProxyConfig holds proxy configuration (optional)
type ProxyConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Enabled  bool   `json:"enabled"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// Don't fail if .env doesn't exist, just continue with system env vars
		fmt.Printf("Warning: .env file not found, using system environment variables\n")
	}

	config := &Config{}

	// Load Discord configuration
	config.Discord.BotToken = getEnvOrDefault("DISCORD_BOT_TOKEN", "")
	guildID := getEnvOrDefault("DISCORD_GUILD_ID", "")
	if guildID != "" {
		config.Discord.GuildIDs = []string{guildID}
	}

	// Load qBittorrent configuration
	config.QBittorrent.URL = getEnvOrDefault("QBITTORRENT_URL", "http://localhost:8080")
	config.QBittorrent.Username = getEnvOrDefault("QBITTORRENT_USERNAME", "admin")
	config.QBittorrent.Password = getEnvOrDefault("QBITTORRENT_PASSWORD", "")
	config.QBittorrent.RequestTimeout = parseDurationOrDefault("QBITTORRENT_REQUEST_TIMEOUT", 30*time.Second)

	// Load save paths
	config.QBittorrent.SavePaths.Default = getEnvOrDefault("QBITTORRENT_DEFAULT_SAVE_PATH", "/downloads/default")
	config.QBittorrent.SavePaths.Series = getEnvOrDefault("QBITTORRENT_SERIES_SAVE_PATH", "")
	config.QBittorrent.SavePaths.Movies = getEnvOrDefault("QBITTORRENT_MOVIES_SAVE_PATH", "")
	config.QBittorrent.SavePaths.Anime = getEnvOrDefault("QBITTORRENT_ANIME_SAVE_PATH", "")

	// Use default path as fallback for category paths if not set
	if config.QBittorrent.SavePaths.Series == "" {
		config.QBittorrent.SavePaths.Series = config.QBittorrent.SavePaths.Default
	}
	if config.QBittorrent.SavePaths.Movies == "" {
		config.QBittorrent.SavePaths.Movies = config.QBittorrent.SavePaths.Default
	}
	if config.QBittorrent.SavePaths.Anime == "" {
		config.QBittorrent.SavePaths.Anime = config.QBittorrent.SavePaths.Default
	}

	config.QBittorrent.DiskSpaceCheckPath = getEnvOrDefault("DISK_SPACE_CHECK_PATH", "/")

	// Load cache configuration
	config.Cache.TorrentListTTL = parseDurationOrDefault("CACHE_TORRENT_LIST_TTL", 30*time.Second)
	config.Cache.TorrentDetailsTTL = parseDurationOrDefault("CACHE_TORRENT_DETAILS_TTL", 5*time.Minute)
	config.Cache.AuthSessionTTL = parseDurationOrDefault("CACHE_AUTH_SESSION_TTL", 1*time.Hour)
	config.Cache.DiskSpaceTTL = parseDurationOrDefault("CACHE_DISK_SPACE_TTL", 5*time.Minute)
	config.Cache.CleanupInterval = parseDurationOrDefault("CACHE_CLEANUP_INTERVAL", 10*time.Minute)
	config.Cache.MaxItems = parseIntOrDefault("CACHE_MAX_ITEMS", 1000)

	// Load logging configuration
	config.Logging.Level = getEnvOrDefault("LOG_LEVEL", "info")
	config.Logging.File = getEnvOrDefault("LOG_FILE", "bot_activity.log")
	config.Logging.MaxSize = parseIntOrDefault("LOG_MAX_SIZE", 100)
	config.Logging.MaxBackups = parseIntOrDefault("LOG_MAX_BACKUPS", 5)
	config.Logging.MaxAge = parseIntOrDefault("LOG_MAX_AGE", 30)
	config.Logging.Compress = parseBoolOrDefault("LOG_COMPRESS", true)
	config.Logging.ToStdout = parseBoolOrDefault("LOG_TO_STDOUT", true)

	// Load seeding configuration
	config.Seeding.TimeMultiplier = parseFloat64OrDefault("SEEDING_TIME_MULTIPLIER", 10.0)
	config.Seeding.CheckInterval = parseDurationOrDefault("SEEDING_CHECK_INTERVAL", 5*time.Minute)
	config.Seeding.TrackingDataFile = getEnvOrDefault("SEEDING_TRACKING_DATA_FILE", "seeding_tracking.json")

	// Load proxy configuration (optional)
	config.Proxy.Host = getEnvOrDefault("PROXY_HOST", "")
	config.Proxy.Port = parseIntOrDefault("PROXY_PORT", 0)
	config.Proxy.Username = getEnvOrDefault("PROXY_USER", "")
	config.Proxy.Password = getEnvOrDefault("PROXY_PASS", "")
	config.Proxy.Enabled = config.Proxy.Host != "" && config.Proxy.Port > 0

	// Validate required configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate checks that all required configuration is present and valid
func (c *Config) Validate() error {
	if c.Discord.BotToken == "" {
		return fmt.Errorf("DISCORD_BOT_TOKEN is required")
	}

	if c.QBittorrent.URL == "" {
		return fmt.Errorf("QBITTORRENT_URL is required")
	}

	if c.QBittorrent.Username == "" {
		return fmt.Errorf("QBITTORRENT_USERNAME is required")
	}

	if c.QBittorrent.Password == "" {
		return fmt.Errorf("QBITTORRENT_PASSWORD is required")
	}

	if c.QBittorrent.SavePaths.Default == "" {
		return fmt.Errorf("QBITTORRENT_DEFAULT_SAVE_PATH is required")
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"trace": true, "debug": true, "info": true, "warn": true, "error": true, "fatal": true, "panic": true,
	}
	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (must be one of: trace, debug, info, warn, error, fatal, panic)", c.Logging.Level)
	}

	// Validate seeding time multiplier
	if c.Seeding.TimeMultiplier <= 0 {
		return fmt.Errorf("seeding time multiplier must be greater than 0, got: %f", c.Seeding.TimeMultiplier)
	}

	return nil
}

// GetSavePathForCategory returns the save path for a given category
func (c *Config) GetSavePathForCategory(category string) string {
	category = strings.ToLower(category)
	switch category {
	case "series":
		return c.QBittorrent.SavePaths.Series
	case "movies":
		return c.QBittorrent.SavePaths.Movies
	case "anime":
		return c.QBittorrent.SavePaths.Anime
	default:
		return c.QBittorrent.SavePaths.Default
	}
}

// GetValidCategories returns a list of valid torrent categories
func (c *Config) GetValidCategories() []string {
	return []string{"series", "movies", "anime", "default"}
}

// Helper functions for parsing environment variables

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func parseFloat64OrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func parseBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func parseDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
