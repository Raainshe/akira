package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/raainshe/akira/internal/config"
	"github.com/raainshe/akira/internal/logging"
	"github.com/raainshe/akira/internal/qbittorrent"
)

// Cache keys for different types of cached data
const (
	KeyAuthSession     = "auth:session"
	KeyDiskSpacePrefix = "disk:space:" // followed by path
	KeyServerState     = "server:state"
	KeyPreferences     = "server:preferences"
)

// CacheManager wraps go-cache with typed methods and statistics
type CacheManager struct {
	cache  *cache.Cache
	config *config.CacheConfig
	logger *logging.Logger
	stats  *CacheStats
	mutex  sync.RWMutex
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	Hits      int64     `json:"hits"`
	Misses    int64     `json:"misses"`
	Sets      int64     `json:"sets"`
	Deletes   int64     `json:"deletes"`
	Evictions int64     `json:"evictions"`
	ItemCount int       `json:"item_count"`
	LastReset time.Time `json:"last_reset"`
}

// AuthSession represents cached authentication data
type AuthSession struct {
	Cookie    string    `json:"cookie"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// DiskSpaceInfo represents cached disk space information
type DiskSpaceInfo struct {
	Path      string    `json:"path"`
	Total     int64     `json:"total"`
	Used      int64     `json:"used"`
	Free      int64     `json:"free"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ServerStateInfo represents cached server state
type ServerStateInfo struct {
	State     *qbittorrent.ServerState `json:"state"`
	UpdatedAt time.Time                `json:"updated_at"`
}

// cacheInstance holds the global cache manager
var cacheInstance *CacheManager

// Initialize creates and configures the cache manager
func Initialize(cfg *config.CacheConfig) (*CacheManager, error) {
	logger := logging.GetCacheLogger()

	// Create go-cache instance with configured settings
	c := cache.New(
		cfg.AuthSessionTTL,  // default expiration (we'll override per item)
		cfg.CleanupInterval, // cleanup interval
	)

	// Set eviction callback to track evictions
	c.OnEvicted(func(key string, value interface{}) {
		if cacheInstance != nil {
			cacheInstance.mutex.Lock()
			cacheInstance.stats.Evictions++
			cacheInstance.mutex.Unlock()
		}
	})

	manager := &CacheManager{
		cache:  c,
		config: cfg,
		logger: logger,
		stats: &CacheStats{
			LastReset: time.Now(),
		},
	}

	// Set global instance
	cacheInstance = manager

	logger.WithFields(map[string]interface{}{
		"auth_session_ttl": cfg.AuthSessionTTL,
		"disk_space_ttl":   cfg.DiskSpaceTTL,
		"cleanup_interval": cfg.CleanupInterval,
		"max_items":        cfg.MaxItems,
	}).Info("Cache manager initialized successfully")

	return manager, nil
}

// GetManager returns the global cache manager instance
func GetManager() *CacheManager {
	return cacheInstance
}

// Authentication Session Caching

// SetAuthSession stores authentication session data
func (cm *CacheManager) SetAuthSession(session *AuthSession) {
	cm.mutex.Lock()
	cm.stats.Sets++
	cm.mutex.Unlock()

	cm.cache.Set(KeyAuthSession, session, cm.config.AuthSessionTTL)

	cm.logger.WithFields(map[string]interface{}{
		"key": KeyAuthSession,
		"ttl": cm.config.AuthSessionTTL,
	}).Debug("Authentication session cached")
}

// GetAuthSession retrieves cached authentication session
func (cm *CacheManager) GetAuthSession() (*AuthSession, bool) {
	value, found := cm.cache.Get(KeyAuthSession)

	cm.mutex.Lock()
	if found {
		cm.stats.Hits++
	} else {
		cm.stats.Misses++
	}
	cm.mutex.Unlock()

	if !found {
		cm.logger.WithField("key", KeyAuthSession).Debug("Authentication session cache miss")
		return nil, false
	}

	session, ok := value.(*AuthSession)
	if !ok {
		cm.logger.WithField("key", KeyAuthSession).Warn("Invalid auth session type in cache")
		cm.DeleteAuthSession() // Remove invalid entry
		return nil, false
	}

	cm.logger.WithField("key", KeyAuthSession).Debug("Authentication session cache hit")
	return session, true
}

// DeleteAuthSession removes cached authentication session
func (cm *CacheManager) DeleteAuthSession() {
	cm.mutex.Lock()
	cm.stats.Deletes++
	cm.mutex.Unlock()

	cm.cache.Delete(KeyAuthSession)
	cm.logger.WithField("key", KeyAuthSession).Debug("Authentication session cache deleted")
}

// IsAuthSessionValid checks if cached session is valid and not expired
func (cm *CacheManager) IsAuthSessionValid() bool {
	session, found := cm.GetAuthSession()
	if !found {
		return false
	}

	// Check if session has expired based on our own expiration logic
	if time.Now().After(session.ExpiresAt) {
		cm.DeleteAuthSession()
		return false
	}

	return true
}

// Disk Space Caching

// SetDiskSpace stores disk space information for a path
func (cm *CacheManager) SetDiskSpace(path string, diskSpace *DiskSpaceInfo) {
	cm.mutex.Lock()
	cm.stats.Sets++
	cm.mutex.Unlock()

	key := KeyDiskSpacePrefix + path
	cm.cache.Set(key, diskSpace, cm.config.DiskSpaceTTL)

	cm.logger.WithFields(map[string]interface{}{
		"key":  key,
		"path": path,
		"ttl":  cm.config.DiskSpaceTTL,
	}).Debug("Disk space information cached")
}

// GetDiskSpace retrieves cached disk space information for a path
func (cm *CacheManager) GetDiskSpace(path string) (*DiskSpaceInfo, bool) {
	key := KeyDiskSpacePrefix + path
	value, found := cm.cache.Get(key)

	cm.mutex.Lock()
	if found {
		cm.stats.Hits++
	} else {
		cm.stats.Misses++
	}
	cm.mutex.Unlock()

	if !found {
		cm.logger.WithFields(map[string]interface{}{
			"key":  key,
			"path": path,
		}).Debug("Disk space cache miss")
		return nil, false
	}

	diskSpace, ok := value.(*DiskSpaceInfo)
	if !ok {
		cm.logger.WithField("key", key).Warn("Invalid disk space type in cache")
		cm.DeleteDiskSpace(path) // Remove invalid entry
		return nil, false
	}

	cm.logger.WithFields(map[string]interface{}{
		"key":  key,
		"path": path,
	}).Debug("Disk space cache hit")
	return diskSpace, true
}

// DeleteDiskSpace removes cached disk space information for a path
func (cm *CacheManager) DeleteDiskSpace(path string) {
	cm.mutex.Lock()
	cm.stats.Deletes++
	cm.mutex.Unlock()

	key := KeyDiskSpacePrefix + path
	cm.cache.Delete(key)
	cm.logger.WithFields(map[string]interface{}{
		"key":  key,
		"path": path,
	}).Debug("Disk space cache deleted")
}

// Server State Caching

// SetServerState stores server state information
func (cm *CacheManager) SetServerState(state *qbittorrent.ServerState) {
	cm.mutex.Lock()
	cm.stats.Sets++
	cm.mutex.Unlock()

	stateInfo := &ServerStateInfo{
		State:     state,
		UpdatedAt: time.Now(),
	}

	// Use a longer TTL for server state as it changes less frequently
	ttl := 10 * time.Minute
	cm.cache.Set(KeyServerState, stateInfo, ttl)

	cm.logger.WithFields(map[string]interface{}{
		"key": KeyServerState,
		"ttl": ttl,
	}).Debug("Server state cached")
}

// GetServerState retrieves cached server state
func (cm *CacheManager) GetServerState() (*qbittorrent.ServerState, bool) {
	value, found := cm.cache.Get(KeyServerState)

	cm.mutex.Lock()
	if found {
		cm.stats.Hits++
	} else {
		cm.stats.Misses++
	}
	cm.mutex.Unlock()

	if !found {
		cm.logger.WithField("key", KeyServerState).Debug("Server state cache miss")
		return nil, false
	}

	stateInfo, ok := value.(*ServerStateInfo)
	if !ok {
		cm.logger.WithField("key", KeyServerState).Warn("Invalid server state type in cache")
		cm.DeleteServerState() // Remove invalid entry
		return nil, false
	}

	cm.logger.WithField("key", KeyServerState).Debug("Server state cache hit")
	return stateInfo.State, true
}

// DeleteServerState removes cached server state
func (cm *CacheManager) DeleteServerState() {
	cm.mutex.Lock()
	cm.stats.Deletes++
	cm.mutex.Unlock()

	cm.cache.Delete(KeyServerState)
	cm.logger.WithField("key", KeyServerState).Debug("Server state cache deleted")
}

// Cache Management Methods

// GetStats returns current cache statistics
func (cm *CacheManager) GetStats() *CacheStats {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// Update item count
	cm.stats.ItemCount = cm.cache.ItemCount()

	// Return a copy to avoid race conditions
	return &CacheStats{
		Hits:      cm.stats.Hits,
		Misses:    cm.stats.Misses,
		Sets:      cm.stats.Sets,
		Deletes:   cm.stats.Deletes,
		Evictions: cm.stats.Evictions,
		ItemCount: cm.stats.ItemCount,
		LastReset: cm.stats.LastReset,
	}
}

// GetHitRatio returns the cache hit ratio as a percentage
func (cm *CacheManager) GetHitRatio() float64 {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	total := cm.stats.Hits + cm.stats.Misses
	if total == 0 {
		return 0.0
	}
	return (float64(cm.stats.Hits) / float64(total)) * 100.0
}

// ResetStats resets cache statistics
func (cm *CacheManager) ResetStats() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.stats = &CacheStats{
		LastReset: time.Now(),
	}
	cm.logger.Info("Cache statistics reset")
}

// Clear removes all items from the cache
func (cm *CacheManager) Clear() {
	cm.cache.Flush()
	cm.ResetStats()
	cm.logger.Info("Cache cleared")
}

// GetItemCount returns the current number of items in cache
func (cm *CacheManager) GetItemCount() int {
	return cm.cache.ItemCount()
}

// DeleteExpired manually triggers cleanup of expired items
func (cm *CacheManager) DeleteExpired() {
	cm.cache.DeleteExpired()
	cm.logger.Debug("Expired cache items cleaned up")
}

// LogStats logs current cache statistics
func (cm *CacheManager) LogStats() {
	stats := cm.GetStats()
	hitRatio := cm.GetHitRatio()

	cm.logger.WithFields(map[string]interface{}{
		"hits":       stats.Hits,
		"misses":     stats.Misses,
		"sets":       stats.Sets,
		"deletes":    stats.Deletes,
		"evictions":  stats.Evictions,
		"item_count": stats.ItemCount,
		"hit_ratio":  fmt.Sprintf("%.2f%%", hitRatio),
		"uptime":     time.Since(stats.LastReset).String(),
	}).Info("Cache statistics")
}

// Shutdown gracefully shuts down the cache manager
func (cm *CacheManager) Shutdown() {
	cm.logger.Info("Shutting down cache manager")
	cm.LogStats()
	cm.Clear()
}

// Helper function to create a new AuthSession
func NewAuthSession(cookie string, expiresIn time.Duration) *AuthSession {
	now := time.Now()
	return &AuthSession{
		Cookie:    cookie,
		ExpiresAt: now.Add(expiresIn),
		CreatedAt: now,
	}
}

// Helper function to create a new DiskSpaceInfo
func NewDiskSpaceInfo(path string, total, used, free int64) *DiskSpaceInfo {
	return &DiskSpaceInfo{
		Path:      path,
		Total:     total,
		Used:      used,
		Free:      free,
		UpdatedAt: time.Now(),
	}
}
