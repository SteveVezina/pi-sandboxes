package microvm

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

const (
	FrameTypeRequest  = "request"
	FrameTypeResponse = "response"
	FrameTypeEvent    = "event"
	FrameTypeStream   = "stream"
)

var validFrameTypes = map[string]struct{}{
	FrameTypeRequest:  {},
	FrameTypeResponse: {},
	FrameTypeEvent:    {},
	FrameTypeStream:   {},
}

var validMethods = map[string]struct{}{
	"hello":         {},
	"ready":         {},
	"exec":          {},
	"file.read":     {},
	"file.write":    {},
	"artifact.list": {},
	"artifact.pull": {},
	"shutdown":      {},
}

// Frame is a newline-delimited JSON control frame exchanged over virtio-vsock.
type Frame struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	SessionID string          `json:"session_id"`
	Method    string          `json:"method,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Error     *FrameError     `json:"error,omitempty"`
}

// FrameError is the protocol error envelope for response frames.
type FrameError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// StreamPayload carries base64-encoded stdout/stderr bytes.
type StreamPayload struct {
	Stream string `json:"stream"`
	Data   string `json:"data"`
}

// ExecRequestPayload is the payload for an exec request frame.
type ExecRequestPayload struct {
	Command        string `json:"command"`
	Cwd            string `json:"cwd,omitempty"`
	TimeoutMs      int64  `json:"timeout_ms,omitempty"`
	MaxOutputBytes int64  `json:"max_output_bytes,omitempty"`
}

// ExecResultPayload is the final exec response payload.
type ExecResultPayload struct {
	ExitCode   int   `json:"exit_code"`
	DurationMs int64 `json:"duration_ms"`
	TimedOut   bool  `json:"timed_out"`
	Truncated  bool  `json:"truncated"`
}

// EncodeFrame writes one validated JSON frame followed by a newline.
func EncodeFrame(w io.Writer, frame Frame) error {
	if err := frame.Validate(); err != nil {
		return err
	}
	encoded, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("encode frame: %w", err)
	}
	if _, err := w.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("write frame: %w", err)
	}
	return nil
}

// DecodeFrame reads one newline-delimited JSON frame and validates it.
func DecodeFrame(r *bufio.Reader) (Frame, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return Frame{}, fmt.Errorf("read frame: %w", err)
	}
	var frame Frame
	if err := json.Unmarshal(line, &frame); err != nil {
		return Frame{}, fmt.Errorf("decode frame: %w", err)
	}
	if err := frame.Validate(); err != nil {
		return Frame{}, err
	}
	return frame, nil
}

// Validate checks the cross-component protocol contract from ADR-002.
func (f Frame) Validate() error {
	if _, ok := validFrameTypes[f.Type]; !ok {
		return fmt.Errorf("invalid frame type %q", f.Type)
	}
	if f.ID == "" {
		return fmt.Errorf("frame id is required")
	}
	if f.SessionID == "" {
		return fmt.Errorf("frame session_id is required")
	}
	if f.Type != FrameTypeStream {
		if _, ok := validMethods[f.Method]; !ok {
			return fmt.Errorf("invalid frame method %q", f.Method)
		}
	}
	return nil
}

// NewStreamFrame creates a stdout/stderr stream frame with base64 payload data.
func NewStreamFrame(id, sessionID, stream string, data []byte) (Frame, error) {
	if stream != "stdout" && stream != "stderr" {
		return Frame{}, fmt.Errorf("invalid stream %q", stream)
	}
	payload, err := json.Marshal(StreamPayload{
		Stream: stream,
		Data:   base64.StdEncoding.EncodeToString(data),
	})
	if err != nil {
		return Frame{}, fmt.Errorf("encode stream payload: %w", err)
	}
	return Frame{
		Type:      FrameTypeStream,
		ID:        id,
		SessionID: sessionID,
		Payload:   payload,
	}, nil
}

// DecodeStreamPayload decodes a stream frame into its stream name and bytes.
func DecodeStreamPayload(frame Frame) (string, []byte, error) {
	if frame.Type != FrameTypeStream {
		return "", nil, fmt.Errorf("frame type %q is not stream", frame.Type)
	}
	var payload StreamPayload
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		return "", nil, fmt.Errorf("decode stream payload: %w", err)
	}
	if payload.Stream != "stdout" && payload.Stream != "stderr" {
		return "", nil, fmt.Errorf("invalid stream %q", payload.Stream)
	}
	data, err := base64.StdEncoding.DecodeString(payload.Data)
	if err != nil {
		return "", nil, fmt.Errorf("decode stream data: %w", err)
	}
	return payload.Stream, data, nil
}

// NewExecRequestFrame creates an exec request frame.
func NewExecRequestFrame(id, sessionID string, payload ExecRequestPayload) (Frame, error) {
	if payload.Command == "" {
		return Frame{}, fmt.Errorf("exec command is required")
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return Frame{}, fmt.Errorf("encode exec request payload: %w", err)
	}
	return Frame{
		Type:      FrameTypeRequest,
		ID:        id,
		SessionID: sessionID,
		Method:    "exec",
		Payload:   encoded,
	}, nil
}

// NewExecResultFrame creates the final exec response frame.
func NewExecResultFrame(id, sessionID string, payload ExecResultPayload) (Frame, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return Frame{}, fmt.Errorf("encode exec result payload: %w", err)
	}
	return Frame{
		Type:      FrameTypeResponse,
		ID:        id,
		SessionID: sessionID,
		Method:    "exec",
		Payload:   encoded,
	}, nil
}
