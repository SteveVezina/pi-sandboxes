package microvm_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/microvm"
)

func TestProtocol_NewFileReadRequestFrame_EncodesPath(t *testing.T) {
	frame, err := microvm.NewFileReadRequestFrame("req-1", "sandbox-1", "/workspace/README.md")
	if err != nil {
		t.Fatalf("NewFileReadRequestFrame failed: %v", err)
	}
	if frame.Type != microvm.FrameTypeRequest || frame.Method != "file.read" {
		t.Fatalf("frame = %+v, want file.read request", frame)
	}

	var payload microvm.FileReadRequestPayload
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		t.Fatalf("payload did not unmarshal: %v", err)
	}
	if payload.Path != "/workspace/README.md" {
		t.Fatalf("path = %q, want /workspace/README.md", payload.Path)
	}
}

func TestProtocol_NewFileWriteRequestFrame_Base64EncodesData(t *testing.T) {
	frame, err := microvm.NewFileWriteRequestFrame("req-1", "sandbox-1", "/workspace/app.txt", []byte("content\n"), 0o644)
	if err != nil {
		t.Fatalf("NewFileWriteRequestFrame failed: %v", err)
	}
	if frame.Type != microvm.FrameTypeRequest || frame.Method != "file.write" {
		t.Fatalf("frame = %+v, want file.write request", frame)
	}

	var payload microvm.FileWriteRequestPayload
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		t.Fatalf("payload did not unmarshal: %v", err)
	}
	data, err := base64.StdEncoding.DecodeString(payload.Data)
	if err != nil {
		t.Fatalf("data is not base64: %v", err)
	}
	if payload.Path != "/workspace/app.txt" || string(data) != "content\n" || payload.Mode != 0o644 {
		t.Fatalf("payload = %+v decoded=%q", payload, data)
	}
}

func TestProtocol_NewArtifactListRequestFrame_UsesArtifactListMethod(t *testing.T) {
	frame, err := microvm.NewArtifactListRequestFrame("req-1", "sandbox-1")
	if err != nil {
		t.Fatalf("NewArtifactListRequestFrame failed: %v", err)
	}
	if frame.Type != microvm.FrameTypeRequest || frame.Method != "artifact.list" {
		t.Fatalf("frame = %+v, want artifact.list request", frame)
	}
}

func TestProtocol_NewArtifactPullRequestFrame_EncodesPath(t *testing.T) {
	frame, err := microvm.NewArtifactPullRequestFrame("req-1", "sandbox-1", "/artifacts/report.json")
	if err != nil {
		t.Fatalf("NewArtifactPullRequestFrame failed: %v", err)
	}
	if frame.Type != microvm.FrameTypeRequest || frame.Method != "artifact.pull" {
		t.Fatalf("frame = %+v, want artifact.pull request", frame)
	}

	var payload microvm.ArtifactPullRequestPayload
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		t.Fatalf("payload did not unmarshal: %v", err)
	}
	if payload.Path != "/artifacts/report.json" {
		t.Fatalf("path = %q, want /artifacts/report.json", payload.Path)
	}
}

func TestProtocol_DecodeFileDataPayload_DecodesBase64Bytes(t *testing.T) {
	payload, err := json.Marshal(microvm.FileDataPayload{
		Path: "/workspace/out.txt",
		Data: base64.StdEncoding.EncodeToString([]byte("hello")),
		Mode: 0o600,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	frame := microvm.Frame{
		Type:      microvm.FrameTypeResponse,
		ID:        "req-1",
		SandboxID: "sandbox-1",
		Method:    "file.read",
		Payload:   payload,
	}

	path, data, mode, err := microvm.DecodeFileDataPayload(frame)
	if err != nil {
		t.Fatalf("DecodeFileDataPayload failed: %v", err)
	}
	if path != "/workspace/out.txt" || string(data) != "hello" || mode != 0o600 {
		t.Fatalf("path=%q data=%q mode=%#o", path, data, mode)
	}
}

func TestTransferClient_ReadWriteFiles_UsesControlPlane(t *testing.T) {
	transport := &fakeTransferTransport{}
	client := microvm.NewTransferClient("sandbox-1", transport)

	data, mode, err := client.ReadFile("/workspace/app.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != "from guest" || mode != 0o640 {
		t.Fatalf("data=%q mode=%#o, want guest file data", data, mode)
	}
	if got := transport.methods[0]; got != "file.read" {
		t.Fatalf("first method = %q, want file.read", got)
	}

	if err := client.WriteFile("/workspace/app.txt", []byte("to guest"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if got := transport.methods[1]; got != "file.write" {
		t.Fatalf("second method = %q, want file.write", got)
	}
}

func TestTransferClient_ListPullArtifacts_UsesControlPlane(t *testing.T) {
	transport := &fakeTransferTransport{}
	client := microvm.NewTransferClient("sandbox-1", transport)

	paths, err := client.ListArtifacts()
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}
	if len(paths) != 2 || paths[0] != "/artifacts/report.json" || paths[1] != "/artifacts/log.txt" {
		t.Fatalf("paths = %v, want artifact list", paths)
	}

	data, mode, err := client.PullArtifact("/artifacts/report.json")
	if err != nil {
		t.Fatalf("PullArtifact failed: %v", err)
	}
	if string(data) != `{"ok":true}` || mode != 0o600 {
		t.Fatalf("data=%q mode=%#o, want artifact payload", data, mode)
	}
	if got := transport.methods[len(transport.methods)-1]; got != "artifact.pull" {
		t.Fatalf("last method = %q, want artifact.pull", got)
	}
}

type fakeTransferTransport struct {
	methods []string
}

func (f *fakeTransferTransport) Send(frame microvm.Frame) (microvm.Frame, error) {
	f.methods = append(f.methods, frame.Method)

	switch frame.Method {
	case "file.read":
		payload, _ := json.Marshal(microvm.FileDataPayload{
			Path: "/workspace/app.txt",
			Data: base64.StdEncoding.EncodeToString([]byte("from guest")),
			Mode: 0o640,
		})
		return microvm.Frame{
			Type:      microvm.FrameTypeResponse,
			ID:        frame.ID,
			SandboxID: frame.SandboxID,
			Method:    frame.Method,
			Payload:   payload,
		}, nil
	case "file.write":
		var payload microvm.FileWriteRequestPayload
		if err := json.Unmarshal(frame.Payload, &payload); err != nil {
			return microvm.Frame{}, err
		}
		if payload.Path != "/workspace/app.txt" || payload.Mode != 0o600 {
			return microvm.Frame{
				Type:      microvm.FrameTypeResponse,
				ID:        frame.ID,
				SandboxID: frame.SandboxID,
				Method:    frame.Method,
				Error:     &microvm.FrameError{Code: "bad_request", Message: "unexpected write payload"},
			}, nil
		}
		return microvm.Frame{
			Type:      microvm.FrameTypeResponse,
			ID:        frame.ID,
			SandboxID: frame.SandboxID,
			Method:    frame.Method,
		}, nil
	case "artifact.list":
		payload, _ := json.Marshal(microvm.ArtifactListPayload{
			Paths: []string{"/artifacts/report.json", "/artifacts/log.txt"},
		})
		return microvm.Frame{
			Type:      microvm.FrameTypeResponse,
			ID:        frame.ID,
			SandboxID: frame.SandboxID,
			Method:    frame.Method,
			Payload:   payload,
		}, nil
	case "artifact.pull":
		payload, _ := json.Marshal(microvm.FileDataPayload{
			Path: "/artifacts/report.json",
			Data: base64.StdEncoding.EncodeToString([]byte(`{"ok":true}`)),
			Mode: 0o600,
		})
		return microvm.Frame{
			Type:      microvm.FrameTypeResponse,
			ID:        frame.ID,
			SandboxID: frame.SandboxID,
			Method:    frame.Method,
			Payload:   payload,
		}, nil
	default:
		return microvm.Frame{
			Type:      microvm.FrameTypeResponse,
			ID:        frame.ID,
			SandboxID: frame.SandboxID,
			Method:    frame.Method,
			Error:     &microvm.FrameError{Code: "unknown_method", Message: frame.Method},
		}, nil
	}
}
