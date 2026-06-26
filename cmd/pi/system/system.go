package system

import (
	"fmt"

	"github.com/pi-sandbox/pi/cmd/pi/cli"
	"github.com/spf13/cobra"
)

var systemCmd = &cobra.Command{
	Use:   "system",
	Short: "Daemon and state management",
	Long:  `Manage the pi-sandbox daemon and local state.`,
}

// Command is exported for initialization.
var Command = systemCmd

func init() {
	cli.AddCommand(systemCmd)
	systemCmd.AddCommand(&cobra.Command{Use: "status", Short: "Show daemon status", Run: func(*cobra.Command, []string) { fmt.Println("stub: status") }})
	systemCmd.AddCommand(&cobra.Command{Use: "doctor", Short: "Validate config", Run: func(*cobra.Command, []string) { fmt.Println("stub: doctor") }})
	systemCmd.AddCommand(&cobra.Command{Use: "prune", Short: "Clean old state", Run: func(*cobra.Command, []string) { fmt.Println("stub: prune") }})
	systemCmd.AddCommand(&cobra.Command{Use: "disk-usage", Short: "Show storage", Run: func(*cobra.Command, []string) { fmt.Println("stub: disk-usage") }})
}
