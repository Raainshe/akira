package core

import (
	"strings"
)

func isWindowsPath(path string) bool {
	if len(path) < 2 || path[1] != ':' {
		return false
	}
	c := path[0]
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func windowsDriveRoot(path string) string {
	if !isWindowsPath(path) {
		return path
	}
	return strings.ToUpper(string(path[0])) + `:\`
}

func driveIdentifierFromWindowsPath(path string) (string, bool) {
	if !isWindowsPath(path) {
		return "", false
	}
	return windowsDriveRoot(path), true
}

func freeSpaceQueryPath(path string) string {
	if isWindowsPath(path) {
		return windowsDriveRoot(path)
	}
	return path
}
