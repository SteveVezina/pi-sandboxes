package template

import (
	"os"
	"path/filepath"

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

	// Subcommands
	templateCmd.AddCommand(List())
	templateCmd.AddCommand(Inspect())
	templateCmd.AddCommand(Build())
	templateCmd.AddCommand(Update())
	templateCmd.AddCommand(Prune())

	// Global flags
	templateCmd.PersistentFlags().StringVar(&tmplDir, "templates-dir", defaultTemplatesDir(), "Templates directory")
}

func defaultTemplatesDir() string {
	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".pi", "templates")
}
