package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ToolCall represents a tool call from the PI agent.
type ToolCall struct {
	Name      string            `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of a tool call.
type ToolResult struct {
	Success bool              `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string            `json:"error,omitempty"`
}

// Adapter connects the PI coding agent to the sandbox runtime.
type Adapter struct {
	baseURL string
	timeout time.Duration
}

// NewAdapter creates a new PI agent adapter.
func NewAdapter(baseURL string) *Adapter {
	return &Adapter{
		baseURL: baseURL,
		timeout: 30 * time.Second,
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
	default:
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("unknown tool: %s", tool.Name),
		}, nil
	}
}

func (a *Adapter) execCreate(args map[string]interface{}) (*ToolResult, error) {
	name, _ := args["name"].(string)
	mode, _ := args["mode"].(string)
	if mode == "" {
		mode = "fast"
	}
	_ = name
	_ = mode

	// In production: HTTP POST to daemon
	// For now, return stub
	return &ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"sandbox_id": "pi-stub",
		},
	}, nil
}

func (a *Adapter) execExec(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	command, _ := args["command"].(string)
	timeoutMs, _ := args["timeout_ms"].(float64)

	_ = sandboxID
	_ = command
	_ = timeoutMs

	return &ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"exit_code":   0,
			"stdout":      "",
			"stderr":      "",
			"duration_ms": int64(timeoutMs),
		},
	}, nil
}

func (a *Adapter) execList(args map[string]interface{}) (*ToolResult, error) {
	return &ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"sandboxes": []map[string]interface{}{},
		},
	}, nil
}

func (a *Adapter) execDestroy(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	_ = sandboxID
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"status": "destroyed"},
	}, nil
}

func (a *Adapter) execClone(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	url, _ := args["url"].(string)
	_ = sandboxID
	_ = url
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"status": "cloned"},
	}, nil
}

func (a *Adapter) execDiff(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	_ = sandboxID
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"diff": ""},
	}, nil
}

func (a *Adapter) execPatch(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	_ = sandboxID
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"patch": ""},
	}, nil
}

func (a *Adapter) execFilesRead(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	path, _ := args["path"].(string)
	_ = sandboxID
	_ = path
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"content": ""},
	}, nil
}

func (a *Adapter) execFilesWrite(args map[string]interface{}) (*ToolResult, error) {
	sandboxID, _ := args["sandbox_id"].(string)
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	_ = sandboxID
	_ = path
	_ = content
	return &ToolResult{
		Success: true,
		Data:    map[string]interface{}{"status": "written"},
	}, nil
}

// FormatResult formats a tool result as JSON for the agent.
func FormatResult(r *ToolResult) string {
	data, _ := json.Marshal(r)
	return string(data)
}
