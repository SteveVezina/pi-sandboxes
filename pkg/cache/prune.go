package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PruneResult holds the results of a cache prune.
type PruneResult struct {
	RemovedScopes int
	RemovedBytes  int64
}

// Prune removes unused cache scopes.
func Prune(activeScopes []Scope, maxSize int64) (*PruneResult, error) {
	result := &PruneResult{}

	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}
	cacheDir := filepath.Join(home, ".pi", "caches")

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return result, fmt.Errorf("read cache dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scopePath := filepath.Join(cacheDir, entry.Name())

		// Check if this scope is active
		active := false
		for _, s := range activeScopes {
			if s.String() == entry.Name() {
				active = true
				break
			}
		}

		if !active {
			// Check size limit
			size, err := dirSize(scopePath)
			if err == nil && size > maxSize {
				// Too large, remove
				if err := os.RemoveAll(scopePath); err == nil {
					result.RemovedScopes++
					result.RemovedBytes += size
				}
			}
		}
	}

	// Also clean old cache files (> 90 days unused)
	cleanOldFiles(cacheDir, 90*24*time.Hour)

	return result, nil
}

// cleanOldFiles removes files older than the given age.
func cleanOldFiles(dir string, age time.Duration) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) > age {
			os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}

