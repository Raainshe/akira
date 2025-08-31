package qbittorrent

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

// Client represents a qBittorrent WebUI API client
type Client struct {
	baseURL    *url.URL
	username   string
	password   string
	httpClient *http.Client
	cookieJar  http.CookieJar
	timeout    time.Duration
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
		return nil, err
	}

	client := &Client{
		baseURL:  parsedURL,
		username: username,
		password: password,
		timeout:  30 * time.Second,
	}

	// Create HTTP client with cookie jar for session management
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
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

	return client, nil
}

// Login authenticates with the qBittorrent WebUI
func (c *Client) Login(ctx context.Context) error {
	// Implementation will be added in the next phase
	return nil
}

// Logout logs out from the qBittorrent WebUI
func (c *Client) Logout(ctx context.Context) error {
	// Implementation will be added in the next phase
	return nil
}

// GetTorrents retrieves all torrents from qBittorrent
func (c *Client) GetTorrents(ctx context.Context) ([]Torrent, error) {
	// Implementation will be added in the next phase
	return nil, nil
}

// GetTorrentProperties retrieves detailed properties for a specific torrent
func (c *Client) GetTorrentProperties(ctx context.Context, hash string) (*TorrentProperties, error) {
	// Implementation will be added in the next phase
	return nil, nil
}

// AddMagnet adds a magnet link to qBittorrent
func (c *Client) AddMagnet(ctx context.Context, magnetURI string, options AddTorrentRequest) error {
	// Implementation will be added in the next phase
	return nil
}

// DeleteTorrents deletes torrents from qBittorrent
func (c *Client) DeleteTorrents(ctx context.Context, hashes []string, deleteFiles bool) error {
	// Implementation will be added in the next phase
	return nil
}

// PauseTorrents pauses torrents in qBittorrent
func (c *Client) PauseTorrents(ctx context.Context, hashes []string) error {
	// Implementation will be added in the next phase
	return nil
}

// ResumeTorrents resumes torrents in qBittorrent
func (c *Client) ResumeTorrents(ctx context.Context, hashes []string) error {
	// Implementation will be added in the next phase
	return nil
}

// GetServerState retrieves global server state information
func (c *Client) GetServerState(ctx context.Context) (*ServerState, error) {
	// Implementation will be added in the next phase
	return nil, nil
}

// GetDiskSpace retrieves disk space information for a given path
func (c *Client) GetDiskSpace(ctx context.Context, path string) (*DiskSpace, error) {
	// Implementation will be added in the next phase
	return nil, nil
}

// IsAuthenticated checks if the client is currently authenticated
func (c *Client) IsAuthenticated(ctx context.Context) bool {
	// Implementation will be added in the next phase
	return false
}
