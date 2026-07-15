package system

import (
	"fmt"
	"os"
	"path/filepath"
)

// DiskUsageInfo holds storage breakdown information.
type DiskUsageInfo struct {
	Sandboxes int64
	Templates int64
	Caches    int64
	Images    int64
	Logs      int64
	Total     int64
}

// GetDiskUsage collects storage usage by category.
func GetDiskUsage() (*DiskUsageInfo, error) {
	piHome := PiHome()
	info := &DiskUsageInfo{}

	// Sandboxes are now at piHome directly (not under a "sandboxes" subdir)

	// Sandboxes are now at piHome directly (not under a "sandboxes" subdir)
	// Only count directories that contain meta.json (actual sandboxes)
	if DirExists(piHome) {
		entries, err := os.ReadDir(piHome)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					metaPath := filepath.Join(piHome, entry.Name(), "meta.json")
					if _, err := os.Stat(metaPath); err == nil {
						size, err := DirSize(filepath.Join(piHome, entry.Name()))
						if err == nil {
							info.Sandboxes += size
						}
					}
				}
			}
		}
	}

	// Templates, caches, images, logs are still under piHome/{name}
	for name, sizePtr := range map[string]*int64{
		"templates": &info.Templates,
		"caches":    &info.Caches,
		"images":    &info.Images,
		"logs":      &info.Logs,
	} {
		path := filepath.Join(piHome, name)
		if DirExists(path) {
			size, err := DirSize(path)
			if err == nil {
				*sizePtr = size
			}
		}
	}

	info.Total = info.Sandboxes + info.Templates + info.Caches + info.Images + info.Logs

	return info, nil
}

// PrintDiskUsage prints the disk usage breakdown.
func PrintDiskUsage(info *DiskUsageInfo) {
	fmt.Println("=== pi-sandbox Disk Usage ===")
	fmt.Println()
	fmt.Printf("Sandboxes:  %s\n", FormatSize(info.Sandboxes))
	fmt.Printf("Templates:  %s\n", FormatSize(info.Templates))
	fmt.Printf("Caches:     %s\n", FormatSize(info.Caches))
	fmt.Printf("Images:     %s\n", FormatSize(info.Images))
	fmt.Printf("Logs:       %s\n", FormatSize(info.Logs))
	fmt.Println()
	fmt.Printf("Total:      %s\n", FormatSize(info.Total))
}

// DirSize returns the total size of a directory in bytes.
// (Duplicate of system.go — will be consolidated)
func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
