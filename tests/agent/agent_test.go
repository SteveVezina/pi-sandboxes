package agent_test

import (
	"encoding/json"
	"testing"

	"github.com/pi-sandbox/pi/pkg/agent"
)

func TestExecuteTool_Create(t *testing.T) {
	adapter := agent.NewAdapter("http://localhost:9001")

	result, err := adapter.ExecuteTool(nil, agent.ToolCall{
		Name: "sandbox.create",
		Arguments: map[string]interface{}{
			"name":     "demo",
			"template": "base",
			"mode":     "fast",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	if result.Data == nil {
		t.Fatal("Expected non-nil data")
	}

	data := result.Data
	sandboxID, ok := data["sandbox_id"].(string)
	if !ok || sandboxID == "" {
		t.Error("Expected sandbox_id in data")
	}
}

func TestExecuteTool_Exec(t *testing.T) {
	adapter := agent.NewAdapter("http://localhost:9001")

	result, err := adapter.ExecuteTool(nil, agent.ToolCall{
		Name: "sandbox.exec",
		Arguments: map[string]interface{}{
			"sandbox_id": "test-123",
			"command":    "echo hello",
			"timeout_ms": 5000,
		},
	})
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	data := result.Data
	if data["exit_code"] != 0 {
		t.Errorf("Expected exit_code 0, got %v", data["exit_code"])
	}
}

func TestExecuteTool_Destroy(t *testing.T) {
	adapter := agent.NewAdapter("http://localhost:9001")

	result, err := adapter.ExecuteTool(nil, agent.ToolCall{
		Name: "sandbox.destroy",
		Arguments: map[string]interface{}{
			"sandbox_id": "test-123",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}
}

func TestExecuteTool_List(t *testing.T) {
	adapter := agent.NewAdapter("http://localhost:9001")

	result, err := adapter.ExecuteTool(nil, agent.ToolCall{
		Name: "sandbox.list",
	})
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}
}

func TestExecuteTool_Clone(t *testing.T) {
	adapter := agent.NewAdapter("http://localhost:9001")

	result, err := adapter.ExecuteTool(nil, agent.ToolCall{
		Name: "sandbox.clone",
		Arguments: map[string]interface{}{
			"sandbox_id": "test-123",
			"url":        "https://github.com/example/repo",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}
}

func TestExecuteTool_Diff(t *testing.T) {
	adapter := agent.NewAdapter("http://localhost:9001")

	result, err := adapter.ExecuteTool(nil, agent.ToolCall{
		Name: "sandbox.diff",
		Arguments: map[string]interface{}{
			"sandbox_id": "test-123",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}
}

func TestExecuteTool_Patch(t *testing.T) {
	adapter := agent.NewAdapter("http://localhost:9001")

	result, err := adapter.ExecuteTool(nil, agent.ToolCall{
		Name: "sandbox.patch",
		Arguments: map[string]interface{}{
			"sandbox_id": "test-123",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}
}

func TestExecuteTool_FilesRead(t *testing.T) {
	adapter := agent.NewAdapter("http://localhost:9001")

	result, err := adapter.ExecuteTool(nil, agent.ToolCall{
		Name: "sandbox.files.read",
		Arguments: map[string]interface{}{
			"sandbox_id": "test-123",
			"path":       "README.md",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}
}

func TestExecuteTool_FilesWrite(t *testing.T) {
	adapter := agent.NewAdapter("http://localhost:9001")

	result, err := adapter.ExecuteTool(nil, agent.ToolCall{
		Name: "sandbox.files.write",
		Arguments: map[string]interface{}{
			"sandbox_id": "test-123",
			"path":       "hello.txt",
			"content":    "world",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}
}

func TestExecuteTool_Unknown(t *testing.T) {
	adapter := agent.NewAdapter("http://localhost:9001")

	result, err := adapter.ExecuteTool(nil, agent.ToolCall{
		Name: "unknown.tool",
	})
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for unknown tool")
	}

	if result.Error == "" {
		t.Error("Expected error message for unknown tool")
	}
}

func TestFormatResult(t *testing.T) {
	result := &agent.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"sandbox_id": "test-123",
			"status":     "created",
		},
	}

	output := agent.FormatResult(result)
	if output == "" {
		t.Error("Expected non-empty output")
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(output), &data); err != nil {
		t.Fatalf("FormatResult output is not valid JSON: %v", err)
	}
}
