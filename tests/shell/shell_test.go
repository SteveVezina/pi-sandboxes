package shell_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pi-sandbox/pi/pkg/api"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func newStoreWithSandbox(t *testing.T) (*sandbox.Store, string) {
	t.Helper()
	store := sandbox.NewStore(t.TempDir())
	id, err := store.Create("test-shell", "base", "fast")
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	return store, id
}

// TestShellSandbox_ConnectsAndEchoes verifies the WebSocket shell endpoint
// accepts a connection and relays output from the shell process (F01 / AC-1.x,
// CORE.md watch-out: true interactive PTY).
func TestShellSandbox_ConnectsAndEchoes(t *testing.T) {
	store, id := newStoreWithSandbox(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Inject the sandbox id via a path variable substitute.
		r = r.WithContext(r.Context())
		// Manually invoke the handler (mux vars not needed for the test).
		h := api.ShellSandboxForID(store, id)
		h.ServeHTTP(w, r)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/shell"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Send a command and read the response.
	if err := conn.WriteMessage(websocket.TextMessage, []byte("echo hello-from-shell\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	output := &strings.Builder{}
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		output.Write(msg)
		if strings.Contains(output.String(), "hello-from-shell") {
			break
		}
	}

	if !strings.Contains(output.String(), "hello-from-shell") {
		t.Errorf("shell did not echo command; got: %q", output.String())
	}
}

// TestShellSandbox_NotFound verifies a 404 is returned for unknown sandbox ids.
func TestShellSandbox_NotFound(t *testing.T) {
	store := sandbox.NewStore(t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := api.ShellSandboxForID(store, "nonexistent-id")
		h.ServeHTTP(w, r)
	}))
	defer srv.Close()

	// A regular HTTP GET should return 404 before the WebSocket upgrade.
	resp, err := http.Get(srv.URL + "/shell")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}
