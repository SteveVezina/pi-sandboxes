package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/pi-sandbox/pi/pkg/api"
	"github.com/pi-sandbox/pi/pkg/sandbox"
	"github.com/pi-sandbox/pi/pkg/template"
)

func TestCompatSandboxCreation(t *testing.T) {
	// Skip if Docker is not available
	if os.Getenv("DOCKER_AVAILABLE") == "" {
		t.Skip("Docker not available, skipping compat e2e test")
	}

	// Create a temporary store
	tmpDir := t.TempDir()
	store := session.NewStore(tmpDir)

	// Create the handler
	handler := api.CreateSandbox(store)

	// Create a request
	body, _ := json.Marshal(map[string]interface{}{
		"name":     "e2e-compat-test",
		"template": "python",
		"mode":     "compat",
	})
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["id"] == "" {
		t.Fatal("Expected sandbox ID in response")
	}

	// Verify the sandbox exists
	meta, err := store.Get(resp["id"])
	if err != nil {
		t.Fatalf("Sandbox not found: %v", err)
	}
	if meta.Template != "python" {
		t.Errorf("Expected template 'python', got '%s'", meta.Template)
	}
	if meta.Mode != "compat" {
		t.Errorf("Expected mode 'compat', got '%s'", meta.Mode)
	}
}

func TestImageResolution(t *testing.T) {
	tests := []struct {
		base     string
		expected string
	}{
		{"debian-slim", "docker.io/library/debian:bookworm-slim"},
		{"debian", "docker.io/library/debian:bookworm"},
		{"node", "docker.io/library/node:22-bookworm"},
		{"python", "docker.io/library/python:3.13-bookworm"},
		{"go", "docker.io/library/golang:1.24-bookworm"},
		{"rust", "docker.io/library/rust:1.80-bookworm"},
		{"ubuntu", "docker.io/library/ubuntu:24.04"},
		{"alpine", "docker.io/library/alpine:3.20"},
		// Already a full reference
		{"docker.io/library/debian:bookworm-slim", "docker.io/library/debian:bookworm-slim"},
		{"ghcr.io/pi-sandbox/myimage:v1", "ghcr.io/pi-sandbox/myimage:v1"},
		// Unknown shorthand
		{"unknown", "docker.io/library/unknown:latest"},
		// Empty
		{"", "docker.io/library/debian:bookworm-slim"},
	}

	for _, tt := range tests {
		t.Run(tt.base, func(t *testing.T) {
			result := template.ResolveImage(tt.base)
			if result != tt.expected {
				t.Errorf("ResolveImage(%q) = %q, want %q", tt.base, result, tt.expected)
			}
		})
	}
}

func TestResolveTemplateImage(t *testing.T) {
	t.Run("explicit image overrides base", func(t *testing.T) {
		tmpl := &template.Template{
			Name:  "custom",
			Base:  "debian-slim",
			Image: "docker.io/library/ubuntu:24.04",
		}
		result := template.ResolveTemplateImage(tmpl)
		if result != "docker.io/library/ubuntu:24.04" {
			t.Errorf("ResolveTemplateImage() = %q, want %q", result, "docker.io/library/ubuntu:24.04")
		}
	})

	t.Run("base is resolved when image is empty", func(t *testing.T) {
		tmpl := &template.Template{
			Name: "python",
			Base: "python",
		}
		result := template.ResolveTemplateImage(tmpl)
		if result != "docker.io/library/python:3.13-bookworm" {
			t.Errorf("ResolveTemplateImage() = %q, want %q", result, "docker.io/library/python:3.13-bookworm")
		}
	})
}
