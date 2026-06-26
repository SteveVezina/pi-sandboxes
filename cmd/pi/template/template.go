package template

import (
	"fmt"
	"os"

	"github.com/pi-sandbox/pi/cmd/pi/cli"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Template management",
	Long:  `Manage sandbox templates.`,
}

// Command is exported for initialization.
var Command = templateCmd

func init() {
	cli.AddCommand(templateCmd)
	templateCmd.AddCommand(&cobra.Command{Use: "list", Short: "List templates", Run: func(*cobra.Command, []string) { fmt.Fprintln(os.Stderr, "stub"); os.Exit(1) }})
	templateCmd.AddCommand(&cobra.Command{Use: "inspect", Short: "Inspect template", Run: func(*cobra.Command, []string) { fmt.Fprintln(os.Stderr, "stub"); os.Exit(1) }})
	templateCmd.AddCommand(&cobra.Command{Use: "build", Short: "Build template", Run: func(*cobra.Command, []string) { fmt.Fprintln(os.Stderr, "stub"); os.Exit(1) }})
	templateCmd.AddCommand(&cobra.Command{Use: "update", Short: "Update template", Run: func(*cobra.Command, []string) { fmt.Fprintln(os.Stderr, "stub"); os.Exit(1) }})
	templateCmd.AddCommand(&cobra.Command{Use: "prune", Short: "Prune templates", Run: func(*cobra.Command, []string) { fmt.Fprintln(os.Stderr, "stub"); os.Exit(1) }})
}
