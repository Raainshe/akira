package core

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/raainshe/akira/internal/cache"
	"github.com/raainshe/akira/internal/config"
	"github.com/raainshe/akira/internal/logging"
	"github.com/raainshe/akira/internal/qbittorrent"
)

// DiskService provides cross-platform disk space operations
type DiskService struct {
	config *config.Config
	cache  *cache.CacheManager
	logger *logging.Logger
}

// DiskInfo represents disk space information for a path
type DiskInfo struct {
	Path        string    `json:"path"`
	Total       int64     `json:"total"`        // Total space in bytes
	Used        int64     `json:"used"`         // Used space in bytes
	Free        int64     `json:"free"`         // Free space in bytes
	Available   int64     `json:"available"`    // Available space for non-root users (Unix)
	UsedPercent float64   `json:"used_percent"` // Used percentage (0-100)
	FreePercent float64   `json:"free_percent"` // Free percentage (0-100)
	Filesystem  string    `json:"filesystem"`   // Filesystem type (if available)
	MountPoint  string    `json:"mount_point"`  // Mount point (Unix)
	LastChecked time.Time `json:"last_checked"` // When this info was last updated
}

// DiskHealthStatus represents the health status of disk space
type DiskHealthStatus string

const (
	DiskHealthGood     DiskHealthStatus = "good"     // > 20% free space
	DiskHealthWarning  DiskHealthStatus = "warning"  // 10-20% free space
	DiskHealthCritical DiskHealthStatus = "critical" // 5-10% free space
	DiskHealthDanger   DiskHealthStatus = "danger"   // < 5% free space
)

// DiskSummary represents a summary of all monitored disk spaces
type DiskSummary struct {
	Paths         map[string]*DiskInfo `json:"paths"`          // Path -> DiskInfo mapping
	TotalSpace    int64                `json:"total_space"`    // Sum of all total space
	TotalUsed     int64                `json:"total_used"`     // Sum of all used space
	TotalFree     int64                `json:"total_free"`     // Sum of all free space
	WorstHealth   DiskHealthStatus     `json:"worst_health"`   // Worst health status across all paths
	WarningPaths  []string             `json:"warning_paths"`  // Paths with warnings
	CriticalPaths []string             `json:"critical_paths"` // Paths with critical status
	LastUpdated   time.Time            `json:"last_updated"`   // When this summary was generated
}

// NewDiskService creates a new disk service instance
func NewDiskService(config *config.Config, cache *cache.CacheManager) *DiskService {
	return &DiskService{
		config: config,
		cache:  cache,
		logger: logging.GetCoreLogger(),
	}
}

// GetDiskSpace retrieves disk space information for a specific path
func (ds *DiskService) GetDiskSpace(ctx context.Context, path string) (*DiskInfo, error) {
	// Validate and normalize path
	normalizedPath, err := ds.normalizePath(path)
	if err != nil {
		ds.logger.WithError(err).WithField("path", path).Error("Failed to normalize path")
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	ds.logger.WithField("path", normalizedPath).Debug("Getting disk space information")

	// Try to get from cache first
	if ds.cache != nil {
		if cachedDisk, found := ds.cache.GetDiskSpace(normalizedPath); found {
			ds.logger.WithField("path", normalizedPath).Debug("Using cached disk space information")
			return &DiskInfo{
				Path:        normalizedPath,
				Total:       cachedDisk.Total,
				Used:        cachedDisk.Used,
				Free:        cachedDisk.Free,
				Available:   cachedDisk.Free, // For cache compatibility
				UsedPercent: ds.calculatePercentage(cachedDisk.Used, cachedDisk.Total),
				FreePercent: ds.calculatePercentage(cachedDisk.Free, cachedDisk.Total),
				LastChecked: cachedDisk.UpdatedAt,
			}, nil
		}
	}

	// Get fresh disk space information
	diskInfo, err := ds.getDiskSpacePlatform(normalizedPath)
	if err != nil {
		ds.logger.WithError(err).WithField("path", normalizedPath).Error("Failed to get disk space")
		return nil, fmt.Errorf("failed to get disk space for %s: %w", normalizedPath, err)
	}

	// Cache the result
	if ds.cache != nil {
		cacheInfo := cache.NewDiskSpaceInfo(normalizedPath, diskInfo.Total, diskInfo.Used, diskInfo.Free)
		ds.cache.SetDiskSpace(normalizedPath, cacheInfo)
	}

	ds.logger.WithFields(map[string]interface{}{
		"path":         normalizedPath,
		"total":        qbittorrent.FormatBytes(diskInfo.Total),
		"used":         qbittorrent.FormatBytes(diskInfo.Used),
		"free":         qbittorrent.FormatBytes(diskInfo.Free),
		"used_percent": fmt.Sprintf("%.1f%%", diskInfo.UsedPercent),
	}).Info("Disk space information retrieved")

	return diskInfo, nil
}

// GetAllDiskSpaces retrieves disk space for all configured torrent paths
func (ds *DiskService) GetAllDiskSpaces(ctx context.Context) (*DiskSummary, error) {
	ds.logger.Debug("Getting disk space for all configured paths")

	summary := &DiskSummary{
		Paths:         make(map[string]*DiskInfo),
		WorstHealth:   DiskHealthGood,
		WarningPaths:  []string{},
		CriticalPaths: []string{},
		LastUpdated:   time.Now(),
	}

	// Get all configured paths
	paths := ds.getAllConfiguredPaths()

	for _, path := range paths {
		diskInfo, err := ds.GetDiskSpace(ctx, path)
		if err != nil {
			ds.logger.WithError(err).WithField("path", path).Warn("Failed to get disk space for configured path")
			continue
		}

		summary.Paths[path] = diskInfo
		summary.TotalSpace += diskInfo.Total
		summary.TotalUsed += diskInfo.Used
		summary.TotalFree += diskInfo.Free

		// Check health status
		health := ds.getDiskHealthStatus(diskInfo)
		if ds.isWorseHealth(health, summary.WorstHealth) {
			summary.WorstHealth = health
		}

		// Add to warning/critical lists
		switch health {
		case DiskHealthWarning:
			summary.WarningPaths = append(summary.WarningPaths, path)
		case DiskHealthCritical, DiskHealthDanger:
			summary.CriticalPaths = append(summary.CriticalPaths, path)
		}
	}

	ds.logger.WithFields(map[string]interface{}{
		"paths_checked":  len(summary.Paths),
		"total_space":    qbittorrent.FormatBytes(summary.TotalSpace),
		"total_free":     qbittorrent.FormatBytes(summary.TotalFree),
		"worst_health":   summary.WorstHealth,
		"warning_paths":  len(summary.WarningPaths),
		"critical_paths": len(summary.CriticalPaths),
	}).Info("Disk space summary generated")

	return summary, nil
}

// CheckDiskHealth performs a health check on all configured disk paths
func (ds *DiskService) CheckDiskHealth(ctx context.Context) (map[string]DiskHealthStatus, error) {
	ds.logger.Debug("Performing disk health check")

	healthStatus := make(map[string]DiskHealthStatus)
	paths := ds.getAllConfiguredPaths()

	for _, path := range paths {
		diskInfo, err := ds.GetDiskSpace(ctx, path)
		if err != nil {
			ds.logger.WithError(err).WithField("path", path).Warn("Failed to check disk health for path")
			healthStatus[path] = DiskHealthDanger // Assume worst case if we can't check
			continue
		}

		health := ds.getDiskHealthStatus(diskInfo)
		healthStatus[path] = health

		// Log warnings for problematic paths
		if health != DiskHealthGood {
			ds.logger.WithFields(map[string]interface{}{
				"path":         path,
				"health":       health,
				"free_space":   qbittorrent.FormatBytes(diskInfo.Free),
				"free_percent": fmt.Sprintf("%.1f%%", diskInfo.FreePercent),
			}).Warn("Disk space health issue detected")
		}
	}

	ds.logger.WithField("health_status", healthStatus).Info("Disk health check completed")
	return healthStatus, nil
}

// FormatDiskInfo formats disk information into a human-readable string
func (ds *DiskService) FormatDiskInfo(diskInfo *DiskInfo) string {
	return fmt.Sprintf(
		"Path: %s\n"+
			"Total: %s\n"+
			"Used: %s (%.1f%%)\n"+
			"Free: %s (%.1f%%)\n"+
			"Health: %s",
		diskInfo.Path,
		qbittorrent.FormatBytes(diskInfo.Total),
		qbittorrent.FormatBytes(diskInfo.Used), diskInfo.UsedPercent,
		qbittorrent.FormatBytes(diskInfo.Free), diskInfo.FreePercent,
		ds.getDiskHealthStatus(diskInfo),
	)
}

// Platform-specific implementations are in disk_service_unix.go and disk_service_windows.go

// Helper methods

// normalizePath normalizes a path for the current operating system
func (ds *DiskService) normalizePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Clean the path
	cleanPath := filepath.Clean(path)

	// Convert to absolute path if relative
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to convert to absolute path: %w", err)
	}

	return absPath, nil
}

// getAllConfiguredPaths returns all configured torrent save paths
func (ds *DiskService) getAllConfiguredPaths() []string {
	paths := []string{}

	// Add all configured save paths
	if ds.config.QBittorrent.SavePaths.Default != "" {
		paths = append(paths, ds.config.QBittorrent.SavePaths.Default)
	}
	if ds.config.QBittorrent.SavePaths.Series != "" && ds.config.QBittorrent.SavePaths.Series != ds.config.QBittorrent.SavePaths.Default {
		paths = append(paths, ds.config.QBittorrent.SavePaths.Series)
	}
	if ds.config.QBittorrent.SavePaths.Movies != "" && ds.config.QBittorrent.SavePaths.Movies != ds.config.QBittorrent.SavePaths.Default {
		paths = append(paths, ds.config.QBittorrent.SavePaths.Movies)
	}
	if ds.config.QBittorrent.SavePaths.Anime != "" && ds.config.QBittorrent.SavePaths.Anime != ds.config.QBittorrent.SavePaths.Default {
		paths = append(paths, ds.config.QBittorrent.SavePaths.Anime)
	}

	// Add disk space check path if different
	if ds.config.QBittorrent.DiskSpaceCheckPath != "" {
		found := false
		for _, existing := range paths {
			if existing == ds.config.QBittorrent.DiskSpaceCheckPath {
				found = true
				break
			}
		}
		if !found {
			paths = append(paths, ds.config.QBittorrent.DiskSpaceCheckPath)
		}
	}

	// Remove duplicates and empty paths
	uniquePaths := []string{}
	seen := make(map[string]bool)
	for _, path := range paths {
		if path != "" && !seen[path] {
			uniquePaths = append(uniquePaths, path)
			seen[path] = true
		}
	}

	return uniquePaths
}

// calculatePercentage calculates percentage with proper handling of zero division
func (ds *DiskService) calculatePercentage(part, total int64) float64 {
	if total == 0 {
		return 0.0
	}
	return (float64(part) / float64(total)) * 100.0
}

// getDiskHealthStatus determines the health status based on free space percentage
func (ds *DiskService) getDiskHealthStatus(diskInfo *DiskInfo) DiskHealthStatus {
	freePercent := diskInfo.FreePercent

	if freePercent < 5.0 {
		return DiskHealthDanger
	} else if freePercent < 10.0 {
		return DiskHealthCritical
	} else if freePercent < 20.0 {
		return DiskHealthWarning
	}
	return DiskHealthGood
}

// isWorseHealth compares two health statuses and returns true if the first is worse
func (ds *DiskService) isWorseHealth(health1, health2 DiskHealthStatus) bool {
	healthOrder := map[DiskHealthStatus]int{
		DiskHealthGood:     0,
		DiskHealthWarning:  1,
		DiskHealthCritical: 2,
		DiskHealthDanger:   3,
	}
	return healthOrder[health1] > healthOrder[health2]
}

// Legacy compatibility method for qBittorrent client
func (ds *DiskService) GetDiskSpaceForClient(ctx context.Context, path string) (*qbittorrent.DiskSpace, error) {
	diskInfo, err := ds.GetDiskSpace(ctx, path)
	if err != nil {
		return nil, err
	}

	return &qbittorrent.DiskSpace{
		Total: diskInfo.Total,
		Used:  diskInfo.Used,
		Free:  diskInfo.Free,
	}, nil
}
