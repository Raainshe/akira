package shared

import (
	"time"

	"github.com/raainshe/akira/internal/core"
	"github.com/raainshe/akira/internal/qbittorrent"
)

// CachedData represents the application cache data
type CachedData struct {
	Torrents    []qbittorrent.Torrent
	Stats       *AppStats
	DiskInfo    map[string]*core.DiskInfo
	SeedingInfo *core.SeedingStatus
	LastFetch   map[string]time.Time
}

// AppStats holds overall application statistics
type AppStats struct {
	TotalTorrents   int
	ActiveDownloads int
	ActiveSeeds     int
	PausedTorrents  int
	ErroredTorrents int
	TotalDownSpeed  int64
	TotalUpSpeed    int64
	LastUpdate      time.Time
}
