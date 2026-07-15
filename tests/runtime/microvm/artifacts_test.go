package microvm_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/microvm"
)

// recordingSender records frames and returns canned responses.
type recordingSender struct {
	sent      []microvm.Frame
	responses []microvm.Frame
}

func (s *recordingSender) Send(frame microvm.Frame) (microvm.Frame, error) {
	s.sent = append(s.sent, frame)
	if len(s.responses) == 0 {
		return microvm.Frame{}, nil
	}
	resp := s.responses[0]
	s.responses = s.responses[1:]
	return resp, nil
}

func TestArtifactExporter_UsesControlPlane(t *testing.T) {
	sender := &recordingSender{}
	listPayload, _ := json.Marshal(microvm.ArtifactListPayload{Paths: []string{"/artifacts/out.tar"}})
	sender.responses = append(sender.responses, microvm.Frame{
		Type: microvm.FrameTypeResponse, ID: "list-1", SandboxID: "s1",
		Method: "artifact.list", Payload: listPayload,
	})
	dataB64 := base64.StdEncoding.EncodeToString([]byte("hello"))
	pullPayload, _ := json.Marshal(microvm.FileDataPayload{Path: "/artifacts/out.tar", Data: dataB64, Mode: 0o644})
	sender.responses = append(sender.responses, microvm.Frame{
		Type: microvm.FrameTypeResponse, ID: "pull-1", SandboxID: "s1",
		Method: "artifact.pull", Payload: pullPayload,
	})

	exp := microvm.NewArtifactExporter("s1", sender)
	dest := t.TempDir()

	exported, err := exp.ExportAll(dest)
	if err != nil {
		t.Fatalf("ExportAll: %v", err)
	}
	if len(exported) != 1 || exported[0] != "/artifacts/out.tar" {
		t.Fatalf("exported = %v, want [/artifacts/out.tar]", exported)
	}

	// Verify control plane was used: 2 requests (list, pull)
	if len(sender.sent) != 2 {
		t.Fatalf("sent %d frames, want 2", len(sender.sent))
	}
	if sender.sent[0].Method != "artifact.list" {
		t.Fatalf("first method = %q, want artifact.list", sender.sent[0].Method)
	}
	if sender.sent[1].Method != "artifact.pull" {
		t.Fatalf("second method = %q, want artifact.pull", sender.sent[1].Method)
	}
}

func TestArtifactExporter_ReturnsEmptyWhenNoArtifacts(t *testing.T) {
	sender := &recordingSender{}
	emptyPayload, _ := json.Marshal(microvm.ArtifactListPayload{Paths: []string{}})
	sender.responses = append(sender.responses, microvm.Frame{
		Type: microvm.FrameTypeResponse, ID: "list-1", SandboxID: "s1",
		Method: "artifact.list", Payload: emptyPayload,
	})

	exp := microvm.NewArtifactExporter("s1", sender)
	exported, err := exp.ExportAll(t.TempDir())
	if err != nil {
		t.Fatalf("ExportAll: %v", err)
	}
	if len(exported) != 0 {
		t.Fatalf("exported = %v, want empty", exported)
	}
}
