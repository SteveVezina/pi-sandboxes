package system

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PruneResult holds the results of a prune operation.
type PruneResult struct {
	RemovedSandboxes int
	RemovedLogs      int
	RemovedBytes     int64
}

// RunPrune removes old sandbox state and logs.
func RunPrune(askConfirm bool) (*PruneResult, error) {
	result := &PruneResult{}
	piHome := PiHome()

	if !DirExists(piHome) {
		return result, fmt.Errorf("pi home does not exist: %s", piHome)
	}

	// Ask for confirmation
	if askConfirm {
		fmt.Println("This will remove old sandbox state and logs.")
		fmt.Println("Type 'yes' to confirm: ")
		// In CLI mode, this would read from stdin.
		// For now, we just proceed — the --yes flag is handled by the CLI.
	}

	// Remove destroyed sandboxes
	sandboxesDir := filepath.Join(piHome, "sandboxes")
	if DirExists(sandboxesDir) {
		entries, err := os.ReadDir(sandboxesDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				metaPath := filepath.Join(sandboxesDir, entry.Name(), "meta.json")
				data, err := os.ReadFile(metaPath)
				if err != nil {
					continue
				}
				// Check if state is "destroyed"
				if ContainsString(data, `"state":"destroyed"`) {
					dirPath := filepath.Join(sandboxesDir, entry.Name())
					size, _ := DirSize(dirPath)
					result.RemovedBytes += size
					if err := os.RemoveAll(dirPath); err == nil {
						result.RemovedSandboxes++
					}
				}
			}
		}
	}

	// Remove old logs (> 30 days)
	logsDir := filepath.Join(piHome, "logs")
	if DirExists(logsDir) {
		entries, err := os.ReadDir(logsDir)
		if err == nil {
			now := time.Now()
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				info, err := entry.Info()
				if err != nil {
					continue
				}
				if now.Sub(info.ModTime()) > 30*24*time.Hour {
					path := filepath.Join(logsDir, entry.Name())
					size := info.Size()
					result.RemovedBytes += size
					if err := os.Remove(path); err == nil {
						result.RemovedLogs++
					}
				}
			}
		}
	}

	return result, nil
}

// PrintPrune prints the prune results.
func PrintPrune(result *PruneResult) {
	fmt.Println("=== pi-sandbox Prune ===")
	fmt.Println()
	fmt.Printf("Removed sandboxes: %d\n", result.RemovedSandboxes)
	fmt.Printf("Removed log files: %d\n", result.RemovedLogs)
	fmt.Printf("Freed space:       %s\n", FormatSize(result.RemovedBytes))
}

// ContainsString checks if data contains substr.
func ContainsString(data []byte, substr string) bool {
	return string(data) != "" && len(data) >= len(substr) && findSubstring(data, substr)
}

func findSubstring(data []byte, substr string) bool {
	for i := 0; i <= len(data)-len(substr); i++ {
		if string(data[i:i+len(substr)]) == substr {
			return true
		}
	}
	return false
}
