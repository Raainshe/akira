package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/raainshe/akira/internal/cache"
	"github.com/raainshe/akira/internal/config"
	"github.com/raainshe/akira/internal/logging"
	"github.com/raainshe/akira/internal/qbittorrent"
)

// DiskService provides cross-platform disk space operations
type DiskService struct {
	config   *config.Config
	cache    *cache.CacheManager
	qbClient *qbittorrent.Client
	logger   *logging.Logger
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

// DriveDiskInfo represents disk space for a unique drive/volume
type DriveDiskInfo struct {
	DriveID string           `json:"drive_id"`
	Free    int64            `json:"free"`
	Health  DiskHealthStatus `json:"health"`
	Paths   []string         `json:"paths"`
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
	Drives        map[string]*DriveDiskInfo `json:"drives"`
	DriveOrder    []string                  `json:"drive_order"`
	Paths         map[string]*DiskInfo      `json:"paths"`
	TotalSpace    int64                     `json:"total_space"`
	TotalUsed     int64                     `json:"total_used"`
	TotalFree     int64                     `json:"total_free"`
	WorstHealth   DiskHealthStatus          `json:"worst_health"`
	WarningPaths  []string                  `json:"warning_paths"`
	CriticalPaths []string                  `json:"critical_paths"`
	LastUpdated   time.Time                 `json:"last_updated"`
}

// NewDiskService creates a new disk service instance
func NewDiskService(config *config.Config, cache *cache.CacheManager, qbClient *qbittorrent.Client) *DiskService {
	return &DiskService{
		config:   config,
		cache:    cache,
		qbClient: qbClient,
		logger:   logging.GetCoreLogger(),
	}
}

// GetDiskSpace retrieves disk space information for a specific path
func (ds *DiskService) GetDiskSpace(ctx context.Context, path string) (*DiskInfo, error) {
	normalizedPath, err := ds.normalizePath(path)
	if err != nil {
		ds.logger.WithError(err).WithField("path", path).Error("Failed to normalize path")
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	ds.logger.WithField("path", normalizedPath).Debug("Getting disk space information")

	cacheKey := ds.diskCacheKey(normalizedPath)
	if ds.cache != nil {
		if cachedDisk, found := ds.cache.GetDiskSpace(cacheKey); found {
			ds.logger.WithField("path", normalizedPath).Debug("Using cached disk space information")
			return ds.diskInfoFromCache(normalizedPath, cachedDisk), nil
		}
	}

	diskInfo, err := ds.fetchDiskSpace(ctx, normalizedPath, freeSpaceQueryPath(normalizedPath))
	if err != nil {
		ds.logger.WithError(err).WithField("path", normalizedPath).Error("Failed to get disk space")
		return nil, fmt.Errorf("failed to get disk space for %s: %w", normalizedPath, err)
	}

	if ds.cache != nil {
		cacheInfo := cache.NewDiskSpaceInfo(cacheKey, diskInfo.Total, diskInfo.Used, diskInfo.Free)
		ds.cache.SetDiskSpace(cacheKey, cacheInfo)
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

// GetAllDiskSpaces retrieves disk space grouped by unique drive/volume
func (ds *DiskService) GetAllDiskSpaces(ctx context.Context) (*DiskSummary, error) {
	ds.logger.Debug("Getting disk space for all configured paths")

	summary := &DiskSummary{
		Drives:        make(map[string]*DriveDiskInfo),
		Paths:         make(map[string]*DiskInfo),
		WorstHealth:   DiskHealthGood,
		WarningPaths:  []string{},
		CriticalPaths: []string{},
		LastUpdated:   time.Now(),
	}

	driveGroups, driveOrder, err := ds.groupConfiguredPathsByDrive()
	if err != nil {
		return nil, err
	}

	for _, driveID := range driveOrder {
		pathsOnDrive := driveGroups[driveID]
		queryPath := freeSpaceQueryPath(pathsOnDrive[0])

		diskInfo, err := ds.fetchDiskSpace(ctx, driveID, queryPath)
		if err != nil {
			ds.logger.WithError(err).WithField("drive", driveID).Warn("Failed to get disk space for drive")
			continue
		}

		health := ds.getDiskHealthStatus(diskInfo)
		summary.Drives[driveID] = &DriveDiskInfo{
			DriveID: driveID,
			Free:    diskInfo.Free,
			Health:  health,
			Paths:   append([]string(nil), pathsOnDrive...),
		}

		summary.TotalFree += diskInfo.Free
		if diskInfo.Total > 0 {
			summary.TotalSpace += diskInfo.Total
			summary.TotalUsed += diskInfo.Used
		}

		for _, path := range pathsOnDrive {
			pathInfo := *diskInfo
			pathInfo.Path = path
			summary.Paths[path] = &pathInfo
		}

		if ds.isWorseHealth(health, summary.WorstHealth) {
			summary.WorstHealth = health
		}

		switch health {
		case DiskHealthWarning:
			summary.WarningPaths = append(summary.WarningPaths, driveID)
		case DiskHealthCritical, DiskHealthDanger:
			summary.CriticalPaths = append(summary.CriticalPaths, driveID)
		}
	}

	summary.DriveOrder = driveOrder

	ds.logger.WithFields(map[string]interface{}{
		"paths_checked":  len(summary.Paths),
		"unique_drives":  len(summary.Drives),
		"total_space":    qbittorrent.FormatBytes(summary.TotalSpace),
		"total_free":     qbittorrent.FormatBytes(summary.TotalFree),
		"worst_health":   summary.WorstHealth,
		"warning_paths":  len(summary.WarningPaths),
		"critical_paths": len(summary.CriticalPaths),
	}).Info("Disk space summary generated")

	return summary, nil
}

// CheckDiskHealth performs a health check on all configured drives
func (ds *DiskService) CheckDiskHealth(ctx context.Context) (map[string]DiskHealthStatus, error) {
	ds.logger.Debug("Performing disk health check")

	summary, err := ds.GetAllDiskSpaces(ctx)
	if err != nil {
		return nil, err
	}

	healthStatus := make(map[string]DiskHealthStatus, len(summary.Drives))
	for driveID, drive := range summary.Drives {
		healthStatus[driveID] = drive.Health
		if drive.Health != DiskHealthGood {
			ds.logger.WithFields(map[string]interface{}{
				"drive":      driveID,
				"health":     drive.Health,
				"free_space": qbittorrent.FormatBytes(drive.Free),
				"paths":      strings.Join(drive.Paths, ", "),
			}).Warn("Disk space health issue detected")
		}
	}

	ds.logger.WithField("health_status", healthStatus).Info("Disk health check completed")
	return healthStatus, nil
}

// FormatDiskInfo formats disk information into a human-readable string
func (ds *DiskService) FormatDiskInfo(diskInfo *DiskInfo) string {
	if diskInfo.Total == 0 {
		return fmt.Sprintf(
			"Path: %s\nAvailable: %s\nHealth: %s",
			diskInfo.Path,
			qbittorrent.FormatBytes(diskInfo.Free),
			ds.getDiskHealthStatus(diskInfo),
		)
	}

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

func (ds *DiskService) getDriveIdentifier(path string) (string, error) {
	return ds.getDriveIdentifierPlatform(path)
}

func (ds *DiskService) normalizePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	cleanPath := filepath.Clean(path)
	if isWindowsPath(cleanPath) {
		return cleanPath, nil
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to convert to absolute path: %w", err)
	}

	return absPath, nil
}

func (ds *DiskService) getAllConfiguredPaths() []string {
	paths := []string{}

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

func (ds *DiskService) groupConfiguredPathsByDrive() (map[string][]string, []string, error) {
	paths := ds.getAllConfiguredPaths()
	if len(paths) == 0 {
		return nil, nil, fmt.Errorf("no configured paths to check disk space")
	}

	driveGroups := make(map[string][]string)
	driveOrder := []string{}
	seenPaths := make(map[string]bool)

	for _, path := range paths {
		if seenPaths[path] {
			continue
		}
		seenPaths[path] = true

		driveID, err := ds.getDriveIdentifier(path)
		if err != nil {
			ds.logger.WithError(err).WithField("path", path).Warn("Failed to get drive identifier")
			continue
		}

		if _, exists := driveGroups[driveID]; !exists {
			driveOrder = append(driveOrder, driveID)
		}
		driveGroups[driveID] = append(driveGroups[driveID], path)
	}

	if len(driveGroups) == 0 {
		return nil, nil, fmt.Errorf("no valid drives found for configured paths")
	}

	return driveGroups, driveOrder, nil
}

func (ds *DiskService) fetchDiskSpace(ctx context.Context, cacheKey, queryPath string) (*DiskInfo, error) {
	if ds.cache != nil {
		if cachedDisk, found := ds.cache.GetDiskSpace(cacheKey); found {
			return ds.diskInfoFromCache(cacheKey, cachedDisk), nil
		}
	}

	var diskInfo *DiskInfo
	var err error

	if ds.qbClient != nil {
		diskInfo, err = ds.getDiskSpaceViaQBittorrent(ctx, queryPath)
		if err != nil && shouldFallbackToLocalDisk(err) {
			ds.logger.WithError(err).WithField("path", queryPath).Warn("qBittorrent disk API unavailable, falling back to local check")
			diskInfo, err = ds.getDiskSpacePlatform(queryPath)
		}
	} else {
		diskInfo, err = ds.getDiskSpacePlatform(queryPath)
	}

	if err != nil {
		return nil, err
	}

	diskInfo.Path = cacheKey
	if ds.cache != nil {
		cacheInfo := cache.NewDiskSpaceInfo(cacheKey, diskInfo.Total, diskInfo.Used, diskInfo.Free)
		ds.cache.SetDiskSpace(cacheKey, cacheInfo)
	}

	return diskInfo, nil
}

func (ds *DiskService) getDiskSpaceViaQBittorrent(ctx context.Context, queryPath string) (*DiskInfo, error) {
	free, err := ds.qbClient.GetFreeSpaceAtPath(ctx, queryPath)
	if err != nil {
		return nil, err
	}

	return &DiskInfo{
		Path:        queryPath,
		Free:        free,
		Available:   free,
		Total:       0,
		Used:        0,
		UsedPercent: 0,
		FreePercent: 0,
		LastChecked: time.Now(),
	}, nil
}

func shouldFallbackToLocalDisk(err error) bool {
	var apiErr *qbittorrent.APIError
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusNotFound {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	return strings.Contains(err.Error(), "request failed")
}

func (ds *DiskService) diskCacheKey(path string) string {
	if driveID, ok := driveIdentifierFromWindowsPath(path); ok {
		return "drive:" + driveID
	}
	return path
}

func (ds *DiskService) diskInfoFromCache(path string, cachedDisk *cache.DiskSpaceInfo) *DiskInfo {
	return &DiskInfo{
		Path:        path,
		Total:       cachedDisk.Total,
		Used:        cachedDisk.Used,
		Free:        cachedDisk.Free,
		Available:   cachedDisk.Free,
		UsedPercent: ds.calculatePercentage(cachedDisk.Used, cachedDisk.Total),
		FreePercent: ds.calculatePercentage(cachedDisk.Free, cachedDisk.Total),
		LastChecked: cachedDisk.UpdatedAt,
	}
}

func (ds *DiskService) calculatePercentage(part, total int64) float64 {
	if total == 0 {
		return 0.0
	}
	return (float64(part) / float64(total)) * 100.0
}

func (ds *DiskService) getDiskHealthStatus(diskInfo *DiskInfo) DiskHealthStatus {
	if diskInfo.Total > 0 {
		freePercent := diskInfo.FreePercent
		if freePercent < 5.0 {
			return DiskHealthDanger
		}
		if freePercent < 10.0 {
			return DiskHealthCritical
		}
		if freePercent < 20.0 {
			return DiskHealthWarning
		}
		return DiskHealthGood
	}

	thresholds := ds.config.QBittorrent.DiskSpaceThresholds
	free := diskInfo.Free
	if free < thresholds.DangerFreeBytes() {
		return DiskHealthDanger
	}
	if free < thresholds.CriticalFreeBytes() {
		return DiskHealthCritical
	}
	if free < thresholds.WarnFreeBytes() {
		return DiskHealthWarning
	}
	return DiskHealthGood
}

func (ds *DiskService) isWorseHealth(health1, health2 DiskHealthStatus) bool {
	healthOrder := map[DiskHealthStatus]int{
		DiskHealthGood:     0,
		DiskHealthWarning:  1,
		DiskHealthCritical: 2,
		DiskHealthDanger:   3,
	}
	return healthOrder[health1] > healthOrder[health2]
}

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
