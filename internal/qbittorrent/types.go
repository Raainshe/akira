package qbittorrent

import (
	"fmt"
	"time"
)

// TorrentState represents the state of a torrent
type TorrentState string

const (
	StateError              TorrentState = "error"              // Some error occurred, applies to paused torrents
	StateMissingFiles       TorrentState = "missingFiles"       // Torrent data files is missing
	StateUploading          TorrentState = "uploading"          // Torrent is being seeded and data is being transferred
	StatePausedUP           TorrentState = "pausedUP"           // Torrent is paused and has finished downloading
	StateQueuedUP           TorrentState = "queuedUP"           // Queuing is enabled and torrent is queued for upload
	StateStalledUP          TorrentState = "stalledUP"          // Torrent is being seeded, but no connection were made
	StateCheckingUP         TorrentState = "checkingUP"         // Torrent has finished downloading and is being checked
	StateForcedUP           TorrentState = "forcedUP"           // Torrent is forced to uploading and ignore queue limit
	StateAllocating         TorrentState = "allocating"         // Torrent is allocating disk space for download
	StateDownloading        TorrentState = "downloading"        // Torrent is being downloaded and data is being transferred
	StateMetaDL             TorrentState = "metaDL"             // Torrent has just started downloading and is fetching metadata
	StatePausedDL           TorrentState = "pausedDL"           // Torrent is paused and has NOT finished downloading
	StateQueuedDL           TorrentState = "queuedDL"           // Queuing is enabled and torrent is queued for download
	StateStalledDL          TorrentState = "stalledDL"          // Torrent is being downloaded, but no connection were made
	StateCheckingDL         TorrentState = "checkingDL"         // Same as checkingUP, but torrent has NOT finished downloading
	StateForcedDL           TorrentState = "forcedDL"           // Torrent is forced to downloading to ignore queue limit
	StateCheckingResumeData TorrentState = "checkingResumeData" // Checking resume data on qBt startup
	StateMoving             TorrentState = "moving"             // Torrent is moving to another location
	StateUnknown            TorrentState = "unknown"            // Unknown status
)

// Torrent represents a torrent in qBittorrent
type Torrent struct {
	AddedOn           int64        `json:"added_on"`           // Time (Unix Timestamp) when the torrent was added to the client
	AmountLeft        int64        `json:"amount_left"`        // Amount of data left to download (bytes)
	AutoTmm           bool         `json:"auto_tmm"`           // Whether this torrent is managed by Automatic Torrent Management
	Availability      float64      `json:"availability"`       // Percentage of file pieces currently available
	Category          string       `json:"category"`           // Category of the torrent
	Completed         int64        `json:"completed"`          // Amount of transfer data completed (bytes)
	CompletionOn      int64        `json:"completion_on"`      // Time (Unix Timestamp) when the torrent completed
	ContentPath       string       `json:"content_path"`       // Absolute path of torrent content (root path for multifile torrents, absolute file path for singlefile torrents)
	DlLimit           int64        `json:"dl_limit"`           // Torrent download speed limit (bytes/s). -1 if no limit is set
	Dlspeed           int64        `json:"dlspeed"`            // Torrent download speed (bytes/s)
	Downloaded        int64        `json:"downloaded"`         // Amount of data downloaded
	DownloadedSession int64        `json:"downloaded_session"` // Amount of data downloaded this session
	Eta               int64        `json:"eta"`                // Torrent ETA (seconds)
	FLPiecePrio       bool         `json:"f_l_piece_prio"`     // True if first last piece are prioritized
	ForceStart        bool         `json:"force_start"`        // True if force start is enabled for this torrent
	Hash              string       `json:"hash"`               // Torrent hash
	LastActivity      int64        `json:"last_activity"`      // Last time (Unix Timestamp) when a chunk was downloaded/uploaded
	MagnetURI         string       `json:"magnet_uri"`         // Magnet URI corresponding to this torrent
	MaxRatio          float64      `json:"max_ratio"`          // Maximum share ratio until torrent is stopped from seeding/uploading
	MaxSeedingTime    int64        `json:"max_seeding_time"`   // Maximum seeding time (seconds) until torrent is stopped from seeding
	Name              string       `json:"name"`               // Torrent name
	NumComplete       int          `json:"num_complete"`       // Number of seeds in the swarm
	NumIncomplete     int          `json:"num_incomplete"`     // Number of leechers in the swarm
	NumLeechs         int          `json:"num_leechs"`         // Number of leechers connected to
	NumSeeds          int          `json:"num_seeds"`          // Number of seeds connected to
	Priority          int          `json:"priority"`           // Torrent priority. Returns -1 if queuing is disabled or torrent is in seed mode
	Progress          float64      `json:"progress"`           // Torrent progress (percentage/100)
	Ratio             float64      `json:"ratio"`              // Torrent share ratio. Max ratio value: 9999.
	RatioLimit        float64      `json:"ratio_limit"`        // TODO (what is different from max_ratio?)
	SavePath          string       `json:"save_path"`          // Path where this torrent's data is stored
	SeedingTime       int64        `json:"seeding_time"`       // Torrent seeding time (seconds)
	SeedingTimeLimit  int64        `json:"seeding_time_limit"` // TODO (what is different from max_seeding_time?)
	SeenComplete      int64        `json:"seen_complete"`      // Time (Unix Timestamp) when this torrent was last seen complete
	SeqDl             bool         `json:"seq_dl"`             // True if sequential download is enabled
	Size              int64        `json:"size"`               // Total size (bytes) of files selected for download
	State             TorrentState `json:"state"`              // Torrent state. See table here below for the possible values
	SuperSeeding      bool         `json:"super_seeding"`      // True if super seeding is enabled
	Tags              string       `json:"tags"`               // Comma-concatenated tag list of the torrent
	TimeActive        int64        `json:"time_active"`        // Total active time (seconds)
	TotalSize         int64        `json:"total_size"`         // Total size (bytes) of all file in this torrent (including unselected ones)
	Tracker           string       `json:"tracker"`            // The first tracker with working status. Returns empty string if no tracker is working.
	TrackersCount     int          `json:"trackers_count"`     // Number of trackers for this torrent
	UpLimit           int64        `json:"up_limit"`           // Torrent upload speed limit (bytes/s). -1 if no limit is set
	Uploaded          int64        `json:"uploaded"`           // Amount of data uploaded
	UploadedSession   int64        `json:"uploaded_session"`   // Amount of data uploaded this session
	Upspeed           int64        `json:"upspeed"`            // Torrent upload speed (bytes/s)
}

// TorrentProperties represents detailed properties of a torrent
type TorrentProperties struct {
	AdditionDate           int64   `json:"addition_date"`            // Time (Unix Timestamp) when the torrent was added to the client
	Comment                string  `json:"comment"`                  // Torrent comment
	CompletionDate         int64   `json:"completion_date"`          // Time (Unix Timestamp) when the torrent completed
	CreatedBy              string  `json:"created_by"`               // Torrent creator
	CreationDate           int64   `json:"creation_date"`            // Time (Unix Timestamp) when the torrent was created
	DlLimit                int64   `json:"dl_limit"`                 // Torrent download speed limit (bytes/s). -1 if no limit is set
	DlSpeed                int64   `json:"dl_speed"`                 // Torrent download speed (bytes/s)
	DlSpeedAvg             int64   `json:"dl_speed_avg"`             // Torrent average download speed (bytes/s)
	Eta                    int64   `json:"eta"`                      // Torrent ETA (seconds)
	LastSeen               int64   `json:"last_seen"`                // Last seen complete date (Unix Timestamp)
	NbConnections          int     `json:"nb_connections"`           // Number of peers connected to
	NbConnectionsLimit     int     `json:"nb_connections_limit"`     // Maximum number of peers allowed to connect to
	Peers                  int     `json:"peers"`                    // Number of peers in the swarm
	PeersTotal             int     `json:"peers_total"`              // Number of peers in the swarm
	PieceSize              int64   `json:"piece_size"`               // Torrent piece size (bytes)
	PiecesHave             int     `json:"pieces_have"`              // Number of pieces owned
	PiecesNum              int     `json:"pieces_num"`               // Number of pieces of the torrent
	Reannounce             int64   `json:"reannounce"`               // Time (seconds) until the next announce
	SavePath               string  `json:"save_path"`                // Torrent save path
	SeedingTime            int64   `json:"seeding_time"`             // Torrent seeding time (seconds)
	Seeds                  int     `json:"seeds"`                    // Number of seeds in the swarm
	SeedsTotal             int     `json:"seeds_total"`              // Number of seeds in the swarm
	ShareRatio             float64 `json:"share_ratio"`              // Torrent share ratio
	TimeElapsed            int64   `json:"time_elapsed"`             // Torrent elapsed time (seconds)
	TotalDownloaded        int64   `json:"total_downloaded"`         // Total data downloaded for torrent (bytes)
	TotalDownloadedSession int64   `json:"total_downloaded_session"` // Total data downloaded this session (bytes)
	TotalSize              int64   `json:"total_size"`               // Torrent total size (bytes)
	TotalUploaded          int64   `json:"total_uploaded"`           // Total data uploaded for torrent (bytes)
	TotalUploadedSession   int64   `json:"total_uploaded_session"`   // Total data uploaded this session (bytes)
	TotalWasted            int64   `json:"total_wasted"`             // Total data wasted for torrent (bytes)
	UpLimit                int64   `json:"up_limit"`                 // Torrent upload speed limit (bytes/s). -1 if no limit is set
	UpSpeed                int64   `json:"up_speed"`                 // Torrent upload speed (bytes/s)
	UpSpeedAvg             int64   `json:"up_speed_avg"`             // Torrent average upload speed (bytes/s)
}

// TorrentFile represents a file within a torrent
type TorrentFile struct {
	Index        int     `json:"index"`        // File index
	Name         string  `json:"name"`         // File name (including relative path)
	Size         int64   `json:"size"`         // File size (bytes)
	Progress     float64 `json:"progress"`     // File progress (percentage/100)
	Priority     int     `json:"priority"`     // File priority. See possible values here below
	IsSeed       bool    `json:"is_seed"`      // True if file is seeding/complete
	PieceRange   []int   `json:"piece_range"`  // The first number is the starting piece index and the second number is the ending piece index (inclusive)
	Availability float64 `json:"availability"` // Percentage of file pieces currently available (percentage/100)
}

// TorrentTracker represents a tracker for a torrent
type TorrentTracker struct {
	URL           string `json:"url"`            // Tracker url
	Status        int    `json:"status"`         // Tracker status. See the table below for possible values
	Tier          int    `json:"tier"`           // Tracker tier
	NumPeers      int    `json:"num_peers"`      // Number of peers for current torrent, as reported by the tracker
	NumSeeds      int    `json:"num_seeds"`      // Number of seeds for current torrent, as reported by the tracker
	NumLeeches    int    `json:"num_leeches"`    // Number of leeches for current torrent, as reported by the tracker
	NumDownloaded int    `json:"num_downloaded"` // Number of completed downloads for current torrent, as reported by the tracker
	Msg           string `json:"msg"`            // Tracker message (there is no way of knowing what this message is - it's up to tracker admins)
}

// AddTorrentRequest represents a request to add a torrent
type AddTorrentRequest struct {
	URLs                   []string `json:"urls,omitempty"`                   // URLs separated with newlines
	Torrents               [][]byte `json:"torrents,omitempty"`               // Raw data of torrent files
	SavePath               string   `json:"savepath,omitempty"`               // Download folder
	Cookie                 string   `json:"cookie,omitempty"`                 // Cookie sent to download the .torrent file
	Category               string   `json:"category,omitempty"`               // Category for the torrent
	Tags                   string   `json:"tags,omitempty"`                   // Tags for the torrent, split by ','
	SkipChecking           bool     `json:"skip_checking,omitempty"`          // Skip hash checking. Possible values are true, false (default)
	Paused                 bool     `json:"paused,omitempty"`                 // Add torrents in the paused state. Possible values are true, false (default)
	RootFolder             bool     `json:"root_folder,omitempty"`            // Create the root folder. Possible values are true, false, unset (default)
	Rename                 string   `json:"rename,omitempty"`                 // Rename torrent
	UpLimit                int64    `json:"upLimit,omitempty"`                // Set torrent upload speed limit. Unit in bytes/second
	DlLimit                int64    `json:"dlLimit,omitempty"`                // Set torrent download speed limit. Unit in bytes/second
	RatioLimit             float64  `json:"ratioLimit,omitempty"`             // Set torrent share ratio limit
	SeedingTimeLimit       int64    `json:"seedingTimeLimit,omitempty"`       // Set torrent seeding time limit. Unit in seconds
	AutoTMM                bool     `json:"autoTMM,omitempty"`                // Whether Automatic Torrent Management should be used
	SequentialDownload     bool     `json:"sequentialDownload,omitempty"`     // Enable sequential download. Possible values are true, false (default)
	FirstLastPiecePriority bool     `json:"firstLastPiecePriority,omitempty"` // Prioritize download first last piece. Possible values are true, false (default)
}

// DeleteTorrentRequest represents a request to delete torrents
type DeleteTorrentRequest struct {
	Hashes      []string `json:"hashes"`      // The hashes of the torrents you want to delete
	DeleteFiles bool     `json:"deleteFiles"` // If set to true, the downloaded data will also be deleted, otherwise has no effect
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ServerState represents the global server state
type ServerState struct {
	ConnectionStatus     string `json:"connection_status"`      // Server connection status
	DhtNodes             int64  `json:"dht_nodes"`              // DHT nodes connected to
	DlInfoData           int64  `json:"dl_info_data"`           // Data downloaded this session (bytes)
	DlInfoSpeed          int64  `json:"dl_info_speed"`          // Global download rate (bytes/s)
	DlRateLimit          int64  `json:"dl_rate_limit"`          // Download rate limit (bytes/s)
	UpInfoData           int64  `json:"up_info_data"`           // Data uploaded this session (bytes)
	UpInfoSpeed          int64  `json:"up_info_speed"`          // Global upload rate (bytes/s)
	UpRateLimit          int64  `json:"up_rate_limit"`          // Upload rate limit (bytes/s)
	QueuedIoJobs         int64  `json:"queued_io_jobs"`         // Queued I/O jobs
	ReadCacheHits        string `json:"read_cache_hits"`        // Read cache hits
	ReadCacheOverload    string `json:"read_cache_overload"`    // Read cache overload
	TotalBuffersSize     int64  `json:"total_buffers_size"`     // Total buffers size (bytes)
	TotalPeerConnections int64  `json:"total_peer_connections"` // Total peer connections
	WriteCacheOverload   string `json:"write_cache_overload"`   // Write cache overload
}

// DiskSpace represents disk space information
type DiskSpace struct {
	Total int64 `json:"total"` // Total space in bytes
	Used  int64 `json:"used"`  // Used space in bytes
	Free  int64 `json:"free"`  // Free space in bytes
}

// APIError represents an error from the qBittorrent API
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("qBittorrent API error %d: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("qBittorrent API error %d: %s", e.Code, e.Message)
}

// SeedingTrackingData represents data for tracking torrent seeding times
type SeedingTrackingData struct {
	Hash                 string        `json:"hash"`                   // Torrent hash
	Name                 string        `json:"name"`                   // Torrent name for display purposes
	DownloadStartTime    time.Time     `json:"download_start_time"`    // When download started
	DownloadCompleteTime time.Time     `json:"download_complete_time"` // When download completed
	DownloadDuration     time.Duration `json:"download_duration"`      // How long download took
	SeedingStopTime      time.Time     `json:"seeding_stop_time"`      // When seeding should stop
	AutoStopped          bool          `json:"auto_stopped"`           // Whether this torrent has been auto-stopped
	CreatedAt            time.Time     `json:"created_at"`             // When this tracking record was created
	UpdatedAt            time.Time     `json:"updated_at"`             // When this tracking record was last updated
}

// IsDownloading returns true if the torrent is currently downloading
func (t *Torrent) IsDownloading() bool {
	return t.State == StateDownloading || t.State == StateMetaDL ||
		t.State == StateStalledDL || t.State == StateCheckingDL ||
		t.State == StateForcedDL || t.State == StateQueuedDL ||
		t.State == StateAllocating
}

// IsSeeding returns true if the torrent is currently seeding
func (t *Torrent) IsSeeding() bool {
	return t.State == StateUploading || t.State == StateStalledUP ||
		t.State == StateCheckingUP || t.State == StateForcedUP ||
		t.State == StateQueuedUP
}

// IsCompleted returns true if the torrent has finished downloading
func (t *Torrent) IsCompleted() bool {
	return t.Progress >= 1.0
}

// IsPaused returns true if the torrent is paused
func (t *Torrent) IsPaused() bool {
	return t.State == StatePausedDL || t.State == StatePausedUP
}

// IsActive returns true if the torrent is actively transferring data
func (t *Torrent) IsActive() bool {
	return t.Dlspeed > 0 || t.Upspeed > 0
}

// GetProgressPercentage returns progress as a percentage (0-100)
func (t *Torrent) GetProgressPercentage() float64 {
	return t.Progress * 100
}

// GetFormattedETA returns a human-readable ETA string
func (t *Torrent) GetFormattedETA() string {
	if t.Eta <= 0 || t.Eta == 8640000 { // 8640000 is qBittorrent's "infinity" value
		return "∞"
	}

	duration := time.Duration(t.Eta) * time.Second
	if duration < time.Minute {
		return "< 1m"
	}

	hours := duration / time.Hour
	minutes := (duration % time.Hour) / time.Minute

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// GetStateDisplayName returns a human-readable state name
func (t *Torrent) GetStateDisplayName() string {
	switch t.State {
	case StateError:
		return "Error"
	case StateMissingFiles:
		return "Missing Files"
	case StateUploading:
		return "Seeding"
	case StatePausedUP:
		return "Paused (Complete)"
	case StateQueuedUP:
		return "Queued (Seeding)"
	case StateStalledUP:
		return "Stalled (Seeding)"
	case StateCheckingUP:
		return "Checking (Complete)"
	case StateForcedUP:
		return "Forced Seeding"
	case StateAllocating:
		return "Allocating"
	case StateDownloading:
		return "Downloading"
	case StateMetaDL:
		return "Fetching Metadata"
	case StatePausedDL:
		return "Paused"
	case StateQueuedDL:
		return "Queued"
	case StateStalledDL:
		return "Stalled"
	case StateCheckingDL:
		return "Checking"
	case StateForcedDL:
		return "Forced Download"
	case StateCheckingResumeData:
		return "Checking Resume Data"
	case StateMoving:
		return "Moving"
	default:
		return "Unknown"
	}
}

// FormatBytes formats bytes into human-readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatSpeed formats bytes per second into human-readable format
func FormatSpeed(bytesPerSecond int64) string {
	if bytesPerSecond == 0 {
		return "0 B/s"
	}
	return FormatBytes(bytesPerSecond) + "/s"
}
