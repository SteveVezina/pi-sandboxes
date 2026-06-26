package system

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PiHome returns the ~/.pi/ directory path.
func PiHome() string {
	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".pi")
}

// DirExists checks if a directory exists.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// DirSize returns the total size of a directory in bytes.
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// FormatSize formats a byte size into human-readable form.
func FormatSize(bytes int64) string {
	if bytes >= 1024*1024*1024 {
		return fmt.Sprintf("%.1f GiB", float64(bytes)/(1024*1024*1024))
	}
	if bytes >= 1024*1024 {
		return fmt.Sprintf("%.1f MiB", float64(bytes)/(1024*1024))
	}
	if bytes >= 1024 {
		return fmt.Sprintf("%.1f KiB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%d B", bytes)
}

// DiskInfo holds disk space information.
type DiskInfo struct {
	Total   int64
	Free    int64
	Used    int64
}

// GetDiskInfo returns disk space info for the given path.
func GetDiskInfo(path string) (*DiskInfo, error) {
	// Simple implementation — just check if path exists
	// Full disk info requires syscall on Unix
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return &DiskInfo{
		Total: info.Size(),
		Free:  0, // Not implemented in MVP
		Used:  0, // Not implemented in MVP
	}, nil
}

// TimeAgo returns a human-readable string for time ago.
func TimeAgo(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", d/time.Minute)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", d/time.Hour)
	}
	return fmt.Sprintf("%dd ago", d/(24*time.Hour))
}
