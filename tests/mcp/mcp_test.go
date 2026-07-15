package mcp_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/pi-sandbox/pi/pkg/mcp"
)

func TestHandleInitialize(t *testing.T) {
	server := mcp.NewServer()

	req := `{
		"jsonrpc": "2.0",
		"method": "initialize",
		"params": {},
		"id": 1
	}`

	resp, err := server.HandleRequest(bytes.NewReader([]byte(req)))
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("Expected protocol version '2024-11-05', got '%v'", result["protocolVersion"])
	}

	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected serverInfo map")
	}
	if serverInfo["name"] != "pi-sandbox-mcp" {
		t.Errorf("Expected server name 'pi-sandbox-mcp', got '%v'", serverInfo["name"])
	}
}

func TestHandleToolsList(t *testing.T) {
	server := mcp.NewServer()

	req := `{
		"jsonrpc": "2.0",
		"method": "tools/list",
		"id": 2
	}`

	resp, err := server.HandleRequest(bytes.NewReader([]byte(req)))
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	tools, ok := result["tools"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected tools array")
	}

	if len(tools) < 4 {
		t.Errorf("Expected >= 4 tools, got %d", len(tools))
	}
}

func TestHandleSandboxCreate(t *testing.T) {
	server := mcp.NewServer()

	req := `{
		"jsonrpc": "2.0",
		"method": "sandbox/create",
		"params": {"name": "test", "template": "base", "mode": "fast"},
		"id": 3
	}`

	resp, err := server.HandleRequest(bytes.NewReader([]byte(req)))
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	if result["status"] != "created" {
		t.Errorf("Expected status 'created', got '%v'", result["status"])
	}
}

func TestHandleSandboxExec(t *testing.T) {
	server := mcp.NewServer()

	req := `{
		"jsonrpc": "2.0",
		"method": "sandbox/exec",
		"params": {"sandbox_id": "test-123", "command": "echo hello"},
		"id": 4
	}`

	resp, err := server.HandleRequest(bytes.NewReader([]byte(req)))
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	if result["exit_code"] != 0 {
		t.Errorf("Expected exit_code 0, got %v", result["exit_code"])
	}
}

func TestHandleSandboxDestroy(t *testing.T) {
	server := mcp.NewServer()

	req := `{
		"jsonrpc": "2.0",
		"method": "sandbox/destroy",
		"params": {"sandbox_id": "test-123"},
		"id": 5
	}`

	resp, err := server.HandleRequest(bytes.NewReader([]byte(req)))
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	if result["status"] != "destroyed" {
		t.Errorf("Expected status 'destroyed', got '%v'", result["status"])
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	server := mcp.NewServer()

	req := `{
		"jsonrpc": "2.0",
		"method": "unknown/method",
		"id": 99
	}`

	resp, err := server.HandleRequest(bytes.NewReader([]byte(req)))
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("Expected error for unknown method")
	}

	if resp.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", resp.Error.Code)
	}
}

func TestHTTPHandler(t *testing.T) {
	server := mcp.NewServer()

	// Create a test HTTP request
	req := bytes.NewReader([]byte(`{
		"jsonrpc": "2.0",
		"method": "tools/list",
		"id": 1
	}`))

	resp, err := server.HandleRequest(req)
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %s", resp.Error.Message)
	}

	// Verify JSON marshaling works
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty JSON response")
	}
}
