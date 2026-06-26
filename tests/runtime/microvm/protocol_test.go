package microvm_test

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/microvm"
)

func TestProtocol_EncodeDecodeRoundTrip_Succeeds(t *testing.T) {
	payload := json.RawMessage(`{"cwd":"/workspace","command":"echo hello"}`)
	frame := microvm.Frame{
		Type:      microvm.FrameTypeRequest,
		ID:        "req-1",
		SessionID: "sandbox-1",
		Method:    "exec",
		Payload:   payload,
	}

	var buf bytes.Buffer
	if err := microvm.EncodeFrame(&buf, frame); err != nil {
		t.Fatalf("EncodeFrame failed: %v", err)
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Fatalf("encoded frame is not newline-delimited: %q", buf.String())
	}

	got, err := microvm.DecodeFrame(bufio.NewReader(&buf))
	if err != nil {
		t.Fatalf("DecodeFrame failed: %v", err)
	}
	if got.Type != frame.Type || got.ID != frame.ID || got.SessionID != frame.SessionID || got.Method != frame.Method {
		t.Fatalf("decoded frame mismatch: got %+v want %+v", got, frame)
	}
	if string(got.Payload) != string(payload) {
		t.Fatalf("payload mismatch: got %s want %s", got.Payload, payload)
	}
}

func TestProtocol_DecodeInvalidType_ReturnsError(t *testing.T) {
	input := `{"type":"wat","id":"req-1","session_id":"sandbox-1","method":"exec"}` + "\n"

	_, err := microvm.DecodeFrame(bufio.NewReader(strings.NewReader(input)))
	if err == nil {
		t.Fatal("expected error for invalid frame type")
	}
	if !strings.Contains(err.Error(), "invalid frame type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProtocol_NewStreamFrame_Base64EncodesPayload(t *testing.T) {
	frame, err := microvm.NewStreamFrame("req-1", "sandbox-1", "stdout", []byte("hello\n"))
	if err != nil {
		t.Fatalf("NewStreamFrame failed: %v", err)
	}
	if frame.Type != microvm.FrameTypeStream {
		t.Fatalf("frame type = %q, want stream", frame.Type)
	}

	var payload microvm.StreamPayload
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		t.Fatalf("stream payload did not unmarshal: %v", err)
	}
	if payload.Stream != "stdout" {
		t.Fatalf("stream = %q, want stdout", payload.Stream)
	}
	decoded, err := base64.StdEncoding.DecodeString(payload.Data)
	if err != nil {
		t.Fatalf("payload data is not base64: %v", err)
	}
	if string(decoded) != "hello\n" {
		t.Fatalf("decoded payload = %q, want hello newline", decoded)
	}
}

func TestProtocol_NewStreamFrame_InvalidStreamReturnsError(t *testing.T) {
	_, err := microvm.NewStreamFrame("req-1", "sandbox-1", "stdin", []byte("nope"))
	if err == nil {
		t.Fatal("expected invalid stream error")
	}
}
