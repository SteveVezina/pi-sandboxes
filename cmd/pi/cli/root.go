package cli

import "github.com/spf13/cobra"

// ContextOverride captures the value of the global --context flag (F22).
// Empty means "use the active context from ~/.pi-box/contexts.yaml".
var ContextOverride string

// Root is the root command for the pi CLI.
var Root = &cobra.Command{
	Use:   "pi-box",
	Short: "PI Agent Sandbox Runtime CLI",
	Long: `pi-sandbox — Local-first sandboxes for coding agents.

Fast warm exec, tiny footprint, selectable isolation.`,
}

func init() {
	Root.PersistentFlags().StringVar(&ContextOverride, "context", "",
		"Override the active daemon context for this command (F22)")
}

// AddCommand adds a subcommand to the root.
func AddCommand(cmd *cobra.Command) {
	Root.AddCommand(cmd)
}
