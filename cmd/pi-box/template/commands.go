package template

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/pi-sandbox/pi/pkg/template"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var tmplDir string
var jsonFlag bool

// List returns the list command.
func List() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		Run: func(*cobra.Command, []string) {
			store := template.NewStore(tmplDir)
			if err := store.InstallDefaults(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			names, err := store.List()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			if jsonFlag {
				data, _ := json.Marshal(names)
				fmt.Println(string(data))
				return
			}

			if len(names) == 0 {
				fmt.Println("No templates found")
				return
			}

			for _, name := range names {
				fmt.Println(name)
			}
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}

// Inspect returns the inspect command.
func Inspect() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "inspect [name]",
		Short: "Inspect a template",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name = args[0]
			store := template.NewStore(tmplDir)
			if err := store.InstallDefaults(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			tmpl, err := store.Get(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			if jsonFlag {
				data, _ := json.Marshal(tmpl)
				fmt.Println(string(data))
				return
			}

			fmt.Printf("Name:    %s\n", tmpl.Name)
			if tmpl.Version != "" {
				fmt.Printf("Version: %s\n", tmpl.Version)
			}
			if tmpl.Summary != "" {
				fmt.Printf("Summary: %s\n", tmpl.Summary)
			}
			src := string(template.SourceBuiltin)
			if tmpl.Source != nil && tmpl.Source.Type != "" {
				src = string(tmpl.Source.Type)
				if tmpl.Source.ForkedFrom != "" {
					src += " (forked from " + tmpl.Source.ForkedFrom + ")"
				}
			}
			fmt.Printf("Source:  %s\n", src)
			if tmpl.Lineage != nil {
				fmt.Printf("Digest:  %s (generation %d)\n", tmpl.ContentDigest(), tmpl.Lineage.Generation)
			}
			fmt.Printf("Runtime: %s\n", tmpl.Runtime)
			fmt.Printf("Base:    %s\n", tmpl.Base)
			if tmpl.Compatibility != nil && len(tmpl.Compatibility.Runtimes) > 0 {
				fmt.Println("\nRuntime compatibility (declared):")
				for mode, state := range tmpl.Compatibility.Runtimes {
					fmt.Printf("  %s: %s\n", mode, state)
				}
			}
			fmt.Println("\nTools:")
			for _, tool := range tmpl.Tools {
				fmt.Printf("  - %s\n", tool)
			}
			fmt.Println("\nMounts:")
			for k, v := range tmpl.Mounts {
				fmt.Printf("  %s: %s\n", k, v)
			}
			if len(tmpl.Caches) > 0 {
				fmt.Println("\nCaches:")
				for k, v := range tmpl.Caches {
					fmt.Printf("  %s: %s\n", k, v)
				}
			}
			fmt.Printf("\nNetwork: %s\n", tmpl.Network)
			if len(tmpl.NetworkDomains) > 0 {
				fmt.Println("Network domains (seed):")
				for _, d := range tmpl.NetworkDomains {
					fmt.Printf("  - %s\n", d)
				}
			}
			if tmpl.Resources != nil {
				fmt.Printf("Resources: cpu=%s memory=%s disk=%s\n",
					tmpl.Resources.CPU, tmpl.Resources.Memory, tmpl.Resources.Disk)
			}
			if problems := tmpl.Validate(); len(problems) > 0 {
				fmt.Printf("\n⚠ %d validation problem(s) — run `pi-box template validate %s`\n", len(problems), tmpl.Name)
			}
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}

// Build returns the build command.
func Build() *cobra.Command {
	var socketPath string
	cmd := &cobra.Command{
		Use:   "build [name]",
		Short: "Build a template OCI image",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			// Call daemon API to build template
			url := fmt.Sprintf("http://localhost/v1/templates/%s/build", name)
			if socketPath != "" {
				// Use curl with unix socket
				cmd := fmt.Sprintf("curl -s -X POST -H 'Content-Type: application/json' --unix-socket %s %s", socketPath, url)
				output, err := exec.Command("sh", "-c", cmd).Output()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error building template: %v\n", err)
					os.Exit(1)
				}
				fmt.Println(string(output))
				return
			}
			// Fallback: HTTP request
			resp, err := http.Post(url, "application/json", nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error building template: %v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
		},
	}
	cmd.Flags().StringVar(&socketPath, "socket", "", "Daemon socket path")
	return cmd
}

// Update returns the update command.
func Update() *cobra.Command {
	var socketPath string
	cmd := &cobra.Command{
		Use:   "update [name]",
		Short: "Update a template from remote",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			url := fmt.Sprintf("http://localhost/v1/templates/%s/update", name)
			if socketPath != "" {
				cmd := fmt.Sprintf("curl -s -X POST -H 'Content-Type: application/json' --unix-socket %s %s", socketPath, url)
				output, err := exec.Command("sh", "-c", cmd).Output()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error updating template: %v\n", err)
					os.Exit(1)
				}
				fmt.Println(string(output))
				return
			}
			resp, err := http.Post(url, "application/json", nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error updating template: %v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
		},
	}
	cmd.Flags().StringVar(&socketPath, "socket", "", "Daemon socket path")
	return cmd
}

// Prune returns the prune command.
func Prune() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove unused templates",
		Run: func(*cobra.Command, []string) {
			store := template.NewStore(tmplDir)
			names, err := store.List()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			if jsonFlag {
				data, _ := json.Marshal(names)
				fmt.Println(string(data))
				return
			}

			fmt.Printf("Pruning %d template(s)...\n", len(names))
			for _, name := range names {
				dir := filepath.Join(store.Dir(), name)
				if err := os.RemoveAll(dir); err != nil {
					fmt.Fprintf(os.Stderr, "  Failed to remove %s: %v\n", name, err)
					continue
				}
				fmt.Printf("  Removed: %s\n", name)
			}
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}

// Fork returns the `template fork <source> <new-name>` command.
func Fork() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fork <source> <new-name>",
		Short: "Create an editable local template from an existing one",
		Args:  cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			source, newName := args[0], args[1]
			store := template.NewStore(tmplDir)
			if err := store.InstallDefaults(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			forked, err := store.Fork(source, newName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Forked %s -> %s (generation %d)\n", source, newName, forked.Lineage.Generation)
		},
	}
	return cmd
}

// Validate returns the `template validate <path-or-name>` command.
func Validate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <path-or-name>",
		Short: "Validate a template definition",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			ref := args[0]
			var t *template.Template
			if data, err := os.ReadFile(ref); err == nil {
				var parsed template.Template
				if err := yaml.Unmarshal(data, &parsed); err != nil {
					fmt.Fprintf(os.Stderr, "Error: parse %s: %v\n", ref, err)
					os.Exit(1)
				}
				t = &parsed
			} else {
				store := template.NewStore(tmplDir)
				_ = store.InstallDefaults()
				loaded, err := store.Get(ref)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}
				t = loaded
			}

			problems := t.Validate()
			if jsonFlag {
				data, _ := json.Marshal(map[string]any{"valid": len(problems) == 0, "problems": problems})
				fmt.Println(string(data))
			} else if len(problems) == 0 {
				fmt.Println("OK")
			} else {
				fmt.Printf("%d problem(s):\n", len(problems))
				for _, p := range problems {
					fmt.Printf("  - %s\n", p)
				}
			}
			if len(problems) > 0 {
				os.Exit(1)
			}
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}

// History returns the `template history <name>` command.
func History() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history <name>",
		Short: "Show local revision history for a template",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			store := template.NewStore(tmplDir)
			_ = store.InstallDefaults()
			revs, err := store.History(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if jsonFlag {
				data, _ := json.Marshal(revs)
				fmt.Println(string(data))
				return
			}
			if len(revs) == 0 {
				fmt.Println("no revisions")
				return
			}
			for _, r := range revs {
				fmt.Printf("rev %-3d  %s  %s\n", r.N, r.Time, r.Digest)
			}
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}

// Diff returns the `template diff <left> <right>` command. Each ref is a
// template name or "name@N" for a revision.
func Diff() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff <left> <right>",
		Short: "Diff two templates or revisions (name or name@N)",
		Args:  cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			store := template.NewStore(tmplDir)
			_ = store.InstallDefaults()
			l, err := store.ResolveRef(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			r, err := store.ResolveRef(args[1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Print(template.Diff(l, r))
		},
	}
	return cmd
}

// Rollback returns the `template rollback <name> <revision>` command.
func Rollback() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback <name> <revision>",
		Short: "Restore a template to a prior local revision",
		Args:  cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			n, err := strconv.Atoi(args[1])
			if err != nil || n < 1 {
				fmt.Fprintln(os.Stderr, "Error: revision must be a positive integer")
				os.Exit(1)
			}
			store := template.NewStore(tmplDir)
			_ = store.InstallDefaults()
			restored, err := store.Rollback(args[0], n)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Rolled %s back to revision %d (now generation %d)\n",
				args[0], n, restored.Lineage.Generation)
		},
	}
	return cmd
}

// Export returns the `template export <name> -o <file>` command.
func Export() *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "export <name>",
		Short: "Export a template as a portable OCI bundle",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			if out == "" {
				out = args[0] + ".oci.tar"
			}
			store := template.NewStore(tmplDir)
			_ = store.InstallDefaults()
			t, err := store.Get(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			bundle, err := template.ExportBundle(t)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := os.WriteFile(out, bundle, 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Wrote %s (%d bytes)\n", out, len(bundle))
		},
	}
	cmd.Flags().StringVarP(&out, "output", "o", "", "Output path (default: <name>.oci.tar)")
	return cmd
}

// Import returns the `template import <file> [--name <n>]` command.
func Import() *cobra.Command {
	var newName string
	cmd := &cobra.Command{
		Use:   "import <bundle.tar>",
		Short: "Import a portable template bundle (installed as source: imported)",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			data, err := os.ReadFile(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			store := template.NewStore(tmplDir)
			_ = store.InstallDefaults()
			t, err := store.Import(data, newName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Imported %s (%s)\n", t.Name, t.ContentDigest())
		},
	}
	cmd.Flags().StringVar(&newName, "name", "", "Install under a different name")
	return cmd
}
