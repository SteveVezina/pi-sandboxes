package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pi-sandbox/pi/pkg/agent"
)

// Example coding-agent loop.
// Demonstrates the full flow: create sandbox → clone → exec → diff → destroy.
func main() {
	adapter := agent.NewAdapter("http://localhost:9001")

	// Step 1: Create sandbox
	fmt.Println("Creating sandbox...")
	result, err := adapter.ExecuteTool(nil, agent.ToolCall{
		Name: "sandbox.create",
		Arguments: map[string]interface{}{
			"name":     "demo",
			"template": "node-python",
			"mode":     "fast",
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created: %s\n", agent.FormatResult(result))

	var sandboxID string
	if data, ok := result.Data["sandbox_id"].(string); ok {
		sandboxID = data
	}

	// Step 2: Clone repository
	fmt.Println("Cloning repository...")
	result, err = adapter.ExecuteTool(nil, agent.ToolCall{
		Name: "sandbox.clone",
		Arguments: map[string]interface{}{
			"sandbox_id": sandboxID,
			"url":        "https://github.com/example/repo",
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Cloned: %s\n", agent.FormatResult(result))

	// Step 3: Run install
	fmt.Println("Running pnpm install...")
	result, err = adapter.ExecuteTool(nil, agent.ToolCall{
		Name: "sandbox.exec",
		Arguments: map[string]interface{}{
			"sandbox_id": sandboxID,
			"command":    "pnpm install",
			"timeout_ms": 120000,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Install: %s\n", agent.FormatResult(result))

	// Step 4: Run tests
	fmt.Println("Running tests...")
	result, err = adapter.ExecuteTool(nil, agent.ToolCall{
		Name: "sandbox.exec",
		Arguments: map[string]interface{}{
			"sandbox_id": sandboxID,
			"command":    "pnpm test",
			"timeout_ms": 60000,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Tests: %s\n", agent.FormatResult(result))

	// Step 5: Get diff
	fmt.Println("Getting diff...")
	result, err = adapter.ExecuteTool(nil, agent.ToolCall{
		Name: "sandbox.diff",
		Arguments: map[string]interface{}{
			"sandbox_id": sandboxID,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Diff: %s\n", agent.FormatResult(result))

	// Step 6: Destroy sandbox
	fmt.Println("Destroying sandbox...")
	result, err = adapter.ExecuteTool(nil, agent.ToolCall{
		Name: "sandbox.destroy",
		Arguments: map[string]interface{}{
			"sandbox_id": sandboxID,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Destroyed: %s\n", agent.FormatResult(result))

	fmt.Println("\nDone!")
}

// JSONType demonstrates JSON mode output.
func JSONType() {
	// Example JSON output for machine consumption
	type SandboxCreateResponse struct {
		SandboxID string `json:"sandbox_id"`
		Status    string `json:"status"`
	}

	resp := SandboxCreateResponse{
		SandboxID: "pi-demo",
		Status:    "created",
	}

	data, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(data))
}
