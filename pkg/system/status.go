package system

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// StatusInfo holds the system status information.
type StatusInfo struct {
	DaemonConnected bool
	ActiveSandboxes int
	TotalSandboxes  int
	PiHomeExists    bool
	PiHomePath      string
}

// GetStatus collects system status information.
func GetStatus(socketPath string) (*StatusInfo, error) {
	info := &StatusInfo{}

	// Check daemon connection
	info.DaemonConnected = checkDaemonConnected(socketPath)

	// Check Pi Box home
	info.PiHomePath = PiHome()
	info.PiHomeExists = DirExists(info.PiHomePath)

	// Count sandboxes
	if info.PiHomeExists {
		entries, err := os.ReadDir(info.PiHomePath)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					metaPath := filepath.Join(info.PiHomePath, entry.Name(), "meta.json")
					if _, err := os.Stat(metaPath); err == nil {
						store := session.NewStore(info.PiHomePath)
						meta, err := store.Get(entry.Name())
						if err == nil {
							info.TotalSandboxes++
							if meta.State == session.StateWarm || meta.State == session.StateExecuting {
								info.ActiveSandboxes++
							}
						}
					}
				}
			}
		}
	}

	return info, nil
}

// PrintStatus prints the status to stdout.
func PrintStatus(info *StatusInfo) {
	fmt.Println("=== pi-sandbox System Status ===")
	fmt.Println()

	// Daemon
	daemonStatus := "disconnected"
	if info.DaemonConnected {
		daemonStatus = "connected"
	}
	fmt.Printf("Daemon:     %s\n", daemonStatus)

	// Sandboxes
	fmt.Printf("Sandboxes:  %d total, %d active\n", info.TotalSandboxes, info.ActiveSandboxes)

	// Pi Box home
	if info.PiHomeExists {
		fmt.Printf("Pi Box Home:    %s (exists)\n", info.PiHomePath)
	} else {
		fmt.Printf("Pi Box Home:    %s (not found)\n", info.PiHomePath)
	}
}

func checkDaemonConnected(socketPath string) bool {
	conn, err := net.DialTimeout("unix", socketPath, 100*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
