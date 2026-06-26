package microvm_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/microvm"
)

func TestExec_EncodeStreamData_RoundTrip(t *testing.T) {
	original := []byte("hello world\nline2\n")
	encoded := microvm.EncodeStreamData(original)
	decoded, err := microvm.DecodeStreamData(encoded)
	if err != nil {
		t.Fatalf("DecodeStreamData failed: %v", err)
	}
	if string(decoded) != string(original) {
		t.Fatalf("decoded = %q, want %q", decoded, original)
	}
}

func TestExec_EncodeStreamData_Base64Format(t *testing.T) {
	data := []byte{0x00, 0x01, 0x02, 0xff}
	encoded := microvm.EncodeStreamData(data)
	// Verify it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("encoded data is not valid base64: %v", err)
	}
	if string(decoded) != string(data) {
		t.Fatalf("decoded = %v, want %v", decoded, data)
	}
}

func TestExec_NewHelloFrame_CorrectType(t *testing.T) {
	frame, err := microvm.NewHelloFrame("h-1", "sess-1")
	if err != nil {
		t.Fatalf("NewHelloFrame failed: %v", err)
	}
	if frame.Type != microvm.FrameTypeRequest {
		t.Errorf("frame type = %q, want %q", frame.Type, microvm.FrameTypeRequest)
	}
	if frame.Method != "hello" {
		t.Errorf("frame method = %q, want hello", frame.Method)
	}
	if frame.ID != "h-1" {
		t.Errorf("frame id = %q, want h-1", frame.ID)
	}
	if frame.SessionID != "sess-1" {
		t.Errorf("frame session_id = %q, want sess-1", frame.SessionID)
	}
}

func TestExec_NewReadyFrame_CorrectType(t *testing.T) {
	frame, err := microvm.NewReadyFrame("r-1", "sess-1")
	if err != nil {
		t.Fatalf("NewReadyFrame failed: %v", err)
	}
	if frame.Type != microvm.FrameTypeEvent {
		t.Errorf("frame type = %q, want %q", frame.Type, microvm.FrameTypeEvent)
	}
	if frame.Method != "ready" {
		t.Errorf("frame method = %q, want ready", frame.Method)
	}
}

func TestExec_NewShutdownFrame_CorrectType(t *testing.T) {
	frame, err := microvm.NewShutdownFrame("s-1", "sess-1")
	if err != nil {
		t.Fatalf("NewShutdownFrame failed: %v", err)
	}
	if frame.Type != microvm.FrameTypeEvent {
		t.Errorf("frame type = %q, want %q", frame.Type, microvm.FrameTypeEvent)
	}
	if frame.Method != "shutdown" {
		t.Errorf("frame method = %q, want shutdown", frame.Method)
	}
}

func TestExec_ExecResultPayload_DecodesCorrectly(t *testing.T) {
	payload := microvm.ExecResultPayload{
		ExitCode:   0,
		DurationMs: 42,
		TimedOut:   false,
		Truncated:  false,
	}
	frame, err := microvm.NewExecResultFrame("e-1", "sess-1", payload)
	if err != nil {
		t.Fatalf("NewExecResultFrame failed: %v", err)
	}

	var decoded microvm.ExecResultPayload
	if err := json.Unmarshal(frame.Payload, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if decoded.ExitCode != 0 || decoded.DurationMs != 42 || decoded.TimedOut || decoded.Truncated {
		t.Errorf("decoded = %+v, want %+v", decoded, payload)
	}
}

func TestExec_ExecResultPayload_NonZeroExit(t *testing.T) {
	payload := microvm.ExecResultPayload{
		ExitCode:   1,
		DurationMs: 100,
		TimedOut:   false,
		Truncated:  true,
	}
	frame, err := microvm.NewExecResultFrame("e-2", "sess-2", payload)
	if err != nil {
		t.Fatalf("NewExecResultFrame failed: %v", err)
	}

	result, err := microvm.ExecFramePayload(frame)
	if err != nil {
		t.Fatalf("ExecFramePayload failed: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("exit_code = %d, want 1", result.ExitCode)
	}
	if result.Truncated != true {
		t.Error("expected truncated to be true")
	}
}

func TestExec_ExecStreamingPayload(t *testing.T) {
	// Test that stream payload decoding works correctly
	frame, err := microvm.NewStreamFrame("s-1", "sess-1", "stdout", []byte("test output"))
	if err != nil {
		t.Fatalf("NewStreamFrame failed: %v", err)
	}

	stream, data, err := microvm.DecodeStreamPayload(frame)
	if err != nil {
		t.Fatalf("DecodeStreamPayload failed: %v", err)
	}
	if stream != "stdout" {
		t.Errorf("stream = %q, want stdout", stream)
	}
	if string(data) != "test output" {
		t.Errorf("data = %q, want test output", data)
	}
}

func TestExec_ExecResult_EmptyPayload(t *testing.T) {
	// Test with minimal payload
	frame, err := microvm.NewExecResultFrame("e-3", "sess-3", microvm.ExecResultPayload{
		ExitCode: 0,
	})
	if err != nil {
		t.Fatalf("NewExecResultFrame failed: %v", err)
	}

	result, err := microvm.ExecFramePayload(frame)
	if err != nil {
		t.Fatalf("ExecFramePayload failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit_code = %d, want 0", result.ExitCode)
	}
}

func TestExec_StreamPayload_InvalidFrameType(t *testing.T) {
	// Try to decode a non-stream frame as stream payload
	frame := microvm.Frame{
		Type:      microvm.FrameTypeRequest,
		ID:        "req-1",
		SessionID: "sess-1",
		Method:    "exec",
	}
	_, _, err := microvm.DecodeStreamPayload(frame)
	if err == nil {
		t.Fatal("expected error for non-stream frame type")
	}
}
