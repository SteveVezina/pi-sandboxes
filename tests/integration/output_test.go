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

// TestOutputChannel_E2E verifies SPEC.md §12.8 / F9: list, pull, and pack
// actions on POST /v1/sandboxes/{id}/output against a real compat
// container, covering artifact files and the workspace patch deliverable.
func TestOutputChannel_E2E(t *testing.T) {
	if os.Getenv("DOCKER_AVAILABLE") == "" {
		t.Skip("Docker not available, skipping compat e2e test")
	}

	tmpDir := t.TempDir()
	store := sandbox.NewStore(tmpDir)
	router := daemon.NewRouter(store)

	createBody, _ := json.Marshal(map[string]interface{}{
		"name":     "e2e-output-channel",
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

	// Produce a real artifact and a workspace change so both deliverable
	// kinds (file + patch) are exercised. /artifacts is outside /workspace,
	// so it's seeded via exec rather than the workspace-scoped files API.
	execBody, _ := json.Marshal(map[string]interface{}{
		"command":   "echo 'build succeeded' > /artifacts/report.txt",
		"timeoutMs": 5000,
	})
	execReq := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/exec", bytes.NewReader(execBody))
	execW := httptest.NewRecorder()
	router.ServeHTTP(execW, execReq)
	if execW.Code != http.StatusOK {
		t.Fatalf("seed artifact: expected 200, got %d: %s", execW.Code, execW.Body.String())
	}

	// git may not be present in the base image (template tool
	// installation for compat mode is a known stub, per F5 spec gaps);
	// when it's missing, skip only the patch-deliverable assertions
	// below rather than the whole test.
	wantPatch := true
	cloneBody, _ := json.Marshal(map[string]interface{}{"url": "https://github.com/octocat/Hello-World.git"})
	cloneReq := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/clone", bytes.NewReader(cloneBody))
	cloneW := httptest.NewRecorder()
	router.ServeHTTP(cloneW, cloneReq)
	if cloneW.Code != http.StatusOK {
		t.Logf("clone unavailable in this environment (%d: %s); skipping patch-deliverable assertions", cloneW.Code, cloneW.Body.String())
		wantPatch = false
	} else {
		writeChange, _ := json.Marshal(map[string]interface{}{
			"path":    "README",
			"content": "modified for output test\n",
		})
		changeReq := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/files/write", bytes.NewReader(writeChange))
		changeW := httptest.NewRecorder()
		router.ServeHTTP(changeW, changeReq)
		if changeW.Code != http.StatusOK {
			t.Fatalf("seed workspace change: expected 200, got %d: %s", changeW.Code, changeW.Body.String())
		}
	}

	// list
	listReq := httptest.NewRequest("GET", "/v1/sandboxes/"+id+"/output", nil)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("output list: expected 200, got %d: %s", listW.Code, listW.Body.String())
	}
	var listResp struct {
		Items []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"items"`
	}
	json.Unmarshal(listW.Body.Bytes(), &listResp)
	var sawArtifact, sawPatch bool
	for _, item := range listResp.Items {
		if item.Path == "/artifacts/report.txt" {
			sawArtifact = true
		}
		if item.Type == "patch" {
			sawPatch = true
		}
	}
	if !sawArtifact {
		t.Errorf("output list missing seeded artifact, got %+v", listResp.Items)
	}
	if wantPatch && !sawPatch {
		t.Errorf("output list missing patch deliverable, got %+v", listResp.Items)
	}

	// pull
	pullDest := filepath.Join(tmpDir, "pulled-output")
	pullBody, _ := json.Marshal(map[string]interface{}{"action": "pull", "dest": pullDest})
	pullReq := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/output", bytes.NewReader(pullBody))
	pullW := httptest.NewRecorder()
	router.ServeHTTP(pullW, pullReq)
	if pullW.Code != http.StatusOK {
		t.Fatalf("output pull: expected 200, got %d: %s", pullW.Code, pullW.Body.String())
	}
	artifactBytes, err := os.ReadFile(filepath.Join(pullDest, "artifacts", "report.txt"))
	if err != nil {
		t.Fatalf("read pulled artifact: %v", err)
	}
	if string(artifactBytes) != "build succeeded\n" {
		t.Errorf("pulled artifact content = %q, want %q", string(artifactBytes), "build succeeded\n")
	}
	_, patchErr := os.Stat(filepath.Join(pullDest, "workspace.patch"))
	if wantPatch && patchErr != nil {
		t.Errorf("expected workspace.patch delivered on pull: %v", patchErr)
	}

	// pack
	packOutput := filepath.Join(tmpDir, "packed.tar.gz")
	packBody, _ := json.Marshal(map[string]interface{}{"action": "pack", "output": packOutput})
	packReq := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/output", bytes.NewReader(packBody))
	packW := httptest.NewRecorder()
	router.ServeHTTP(packW, packReq)
	if packW.Code != http.StatusOK {
		t.Fatalf("output pack: expected 200, got %d: %s", packW.Code, packW.Body.String())
	}
	info, err := os.Stat(packOutput)
	if err != nil {
		t.Fatalf("stat packed archive: %v", err)
	}
	if info.Size() == 0 {
		t.Error("packed archive is empty")
	}
}
