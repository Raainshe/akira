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

// getDriveIdentifierPlatform returns a unique identifier for the filesystem (Unix)
// Uses filesystem device ID to identify unique filesystems
func (ds *DiskService) getDriveIdentifierPlatform(path string) (string, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return "", fmt.Errorf("failed to get filesystem stats: %w", err)
	}
	// Use filesystem ID (combination of type and device) to identify unique filesystems
	// Format: "type:fsid0:fsid1" where fsid is a struct with two int32 values
	// Access the internal array field X__val
	fsid := stat.Fsid
	return fmt.Sprintf("%d:%d:%d", stat.Type, fsid.X__val[0], fsid.X__val[1]), nil
}
