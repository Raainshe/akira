//go:build linux || darwin || freebsd

package core

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"
)

// getDiskSpacePlatform gets disk space using Unix-specific syscalls
func (ds *DiskService) getDiskSpacePlatform(path string) (*DiskInfo, error) {
	ds.logger.WithField("platform", runtime.GOOS).Debug("Getting real disk space information")

	// Get file info to ensure path exists
	_, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}

	// Get filesystem statistics
	var stat syscall.Statfs_t
	err = syscall.Statfs(path, &stat)
	if err != nil {
		return nil, fmt.Errorf("failed to get filesystem stats: %w", err)
	}

	// Calculate space in bytes
	blockSize := int64(stat.Bsize)
	total := int64(stat.Blocks) * blockSize
	free := int64(stat.Bavail) * blockSize // Available to non-root users
	used := total - (int64(stat.Bfree) * blockSize)

	// Calculate percentages
	usedPercent := ds.calculatePercentage(used, total)
	freePercent := ds.calculatePercentage(free, total)

	return &DiskInfo{
		Path:        path,
		Total:       total,
		Used:        used,
		Free:        free,
		Available:   free,
		UsedPercent: usedPercent,
		FreePercent: freePercent,
		Filesystem:  "unknown", // Could be enhanced to detect filesystem type
		MountPoint:  path,
		LastChecked: time.Now(),
	}, nil
}
