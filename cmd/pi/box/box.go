package box

import (
	"fmt"
	"os"

	"github.com/pi-sandbox/pi/cmd/pi/cli"
	"github.com/spf13/cobra"
)

var boxCmd = &cobra.Command{
	Use:   "box",
	Short: "Sandbox lifecycle management",
	Long:  `Manage sandbox sessions: create, exec, clone, files, artifacts, etc.`,
}

// Command is exported for initialization.
var Command = boxCmd

func init() {
	cli.AddCommand(boxCmd)
	boxCmd.AddCommand(createCmd, listCmd, inspectCmd, destroyCmd, cloneCmd, execCmd, shellCmd, filesCmd, diffCmd, patchCmd, artifactsCmd, snapshotCmd, logsCmd)
}

var createCmd = &cobra.Command{
	Use:   "create [name] [template]",
	Short: "Create a new sandbox session",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "stub: create — not yet implemented")
		os.Exit(1)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List sandbox sessions",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "stub: list — not yet implemented")
		os.Exit(1)
	},
}

var inspectCmd = &cobra.Command{
	Use:   "inspect <name>",
	Short: "Inspect a sandbox session",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "stub: inspect — not yet implemented")
		os.Exit(1)
	},
}

var destroyCmd = &cobra.Command{
	Use:   "destroy <name>",
	Short: "Destroy a sandbox session",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "stub: destroy — not yet implemented")
		os.Exit(1)
	},
}

var cloneCmd = &cobra.Command{
	Use:   "clone <name> <url>",
	Short: "Clone a repository into a sandbox",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "stub: clone — not yet implemented")
		os.Exit(1)
	},
}

var execCmd = &cobra.Command{
	Use:   "exec <name> -- <command>",
	Short: "Execute a command in a sandbox",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "stub: exec — not yet implemented")
		os.Exit(1)
	},
}

var shellCmd = &cobra.Command{
	Use:   "shell <name>",
	Short: "Open an interactive shell in a sandbox",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "stub: shell — not yet implemented")
		os.Exit(1)
	},
}

var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "File operations in a sandbox",
}

var diffCmd = &cobra.Command{
	Use:   "diff <name>",
	Short: "Show workspace diff",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "stub: diff — not yet implemented")
		os.Exit(1)
	},
}

var patchCmd = &cobra.Command{
	Use:   "patch <name>",
	Short: "Export workspace as patch",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "stub: patch — not yet implemented")
		os.Exit(1)
	},
}

var artifactsCmd = &cobra.Command{
	Use:   "artifacts",
	Short: "Artifact management",
}

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Snapshot management",
}

var logsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Show sandbox logs",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "stub: logs — not yet implemented")
		os.Exit(1)
	},
}

func init() {
	filesCmd.AddCommand(&cobra.Command{Use: "list", Short: "List files", Run: func(*cobra.Command, []string) { fmt.Fprintln(os.Stderr, "stub"); os.Exit(1) }})
	filesCmd.AddCommand(&cobra.Command{Use: "read", Short: "Read file", Run: func(*cobra.Command, []string) { fmt.Fprintln(os.Stderr, "stub"); os.Exit(1) }})
	filesCmd.AddCommand(&cobra.Command{Use: "write", Short: "Write file", Run: func(*cobra.Command, []string) { fmt.Fprintln(os.Stderr, "stub"); os.Exit(1) }})
	artifactsCmd.AddCommand(&cobra.Command{Use: "list", Short: "List artifacts", Run: func(*cobra.Command, []string) { fmt.Fprintln(os.Stderr, "stub"); os.Exit(1) }})
	artifactsCmd.AddCommand(&cobra.Command{Use: "pull", Short: "Pull artifacts", Run: func(*cobra.Command, []string) { fmt.Fprintln(os.Stderr, "stub"); os.Exit(1) }})
	artifactsCmd.AddCommand(&cobra.Command{Use: "pack", Short: "Pack artifacts", Run: func(*cobra.Command, []string) { fmt.Fprintln(os.Stderr, "stub"); os.Exit(1) }})
	snapshotCmd.AddCommand(&cobra.Command{Use: "create", Short: "Create snapshot", Run: func(*cobra.Command, []string) { fmt.Fprintln(os.Stderr, "stub"); os.Exit(1) }})
	snapshotCmd.AddCommand(&cobra.Command{Use: "list", Short: "List snapshots", Run: func(*cobra.Command, []string) { fmt.Fprintln(os.Stderr, "stub"); os.Exit(1) }})
	snapshotCmd.AddCommand(&cobra.Command{Use: "rollback", Short: "Rollback", Run: func(*cobra.Command, []string) { fmt.Fprintln(os.Stderr, "stub"); os.Exit(1) }})
	snapshotCmd.AddCommand(&cobra.Command{Use: "delete", Short: "Delete snapshot", Run: func(*cobra.Command, []string) { fmt.Fprintln(os.Stderr, "stub"); os.Exit(1) }})
}
