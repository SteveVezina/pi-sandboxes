package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/daemon"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// TestFilesPullPush_E2E verifies SPEC.md §12.6: `files pull` copies a
// workspace path out of a compat sandbox to a host destination, and
// `files push` copies a host path into the sandbox workspace.
func TestFilesPullPush_E2E(t *testing.T) {
	if os.Getenv("DOCKER_AVAILABLE") == "" {
		t.Skip("Docker not available, skipping compat e2e test")
	}

	tmpDir := t.TempDir()
	store := sandbox.NewStore(tmpDir)
	router := daemon.NewRouter(store)

	// Create a compat sandbox.
	createBody, _ := json.Marshal(map[string]interface{}{
		"name":     "e2e-files-pull-push",
		"template": "base",
		"mode":     "compat",
	})
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewReader(createBody))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create sandbox: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created map[string]string
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created["id"]
	if id == "" {
		t.Fatal("expected sandbox ID in create response")
	}
	t.Cleanup(func() {
		delReq := httptest.NewRequest("DELETE", "/v1/sandboxes/"+id, nil)
		delW := httptest.NewRecorder()
		router.ServeHTTP(delW, delReq)
	})

	// Write a file inside the workspace so there's something to pull.
	writeBody, _ := json.Marshal(map[string]interface{}{
		"path":    "hello.txt",
		"content": "pulled from sandbox\n",
	})
	writeReq := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/files/write", bytes.NewReader(writeBody))
	writeW := httptest.NewRecorder()
	router.ServeHTTP(writeW, writeReq)
	if writeW.Code != http.StatusOK {
		t.Fatalf("files/write: expected 200, got %d: %s", writeW.Code, writeW.Body.String())
	}

	// Pull it to a host destination.
	hostDest := filepath.Join(tmpDir, "pulled", "hello.txt")
	pullBody, _ := json.Marshal(map[string]interface{}{
		"src":  "/workspace/hello.txt",
		"dest": hostDest,
	})
	pullReq := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/files/pull", bytes.NewReader(pullBody))
	pullW := httptest.NewRecorder()
	router.ServeHTTP(pullW, pullReq)
	if pullW.Code != http.StatusOK {
		t.Fatalf("files/pull: expected 200, got %d: %s", pullW.Code, pullW.Body.String())
	}
	pulled, err := os.ReadFile(hostDest)
	if err != nil {
		t.Fatalf("read pulled file: %v", err)
	}
	if string(pulled) != "pulled from sandbox\n" {
		t.Errorf("pulled content = %q, want %q", string(pulled), "pulled from sandbox\n")
	}

	// Push a host file into the workspace.
	hostSrc := filepath.Join(tmpDir, "pushed.txt")
	if err := os.WriteFile(hostSrc, []byte("pushed to sandbox\n"), 0644); err != nil {
		t.Fatalf("write host src: %v", err)
	}
	pushBody, _ := json.Marshal(map[string]interface{}{
		"src":  hostSrc,
		"dest": "/workspace/pushed.txt",
	})
	pushReq := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/files/push", bytes.NewReader(pushBody))
	pushW := httptest.NewRecorder()
	router.ServeHTTP(pushW, pushReq)
	if pushW.Code != http.StatusOK {
		t.Fatalf("files/push: expected 200, got %d: %s", pushW.Code, pushW.Body.String())
	}

	// Verify it landed by reading it back through the sandbox.
	readReq := httptest.NewRequest("GET", "/v1/sandboxes/"+id+"/files/read?path=/workspace/pushed.txt", nil)
	readW := httptest.NewRecorder()
	router.ServeHTTP(readW, readReq)
	if readW.Code != http.StatusOK {
		t.Fatalf("files/read: expected 200, got %d: %s", readW.Code, readW.Body.String())
	}
	var readResp map[string]string
	json.Unmarshal(readW.Body.Bytes(), &readResp)
	if readResp["content"] != "pushed to sandbox\n" {
		t.Errorf("pushed content = %q, want %q", readResp["content"], "pushed to sandbox\n")
	}
}

// TestFilesPullPush_RejectsNonCompatMode verifies pull/push are rejected
// for sandbox modes without a container-backed workspace.
func TestFilesPullPush_RejectsNonCompatMode(t *testing.T) {
	tmpDir := t.TempDir()
	store := sandbox.NewStore(tmpDir)
	router := daemon.NewRouter(store)

	id, err := store.CreateWithOptions(sandbox.CreateOptions{
		Name:     "fast-mode-sandbox",
		Template: "base",
		Mode:     "fast",
	})
	if err != nil {
		t.Fatalf("CreateWithOptions: %v", err)
	}
	store.UpdateState(id, sandbox.StateWarm)

	for _, tc := range []struct {
		name string
		path string
		body map[string]interface{}
	}{
		{"pull", "/files/pull", map[string]interface{}{"src": "/workspace/x", "dest": "/tmp/x"}},
		{"push", "/files/push", map[string]interface{}{"src": "/tmp/x", "dest": "/workspace/x"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest("POST", "/v1/sandboxes/"+id+tc.path, bytes.NewReader(body))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for fast-mode sandbox, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}
