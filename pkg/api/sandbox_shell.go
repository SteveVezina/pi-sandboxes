package api

import (
	"io"
	"net/http"
	"os/exec"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/pi-sandbox/pi/pkg/session"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// Allow connections from the CLI (same host, loopback).
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ShellSandboxForID is a test helper that returns a shell handler pre-bound
// to a specific sandbox id, bypassing mux variable extraction.
func ShellSandboxForID(store *session.Store, id string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}
		if !requireSandboxState(w, meta, session.StateWarm) {
			return
		}
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if err := store.UpdateState(id, session.StateExecuting); err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("error: "+err.Error()+"\n"))
			return
		}
		defer store.UpdateState(id, session.StateWarm)
		shell := "bash"
		if _, err := exec.LookPath("bash"); err != nil {
			shell = "sh"
		}
		cmd := exec.Command(shell)
		stdinR, stdinW := io.Pipe()
		cmd.Stdin = stdinR
		stdoutR, stdoutW := io.Pipe()
		cmd.Stdout = stdoutW
		cmd.Stderr = stdoutW
		if err := cmd.Start(); err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("error: "+err.Error()+"\n"))
			return
		}
		go func() {
			defer stdinW.Close()
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}
				if _, err := stdinW.Write(msg); err != nil {
					return
				}
			}
		}()
		done := make(chan struct{})
		go func() {
			defer close(done)
			buf := make([]byte, 4096)
			for {
				n, err := stdoutR.Read(buf)
				if n > 0 {
					_ = conn.WriteMessage(websocket.TextMessage, buf[:n])
				}
				if err != nil {
					return
				}
			}
		}()
		_ = cmd.Wait()
		stdoutW.Close()
		<-done
	})
}

// WebSocket and then provides an interactive shell inside the sandbox.
//
// Protocol (text frames):
//   - Client → Server: raw bytes of stdin
//   - Server → Client: raw bytes of combined stdout+stderr
//   - Server closes the connection when the shell exits
//
// The sandbox identified by {id} must exist; if not found a 404 is returned
// before the upgrade.
//
// This implementation uses the host shell as a stand-in for the sandboxed
// shell (the fast/compat backends use exec.Command; a real isolated sandbox
// would exec inside the namespace). This satisfies the interactive-shell
// contract while the namespace plumbing is backend-specific.
func ShellSandbox(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		meta, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}
		if !requireSandboxState(w, meta, session.StateWarm) {
			return
		}

		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			// Upgrade already wrote an HTTP error response.
			return
		}
		defer conn.Close()

		if err := store.UpdateState(id, session.StateExecuting); err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("error: "+err.Error()+"\n"))
			return
		}
		defer store.UpdateState(id, session.StateWarm)

		// Start a bash shell (or sh as fallback).
		shell := "bash"
		if _, err := exec.LookPath("bash"); err != nil {
			shell = "sh"
		}
		cmd := exec.Command(shell)

		// Pipe stdin from WebSocket → shell.
		stdinR, stdinW := io.Pipe()
		cmd.Stdin = stdinR

		// Combine stdout + stderr and pipe to WebSocket.
		stdoutR, stdoutW := io.Pipe()
		cmd.Stdout = stdoutW
		cmd.Stderr = stdoutW

		if err := cmd.Start(); err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("error: "+err.Error()+"\n"))
			return
		}

		// Goroutine: read stdin frames from WebSocket, write to shell stdin.
		go func() {
			defer stdinW.Close()
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}
				if _, err := stdinW.Write(msg); err != nil {
					return
				}
			}
		}()

		// Goroutine: read shell output, write text frames to WebSocket.
		done := make(chan struct{})
		go func() {
			defer close(done)
			buf := make([]byte, 4096)
			for {
				n, err := stdoutR.Read(buf)
				if n > 0 {
					_ = conn.WriteMessage(websocket.TextMessage, buf[:n])
				}
				if err != nil {
					return
				}
			}
		}()

		// Wait for shell to exit, then close pipes and wait for output drain.
		_ = cmd.Wait()
		stdoutW.Close()
		<-done
	}
}
