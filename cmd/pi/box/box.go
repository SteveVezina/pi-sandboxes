package box

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/pi-sandbox/pi/cmd/pi/cli"
	pictx "github.com/pi-sandbox/pi/pkg/context"
	"github.com/pi-sandbox/pi/pkg/remote"
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

	// Files subcommands
	filesCmd.AddCommand(filesListCmd, filesReadCmd, filesWriteCmd)

	// Artifacts subcommands
	artifactsCmd.AddCommand(artifactsListCmd, artifactsPullCmd, artifactsPackCmd)

	// Snapshot subcommands
	snapshotCmd.AddCommand(snapshotCreateCmd, snapshotListCmd, snapshotRollbackCmd, snapshotDeleteCmd)

	// Set up flags
	artifactsPackCmd.Flags().StringP("output", "o", "/tmp/artifacts.tar.gz", "Output path")
}

// getSocketPath returns the daemon socket path.
func getSocketPath() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = "."
	}
	return fmt.Sprintf("%s/.pi/sandboxd.sock", home)
}

// resolveContext returns the context to use, honoring --context override (F22).
func resolveContext() (pictx.Context, error) {
	store, err := pictx.NewStore(pictx.DefaultPath())
	if err != nil {
		return pictx.Context{}, err
	}
	return store.Resolve(cli.ContextOverride)
}

// callAPI makes an HTTP call to the daemon, routing via the active context.
// For the local unix context (the default) it preserves the curl-based path;
// for remote contexts it uses pkg/remote with proper auth.
func callAPI(method, endpoint string, body io.Reader) (map[string]interface{}, error) {
	ctx, err := resolveContext()
	if err != nil {
		return nil, fmt.Errorf("context: %w", err)
	}
	if ctx.Transport == pictx.TransportUnix && cli.ContextOverride == "" {
		return callAPIUnix(method, endpoint, body)
	}
	return callAPIRemote(ctx, method, endpoint, body)
}

func callAPIUnix(method, endpoint string, body io.Reader) (map[string]interface{}, error) {
	args := []string{"-s", "-X", method, "-H", "Content-Type: application/json",
		"--unix-socket", getSocketPath(), endpoint}
	if body != nil {
		args = append(args, "-d", "@-")
	}
	cmd := exec.Command("curl", args...)
	if body != nil {
		cmd.Stdin = body
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if len(output) == 0 {
		return result, nil
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func callAPIRemote(ctx pictx.Context, method, endpoint string, body io.Reader) (map[string]interface{}, error) {
	client, err := remote.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("remote client: %w", err)
	}
	resp, err := client.Do(method, endpoint, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read remote response: %w", err)
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("remote auth failed (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("remote API error (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var result map[string]interface{}
	if len(data) == 0 {
		return result, nil
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode remote response: %w", err)
	}
	return result, nil
}

// callAPIList makes an HTTP call and returns a JSON array.
func callAPIList(method, endpoint string) ([]interface{}, error) {
	ctx, err := resolveContext()
	if err != nil {
		return nil, fmt.Errorf("context: %w", err)
	}
	var data []byte
	if ctx.Transport == pictx.TransportUnix && cli.ContextOverride == "" {
		data, err = curlList(method, endpoint)
	} else {
		data, err = remoteList(ctx, method, endpoint)
	}
	if err != nil {
		return nil, err
	}
	var result []interface{}
	if len(data) == 0 {
		return result, nil
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode list response: %w", err)
	}
	return result, nil
}

func curlList(method, endpoint string) ([]byte, error) {
	args := []string{"-s", "-X", method, "-H", "Content-Type: application/json",
		"--unix-socket", getSocketPath(), endpoint}
	return exec.Command("curl", args...).Output()
}

func remoteList(ctx pictx.Context, method, endpoint string) ([]byte, error) {
	client, err := remote.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("remote client: %w", err)
	}
	resp, err := client.Do(method, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("remote auth failed (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("remote API error (HTTP %d)", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// createCmd creates a new sandbox session.
var createCmd = &cobra.Command{
	Use:   "create [name] [template]",
	Short: "Create a new sandbox session",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := "default"
		template := "base"
		mode := "fast"
		if len(args) > 0 {
			name = args[0]
		}
		if len(args) > 1 {
			template = args[1]
		}
		if m, _ := cmd.Flags().GetString("mode"); m != "" {
			mode = m
		}
		payload := map[string]interface{}{
			"name":     name,
			"template": template,
			"mode":     mode,
		}
		data, _ := json.Marshal(payload)
		result, err := callAPI("POST", "/v1/sandboxes", bytes.NewReader(data))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to create sandbox: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created sandbox: %v\n", result["id"])
	},
}

func init() {
	createCmd.Flags().StringP("mode", "m", "fast", "Runtime mode: fast, compat, secure")
}

// listCmd lists sandbox sessions.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List sandbox sessions",
	Run: func(cmd *cobra.Command, args []string) {
		sandboxes, err := callAPIList("GET", "/v1/sandboxes")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to list sandboxes: %v\n", err)
			os.Exit(1)
		}
		data, _ := json.MarshalIndent(sandboxes, "", "  ")
		fmt.Println(string(data))
	},
}

// inspectCmd inspects a sandbox session.
var inspectCmd = &cobra.Command{
	Use:   "inspect <name>",
	Short: "Inspect a sandbox session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		result, err := callAPI("GET", "/v1/sandboxes/"+args[0], nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to inspect sandbox: %v\n", err)
			os.Exit(1)
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	},
}

// destroyCmd destroys a sandbox session.
var destroyCmd = &cobra.Command{
	Use:   "destroy <name>",
	Short: "Destroy a sandbox session",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if args[0] == "--all" {
			sandboxes, err := callAPIList("GET", "/v1/sandboxes")
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: listing sandboxes failed: %v\n", err)
				os.Exit(1)
			}
			count := 0
			for _, sb := range sandboxes {
				sbMap, ok := sb.(map[string]interface{})
				if !ok {
					continue
				}
				id, _ := sbMap["id"].(string)
				if id == "" {
					id, _ = sbMap["name"].(string)
				}
				if id == "" {
					continue
				}
				_, err := callAPI("DELETE", "/v1/sandboxes/"+id, nil)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to destroy %s: %v\n", id, err)
					continue
				}
				fmt.Printf("Destroyed sandbox: %s\n", id)
				count++
			}
			fmt.Printf("Destroyed %d sandbox(es)\n", count)
			return
		}
		_, err := callAPI("DELETE", "/v1/sandboxes/"+args[0], nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to destroy sandbox: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Destroyed sandbox: %s\n", args[0])
	},
}

// cloneCmd clones a repository into a sandbox.
var cloneCmd = &cobra.Command{
	Use:   "clone <name> <url>",
	Short: "Clone a repository into a sandbox",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		payload := map[string]interface{}{"url": args[1]}
		data, _ := json.Marshal(payload)
		result, err := callAPI("POST", "/v1/sandboxes/"+args[0]+"/clone", bytes.NewReader(data))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: clone failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Cloned %s into sandbox %s\n", args[1], result["id"])
	},
}

// execCmd executes a command in a sandbox.
var execCmd = &cobra.Command{
	Use:   "exec <name> -- <command>",
	Short: "Execute a command in a sandbox",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cmdIdx := -1
		for i, a := range os.Args {
			if a == "--" && i < len(os.Args)-1 {
				cmdIdx = i
				break
			}
		}
		if cmdIdx == -1 {
			fmt.Fprintln(os.Stderr, "error: command required after --")
			os.Exit(1)
		}
		command := strings.Join(os.Args[cmdIdx+1:], " ")
		timeout := int64(120)
		if t, _ := cmd.Flags().GetInt64("timeout"); t > 0 {
			timeout = t
		}
		cwd := "/workspace"
		if c, _ := cmd.Flags().GetString("cwd"); c != "" {
			cwd = c
		}
		payload := map[string]interface{}{
			"command":        command,
			"cwd":            cwd,
			"timeoutMs":      timeout * 1000,
			"maxOutputBytes": 8388608,
		}
		data, _ := json.Marshal(payload)
		result, err := callAPI("POST", "/v1/sandboxes/"+args[0]+"/exec", bytes.NewReader(data))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: exec failed: %v\n", err)
			os.Exit(1)
		}
		if stdout, ok := result["stdout"].(string); ok && stdout != "" {
			fmt.Print(stdout)
		}
		if stderr, ok := result["stderr"].(string); ok && stderr != "" {
			fmt.Fprint(os.Stderr, stderr)
		}
		if jsonFlag, _ := cmd.Flags().GetBool("json"); jsonFlag {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
		}
	},
}

func init() {
	execCmd.Flags().StringP("cwd", "c", "/workspace", "Working directory")
	execCmd.Flags().Int64P("timeout", "t", 120, "Timeout in seconds")
	execCmd.Flags().BoolP("json", "j", false, "Output as JSON")
}

// shellCmd opens an interactive shell in a sandbox.
var shellCmd = &cobra.Command{
	Use:   "shell <name>",
	Short: "Open an interactive shell in a sandbox",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)
		sandboxID := args[0]
		fmt.Fprintf(os.Stderr, "Interactive shell for sandbox %s. Type 'exit' to quit.\n", sandboxID)
		for {
			fmt.Printf("%s> ", sandboxID)
			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
				break
			}
			line = strings.TrimSpace(line)
			if line == "exit" || line == "quit" {
				break
			}
			if line == "" {
				continue
			}
			result, err := callAPI("POST", "/v1/sandboxes/"+sandboxID+"/exec",
				bytes.NewReader([]byte(`{"command":"`+escapeJSON(line)+`","cwd":"/workspace"}`)))
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				continue
			}
			if stdout, ok := result["stdout"].(string); ok && stdout != "" {
				fmt.Print(stdout)
			}
			if stderr, ok := result["stderr"].(string); ok && stderr != "" {
				fmt.Fprint(os.Stderr, stderr)
			}
		}
	},
}

// escapeJSON escapes a string for JSON embedding.
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// filesCmd handles file operations.
var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "File operations in a sandbox",
}

// filesListCmd lists files in workspace.
var filesListCmd = &cobra.Command{
	Use:   "list <name> [path]",
	Short: "List files in workspace",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := ""
		if len(args) > 1 {
			path = args[1]
		}
		result, err := callAPI("GET", "/v1/sandboxes/"+args[0]+"/files/read?path="+path, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: list files failed: %v\n", err)
			os.Exit(1)
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	},
}

// filesReadCmd reads a file from workspace.
var filesReadCmd = &cobra.Command{
	Use:   "read <name> <path>",
	Short: "Read a file from workspace",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		result, err := callAPI("GET", "/v1/sandboxes/"+args[0]+"/files/read?path="+args[1], nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: read file failed: %v\n", err)
			os.Exit(1)
		}
		if content, ok := result["content"].(string); ok {
			fmt.Print(content)
		}
	},
}

// filesWriteCmd writes a file to workspace.
var filesWriteCmd = &cobra.Command{
	Use:   "write <name> <path>",
	Short: "Write a file to workspace",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if !isPipe() {
			fmt.Fprintln(os.Stderr, "error: pipe content via stdin: cat file | pi box files write name path")
			os.Exit(1)
		}
		reader := bufio.NewReader(os.Stdin)
		data, _ := io.ReadAll(reader)
		payload := map[string]interface{}{
			"path":    args[1],
			"content": string(data),
		}
		dataJSON, _ := json.Marshal(payload)
		result, err := callAPI("POST", "/v1/sandboxes/"+args[0]+"/files/write", bytes.NewReader(dataJSON))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: write file failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Wrote %v bytes to %s\n", result["bytes"], args[1])
	},
}

// diffCmd shows workspace diff.
var diffCmd = &cobra.Command{
	Use:   "diff <name>",
	Short: "Show workspace diff",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		result, err := callAPI("GET", "/v1/sandboxes/"+args[0]+"/diff", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: diff failed: %v\n", err)
			os.Exit(1)
		}
		if diff, ok := result["diff"].(string); ok {
			fmt.Print(diff)
		}
	},
}

// patchCmd exports workspace as patch.
var patchCmd = &cobra.Command{
	Use:   "patch <name>",
	Short: "Export workspace as patch",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		result, err := callAPI("GET", "/v1/sandboxes/"+args[0]+"/patch", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: patch failed: %v\n", err)
			os.Exit(1)
		}
		if patch, ok := result["patch"].(string); ok {
			fmt.Print(patch)
		}
	},
}

// artifactsCmd handles artifact management.
var artifactsCmd = &cobra.Command{
	Use:   "artifacts",
	Short: "Artifact management",
}

// artifactsListCmd lists artifacts.
var artifactsListCmd = &cobra.Command{
	Use:   "list <name>",
	Short: "List artifacts",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		result, err := callAPI("GET", "/v1/sandboxes/"+args[0]+"/artifacts/list", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: list artifacts failed: %v\n", err)
			os.Exit(1)
		}
		files, _ := result["files"].([]interface{})
		for _, f := range files {
			fileMap, _ := f.(map[string]interface{})
			fmt.Printf("  %s (%d bytes)\n", fileMap["path"], fileMap["size"])
		}
	},
}

// artifactsPullCmd pulls artifacts to host.
var artifactsPullCmd = &cobra.Command{
	Use:   "pull <name> <dest>",
	Short: "Pull artifacts to host",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		payload := map[string]interface{}{"destination": args[1]}
		data, _ := json.Marshal(payload)
		_, err := callAPI("POST", "/v1/sandboxes/"+args[0]+"/artifacts/pull", bytes.NewReader(data))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: pull artifacts failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Pulled artifacts to %s\n", args[1])
	},
}

// artifactsPackCmd packs artifacts into archive.
var artifactsPackCmd = &cobra.Command{
	Use:   "pack <name>",
	Short: "Pack artifacts into archive",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		output := "/tmp/artifacts.tar.gz"
		if o, _ := cmd.Flags().GetString("output"); o != "" {
			output = o
		}
		payload := map[string]interface{}{"output": output}
		data, _ := json.Marshal(payload)
		result, err := callAPI("POST", "/v1/sandboxes/"+args[0]+"/artifacts/pack", bytes.NewReader(data))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: pack artifacts failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Packed artifacts to %s (%v bytes)\n", output, result["bytes"])
	},
}

// snapshotCmd handles snapshot management.
var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Snapshot management",
}

// snapshotCreateCmd creates a snapshot.
var snapshotCreateCmd = &cobra.Command{
	Use:   "create <name> <snapshot>",
	Short: "Create a snapshot",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		payload := map[string]interface{}{"name": args[1]}
		data, _ := json.Marshal(payload)
		_, err := callAPI("POST", "/v1/sandboxes/"+args[0]+"/snapshot/create", bytes.NewReader(data))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: create snapshot failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created snapshot: %s\n", args[1])
	},
}

// snapshotListCmd lists snapshots.
var snapshotListCmd = &cobra.Command{
	Use:   "list <name>",
	Short: "List snapshots",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		result, err := callAPI("GET", "/v1/sandboxes/"+args[0]+"/snapshot/list", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: list snapshots failed: %v\n", err)
			os.Exit(1)
		}
		snapshots, _ := result["snapshots"].([]interface{})
		for _, s := range snapshots {
			snapMap, _ := s.(map[string]interface{})
			fmt.Printf("  %s (%d bytes, %s)\n", snapMap["name"], snapMap["sizeBytes"], snapMap["method"])
		}
	},
}

// snapshotRollbackCmd rolls back to a snapshot.
var snapshotRollbackCmd = &cobra.Command{
	Use:   "rollback <name> <snapshot>",
	Short: "Rollback to snapshot",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		payload := map[string]interface{}{"name": args[1]}
		data, _ := json.Marshal(payload)
		_, err := callAPI("POST", "/v1/sandboxes/"+args[0]+"/snapshot/rollback", bytes.NewReader(data))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: rollback failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Rolled back to snapshot: %s\n", args[1])
	},
}

// snapshotDeleteCmd deletes a snapshot.
var snapshotDeleteCmd = &cobra.Command{
	Use:   "delete <name> <snapshot>",
	Short: "Delete a snapshot",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		payload := map[string]interface{}{"name": args[1]}
		data, _ := json.Marshal(payload)
		_, err := callAPI("POST", "/v1/sandboxes/"+args[0]+"/snapshot/delete", bytes.NewReader(data))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: delete snapshot failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Deleted snapshot: %s\n", args[1])
	},
}

// logsCmd shows sandbox logs.
var logsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Show sandbox logs",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		result, err := callAPI("GET", "/v1/sandboxes/"+args[0]+"/logs", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: logs failed: %v\n", err)
			os.Exit(1)
		}
		entries, _ := result["entries"].([]interface{})
		for _, e := range entries {
			entryMap, _ := e.(map[string]interface{})
			seq := entryMap["sequence"]
			cmd := entryMap["command"]
			exitCode := entryMap["exitCode"]
			fmt.Printf("[%v] %s (exit: %v)\n", seq, cmd, exitCode)
		}
	},
}

// isPipe checks if stdin is a pipe (not a terminal).
func isPipe() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeNamedPipe) != 0
}
