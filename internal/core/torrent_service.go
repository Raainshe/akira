package core

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/raainshe/akira/internal/cache"
	"github.com/raainshe/akira/internal/config"
	"github.com/raainshe/akira/internal/logging"
	"github.com/raainshe/akira/internal/qbittorrent"
)

// TorrentService provides high-level business logic for torrent operations
type TorrentService struct {
	client *qbittorrent.Client
	config *config.Config
	cache  *cache.CacheManager
	logger *logging.Logger
}

// TorrentFilter represents filtering options for torrent queries
type TorrentFilter struct {
	Category    string                     // Filter by category (series, movies, anime, etc.)
	State       qbittorrent.TorrentState   // Filter by torrent state
	States      []qbittorrent.TorrentState // Filter by multiple states
	NamePattern string                     // Filter by name pattern (regex)
	OnlyActive  bool                       // Only show active torrents (downloading/uploading)
	OnlySeeding bool                       // Only show seeding torrents
	SortBy      TorrentSortField           // Sort field
	SortDesc    bool                       // Sort in descending order
	Limit       int                        // Limit number of results (0 = no limit)
}

// TorrentSortField represents fields that can be used for sorting
type TorrentSortField string

const (
	SortByName          TorrentSortField = "name"
	SortBySize          TorrentSortField = "size"
	SortByProgress      TorrentSortField = "progress"
	SortByDownloadSpeed TorrentSortField = "download_speed"
	SortByUploadSpeed   TorrentSortField = "upload_speed"
	SortByAddedDate     TorrentSortField = "added_date"
	SortByCompletedDate TorrentSortField = "completed_date"
	SortByRatio         TorrentSortField = "ratio"
	SortBySeedingTime   TorrentSortField = "seeding_time"
)

// AddTorrentOptions represents options for adding torrents with business logic
type AddTorrentOptions struct {
	Category           string        // Category (will be validated and mapped to save path)
	SavePath           string        // Override save path (optional)
	Paused             bool          // Start paused
	SkipChecking       bool          // Skip hash checking
	Tags               []string      // Tags to apply
	UploadLimit        int64         // Upload speed limit (bytes/s)
	DownloadLimit      int64         // Download speed limit (bytes/s)
	RatioLimit         float64       // Share ratio limit
	SeedingTimeLimit   time.Duration // Seeding time limit
	SequentialDownload bool          // Enable sequential download
	FirstLastPriority  bool          // Prioritize first/last pieces
}

// TorrentStats represents statistics about torrents
type TorrentStats struct {
	Total         int   `json:"total"`
	Downloading   int   `json:"downloading"`
	Seeding       int   `json:"seeding"`
	Completed     int   `json:"completed"`
	Paused        int   `json:"paused"`
	Error         int   `json:"error"`
	TotalSize     int64 `json:"total_size"`
	Downloaded    int64 `json:"downloaded"`
	Uploaded      int64 `json:"uploaded"`
	DownloadSpeed int64 `json:"download_speed"`
	UploadSpeed   int64 `json:"upload_speed"`
}

// NewTorrentService creates a new torrent service instance
func NewTorrentService(client *qbittorrent.Client, config *config.Config, cache *cache.CacheManager) *TorrentService {
	return &TorrentService{
		client: client,
		config: config,
		cache:  cache,
		logger: logging.GetCoreLogger(),
	}
}

// GetTorrents retrieves torrents with optional filtering
func (ts *TorrentService) GetTorrents(ctx context.Context, filter *TorrentFilter) ([]qbittorrent.Torrent, error) {
	ts.logger.Debug("Fetching torrents with filtering")

	// Get all torrents from qBittorrent
	torrents, err := ts.client.GetTorrents(ctx)
	if err != nil {
		ts.logger.WithError(err).Error("Failed to fetch torrents from client")
		return nil, fmt.Errorf("failed to fetch torrents: %w", err)
	}

	// Apply filtering if provided
	if filter != nil {
		torrents = ts.applyFilter(torrents, filter)
	}

	ts.logger.WithFields(map[string]interface{}{
		"total_count":    len(torrents),
		"filter_applied": filter != nil,
	}).Info("Torrents retrieved and filtered")

	return torrents, nil
}

// GetTorrentsByCategory retrieves torrents filtered by category
func (ts *TorrentService) GetTorrentsByCategory(ctx context.Context, category string) ([]qbittorrent.Torrent, error) {
	// Validate category
	if !ts.isValidCategory(category) {
		return nil, fmt.Errorf("invalid category: %s", category)
	}

	filter := &TorrentFilter{
		Category: category,
		SortBy:   SortByAddedDate,
		SortDesc: true,
	}

	return ts.GetTorrents(ctx, filter)
}

// GetTorrentsByState retrieves torrents filtered by state
func (ts *TorrentService) GetTorrentsByState(ctx context.Context, state qbittorrent.TorrentState) ([]qbittorrent.Torrent, error) {
	filter := &TorrentFilter{
		State:    state,
		SortBy:   SortByAddedDate,
		SortDesc: true,
	}

	return ts.GetTorrents(ctx, filter)
}

// GetSeedingTorrents retrieves only torrents that are currently seeding
func (ts *TorrentService) GetSeedingTorrents(ctx context.Context) ([]qbittorrent.Torrent, error) {
	filter := &TorrentFilter{
		OnlySeeding: true,
		SortBy:      SortBySeedingTime,
		SortDesc:    true,
	}

	return ts.GetTorrents(ctx, filter)
}

// GetActiveTorrents retrieves only torrents that are actively transferring data
func (ts *TorrentService) GetActiveTorrents(ctx context.Context) ([]qbittorrent.Torrent, error) {
	filter := &TorrentFilter{
		OnlyActive: true,
		SortBy:     SortByDownloadSpeed,
		SortDesc:   true,
	}

	return ts.GetTorrents(ctx, filter)
}

// SearchTorrents searches torrents by name pattern
func (ts *TorrentService) SearchTorrents(ctx context.Context, pattern string) ([]qbittorrent.Torrent, error) {
	if pattern == "" {
		return nil, fmt.Errorf("search pattern cannot be empty")
	}

	filter := &TorrentFilter{
		NamePattern: pattern,
		SortBy:      SortByName,
		SortDesc:    false,
	}

	return ts.GetTorrents(ctx, filter)
}

// AddMagnet adds a magnet link with business logic and validation
func (ts *TorrentService) AddMagnet(ctx context.Context, magnetURI string, options AddTorrentOptions) error {
	ts.logger.WithFields(map[string]interface{}{
		"category":  options.Category,
		"save_path": options.SavePath,
		"tags":      options.Tags,
	}).Info("Adding magnet link with business logic")

	// Validate magnet URI
	if err := ts.validateMagnetURI(magnetURI); err != nil {
		ts.logger.WithError(err).Error("Invalid magnet URI")
		return fmt.Errorf("invalid magnet URI: %w", err)
	}

	// Validate and normalize category
	if options.Category != "" {
		if !ts.isValidCategory(options.Category) {
			return fmt.Errorf("invalid category: %s (valid: %v)", options.Category, ts.config.GetValidCategories())
		}
	} else {
		options.Category = "default"
	}

	// Determine save path
	savePath := options.SavePath
	if savePath == "" {
		savePath = ts.config.GetSavePathForCategory(options.Category)
	}

	// Convert to qBittorrent request format
	qbitOptions := qbittorrent.AddTorrentRequest{
		Category:               options.Category,
		SavePath:               savePath,
		Paused:                 options.Paused,
		SkipChecking:           options.SkipChecking,
		Tags:                   strings.Join(options.Tags, ","),
		UpLimit:                options.UploadLimit,
		DlLimit:                options.DownloadLimit,
		RatioLimit:             options.RatioLimit,
		SeedingTimeLimit:       int64(options.SeedingTimeLimit.Seconds()),
		SequentialDownload:     options.SequentialDownload,
		FirstLastPiecePriority: options.FirstLastPriority,
	}

	// Add the magnet link
	err := ts.client.AddMagnet(ctx, magnetURI, qbitOptions)
	if err != nil {
		ts.logger.WithError(err).Error("Failed to add magnet link")
		return fmt.Errorf("failed to add magnet link: %w", err)
	}

	// Log the successful addition
	logging.LogTorrentAdded(magnetURI, options.Category, savePath)

	ts.logger.WithFields(map[string]interface{}{
		"category":  options.Category,
		"save_path": savePath,
		"tags":      options.Tags,
	}).Info("Magnet link added successfully")

	return nil
}

// DeleteTorrents deletes torrents with category-based filtering
func (ts *TorrentService) DeleteTorrents(ctx context.Context, hashes []string, deleteFiles bool) error {
	if len(hashes) == 0 {
		return fmt.Errorf("no torrent hashes provided")
	}

	ts.logger.WithFields(map[string]interface{}{
		"hashes":       hashes,
		"delete_files": deleteFiles,
		"count":        len(hashes),
	}).Info("Deleting torrents")

	// Get torrent details before deletion for logging
	var torrentNames []string
	for _, hash := range hashes {
		// Try to get torrent info for logging (best effort)
		torrents, err := ts.client.GetTorrents(ctx)
		if err == nil {
			for _, torrent := range torrents {
				if torrent.Hash == hash {
					torrentNames = append(torrentNames, torrent.Name)
					break
				}
			}
		}
	}

	// Delete torrents
	err := ts.client.DeleteTorrents(ctx, hashes, deleteFiles)
	if err != nil {
		ts.logger.WithError(err).Error("Failed to delete torrents")
		return fmt.Errorf("failed to delete torrents: %w", err)
	}

	// Log deletions
	for i, hash := range hashes {
		name := "unknown"
		if i < len(torrentNames) {
			name = torrentNames[i]
		}
		logging.LogTorrentDeleted(name, hash, deleteFiles)
	}

	ts.logger.WithField("count", len(hashes)).Info("Torrents deleted successfully")
	return nil
}

// PauseTorrents pauses the specified torrents
func (ts *TorrentService) PauseTorrents(ctx context.Context, hashes []string) error {
	if len(hashes) == 0 {
		return fmt.Errorf("no torrent hashes provided")
	}

	ts.logger.WithField("count", len(hashes)).Info("Pausing torrents")

	err := ts.client.PauseTorrents(ctx, hashes)
	if err != nil {
		ts.logger.WithError(err).Error("Failed to pause torrents")
		return fmt.Errorf("failed to pause torrents: %w", err)
	}

	ts.logger.WithField("count", len(hashes)).Info("Torrents paused successfully")
	return nil
}

// ResumeTorrents resumes the specified torrents
func (ts *TorrentService) ResumeTorrents(ctx context.Context, hashes []string) error {
	if len(hashes) == 0 {
		return fmt.Errorf("no torrent hashes provided")
	}

	ts.logger.WithField("count", len(hashes)).Info("Resuming torrents")

	err := ts.client.ResumeTorrents(ctx, hashes)
	if err != nil {
		ts.logger.WithError(err).Error("Failed to resume torrents")
		return fmt.Errorf("failed to resume torrents: %w", err)
	}

	ts.logger.WithField("count", len(hashes)).Info("Torrents resumed successfully")
	return nil
}

// GetTorrentStats calculates statistics for all torrents
func (ts *TorrentService) GetTorrentStats(ctx context.Context) (*TorrentStats, error) {
	ts.logger.Debug("Calculating torrent statistics")

	torrents, err := ts.client.GetTorrents(ctx)
	if err != nil {
		ts.logger.WithError(err).Error("Failed to fetch torrents for stats")
		return nil, fmt.Errorf("failed to fetch torrents: %w", err)
	}

	stats := &TorrentStats{}

	for _, torrent := range torrents {
		stats.Total++
		stats.TotalSize += torrent.Size
		stats.Downloaded += torrent.Downloaded
		stats.Uploaded += torrent.Uploaded
		stats.DownloadSpeed += torrent.Dlspeed
		stats.UploadSpeed += torrent.Upspeed

		// Count by state
		if torrent.IsDownloading() {
			stats.Downloading++
		} else if torrent.IsSeeding() {
			stats.Seeding++
		} else if torrent.IsCompleted() {
			stats.Completed++
		} else if torrent.IsPaused() {
			stats.Paused++
		} else if torrent.State == qbittorrent.StateError {
			stats.Error++
		}
	}

	ts.logger.WithFields(map[string]interface{}{
		"total":       stats.Total,
		"downloading": stats.Downloading,
		"seeding":     stats.Seeding,
		"completed":   stats.Completed,
	}).Info("Torrent statistics calculated")

	return stats, nil
}

// Helper methods

// applyFilter applies filtering logic to torrents
func (ts *TorrentService) applyFilter(torrents []qbittorrent.Torrent, filter *TorrentFilter) []qbittorrent.Torrent {
	var filtered []qbittorrent.Torrent

	// Compile regex pattern if provided
	var nameRegex *regexp.Regexp
	if filter.NamePattern != "" {
		var err error
		nameRegex, err = regexp.Compile("(?i)" + filter.NamePattern) // Case insensitive
		if err != nil {
			ts.logger.WithError(err).Warn("Invalid regex pattern, skipping name filter")
			nameRegex = nil
		}
	}

	for _, torrent := range torrents {
		// Filter by category
		if filter.Category != "" {
			torrentCategory := ts.getTorrentCategory(torrent)
			if torrentCategory != filter.Category {
				continue
			}
		}

		// Filter by state
		if filter.State != "" && torrent.State != filter.State {
			continue
		}

		// Filter by multiple states
		if len(filter.States) > 0 {
			found := false
			for _, state := range filter.States {
				if torrent.State == state {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by active status
		if filter.OnlyActive && !torrent.IsActive() {
			continue
		}

		// Filter by seeding status
		if filter.OnlySeeding && !torrent.IsSeeding() {
			continue
		}

		// Filter by name pattern
		if nameRegex != nil && !nameRegex.MatchString(torrent.Name) {
			continue
		}

		filtered = append(filtered, torrent)
	}

	// Apply sorting
	ts.sortTorrents(filtered, filter.SortBy, filter.SortDesc)

	// Apply limit
	if filter.Limit > 0 && len(filtered) > filter.Limit {
		filtered = filtered[:filter.Limit]
	}

	return filtered
}

// sortTorrents sorts torrents by the specified field
func (ts *TorrentService) sortTorrents(torrents []qbittorrent.Torrent, sortBy TorrentSortField, desc bool) {
	sort.Slice(torrents, func(i, j int) bool {
		var less bool

		switch sortBy {
		case SortByName:
			less = strings.ToLower(torrents[i].Name) < strings.ToLower(torrents[j].Name)
		case SortBySize:
			less = torrents[i].Size < torrents[j].Size
		case SortByProgress:
			less = torrents[i].Progress < torrents[j].Progress
		case SortByDownloadSpeed:
			less = torrents[i].Dlspeed < torrents[j].Dlspeed
		case SortByUploadSpeed:
			less = torrents[i].Upspeed < torrents[j].Upspeed
		case SortByAddedDate:
			less = torrents[i].AddedOn < torrents[j].AddedOn
		case SortByCompletedDate:
			less = torrents[i].CompletionOn < torrents[j].CompletionOn
		case SortByRatio:
			less = torrents[i].Ratio < torrents[j].Ratio
		case SortBySeedingTime:
			less = torrents[i].SeedingTime < torrents[j].SeedingTime
		default:
			less = strings.ToLower(torrents[i].Name) < strings.ToLower(torrents[j].Name)
		}

		if desc {
			return !less
		}
		return less
	})
}

// getTorrentCategory determines the category of a torrent based on its save path
func (ts *TorrentService) getTorrentCategory(torrent qbittorrent.Torrent) string {
	// First check the category field
	if torrent.Category != "" {
		return torrent.Category
	}

	// Fall back to path-based detection
	savePath := strings.ToLower(torrent.SavePath)

	seriesPath := strings.ToLower(ts.config.QBittorrent.SavePaths.Series)
	moviesPath := strings.ToLower(ts.config.QBittorrent.SavePaths.Movies)
	animePath := strings.ToLower(ts.config.QBittorrent.SavePaths.Anime)

	if strings.Contains(savePath, seriesPath) {
		return "series"
	} else if strings.Contains(savePath, moviesPath) {
		return "movies"
	} else if strings.Contains(savePath, animePath) {
		return "anime"
	}

	return "default"
}

// validateMagnetURI validates that a string is a valid magnet URI
func (ts *TorrentService) validateMagnetURI(magnetURI string) error {
	if magnetURI == "" {
		return fmt.Errorf("magnet URI cannot be empty")
	}

	if !strings.HasPrefix(strings.ToLower(magnetURI), "magnet:?") {
		return fmt.Errorf("invalid magnet URI format")
	}

	// Check for required xt parameter (exact topic)
	if !strings.Contains(magnetURI, "xt=urn:btih:") {
		return fmt.Errorf("magnet URI missing required xt parameter")
	}

	return nil
}

// isValidCategory checks if a category is valid according to configuration
func (ts *TorrentService) isValidCategory(category string) bool {
	validCategories := ts.config.GetValidCategories()
	for _, valid := range validCategories {
		if category == valid {
			return true
		}
	}
	return false
}
