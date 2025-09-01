//go:build windows

package core

import (
	"fmt"
	"os"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// getDiskSpacePlatform gets disk space using Windows-specific API
func (ds *DiskService) getDiskSpacePlatform(path string) (*DiskInfo, error) {
	ds.logger.WithField("platform", runtime.GOOS).Debug("Getting real disk space information")

	// Get file info to ensure path exists
	_, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}

	// Convert path to UTF-16 for Windows API
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("failed to convert path to UTF-16: %w", err)
	}

	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64

	// Call GetDiskFreeSpaceEx Windows API
	err = windows.GetDiskFreeSpaceEx(
		pathPtr,
		(*uint64)(unsafe.Pointer(&freeBytesAvailable)),
		(*uint64)(unsafe.Pointer(&totalNumberOfBytes)),
		(*uint64)(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk space: %w", err)
	}

	// Convert to int64 for consistency with Unix implementation
	total := int64(totalNumberOfBytes)
	free := int64(freeBytesAvailable)
	used := total - int64(totalNumberOfFreeBytes)

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
		Filesystem:  "NTFS", // Most common on Windows, could be enhanced to detect actual type
		MountPoint:  path,
		LastChecked: time.Now(),
	}, nil
}
