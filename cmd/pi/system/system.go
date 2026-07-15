package system

import (
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
	systemCmd.AddCommand(StatusCmd())
	systemCmd.AddCommand(DoctorCmd())
	systemCmd.AddCommand(PruneCmd())
	systemCmd.AddCommand(DiskUsageCmd())
	systemCmd.AddCommand(RuntimesCmd())
}
