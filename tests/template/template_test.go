package template_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/template"
)

func TestStore_CreateAndGet(t *testing.T) {
	store := newTestStore(t)

	tmpl := &template.Template{
		Runtime: "auto",
		Base:    "debian-slim",
		Tools:   []string{"git", "curl"},
		Mounts:  map[string]string{"workspace": "/workspace"},
		Network: "restricted",
	}

	if err := store.Create("my-test", tmpl); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	loaded, err := store.Get("my-test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if loaded.Name != "my-test" {
		t.Errorf("Expected name 'my-test', got '%s'", loaded.Name)
	}
	if len(loaded.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(loaded.Tools))
	}
}

func TestStore_List(t *testing.T) {
	store := newTestStore(t)

	// Create two templates
	store.Create("tpl-a", &template.Template{Name: "tpl-a", Tools: []string{"git"}})
	store.Create("tpl-b", &template.Template{Name: "tpl-b", Tools: []string{"python"}})

	names, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(names) != 2 {
		t.Errorf("Expected 2 templates, got %d: %v", len(names), names)
	}
}

func TestStore_Delete(t *testing.T) {
	store := newTestStore(t)

	store.Create("to-delete", &template.Template{Name: "to-delete"})

	_, err := store.Get("to-delete")
	if err != nil {
		t.Fatalf("Template should exist before delete: %v", err)
	}

	if err := store.Delete("to-delete"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.Get("to-delete")
	if err == nil {
		t.Fatal("Expected error after delete")
	}
}

func TestStore_Get_NotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent template")
	}
}

func TestStore_InstallDefaults(t *testing.T) {
	store := newTestStore(t)

	if err := store.InstallDefaults(); err != nil {
		t.Fatalf("InstallDefaults failed: %v", err)
	}

	names, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Should have 7 default templates
	if len(names) != 7 {
		t.Errorf("Expected 7 templates, got %d: %v", len(names), names)
	}
}

func TestStore_GetBaseTemplate(t *testing.T) {
	store := newTestStore(t)
	store.InstallDefaults()

	tmpl, err := store.Get("base")
	if err != nil {
		t.Fatalf("Get base failed: %v", err)
	}

	if tmpl.Name != "base" {
		t.Errorf("Expected name 'base', got '%s'", tmpl.Name)
	}
	if tmpl.Runtime != "auto" {
		t.Errorf("Expected runtime 'auto', got '%s'", tmpl.Runtime)
	}
	if tmpl.Base != "debian-slim" {
		t.Errorf("Expected base 'debian-slim', got '%s'", tmpl.Base)
	}
}

func TestStore_GetNodeTemplate(t *testing.T) {
	store := newTestStore(t)
	store.InstallDefaults()

	tmpl, err := store.Get("node")
	if err != nil {
		t.Fatalf("Get node failed: %v", err)
	}

	hasNode := false
	hasPNPM := false
	for _, tool := range tmpl.Tools {
		if tool == "node:22" {
			hasNode = true
		}
		if tool == "pnpm" {
			hasPNPM = true
		}
	}
	if !hasNode {
		t.Error("Expected node:22 in tools")
	}
	if !hasPNPM {
		t.Error("Expected pnpm in tools")
	}
}

func TestStore_GetPythonTemplate(t *testing.T) {
	store := newTestStore(t)
	store.InstallDefaults()

	tmpl, err := store.Get("python")
	if err != nil {
		t.Fatalf("Get python failed: %v", err)
	}

	hasPython := false
	hasUV := false
	for _, tool := range tmpl.Tools {
		if tool == "python:3.13" {
			hasPython = true
		}
		if tool == "uv" {
			hasUV = true
		}
	}
	if !hasPython {
		t.Error("Expected python:3.13 in tools")
	}
	if !hasUV {
		t.Error("Expected uv in tools")
	}
}

func TestStore_GetNodePythonTemplate(t *testing.T) {
	store := newTestStore(t)
	store.InstallDefaults()

	tmpl, err := store.Get("node-python")
	if err != nil {
		t.Fatalf("Get node-python failed: %v", err)
	}

	hasNode := false
	hasPython := false
	hasUV := false
	for _, tool := range tmpl.Tools {
		if tool == "node:22" {
			hasNode = true
		}
		if tool == "python:3.13" {
			hasPython = true
		}
		if tool == "uv" {
			hasUV = true
		}
	}
	if !hasNode {
		t.Error("Expected node:22 in node-python template")
	}
	if !hasPython {
		t.Error("Expected python:3.13 in node-python template")
	}
	if !hasUV {
		t.Error("Expected uv in node-python template")
	}
}

func TestStore_GetPolyglotTemplate(t *testing.T) {
	store := newTestStore(t)
	store.InstallDefaults()

	tmpl, err := store.Get("polyglot")
	if err != nil {
		t.Fatalf("Get polyglot failed: %v", err)
	}

	// Should have tools from all templates
	if len(tmpl.Tools) < 10 {
		t.Errorf("Expected >= 10 tools in polyglot, got %d", len(tmpl.Tools))
	}
	if len(tmpl.Caches) < 5 {
		t.Errorf("Expected >= 5 caches in polyglot, got %d", len(tmpl.Caches))
	}
}

func TestStore_DefaultTemplates(t *testing.T) {
	defaults := template.DefaultTemplates()
	if len(defaults) != 7 {
		t.Errorf("Expected 7 default templates, got %d", len(defaults))
	}

	expected := []string{"base", "node", "python", "go", "rust", "node-python", "polyglot"}
	for _, name := range expected {
		if _, ok := defaults[name]; !ok {
			t.Errorf("Missing default template: %s", name)
		}
	}
}

func TestStore_RecreateExisting(t *testing.T) {
	store := newTestStore(t)

	// Create a template
	store.Create("my-tpl", &template.Template{Name: "my-tpl", Tools: []string{"git"}})

	// Try to create again — should succeed (overwrites)
	err := store.Create("my-tpl", &template.Template{Name: "my-tpl", Tools: []string{"curl"}})
	if err != nil {
		t.Fatalf("Recreate should succeed: %v", err)
	}
}

func TestStore_MissingTemplatesDir(t *testing.T) {
	// Create store with non-existent dir
	store := template.NewStore("/tmp/pi-nonexistent-dir-" + randomID())

	names, err := store.List()
	if err == nil {
		t.Fatal("Expected error for non-existent dir")
	}
	if names != nil {
		t.Error("Expected nil names for non-existent dir")
	}
}

func TestCLI_ListJSON(t *testing.T) {
	store := newTestStore(t)
	store.Create("tpl1", &template.Template{Name: "tpl1"})
	store.Create("tpl2", &template.Template{Name: "tpl2"})

	names, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Verify JSON marshaling works
	data, err := json.Marshal(names)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	// Verify JSON is valid
	var result []string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
}

func newTestStore(t *testing.T) *template.Store {
	tmpDir := filepath.Join(os.TempDir(), "pi-tpl-test-"+randomID())
	os.MkdirAll(tmpDir, 0755)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	return template.NewStore(tmpDir)
}

func randomID() string {
	b := []byte("abcdefghijklmnopqrstuvwxyz012345")
	n := len(b)
	result := make([]byte, 8)
	for i := range result {
		result[i] = b[i%n]
	}
	return string(result)
}
