package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/raainshe/akira/internal/config"
)

// Component represents different parts of the application for contextualized logging
type Component string

const (
	ComponentDiscordBot  Component = "discord_bot"
	ComponentQBittorrent Component = "qbittorrent"
	ComponentCLI         Component = "cli"
	ComponentSeedingMgr  Component = "seeding_manager"
	ComponentCache       Component = "cache"
	ComponentConfig      Component = "config"
	ComponentCore        Component = "core"
	ComponentMain        Component = "main"
)

// Logger wraps logrus.Logger with additional functionality
type Logger struct {
	*logrus.Logger
	config    *config.LoggingConfig
	component Component
}

// loggerInstance holds the global logger instance
var loggerInstance *Logger

// Initialize sets up the global logger with the provided configuration
func Initialize(cfg *config.LoggingConfig) (*Logger, error) {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level '%s': %w", cfg.Level, err)
	}
	logger.SetLevel(level)

	// Create multi-writer for outputs
	var writers []io.Writer

	// Add stdout writer if enabled
	if cfg.ToStdout {
		writers = append(writers, os.Stdout)
	}

	// Add file writer with rotation
	if cfg.File != "" {
		// Ensure log directory exists
		logDir := filepath.Dir(cfg.File)
		if logDir != "." {
			if err := os.MkdirAll(logDir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create log directory '%s': %w", logDir, err)
			}
		}

		fileWriter := &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSize,    // megabytes
			MaxBackups: cfg.MaxBackups, // number of backup files
			MaxAge:     cfg.MaxAge,     // days
			Compress:   cfg.Compress,   // compress rotated files
		}
		writers = append(writers, fileWriter)
	}

	if len(writers) == 0 {
		// Fallback to stdout if no writers configured
		writers = append(writers, os.Stdout)
	}

	// Set multi-writer output
	logger.SetOutput(io.MultiWriter(writers...))

	// Set formatter based on output type
	if cfg.ToStdout && len(writers) == 1 {
		// Human-readable format for stdout-only
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	} else {
		// JSON format for file logging or mixed outputs
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	}

	// Create wrapper logger
	appLogger := &Logger{
		Logger:    logger,
		config:    cfg,
		component: ComponentMain,
	}

	// Set global instance
	loggerInstance = appLogger

	// Log initialization only if level is info or below (to reduce CLI verbosity)
	if level <= logrus.InfoLevel {
		appLogger.WithField("component", ComponentMain).Info("Logger initialized successfully")
	}

	return appLogger, nil
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if loggerInstance == nil {
		// Create a fallback logger if not initialized
		fallbackLogger := logrus.New()
		fallbackLogger.SetLevel(logrus.InfoLevel)
		fallbackLogger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})

		return &Logger{
			Logger:    fallbackLogger,
			component: ComponentMain,
		}
	}
	return loggerInstance
}

// WithComponent creates a new logger instance with a specific component context
func (l *Logger) WithComponent(component Component) *Logger {
	return &Logger{
		Logger:    l.Logger,
		config:    l.config,
		component: component,
	}
}

// WithField adds a field to the logger entry and ensures component is included
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithFields(logrus.Fields{
		"component": l.component,
		key:         value,
	})
}

// WithFields adds multiple fields to the logger entry and ensures component is included
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	if fields == nil {
		fields = make(logrus.Fields)
	}
	fields["component"] = l.component
	return l.Logger.WithFields(fields)
}

// WithError adds an error field to the logger entry and ensures component is included
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithFields(logrus.Fields{
		"component": l.component,
		"error":     err,
	})
}

// Component-specific logging methods that automatically include component context

// Debug logs a debug message with component context
func (l *Logger) Debug(args ...interface{}) {
	l.WithField("component", l.component).Debug(args...)
}

// Debugf logs a formatted debug message with component context
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.WithField("component", l.component).Debugf(format, args...)
}

// Info logs an info message with component context
func (l *Logger) Info(args ...interface{}) {
	l.WithField("component", l.component).Info(args...)
}

// Infof logs a formatted info message with component context
func (l *Logger) Infof(format string, args ...interface{}) {
	l.WithField("component", l.component).Infof(format, args...)
}

// Warn logs a warning message with component context
func (l *Logger) Warn(args ...interface{}) {
	l.WithField("component", l.component).Warn(args...)
}

// Warnf logs a formatted warning message with component context
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.WithField("component", l.component).Warnf(format, args...)
}

// Error logs an error message with component context
func (l *Logger) Error(args ...interface{}) {
	l.WithField("component", l.component).Error(args...)
}

// Errorf logs a formatted error message with component context
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.WithField("component", l.component).Errorf(format, args...)
}

// Fatal logs a fatal message with component context and exits
func (l *Logger) Fatal(args ...interface{}) {
	l.WithField("component", l.component).Fatal(args...)
}

// Fatalf logs a formatted fatal message with component context and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.WithField("component", l.component).Fatalf(format, args...)
}

// Panic logs a panic message with component context and panics
func (l *Logger) Panic(args ...interface{}) {
	l.WithField("component", l.component).Panic(args...)
}

// Panicf logs a formatted panic message with component context and panics
func (l *Logger) Panicf(format string, args ...interface{}) {
	l.WithField("component", l.component).Panicf(format, args...)
}

// Convenience functions for getting component-specific loggers

// GetDiscordLogger returns a logger instance configured for Discord bot operations
func GetDiscordLogger() *Logger {
	return GetLogger().WithComponent(ComponentDiscordBot)
}

// GetQBittorrentLogger returns a logger instance configured for qBittorrent operations
func GetQBittorrentLogger() *Logger {
	return GetLogger().WithComponent(ComponentQBittorrent)
}

// GetCLILogger returns a logger instance configured for CLI operations
func GetCLILogger() *Logger {
	return GetLogger().WithComponent(ComponentCLI)
}

// GetSeedingLogger returns a logger instance configured for seeding manager operations
func GetSeedingLogger() *Logger {
	return GetLogger().WithComponent(ComponentSeedingMgr)
}

// GetCacheLogger returns a logger instance configured for cache operations
func GetCacheLogger() *Logger {
	return GetLogger().WithComponent(ComponentCache)
}

// GetConfigLogger returns a logger instance configured for configuration operations
func GetConfigLogger() *Logger {
	return GetLogger().WithComponent(ComponentConfig)
}

// GetCoreLogger returns a logger instance configured for core business logic operations
func GetCoreLogger() *Logger {
	return GetLogger().WithComponent(ComponentCore)
}

// SetLogLevel changes the log level at runtime
func SetLogLevel(levelStr string) error {
	logger := GetLogger()
	level, err := logrus.ParseLevel(strings.ToLower(levelStr))
	if err != nil {
		return fmt.Errorf("invalid log level '%s': %w", levelStr, err)
	}

	logger.Logger.SetLevel(level)
	logger.Infof("Log level changed to: %s", level.String())
	return nil
}

// GetLogLevel returns the current log level
func GetLogLevel() string {
	return GetLogger().Logger.GetLevel().String()
}

// LogCommand logs a CLI command execution
func LogCommand(command string, args []string, user string) {
	logger := GetCLILogger()
	logger.WithFields(logrus.Fields{
		"command": command,
		"args":    args,
		"user":    user,
	}).Info("CLI command executed")
}

// LogTorrentAdded logs when a torrent is added
func LogTorrentAdded(magnetURI, category, savePath string) {
	logger := GetQBittorrentLogger()
	logger.WithFields(logrus.Fields{
		"action":     "torrent_added",
		"magnet_uri": maskMagnetURI(magnetURI),
		"category":   category,
		"save_path":  savePath,
	}).Info("Torrent added successfully")
}

// LogTorrentCompleted logs when a torrent completes downloading
func LogTorrentCompleted(torrentName, hash string, downloadDuration string) {
	logger := GetQBittorrentLogger()
	logger.WithFields(logrus.Fields{
		"action":            "torrent_completed",
		"torrent_name":      torrentName,
		"torrent_hash":      hash,
		"download_duration": downloadDuration,
	}).Info("Torrent download completed")
}

// LogTorrentDeleted logs when a torrent is deleted
func LogTorrentDeleted(torrentName, hash string, deleteFiles bool) {
	logger := GetQBittorrentLogger()
	logger.WithFields(logrus.Fields{
		"action":       "torrent_deleted",
		"torrent_name": torrentName,
		"torrent_hash": hash,
		"delete_files": deleteFiles,
	}).Info("Torrent deleted")
}

// LogSeedingStopped logs when seeding is automatically stopped
func LogSeedingStopped(torrentName, hash string, seedingDuration string) {
	logger := GetSeedingLogger()
	logger.WithFields(logrus.Fields{
		"action":           "seeding_stopped",
		"torrent_name":     torrentName,
		"torrent_hash":     hash,
		"seeding_duration": seedingDuration,
	}).Info("Automatic seeding stopped")
}

// LogDiscordCommand logs Discord slash command usage
func LogDiscordCommand(command string, userID, guildID string, options map[string]interface{}) {
	logger := GetDiscordLogger()
	logger.WithFields(logrus.Fields{
		"command":  command,
		"user_id":  userID,
		"guild_id": guildID,
		"options":  options,
	}).Info("Discord command executed")
}

// LogError logs an error with additional context
func LogError(component Component, operation string, err error, context map[string]interface{}) {
	logger := GetLogger().WithComponent(component)
	fields := logrus.Fields{
		"operation": operation,
		"error":     err.Error(),
	}

	// Add context fields
	for k, v := range context {
		fields[k] = v
	}

	logger.WithFields(fields).Error("Operation failed")
}

// Helper functions

// maskMagnetURI masks sensitive parts of magnet URI for logging
func maskMagnetURI(magnetURI string) string {
	if len(magnetURI) < 50 {
		return magnetURI
	}

	// Show first 20 and last 10 characters, mask the middle
	return magnetURI[:20] + "..." + magnetURI[len(magnetURI)-10:]
}

// Shutdown gracefully shuts down the logging system
func Shutdown() {
	logger := GetLogger()
	if logger.config != nil && logger.config.File != "" {
		logger.Info("Shutting down logging system")

		// If using lumberjack, we need to close it properly
		if writer, ok := logger.Logger.Out.(io.Closer); ok {
			writer.Close()
		}
	}
}
