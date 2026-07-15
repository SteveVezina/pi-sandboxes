// Package context implements the `pi-box context` CLI command group (F22).
package context

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pi-sandbox/pi/cmd/pi-box/cli"
	pictx "github.com/pi-sandbox/pi/pkg/context"
	"github.com/spf13/cobra"
)

var (
	contextStorePath string
	createTransport  string
	createAuthType   string
	createTokenEnv   string
	listJSON         bool
	inspectJSON      bool
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage CLI daemon contexts (local/remote)",
	Long:  `Create, switch, list, inspect and delete daemon contexts. See ADR-003.`,
}

// Command is exported for initialization.
var Command = contextCmd

func init() {
	cli.AddCommand(contextCmd)
	contextCmd.PersistentFlags().StringVar(&contextStorePath, "store", "",
		"Path to contexts.yaml (default: ~/.pi-box/contexts.yaml)")

	contextCmd.AddCommand(createCmd, useCmd, listCmd, inspectCmd, deleteCmd)

	createCmd.Flags().StringVar(&createTransport, "transport", "", "Transport: unix, http, ssh (auto-detected from target if omitted)")
	createCmd.Flags().StringVar(&createAuthType, "auth", "", "Auth type: none, bearer-token, ssh-agent (auto-detected from transport if omitted)")
	createCmd.Flags().StringVar(&createTokenEnv, "token-env", "", "Env var holding bearer token (required for http transport)")

	listCmd.Flags().BoolVar(&listJSON, "json", false, "JSON output")
	inspectCmd.Flags().BoolVar(&inspectJSON, "json", false, "JSON output")
}

func openStore() *pictx.Store {
	path := contextStorePath
	if path == "" {
		path = pictx.DefaultPath()
	}
	store, err := pictx.NewStore(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: open context store: %v\n", err)
		os.Exit(1)
	}
	return store
}

var createCmd = &cobra.Command{
	Use:   "create <name> <target>",
	Short: "Create a new context",
	Long:  "Create a remote daemon context. Target examples: ssh://gpu-box.local, https://daemon:7777",
	Args:  cobra.ExactArgs(2),
	Run: func(_ *cobra.Command, args []string) {
		name, target := args[0], args[1]
		transport := createTransport
		if transport == "" {
			transport = inferTransport(target)
		}
		auth := createAuthType
		if auth == "" {
			auth = inferAuth(transport)
		}
		ctx := pictx.Context{
			Name:      name,
			Target:    target,
			Transport: transport,
			Auth: pictx.AuthConfig{
				Type:     auth,
				TokenEnv: createTokenEnv,
			},
		}
		store := openStore()
		if err := store.Create(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created context %q (transport=%s, auth=%s)\n", name, transport, auth)
	},
}

var useCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch the active context",
	Args:  cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		store := openStore()
		if err := store.Use(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Switched to context %q\n", args[0])
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List contexts",
	Run: func(*cobra.Command, []string) {
		store := openStore()
		ctxs := store.List()
		active := store.ActiveName()
		if listJSON {
			out := map[string]interface{}{
				"active":   active,
				"contexts": ctxs,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(out)
			return
		}
		fmt.Printf("%-3s %-20s %-12s %-15s %s\n", "*", "NAME", "TRANSPORT", "AUTH", "TARGET")
		for _, c := range ctxs {
			marker := " "
			if c.Name == active {
				marker = "*"
			}
			fmt.Printf("%-3s %-20s %-12s %-15s %s\n", marker, c.Name, c.Transport, c.Auth.Type, c.Target)
		}
	},
}

var inspectCmd = &cobra.Command{
	Use:   "inspect <name>",
	Short: "Show context details",
	Args:  cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		store := openStore()
		ctx, err := store.Get(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if inspectJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(ctx)
			return
		}
		fmt.Printf("name:      %s\ntarget:    %s\ntransport: %s\nauth.type: %s\n",
			ctx.Name, ctx.Target, ctx.Transport, ctx.Auth.Type)
		if ctx.Auth.TokenEnv != "" {
			fmt.Printf("token_env: %s\n", ctx.Auth.TokenEnv)
		}
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a context",
	Args:  cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		store := openStore()
		if err := store.Delete(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Deleted context %q\n", args[0])
	},
}

func inferTransport(target string) string {
	switch {
	case starts(target, "unix://"), starts(target, "/"):
		return pictx.TransportUnix
	case starts(target, "http://"), starts(target, "https://"):
		return pictx.TransportHTTP
	case starts(target, "ssh://"):
		return pictx.TransportSSH
	}
	return pictx.TransportHTTP
}

func inferAuth(transport string) string {
	switch transport {
	case pictx.TransportHTTP:
		return pictx.AuthBearerToken
	case pictx.TransportSSH:
		return pictx.AuthSSHAgent
	default:
		return pictx.AuthNone
	}
}

func starts(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
