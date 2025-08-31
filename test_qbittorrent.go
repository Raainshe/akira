// test_qbittorrent.go - Comprehensive test program for qBittorrent integration
// Run with: go run test_qbittorrent.go
// This file can be deleted after testing

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/raainshe/akira/internal/config"
	"github.com/raainshe/akira/internal/logging"
	"github.com/raainshe/akira/internal/qbittorrent"
)

func main() {
	fmt.Println("🧪 Testing qBittorrent Integration...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	_, err = logging.Initialize(&cfg.Logging)
	if err != nil {
		fmt.Printf("❌ Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Configuration loaded successfully\n")
	fmt.Printf("🔗 qBittorrent URL: %s\n", cfg.QBittorrent.URL)
	fmt.Printf("👤 Username: %s\n", cfg.QBittorrent.Username)

	// Create qBittorrent client
	client, err := qbittorrent.NewClient(
		cfg.QBittorrent.URL,
		cfg.QBittorrent.Username,
		cfg.QBittorrent.Password,
		qbittorrent.WithTimeout(cfg.QBittorrent.RequestTimeout),
	)
	if err != nil {
		fmt.Printf("❌ Failed to create qBittorrent client: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ qBittorrent client created successfully")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test 1: Authentication
	fmt.Println("\n🔐 Testing Authentication...")
	testAuthentication(ctx, client)

	// Test 2: Get Torrents
	fmt.Println("\n📋 Testing Torrent Listing...")
	torrents := testGetTorrents(ctx, client)

	// Test 3: Get Torrent Properties (if we have torrents)
	if len(torrents) > 0 {
		fmt.Println("\n🔍 Testing Torrent Properties...")
		testGetTorrentProperties(ctx, client, torrents[0].Hash)
	}

	// Test 4: Server State
	fmt.Println("\n🖥️ Testing Server State...")
	testGetServerState(ctx, client)

	// Test 5: Add Magnet Link (optional, commented out to avoid adding test torrents)
	fmt.Println("\n🧲 Testing Magnet Link Addition (SKIPPED - uncomment to test)...")
	// testAddMagnet(ctx, client, cfg)

	// Test 6: Torrent Operations (pause/resume) - only if we have torrents
	if len(torrents) > 0 {
		fmt.Println("\n⏸️ Testing Torrent Operations...")
		testTorrentOperations(ctx, client, []string{torrents[0].Hash})
	}

	// Test 7: Error Handling
	fmt.Println("\n❌ Testing Error Handling...")
	testErrorHandling(ctx, client)

	// Test 8: Logout
	fmt.Println("\n🚪 Testing Logout...")
	testLogout(ctx, client)

	fmt.Println("\n✅ qBittorrent integration test completed!")
	fmt.Println("🗑️  You can delete this test file: test_qbittorrent.go")
}

func testAuthentication(ctx context.Context, client *qbittorrent.Client) {
	// Test login
	err := client.Login(ctx)
	if err != nil {
		fmt.Printf("❌ Authentication failed: %v\n", err)
		fmt.Println("💡 Check your qBittorrent URL, username, and password in .env file")
		os.Exit(1)
	}
	fmt.Println("✅ Authentication successful")

	// Test authentication check
	if client.IsAuthenticated(ctx) {
		fmt.Println("✅ Authentication verification successful")
	} else {
		fmt.Println("⚠️ Authentication verification failed")
	}
}

func testGetTorrents(ctx context.Context, client *qbittorrent.Client) []qbittorrent.Torrent {
	torrents, err := client.GetTorrents(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to get torrents: %v\n", err)
		return nil
	}

	fmt.Printf("✅ Successfully retrieved %d torrents\n", len(torrents))

	if len(torrents) == 0 {
		fmt.Println("ℹ️  No torrents found - this is normal if you don't have any active torrents")
		return torrents
	}

	// Display first few torrents
	displayCount := len(torrents)
	if displayCount > 3 {
		displayCount = 3
	}

	fmt.Printf("📋 Showing first %d torrents:\n", displayCount)
	for i := 0; i < displayCount; i++ {
		torrent := torrents[i]
		fmt.Printf("  %d. %s\n", i+1, torrent.Name)
		fmt.Printf("     State: %s | Progress: %.1f%% | Size: %s\n",
			torrent.GetStateDisplayName(),
			torrent.GetProgressPercentage(),
			qbittorrent.FormatBytes(torrent.Size))
		if torrent.IsActive() {
			fmt.Printf("     ↓ %s | ↑ %s\n",
				qbittorrent.FormatSpeed(torrent.Dlspeed),
				qbittorrent.FormatSpeed(torrent.Upspeed))
		}
		fmt.Println()
	}

	return torrents
}

func testGetTorrentProperties(ctx context.Context, client *qbittorrent.Client, hash string) {
	properties, err := client.GetTorrentProperties(ctx, hash)
	if err != nil {
		fmt.Printf("❌ Failed to get torrent properties: %v\n", err)
		return
	}

	fmt.Println("✅ Successfully retrieved torrent properties")
	fmt.Printf("📊 Torrent Details:\n")
	fmt.Printf("  Save Path: %s\n", properties.SavePath)
	fmt.Printf("  Total Size: %s\n", qbittorrent.FormatBytes(properties.TotalSize))
	fmt.Printf("  Downloaded: %s\n", qbittorrent.FormatBytes(properties.TotalDownloaded))
	fmt.Printf("  Uploaded: %s\n", qbittorrent.FormatBytes(properties.TotalUploaded))
	fmt.Printf("  Share Ratio: %.2f\n", properties.ShareRatio)
	fmt.Printf("  Seeding Time: %s\n", time.Duration(properties.SeedingTime*int64(time.Second)).String())
	fmt.Printf("  Peers: %d connected, %d total\n", properties.Peers, properties.PeersTotal)
	fmt.Printf("  Seeds: %d connected, %d total\n", properties.Seeds, properties.SeedsTotal)
}

func testGetServerState(ctx context.Context, client *qbittorrent.Client) {
	state, err := client.GetServerState(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to get server state: %v\n", err)
		return
	}

	fmt.Println("✅ Successfully retrieved server state")
	fmt.Printf("🖥️ Server Information:\n")
	fmt.Printf("  Connection Status: %s\n", state.ConnectionStatus)
	fmt.Printf("  DHT Nodes: %d\n", state.DhtNodes)
	fmt.Printf("  Global Download Speed: %s\n", qbittorrent.FormatSpeed(state.DlInfoSpeed))
	fmt.Printf("  Global Upload Speed: %s\n", qbittorrent.FormatSpeed(state.UpInfoSpeed))
	fmt.Printf("  Downloaded this session: %s\n", qbittorrent.FormatBytes(state.DlInfoData))
	fmt.Printf("  Uploaded this session: %s\n", qbittorrent.FormatBytes(state.UpInfoData))
}

func testAddMagnet(ctx context.Context, client *qbittorrent.Client, cfg *config.Config) {
	// Test magnet URI (Ubuntu 20.04 LTS - small and safe)
	testMagnet := "magnet:?xt=urn:btih:e2467cbf021192c241367b892230dc1e05c0580e&dn=ubuntu-20.04.6-desktop-amd64.iso"

	fmt.Printf("🧲 Adding test magnet link...\n")
	fmt.Printf("  Magnet: %s...\n", testMagnet[:50])

	options := qbittorrent.AddTorrentRequest{
		Category: "test",
		SavePath: cfg.GetSavePathForCategory("test"),
		Paused:   true, // Start paused so we can delete it immediately
		Tags:     "test,integration",
	}

	err := client.AddMagnet(ctx, testMagnet, options)
	if err != nil {
		fmt.Printf("❌ Failed to add magnet link: %v\n", err)
		return
	}

	fmt.Println("✅ Successfully added magnet link")
	fmt.Println("⚠️  NOTE: Test torrent was added in paused state. You may want to delete it manually.")
}

func testTorrentOperations(ctx context.Context, client *qbittorrent.Client, hashes []string) {
	if len(hashes) == 0 {
		fmt.Println("ℹ️  No torrents to test operations on")
		return
	}

	hash := hashes[0]
	fmt.Printf("🔧 Testing operations on torrent: %s...\n", hash[:16])

	// Test pause
	err := client.PauseTorrents(ctx, []string{hash})
	if err != nil {
		fmt.Printf("❌ Failed to pause torrent: %v\n", err)
	} else {
		fmt.Println("✅ Successfully paused torrent")
	}

	// Wait a moment
	time.Sleep(2 * time.Second)

	// Test resume
	err = client.ResumeTorrents(ctx, []string{hash})
	if err != nil {
		fmt.Printf("❌ Failed to resume torrent: %v\n", err)
	} else {
		fmt.Println("✅ Successfully resumed torrent")
	}
}

func testErrorHandling(ctx context.Context, client *qbittorrent.Client) {
	// Test invalid torrent hash
	_, err := client.GetTorrentProperties(ctx, "invalid_hash_12345")
	if err != nil {
		fmt.Printf("✅ Error handling works: %v\n", err)
	} else {
		fmt.Println("⚠️  Expected error for invalid hash, but got success")
	}

	// Test invalid torrent operation
	err = client.PauseTorrents(ctx, []string{"nonexistent_hash"})
	if err != nil {
		fmt.Printf("✅ Error handling for invalid operations works\n")
	} else {
		fmt.Println("⚠️  Expected error for invalid torrent hash, but operation succeeded")
	}
}

func testLogout(ctx context.Context, client *qbittorrent.Client) {
	err := client.Logout(ctx)
	if err != nil {
		fmt.Printf("⚠️ Logout failed: %v\n", err)
	} else {
		fmt.Println("✅ Successfully logged out")
	}

	// Verify we're logged out
	if !client.IsAuthenticated(ctx) {
		fmt.Println("✅ Logout verification successful")
	} else {
		fmt.Println("⚠️ Still appears to be authenticated after logout")
	}
}
