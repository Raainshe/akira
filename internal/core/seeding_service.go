package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/raainshe/akira/internal/config"
	"github.com/raainshe/akira/internal/logging"
	"github.com/raainshe/akira/internal/qbittorrent"
)

// SeedingService manages automatic seeding time limits and tracking
type SeedingService struct {
	config         *config.Config
	torrentService *TorrentService
	client         *qbittorrent.Client
	logger         *logging.Logger

	// Tracking data
	trackingData map[string]*qbittorrent.SeedingTrackingData
	dataMutex    sync.RWMutex

	// Background processing
	stopChan     chan struct{}
	ticker       *time.Ticker
	isRunning    bool
	runningMutex sync.RWMutex
}

// SeedingStatus represents the current status of seeding management
type SeedingStatus struct {
	TrackedTorrents   int                              `json:"tracked_torrents"`
	ActiveSeeding     int                              `json:"active_seeding"`
	CompletedSeeding  int                              `json:"completed_seeding"`
	OverdueSeeding    int                              `json:"overdue_seeding"`
	TotalDownloadTime time.Duration                    `json:"total_download_time"`
	TotalSeedingTime  time.Duration                    `json:"total_seeding_time"`
	Details           map[string]*SeedingTorrentStatus `json:"details"`
	LastChecked       time.Time                        `json:"last_checked"`
}

// SeedingTorrentStatus represents the seeding status of an individual torrent
type SeedingTorrentStatus struct {
	Hash             string        `json:"hash"`
	Name             string        `json:"name"`
	DownloadDuration time.Duration `json:"download_duration"`
	SeedingDuration  time.Duration `json:"seeding_duration"`
	SeedingLimit     time.Duration `json:"seeding_limit"`
	TimeRemaining    time.Duration `json:"time_remaining"`
	IsOverdue        bool          `json:"is_overdue"`
	AutoStopped      bool          `json:"auto_stopped"`
	CurrentState     string        `json:"current_state"`
	SeedingStopTime  time.Time     `json:"seeding_stop_time"`
}

// NewSeedingService creates a new seeding service instance
func NewSeedingService(config *config.Config, torrentService *TorrentService, client *qbittorrent.Client) *SeedingService {
	return &SeedingService{
		config:         config,
		torrentService: torrentService,
		client:         client,
		logger:         logging.GetSeedingLogger(),
		trackingData:   make(map[string]*qbittorrent.SeedingTrackingData),
		stopChan:       make(chan struct{}),
	}
}

// Start begins the background seeding management service
func (ss *SeedingService) Start(ctx context.Context) error {
	ss.runningMutex.Lock()
	defer ss.runningMutex.Unlock()

	if ss.isRunning {
		return fmt.Errorf("seeding service is already running")
	}

	ss.logger.Info("Starting seeding management service")

	// Load existing tracking data
	if err := ss.LoadTrackingData(); err != nil {
		ss.logger.WithError(err).Warn("Failed to load tracking data, starting fresh")
	}

	// Set up periodic checking
	ss.ticker = time.NewTicker(ss.config.Seeding.CheckInterval)
	ss.isRunning = true

	// Start background goroutine
	go ss.backgroundProcessor(ctx)

	ss.logger.WithFields(map[string]interface{}{
		"check_interval":   ss.config.Seeding.CheckInterval,
		"time_multiplier":  ss.config.Seeding.TimeMultiplier,
		"tracking_file":    ss.config.Seeding.TrackingDataFile,
		"tracked_torrents": len(ss.trackingData),
	}).Info("Seeding management service started")

	return nil
}

// Stop gracefully stops the background seeding management service
func (ss *SeedingService) Stop() error {
	ss.runningMutex.Lock()
	defer ss.runningMutex.Unlock()

	if !ss.isRunning {
		return nil
	}

	ss.logger.Info("Stopping seeding management service")

	// Stop background processing
	close(ss.stopChan)
	if ss.ticker != nil {
		ss.ticker.Stop()
	}

	// Save current tracking data
	if err := ss.SaveTrackingData(); err != nil {
		ss.logger.WithError(err).Error("Failed to save tracking data during shutdown")
	}

	ss.isRunning = false
	ss.logger.Info("Seeding management service stopped")

	return nil
}

// StartTracking begins tracking a new torrent for seeding management
func (ss *SeedingService) StartTracking(ctx context.Context, hash, name string) error {
	ss.dataMutex.Lock()
	defer ss.dataMutex.Unlock()

	// Check if already tracking
	if _, exists := ss.trackingData[hash]; exists {
		ss.logger.WithField("hash", hash).Debug("Torrent already being tracked")
		return nil
	}

	now := time.Now()
	trackingData := &qbittorrent.SeedingTrackingData{
		Hash:              hash,
		Name:              name,
		DownloadStartTime: now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	ss.trackingData[hash] = trackingData

	ss.logger.WithFields(map[string]interface{}{
		"hash": hash,
		"name": name,
	}).Info("Started tracking torrent for seeding management")

	// Save tracking data (call without holding lock to avoid deadlock)
	go func() {
		if err := ss.SaveTrackingData(); err != nil {
			ss.logger.WithError(err).Error("Failed to save tracking data after starting tracking")
		}
	}()

	return nil
}

// StopTracking stops tracking a torrent (manual removal)
func (ss *SeedingService) StopTracking(hash string) error {
	ss.dataMutex.Lock()
	defer ss.dataMutex.Unlock()

	trackingData, exists := ss.trackingData[hash]
	if !exists {
		return fmt.Errorf("torrent %s is not being tracked", hash)
	}

	delete(ss.trackingData, hash)

	ss.logger.WithFields(map[string]interface{}{
		"hash": hash,
		"name": trackingData.Name,
	}).Info("Stopped tracking torrent")

	// Save tracking data (call without holding lock to avoid deadlock)
	go func() {
		if err := ss.SaveTrackingData(); err != nil {
			ss.logger.WithError(err).Error("Failed to save tracking data after stopping tracking")
		}
	}()

	return nil
}

// CheckSeedingLimits checks all tracked torrents and stops seeding for those that have exceeded limits
func (ss *SeedingService) CheckSeedingLimits(ctx context.Context) error {
	ss.logger.Debug("Checking seeding limits for all tracked torrents")

	// Get current torrents from qBittorrent
	torrents, err := ss.torrentService.GetTorrents(ctx, nil)
	if err != nil {
		ss.logger.WithError(err).Error("Failed to get torrents for seeding limit check")
		return fmt.Errorf("failed to get torrents: %w", err)
	}

	// Create hash map for quick lookup
	torrentMap := make(map[string]qbittorrent.Torrent)
	for _, torrent := range torrents {
		torrentMap[torrent.Hash] = torrent
	}

	ss.dataMutex.Lock()
	defer ss.dataMutex.Unlock()

	now := time.Now()
	stoppedCount := 0
	checkedCount := 0

	for hash, trackingData := range ss.trackingData {
		checkedCount++

		// Skip if already auto-stopped
		if trackingData.AutoStopped {
			continue
		}

		// Get current torrent info
		torrent, exists := torrentMap[hash]
		if !exists {
			ss.logger.WithField("hash", hash).Debug("Tracked torrent not found in current torrent list")
			continue
		}

		// Check if download is complete and update tracking data
		if trackingData.DownloadCompleteTime.IsZero() && torrent.IsCompleted() {
			// Download just completed
			trackingData.DownloadCompleteTime = now
			trackingData.DownloadDuration = trackingData.DownloadCompleteTime.Sub(trackingData.DownloadStartTime)

			// Calculate seeding stop time
			seedingDuration := time.Duration(float64(trackingData.DownloadDuration) * ss.config.Seeding.TimeMultiplier)
			trackingData.SeedingStopTime = trackingData.DownloadCompleteTime.Add(seedingDuration)
			trackingData.UpdatedAt = now

			ss.logger.WithFields(map[string]interface{}{
				"hash":              hash,
				"name":              trackingData.Name,
				"download_duration": trackingData.DownloadDuration,
				"seeding_duration":  seedingDuration,
				"seeding_stop_time": trackingData.SeedingStopTime,
			}).Info("Torrent download completed, seeding time limit calculated")

			// Log the completion
			logging.LogTorrentCompleted(trackingData.Name, hash, trackingData.DownloadDuration.String())
		}

		// Check if seeding should be stopped
		if !trackingData.DownloadCompleteTime.IsZero() && now.After(trackingData.SeedingStopTime) {
			// Time to stop seeding
			if torrent.IsSeeding() {
				err := ss.client.PauseTorrents(ctx, []string{hash})
				if err != nil {
					ss.logger.WithError(err).WithField("hash", hash).Error("Failed to pause torrent for seeding limit")
					continue
				}

				trackingData.AutoStopped = true
				trackingData.UpdatedAt = now
				stoppedCount++

				seedingDuration := now.Sub(trackingData.DownloadCompleteTime)
				ss.logger.WithFields(map[string]interface{}{
					"hash":             hash,
					"name":             trackingData.Name,
					"seeding_duration": seedingDuration,
				}).Info("Automatically stopped seeding due to time limit")

				// Log the seeding stop
				logging.LogSeedingStopped(trackingData.Name, hash, seedingDuration.String())
			}
		}
	}

	ss.logger.WithFields(map[string]interface{}{
		"checked_count": checkedCount,
		"stopped_count": stoppedCount,
	}).Debug("Seeding limit check completed")

	// Save tracking data if any changes were made
	if stoppedCount > 0 {
		if err := ss.SaveTrackingData(); err != nil {
			ss.logger.WithError(err).Error("Failed to save tracking data after seeding limit check")
		}
	}

	return nil
}

// GetSeedingStatus returns the current status of all tracked torrents
func (ss *SeedingService) GetSeedingStatus(ctx context.Context) (*SeedingStatus, error) {
	ss.logger.Debug("Generating seeding status report")

	// Get current torrents
	torrents, err := ss.torrentService.GetTorrents(ctx, nil)
	if err != nil {
		ss.logger.WithError(err).Error("Failed to get torrents for seeding status")
		return nil, fmt.Errorf("failed to get torrents: %w", err)
	}

	torrentMap := make(map[string]qbittorrent.Torrent)
	for _, torrent := range torrents {
		torrentMap[torrent.Hash] = torrent
	}

	ss.dataMutex.RLock()
	defer ss.dataMutex.RUnlock()

	status := &SeedingStatus{
		Details:     make(map[string]*SeedingTorrentStatus),
		LastChecked: time.Now(),
	}

	now := time.Now()

	for hash, trackingData := range ss.trackingData {
		torrent, exists := torrentMap[hash]
		if !exists {
			continue // Skip torrents that no longer exist
		}

		torrentStatus := &SeedingTorrentStatus{
			Hash:             hash,
			Name:             trackingData.Name,
			DownloadDuration: trackingData.DownloadDuration,
			AutoStopped:      trackingData.AutoStopped,
			CurrentState:     torrent.GetStateDisplayName(),
			SeedingStopTime:  trackingData.SeedingStopTime,
		}

		// Calculate seeding duration
		if !trackingData.DownloadCompleteTime.IsZero() {
			if trackingData.AutoStopped {
				torrentStatus.SeedingDuration = trackingData.SeedingStopTime.Sub(trackingData.DownloadCompleteTime)
			} else {
				torrentStatus.SeedingDuration = now.Sub(trackingData.DownloadCompleteTime)
			}

			// Calculate seeding limit and time remaining
			seedingLimit := time.Duration(float64(trackingData.DownloadDuration) * ss.config.Seeding.TimeMultiplier)
			torrentStatus.SeedingLimit = seedingLimit

			timeRemaining := trackingData.SeedingStopTime.Sub(now)
			if timeRemaining < 0 {
				timeRemaining = 0
				torrentStatus.IsOverdue = true
			}
			torrentStatus.TimeRemaining = timeRemaining
		}

		status.Details[hash] = torrentStatus
		status.TrackedTorrents++
		status.TotalDownloadTime += torrentStatus.DownloadDuration
		status.TotalSeedingTime += torrentStatus.SeedingDuration

		// Count by status
		if torrent.IsSeeding() && !trackingData.AutoStopped {
			status.ActiveSeeding++
		} else if trackingData.AutoStopped {
			status.CompletedSeeding++
		}

		if torrentStatus.IsOverdue && !trackingData.AutoStopped {
			status.OverdueSeeding++
		}
	}

	ss.logger.WithFields(map[string]interface{}{
		"tracked_torrents":  status.TrackedTorrents,
		"active_seeding":    status.ActiveSeeding,
		"completed_seeding": status.CompletedSeeding,
		"overdue_seeding":   status.OverdueSeeding,
	}).Info("Seeding status report generated")

	return status, nil
}

// SaveTrackingData saves the current tracking data to disk
func (ss *SeedingService) SaveTrackingData() error {
	ss.dataMutex.RLock()
	defer ss.dataMutex.RUnlock()

	data, err := json.MarshalIndent(ss.trackingData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tracking data: %w", err)
	}

	err = os.WriteFile(ss.config.Seeding.TrackingDataFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write tracking data file: %w", err)
	}

	ss.logger.WithFields(map[string]interface{}{
		"file":             ss.config.Seeding.TrackingDataFile,
		"tracked_torrents": len(ss.trackingData),
	}).Debug("Tracking data saved to disk")

	return nil
}

// LoadTrackingData loads tracking data from disk
func (ss *SeedingService) LoadTrackingData() error {
	ss.dataMutex.Lock()
	defer ss.dataMutex.Unlock()

	data, err := os.ReadFile(ss.config.Seeding.TrackingDataFile)
	if err != nil {
		if os.IsNotExist(err) {
			ss.logger.Debug("Tracking data file does not exist, starting with empty data")
			return nil
		}
		return fmt.Errorf("failed to read tracking data file: %w", err)
	}

	err = json.Unmarshal(data, &ss.trackingData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal tracking data: %w", err)
	}

	ss.logger.WithFields(map[string]interface{}{
		"file":             ss.config.Seeding.TrackingDataFile,
		"tracked_torrents": len(ss.trackingData),
	}).Info("Tracking data loaded from disk")

	return nil
}

// IsRunning returns whether the seeding service is currently running
func (ss *SeedingService) IsRunning() bool {
	ss.runningMutex.RLock()
	defer ss.runningMutex.RUnlock()
	return ss.isRunning
}

// backgroundProcessor runs the periodic seeding limit checks
func (ss *SeedingService) backgroundProcessor(ctx context.Context) {
	ss.logger.Info("Background seeding processor started")

	defer func() {
		ss.logger.Info("Background seeding processor stopped")
	}()

	for {
		select {
		case <-ss.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ss.ticker.C:
			if err := ss.CheckSeedingLimits(ctx); err != nil {
				ss.logger.WithError(err).Error("Failed to check seeding limits")
			}
		}
	}
}

// Helper methods

// GetTrackedTorrentCount returns the number of currently tracked torrents
func (ss *SeedingService) GetTrackedTorrentCount() int {
	ss.dataMutex.RLock()
	defer ss.dataMutex.RUnlock()
	return len(ss.trackingData)
}

// GetTrackedTorrents returns a copy of all tracked torrent data
func (ss *SeedingService) GetTrackedTorrents() map[string]*qbittorrent.SeedingTrackingData {
	ss.dataMutex.RLock()
	defer ss.dataMutex.RUnlock()

	result := make(map[string]*qbittorrent.SeedingTrackingData)
	for hash, data := range ss.trackingData {
		// Create a copy to avoid race conditions
		dataCopy := *data
		result[hash] = &dataCopy
	}
	return result
}

// ForceStopSeeding manually stops seeding for specific torrents (emergency override)
func (ss *SeedingService) ForceStopSeeding(ctx context.Context, hashes []string) error {
	if len(hashes) == 0 {
		return fmt.Errorf("no torrent hashes provided")
	}

	ss.logger.WithField("hashes", hashes).Info("Force stopping seeding for torrents")

	// Pause the torrents
	err := ss.client.PauseTorrents(ctx, hashes)
	if err != nil {
		ss.logger.WithError(err).Error("Failed to force stop seeding")
		return fmt.Errorf("failed to pause torrents: %w", err)
	}

	// Update tracking data
	ss.dataMutex.Lock()
	defer ss.dataMutex.Unlock()

	now := time.Now()
	for _, hash := range hashes {
		if trackingData, exists := ss.trackingData[hash]; exists {
			trackingData.AutoStopped = true
			trackingData.UpdatedAt = now
		}
	}

	// Save tracking data
	if err := ss.SaveTrackingData(); err != nil {
		ss.logger.WithError(err).Error("Failed to save tracking data after force stop")
	}

	ss.logger.WithField("count", len(hashes)).Info("Force stopped seeding for torrents")
	return nil
}
