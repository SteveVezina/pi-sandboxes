package events_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/pi-sandbox/pi/pkg/events"
)

// chanSink captures delivered events for assertions.
type chanSink struct{ ch chan events.Event }

func (s chanSink) Deliver(e events.Event) { s.ch <- e }

func waitEvent(t *testing.T, ch chan events.Event) events.Event {
	t.Helper()
	select {
	case e := <-ch:
		return e
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
		return events.Event{}
	}
}

func TestEmitter_FanOutAndStampsTime(t *testing.T) {
	a := chanSink{make(chan events.Event, 1)}
	b := chanSink{make(chan events.Event, 1)}
	em := events.New(a, b)

	em.Emit(events.Event{Type: events.SandboxCreated, SandboxID: "s1"})

	for _, s := range []chanSink{a, b} {
		e := waitEvent(t, s.ch)
		if e.Type != events.SandboxCreated || e.SandboxID != "s1" {
			t.Fatalf("bad event: %+v", e)
		}
		if e.Time.IsZero() {
			t.Error("Emit should stamp Time")
		}
	}
}

func TestSetDefault_EmitRoutesThroughInstalledEmitter(t *testing.T) {
	s := chanSink{make(chan events.Event, 1)}
	events.SetDefault(events.New(s))
	t.Cleanup(func() { events.SetDefault(events.New(events.SlogSink{})) })

	events.Emit(events.Event{Type: events.ArtifactDelivered, SandboxID: "s2"})
	if e := waitEvent(t, s.ch); e.Type != events.ArtifactDelivered {
		t.Fatalf("got %+v", e)
	}
}

func TestWebhookSink_PostsEnvelope(t *testing.T) {
	var (
		mu   sync.Mutex
		body []byte
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		body = b
		mu.Unlock()
	}))
	defer srv.Close()

	sink := events.NewWebhookSink(srv.URL)
	sink.Deliver(events.Event{Type: events.SandboxCreated, SandboxID: "s3", Time: time.Now().UTC()})

	// Deliver is synchronous for the sink itself; give the handler a beat.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		got := body
		mu.Unlock()
		if got != nil {
			var e events.Event
			if err := json.Unmarshal(got, &e); err != nil {
				t.Fatalf("unmarshal webhook body: %v", err)
			}
			if e.Type != events.SandboxCreated || e.SandboxID != "s3" {
				t.Fatalf("webhook envelope = %+v", e)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("webhook never received a POST")
}

func TestWebhookSink_DropsAfterRetryOnBadURL(t *testing.T) {
	// Unroutable URL — Deliver should return without panicking.
	events.NewWebhookSink("http://127.0.0.1:1/never").Deliver(events.Event{Type: events.RunStarted})
}
