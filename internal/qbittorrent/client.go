package qbittorrent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/raainshe/akira/internal/logging"
)

// Client represents a qBittorrent WebUI API client
type Client struct {
	baseURL    *url.URL
	username   string
	password   string
	httpClient *http.Client
	cookieJar  http.CookieJar
	timeout    time.Duration
	logger     *logging.Logger
}

// ClientOption represents a configuration option for the qBittorrent client
type ClientOption func(*Client)

// WithTimeout sets the HTTP request timeout for the client
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
		c.httpClient.Timeout = timeout
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// NewClient creates a new qBittorrent API client
func NewClient(baseURL, username, password string, options ...ClientOption) (*Client, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	client := &Client{
		baseURL:  parsedURL,
		username: username,
		password: password,
		timeout:  30 * time.Second,
		logger:   logging.GetQBittorrentLogger(),
	}

	// Create HTTP client with cookie jar for session management
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client.httpClient = &http.Client{
		Timeout: client.timeout,
		Jar:     jar,
	}
	client.cookieJar = jar

	// Apply options
	for _, option := range options {
		option(client)
	}

	client.logger.WithFields(map[string]interface{}{
		"base_url": baseURL,
		"username": username,
		"timeout":  client.timeout,
	}).Info("qBittorrent client created")

	return client, nil
}

// makeRequest performs an HTTP request with error handling and retries
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, data interface{}, result interface{}) error {
	reqURL := c.baseURL.ResolveReference(&url.URL{Path: endpoint})

	var body io.Reader
	var contentType string

	// Prepare request body based on data type
	if data != nil {
		switch v := data.(type) {
		case url.Values:
			body = strings.NewReader(v.Encode())
			contentType = "application/x-www-form-urlencoded"
		case *bytes.Buffer:
			body = v
			contentType = "multipart/form-data"
		default:
			jsonData, err := json.Marshal(data)
			if err != nil {
				return fmt.Errorf("failed to marshal request data: %w", err)
			}
			body = bytes.NewReader(jsonData)
			contentType = "application/json"
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	c.logger.WithFields(map[string]interface{}{
		"method":   method,
		"endpoint": endpoint,
		"url":      reqURL.String(),
	}).Debug("Making API request")

	// Perform request with retries
	var resp *http.Response
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err = c.httpClient.Do(req)
		if err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("request failed after %d attempts: %w", maxRetries, err)
			}
			c.logger.WithFields(map[string]interface{}{
				"attempt": attempt,
				"error":   err,
			}).Warn("Request attempt failed, retrying")
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		break
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	c.logger.WithFields(map[string]interface{}{
		"status_code": resp.StatusCode,
		"body_length": len(respBody),
	}).Debug("Received API response")

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		apiErr := &APIError{
			Code:    resp.StatusCode,
			Message: resp.Status,
			Details: string(respBody),
		}
		return apiErr
	}

	// Parse response if result is provided
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// Login authenticates with the qBittorrent WebUI
func (c *Client) Login(ctx context.Context) error {
	c.logger.Info("Authenticating with qBittorrent")

	data := url.Values{}
	data.Set("username", c.username)
	data.Set("password", c.password)

	err := c.makeRequest(ctx, "POST", "/api/v2/auth/login", data, nil)
	if err != nil {
		c.logger.WithError(err).Error("Authentication failed")
		return fmt.Errorf("authentication failed: %w", err)
	}

	c.logger.Info("Authentication successful")
	return nil
}

// Logout logs out from the qBittorrent WebUI
func (c *Client) Logout(ctx context.Context) error {
	c.logger.Info("Logging out from qBittorrent")

	err := c.makeRequest(ctx, "POST", "/api/v2/auth/logout", nil, nil)
	if err != nil {
		c.logger.WithError(err).Warn("Logout request failed")
		// Don't return error as logout might fail if not logged in
	}

	c.logger.Info("Logout completed")
	return nil
}

// IsAuthenticated checks if the client is currently authenticated
func (c *Client) IsAuthenticated(ctx context.Context) bool {
	// Try to make a simple authenticated request
	err := c.makeRequest(ctx, "GET", "/api/v2/app/version", nil, nil)
	return err == nil
}

// ensureAuthenticated ensures the client is authenticated before making API calls
func (c *Client) ensureAuthenticated(ctx context.Context) error {
	if c.IsAuthenticated(ctx) {
		return nil
	}
	return c.Login(ctx)
}

// GetTorrents retrieves all torrents from qBittorrent
func (c *Client) GetTorrents(ctx context.Context) ([]Torrent, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}

	c.logger.Debug("Fetching torrent list")

	var torrents []Torrent
	err := c.makeRequest(ctx, "GET", "/api/v2/torrents/info", nil, &torrents)
	if err != nil {
		c.logger.WithError(err).Error("Failed to fetch torrents")
		return nil, fmt.Errorf("failed to fetch torrents: %w", err)
	}

	c.logger.WithField("count", len(torrents)).Info("Torrents fetched successfully")
	return torrents, nil
}

// GetTorrentProperties retrieves detailed properties for a specific torrent
func (c *Client) GetTorrentProperties(ctx context.Context, hash string) (*TorrentProperties, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}

	c.logger.WithField("hash", hash).Debug("Fetching torrent properties")

	data := url.Values{}
	data.Set("hash", hash)

	var properties TorrentProperties
	err := c.makeRequest(ctx, "GET", "/api/v2/torrents/properties?"+data.Encode(), nil, &properties)
	if err != nil {
		c.logger.WithError(err).WithField("hash", hash).Error("Failed to fetch torrent properties")
		return nil, fmt.Errorf("failed to fetch torrent properties: %w", err)
	}

	c.logger.WithField("hash", hash).Debug("Torrent properties fetched successfully")
	return &properties, nil
}

// AddMagnet adds a magnet link to qBittorrent
func (c *Client) AddMagnet(ctx context.Context, magnetURI string, options AddTorrentRequest) error {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return err
	}

	// Mask magnet URI for logging (show only first 50 chars)
	maskedMagnet := magnetURI
	if len(magnetURI) > 50 {
		maskedMagnet = magnetURI[:50] + "..."
	}

	c.logger.WithFields(map[string]interface{}{
		"magnet_uri": maskedMagnet,
		"category":   options.Category,
		"save_path":  options.SavePath,
	}).Info("Adding magnet link")

	// Prepare form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add magnet URI
	writer.WriteField("urls", magnetURI)

	// Add optional fields
	if options.SavePath != "" {
		writer.WriteField("savepath", options.SavePath)
	}
	if options.Category != "" {
		writer.WriteField("category", options.Category)
	}
	if options.Tags != "" {
		writer.WriteField("tags", options.Tags)
	}
	if options.SkipChecking {
		writer.WriteField("skip_checking", "true")
	}
	if options.Paused {
		writer.WriteField("paused", "true")
	}
	if options.RootFolder {
		writer.WriteField("root_folder", "true")
	}
	if options.Rename != "" {
		writer.WriteField("rename", options.Rename)
	}
	if options.UpLimit > 0 {
		writer.WriteField("upLimit", strconv.FormatInt(options.UpLimit, 10))
	}
	if options.DlLimit > 0 {
		writer.WriteField("dlLimit", strconv.FormatInt(options.DlLimit, 10))
	}
	if options.RatioLimit > 0 {
		writer.WriteField("ratioLimit", strconv.FormatFloat(options.RatioLimit, 'f', 2, 64))
	}
	if options.SeedingTimeLimit > 0 {
		writer.WriteField("seedingTimeLimit", strconv.FormatInt(options.SeedingTimeLimit, 10))
	}
	if options.AutoTMM {
		writer.WriteField("autoTMM", "true")
	}
	if options.SequentialDownload {
		writer.WriteField("sequentialDownload", "true")
	}
	if options.FirstLastPiecePriority {
		writer.WriteField("firstLastPiecePriority", "true")
	}

	writer.Close()

	// Set content type for multipart form
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL.ResolveReference(&url.URL{Path: "/api/v2/torrents/add"}).String(), &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Error("Failed to add magnet link")
		return fmt.Errorf("failed to add magnet link: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		c.logger.WithFields(map[string]interface{}{
			"status_code": resp.StatusCode,
			"response":    string(body),
		}).Error("Add magnet request failed")
		return &APIError{
			Code:    resp.StatusCode,
			Message: resp.Status,
			Details: string(body),
		}
	}

	c.logger.Info("Magnet link added successfully")
	return nil
}

// DeleteTorrents deletes torrents from qBittorrent
func (c *Client) DeleteTorrents(ctx context.Context, hashes []string, deleteFiles bool) error {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return err
	}

	c.logger.WithFields(map[string]interface{}{
		"hashes":       hashes,
		"delete_files": deleteFiles,
		"count":        len(hashes),
	}).Info("Deleting torrents")

	data := url.Values{}
	data.Set("hashes", strings.Join(hashes, "|"))
	data.Set("deleteFiles", strconv.FormatBool(deleteFiles))

	err := c.makeRequest(ctx, "POST", "/api/v2/torrents/delete", data, nil)
	if err != nil {
		c.logger.WithError(err).Error("Failed to delete torrents")
		return fmt.Errorf("failed to delete torrents: %w", err)
	}

	c.logger.WithField("count", len(hashes)).Info("Torrents deleted successfully")
	return nil
}

// PauseTorrents pauses torrents in qBittorrent
func (c *Client) PauseTorrents(ctx context.Context, hashes []string) error {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return err
	}

	c.logger.WithFields(map[string]interface{}{
		"hashes": hashes,
		"count":  len(hashes),
	}).Info("Pausing torrents")

	data := url.Values{}
	data.Set("hashes", strings.Join(hashes, "|"))

	err := c.makeRequest(ctx, "POST", "/api/v2/torrents/pause", data, nil)
	if err != nil {
		c.logger.WithError(err).Error("Failed to pause torrents")
		return fmt.Errorf("failed to pause torrents: %w", err)
	}

	c.logger.WithField("count", len(hashes)).Info("Torrents paused successfully")
	return nil
}

// ResumeTorrents resumes torrents in qBittorrent
func (c *Client) ResumeTorrents(ctx context.Context, hashes []string) error {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return err
	}

	c.logger.WithFields(map[string]interface{}{
		"hashes": hashes,
		"count":  len(hashes),
	}).Info("Resuming torrents")

	data := url.Values{}
	data.Set("hashes", strings.Join(hashes, "|"))

	err := c.makeRequest(ctx, "POST", "/api/v2/torrents/resume", data, nil)
	if err != nil {
		c.logger.WithError(err).Error("Failed to resume torrents")
		return fmt.Errorf("failed to resume torrents: %w", err)
	}

	c.logger.WithField("count", len(hashes)).Info("Torrents resumed successfully")
	return nil
}

// GetServerState retrieves global server state information
func (c *Client) GetServerState(ctx context.Context) (*ServerState, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}

	c.logger.Debug("Fetching server state")

	var state ServerState
	err := c.makeRequest(ctx, "GET", "/api/v2/sync/maindata", nil, &state)
	if err != nil {
		c.logger.WithError(err).Error("Failed to fetch server state")
		return nil, fmt.Errorf("failed to fetch server state: %w", err)
	}

	c.logger.Debug("Server state fetched successfully")
	return &state, nil
}

// GetDiskSpace retrieves disk space information for a given path
func (c *Client) GetDiskSpace(ctx context.Context, path string) (*DiskSpace, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}

	c.logger.WithField("path", path).Debug("Fetching disk space")

	// qBittorrent doesn't have a direct disk space API, so we'll use a system call
	// This is a placeholder - in a real implementation, you'd use syscall or a library
	// For now, we'll return an error indicating this needs platform-specific implementation
	return nil, fmt.Errorf("disk space checking not implemented - requires platform-specific code")
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
