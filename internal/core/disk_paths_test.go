package core

import "testing"

func TestWindowsDriveRoot(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{`E:\Series`, `E:\`},
		{`e:/Movies`, `E:\`},
		{`/downloads/series`, `/downloads/series`},
	}

	for _, tt := range tests {
		got := freeSpaceQueryPath(tt.path)
		if got != tt.want {
			t.Fatalf("freeSpaceQueryPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestDriveIdentifierFromWindowsPath(t *testing.T) {
	driveID, ok := driveIdentifierFromWindowsPath(`F:\Anime`)
	if !ok || driveID != `F:\` {
		t.Fatalf("expected F:\\, got %q ok=%v", driveID, ok)
	}

	_, ok = driveIdentifierFromWindowsPath("/tmp")
	if ok {
		t.Fatal("expected unix path to not match windows drive")
	}
}
