package cli

import "github.com/spf13/cobra"

// Root is the root command for the pi CLI.
var Root = &cobra.Command{
	Use:   "pi",
	Short: "PI Agent Sandbox Runtime CLI",
	Long: `pi-sandbox — Local-first sandboxes for coding agents.

Fast warm exec, tiny footprint, selectable isolation.`,
}

// AddCommand adds a subcommand to the root.
func AddCommand(cmd *cobra.Command) {
	Root.AddCommand(cmd)
}
