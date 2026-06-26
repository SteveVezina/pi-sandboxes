package sdk_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTypeScriptClientExists(t *testing.T) {
	tsDir := filepath.Join("..", "..", "sdk", "typescript")

	// Check package.json exists
	data, err := os.ReadFile(filepath.Join(tsDir, "package.json"))
	if err != nil {
		t.Fatalf("package.json not found: %v", err)
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		t.Fatalf("Invalid package.json: %v", err)
	}

	if pkg["name"] != "@pi-sandbox/sdk" {
		t.Errorf("Expected name '@pi-sandbox/sdk', got '%v'", pkg["name"])
	}
}

func TestTypeScriptConfigExists(t *testing.T) {
	tsDir := filepath.Join("..", "..", "sdk", "typescript")

	// Check tsconfig.json exists
	_, err := os.Stat(filepath.Join(tsDir, "tsconfig.json"))
	if os.IsNotExist(err) {
		t.Fatal("tsconfig.json not found")
	}
}

func TestTypeScriptSourceExists(t *testing.T) {
	tsDir := filepath.Join("..", "..", "sdk", "typescript")

	// Check client.ts exists
	_, err := os.Stat(filepath.Join(tsDir, "src", "client.ts"))
	if os.IsNotExist(err) {
		t.Fatal("client.ts not found")
	}
}

func TestPythonSDKExists(t *testing.T) {
	pyDir := filepath.Join("..", "..", "sdk", "python")

	// Check pyproject.toml exists
	_, err := os.Stat(filepath.Join(pyDir, "pyproject.toml"))
	if os.IsNotExist(err) {
		t.Fatal("pyproject.toml not found")
	}
}

func TestPythonSourceExists(t *testing.T) {
	pyDir := filepath.Join("..", "..", "sdk", "python")

	// Check __init__.py exists
	_, err := os.Stat(filepath.Join(pyDir, "src", "pi_sandbox", "__init__.py"))
	if os.IsNotExist(err) {
		t.Fatal("__init__.py not found")
	}
}

func TestPythonInitHasClient(t *testing.T) {
	pyDir := filepath.Join("..", "..", "sdk", "python")

	data, err := os.ReadFile(filepath.Join(pyDir, "src", "pi_sandbox", "__init__.py"))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	content := string(data)
	// Check for key SDK methods
	methods := []string{
		"def create(",
		"def list(",
		"def get(",
		"def destroy(",
		"def exec(",
		"def clone(",
		"def diff(",
		"def patch(",
		"def files_read(",
		"def files_write(",
		"def logs(",
	}

	for _, method := range methods {
		if !contains(content, method) {
			t.Errorf("Expected method '%s' in __init__.py", method)
		}
	}
}

func TestPythonInitHasExecResult(t *testing.T) {
	pyDir := filepath.Join("..", "..", "sdk", "python")

	data, err := os.ReadFile(filepath.Join(pyDir, "src", "pi_sandbox", "__init__.py"))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	content := string(data)
	if !contains(content, "class ExecResult:") {
		t.Error("Expected ExecResult class in __init__.py")
	}
	if !contains(content, "class SandboxInfo:") {
		t.Error("Expected SandboxInfo class in __init__.py")
	}
}

func TestPythonInitHasCreateClient(t *testing.T) {
	pyDir := filepath.Join("..", "..", "sdk", "python")

	data, err := os.ReadFile(filepath.Join(pyDir, "src", "pi_sandbox", "__init__.py"))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	content := string(data)
	if !contains(content, "def create_client(") {
		t.Error("Expected create_client function in __init__.py")
	}
}

func TestTypeScriptClientHasMethods(t *testing.T) {
	tsDir := filepath.Join("..", "..", "sdk", "typescript")

	data, err := os.ReadFile(filepath.Join(tsDir, "src", "client.ts"))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	content := string(data)
	methods := []string{
		"async create(",
		"async list(",
		"async get(",
		"async destroy(",
		"async exec(",
		"async clone(",
		"async diff(",
		"async patch(",
		"async filesRead(",
		"async filesWrite(",
		"async logs(",
		"async artifactsList(",
	}

	for _, method := range methods {
		if !contains(content, method) {
			t.Errorf("Expected method '%s' in client.ts", method)
		}
	}
}

func TestTypeScriptClientHasInterfaces(t *testing.T) {
	tsDir := filepath.Join("..", "..", "sdk", "typescript")

	data, err := os.ReadFile(filepath.Join(tsDir, "src", "client.ts"))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	content := string(data)
	if !contains(content, "export interface CreateOptions") {
		t.Error("Expected CreateOptions interface")
	}
	if !contains(content, "export interface ExecOptions") {
		t.Error("Expected ExecOptions interface")
	}
	if !contains(content, "export interface ExecResult") {
		t.Error("Expected ExecResult interface")
	}
	if !contains(content, "export interface SandboxInfo") {
		t.Error("Expected SandboxInfo interface")
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && find(haystack, needle)
}

func find(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
