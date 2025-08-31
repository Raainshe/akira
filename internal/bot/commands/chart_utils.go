package commands

import (
	"bytes"
	"fmt"

	"github.com/raainshe/akira/internal/core"
	"github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

// generateDiskUsagePieChart creates a pie chart for disk usage and returns it as PNG bytes
func generateDiskUsagePieChart(path string, used, total int64) ([]byte, error) {
	// Calculate available space
	available := total - used
	usedPercent := float64(used) / float64(total) * 100

	// Create pie chart with better styling
	pie := chart.PieChart{
		Title:  fmt.Sprintf("Disk Usage: %s (%.1f%% Used)", path, usedPercent),
		Width:  600,
		Height: 400,
		Background: chart.Style{
			FillColor: drawing.ColorWhite,
		},
		Values: []chart.Value{
			{
				Value: float64(used),
				Label: "Used",
				Style: chart.Style{
					FillColor:   drawing.ColorFromHex("FF6B6B"), // Red for used
					StrokeColor: drawing.ColorFromHex("CC5555"),
					StrokeWidth: 2,
				},
			},
			{
				Value: float64(available),
				Label: "Available",
				Style: chart.Style{
					FillColor:   drawing.ColorFromHex("4ECDC4"), // Teal for available
					StrokeColor: drawing.ColorFromHex("3DA89E"),
					StrokeWidth: 2,
				},
			},
		},
	}

	// Render to PNG
	var buf bytes.Buffer
	if err := pie.Render(chart.PNG, &buf); err != nil {
		return nil, fmt.Errorf("failed to render pie chart: %w", err)
	}

	return buf.Bytes(), nil
}

// generateMultiDiskPieChart creates a pie chart showing usage for the main disk path only
func generateMultiDiskPieChart(diskSummary *core.DiskSummary) ([]byte, error) {
	if diskSummary == nil || len(diskSummary.Paths) == 0 {
		return nil, fmt.Errorf("no disk data available")
	}

	// Find the main disk path (DISK_SPACE_CHECK_PATH) - should be "/tmp"
	var mainDiskInfo *core.DiskInfo
	mainPath := "/tmp" // This should match DISK_SPACE_CHECK_PATH from .env

	for path, diskInfo := range diskSummary.Paths {
		if path == mainPath {
			mainDiskInfo = diskInfo
			break
		}
	}

	// If we can't find the main path, use the first available one
	if mainDiskInfo == nil {
		for _, diskInfo := range diskSummary.Paths {
			mainDiskInfo = diskInfo
			mainPath = diskInfo.Path
			break
		}
	}

	if mainDiskInfo == nil {
		return nil, fmt.Errorf("no disk information available")
	}

	// Calculate available space
	available := mainDiskInfo.Total - mainDiskInfo.Used

	// Use red for used space regardless of percentage
	usedColor := drawing.ColorFromHex("E74C3C") // Red for used space

	// Create pie chart with just the main disk usage
	pie := chart.PieChart{
		Title:  fmt.Sprintf("Disk Usage: %s (%.1f%% Used)", mainPath, mainDiskInfo.UsedPercent),
		Width:  600,
		Height: 500, // Increased height to accommodate padding
		Background: chart.Style{
			FillColor: drawing.ColorWhite,
		},
		// Add padding between title and chart
		TitleStyle: chart.Style{
			Padding: chart.Box{
				Top:    40,
				Left:   0,
				Right:  0,
				Bottom: 80,
			},
		},
		Values: []chart.Value{
			{
				Value: float64(mainDiskInfo.Used),
				Label: fmt.Sprintf("Used %.1f%%", mainDiskInfo.UsedPercent),
				Style: chart.Style{
					FillColor:   usedColor,
					StrokeColor: drawing.ColorFromHex("2C3E50"),
					StrokeWidth: 2,
				},
			},
			{
				Value: float64(available),
				Label: fmt.Sprintf("Available %.1f%%", float64(available)/float64(mainDiskInfo.Total)*100),
				Style: chart.Style{
					FillColor:   drawing.ColorFromHex("3498DB"), // Blue for available space
					StrokeColor: drawing.ColorFromHex("2980B9"),
					StrokeWidth: 2,
				},
			},
		},
	}

	// Render to PNG
	var buf bytes.Buffer
	if err := pie.Render(chart.PNG, &buf); err != nil {
		return nil, fmt.Errorf("failed to render pie chart: %w", err)
	}

	return buf.Bytes(), nil
}
