// Package agent provides the PI coding agent adapter for sandbox operations.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ToolCall represents a tool call from the PI agent.
type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of a tool call.
type ToolResult struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// FormatResult returns a human-readable string representation of a ToolResult.
func FormatResult(r *ToolResult) string {
	if r == nil {
		return "<nil>"
	}
	if !r.Success {
		return fmt.Sprintf("error: %s", r.Error)
	}
	data, _ := json.Marshal(r.Data)
	return string(data)
}

// Adapter connects the PI coding agent to the sandbox runtime.
type Adapter struct {
	baseURL string
	timeout time.Duration
	client  *http.Client
}

// NewAdapter creates a new PI agent adapter.
func NewAdapter(baseURL string) *Adapter {
	return &Adapter{
		baseURL: baseURL,
		timeout: 30 * time.Second,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExecuteTool executes a tool call from the PI agent.
func (a *Adapter) ExecuteTool(ctx context.Context, tool ToolCall) (*ToolResult, error) {
	switch tool.Name {
	case "sandbox.create":
		return a.execCreate(tool.Arguments)
	case "sandbox.exec":
		return a.execExec(tool.Arguments)
	case "sandbox.list":
		return a.execList(tool.Arguments)
	case "sandbox.destroy":
		return a.execDestroy(tool.Arguments)
	case "sandbox.clone":
		return a.execClone(tool.Arguments)
	case "sandbox.diff":
		return a.execDiff(tool.Arguments)
	case "sandbox.patch":
		return a.execPatch(tool.Arguments)
	case "sandbox.files.read":
		return a.execFilesRead(tool.Arguments)
	case "sandbox.files.write":
		return a.execFilesWrite(tool.Arguments)
	case "sandbox.artifacts.list":
		return a.execArtifactsList(tool.Arguments)
	case "sandbox.artifacts.pull":
		return a.execArtifactsPull(tool.Arguments)
	case "sandbox.snapshot.create":
		return a.execSnapshotCreate(tool.Arguments)
	case "sandbox.snapshot.rollback":
		return a.execSnapshotRollback(tool.Arguments)
	case "sandbox.logs":
		return a.execLogs(tool.Arguments)
	default:
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("unknown tool: %s", tool.Name),
		}, nil
	}
}

func (a *Adapter) execCreate(args map[string]interface{}) (*ToolResult, error) {
	name, _ := args["name"].(string)
	template, _ := args["template"].(string)
	mode, _ := args["mode"].(string)
	if mode == "" {
		mode = "fast"
	}
	if template == "" {
		template = "base"
	}

	body := map[string]interface{}{
		"name":     name,
		"template": template,
		"mode":     mode,
	}

	respBody, err := a.doJSON("POST", "/v1/sandboxes", body)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"sandbox_id": result.ID, "status": "created"},
	}, nil
}

func (a *Adapter) execExec(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	command, _ := args["command"].(string)
	timeoutMs, _ := args["timeout_ms"].(float64)
	if timeoutMs == 0 {
		timeoutMs = 120000
	}

	body := map[string]interface{}{
		"command":   command,
		"timeoutMs": int(timeoutMs),
	}

	respBody, err := a.doJSON("POST", fmt.Sprintf("/v1/sandboxes/%s/exec", sandboxID), body)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	var result struct {
		ExitCode   int    `json:"exitCode"`
		DurationMs int64  `json:"durationMs"`
		Stdout     string `json:"stdout"`
		Stderr     string `json:"stderr"`
		TimedOut   bool   `json:"timedOut"`
		Truncated  bool   `json:"truncated"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	return &ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"exit_code":   result.ExitCode,
			"stdout":      result.Stdout,
			"stderr":      result.Stderr,
			"duration_ms": result.DurationMs,
			"timed_out":   result.TimedOut,
			"truncated":   result.Truncated,
		},
	}, nil
}

func (a *Adapter) execList(args map[string]interface{}) (*ToolResult, error) {
	respBody, err := a.doJSON("GET", "/v1/sandboxes", nil)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	var sandboxes []map[string]interface{}
	if err := json.Unmarshal(respBody, &sandboxes); err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	return &ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"sandboxes": sandboxes,
			"count":     len(sandboxes),
		},
	}, nil
}

func (a *Adapter) execDestroy(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	_, err := a.doJSON("DELETE", fmt.Sprintf("/v1/sandboxes/%s", sandboxID), nil)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"sandbox_id": sandboxID, "status": "destroyed"},
	}, nil
}

func (a *Adapter) execClone(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	url, _ := args["url"].(string)

	body := map[string]interface{}{"url": url}
	respBody, err := a.doJSON("POST", fmt.Sprintf("/v1/sandboxes/%s/clone", sandboxID), body)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"sandbox_id": result.ID, "status": "cloned"},
	}, nil
}

func (a *Adapter) execDiff(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	respBody, err := a.doJSON("GET", fmt.Sprintf("/v1/sandboxes/%s/diff", sandboxID), nil)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	var result struct {
		Diff string `json:"diff"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"sandbox_id": sandboxID, "diff": result.Diff},
	}, nil
}

func (a *Adapter) execPatch(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	respBody, err := a.doJSON("GET", fmt.Sprintf("/v1/sandboxes/%s/patch", sandboxID), nil)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	var result struct {
		Patch string `json:"patch"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"sandbox_id": sandboxID, "patch": result.Patch},
	}, nil
}

func (a *Adapter) execFilesRead(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	path, _ := args["path"].(string)

	respBody, err := a.doJSON("GET", fmt.Sprintf("/v1/sandboxes/%s/files/read?path=%s", sandboxID, path), nil)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"sandbox_id": sandboxID, "path": path, "content": string(respBody)},
	}, nil
}

func (a *Adapter) execFilesWrite(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)

	body := map[string]interface{}{"path": path, "content": content}
	_, err := a.doJSON("POST", fmt.Sprintf("/v1/sandboxes/%s/files/write", sandboxID), body)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"sandbox_id": sandboxID, "path": path, "status": "written"},
	}, nil
}

func (a *Adapter) execArtifactsList(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)

	respBody, err := a.doJSON("GET", fmt.Sprintf("/v1/sandboxes/%s/artifacts/list", sandboxID), nil)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	var result struct {
		Files []string `json:"files"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"sandbox_id": sandboxID, "files": result.Files, "count": len(result.Files)},
	}, nil
}

func (a *Adapter) execArtifactsPull(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	destination, _ := args["destination"].(string)

	body := map[string]interface{}{"destination": destination}
	_, err := a.doJSON("POST", fmt.Sprintf("/v1/sandboxes/%s/artifacts/pull", sandboxID), body)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"sandbox_id": sandboxID, "destination": destination, "status": "pulled"},
	}, nil
}

func (a *Adapter) execSnapshotCreate(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	name, _ := args["name"].(string)

	body := map[string]interface{}{"name": name}
	_, err := a.doJSON("POST", fmt.Sprintf("/v1/sandboxes/%s/snapshot/create", sandboxID), body)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"sandbox_id": sandboxID, "name": name, "status": "created"},
	}, nil
}

func (a *Adapter) execSnapshotRollback(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	name, _ := args["name"].(string)

	body := map[string]interface{}{"name": name}
	_, err := a.doJSON("POST", fmt.Sprintf("/v1/sandboxes/%s/snapshot/rollback", sandboxID), body)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"sandbox_id": sandboxID, "name": name, "status": "rolled_back"},
	}, nil
}

func (a *Adapter) execLogs(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)

	respBody, err := a.doJSON("GET", fmt.Sprintf("/v1/sandboxes/%s/logs", sandboxID), nil)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"sandbox_id": sandboxID, "logs": string(respBody)},
	}, nil
}

// doJSON performs an HTTP request and returns the JSON response body.
func (a *Adapter) doJSON(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(data)
	}

	url := a.baseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return respBody, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
