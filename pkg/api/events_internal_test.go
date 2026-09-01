package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/events"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

type evChanSink struct{ ch chan events.Event }

func (s evChanSink) Deliver(e events.Event) { s.ch <- e }

func captureEvents(t *testing.T) chan events.Event {
	t.Helper()
	ch := make(chan events.Event, 8)
	events.SetDefault(events.New(evChanSink{ch}))
	t.Cleanup(func() { events.SetDefault(events.New(events.SlogSink{})) })
	return ch
}

func TestDeleteSandbox_EmitsSandboxDestroyed(t *testing.T) {
	ch := captureEvents(t)

	dir := filepath.Join(t.TempDir(), "sandboxes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	store := sandbox.NewStore(dir)
	id, err := store.CreateWithOptions(sandbox.CreateOptions{Name: "d", Template: "base", Mode: "fast"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/v1/sandboxes/"+id, nil)
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w := httptest.NewRecorder()
	DeleteSandbox(store)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", w.Code, w.Body)
	}

	select {
	case e := <-ch:
		if e.Type != events.SandboxDestroyed || e.SandboxID != id {
			t.Fatalf("event = %+v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("no pi.sandbox.destroyed event")
	}
}
