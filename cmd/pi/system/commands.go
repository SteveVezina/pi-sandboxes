package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pi-sandbox/pi/pkg/system"
	"github.com/spf13/cobra"
)

var socketPath string

// StatusCmd returns the status command.
func StatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show daemon and sandbox status",
		Run: func(*cobra.Command, []string) {
			info, err := system.GetStatus(socketPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			system.PrintStatus(info)
		},
	}
	cmd.Flags().StringVar(&socketPath, "socket", defaultSocketPath(), "Daemon socket path")
	return cmd
}

// DoctorCmd returns the doctor command.
func DoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Validate configuration and check for issues",
		Run: func(*cobra.Command, []string) {
			result := system.RunDoctor()
			system.PrintDoctor(result)
		},
	}
}

// PruneCmd returns the prune command.
func PruneCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove old sandbox state and logs",
		Run: func(*cobra.Command, []string) {
			result, err := system.RunPrune(!yes)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			system.PrintPrune(result)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation")
	return cmd
}

// DiskUsageCmd returns the disk-usage command.
func DiskUsageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disk-usage",
		Short: "Show storage breakdown by category",
		Run: func(*cobra.Command, []string) {
			info, err := system.GetDiskUsage()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			system.PrintDiskUsage(info)
		},
	}
	return cmd
}

func defaultSocketPath() string {
	return filepath.Join(system.PiHome(), "sandboxd.sock")
}

// RuntimesCmd returns the runtimes command.
func RuntimesCmd() *cobra.Command {
	var jsonFlag bool
	cmd := &cobra.Command{
		Use:   "runtimes",
		Short: "Show available runtime backends and their status",
		Run: func(*cobra.Command, []string) {
			info, err := system.GetRuntimes(socketPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if jsonFlag {
				data, _ := json.MarshalIndent(info, "", "  ")
				fmt.Println(string(data))
				return
			}
			system.PrintRuntimes(info)
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}
