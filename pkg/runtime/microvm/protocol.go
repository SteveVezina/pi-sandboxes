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
	SandboxID string          `json:"sandbox_id"`
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

// FileReadRequestPayload is the payload for file.read requests.
type FileReadRequestPayload struct {
	Path string `json:"path"`
}

// FileWriteRequestPayload is the payload for file.write requests.
type FileWriteRequestPayload struct {
	Path string `json:"path"`
	Data string `json:"data"`
	Mode uint32 `json:"mode,omitempty"`
}

// FileDataPayload carries base64-encoded file or artifact bytes.
type FileDataPayload struct {
	Path string `json:"path"`
	Data string `json:"data"`
	Mode uint32 `json:"mode,omitempty"`
}

// ArtifactPullRequestPayload is the payload for artifact.pull requests.
type ArtifactPullRequestPayload struct {
	Path string `json:"path"`
}

// ArtifactListPayload is a response payload for artifact.list.
type ArtifactListPayload struct {
	Paths []string `json:"paths"`
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
	if f.SandboxID == "" {
		return fmt.Errorf("frame sandbox_id is required")
	}
	if f.Type != FrameTypeStream {
		if _, ok := validMethods[f.Method]; !ok {
			return fmt.Errorf("invalid frame method %q", f.Method)
		}
	}
	return nil
}

// NewStreamFrame creates a stdout/stderr stream frame with base64 payload data.
func NewStreamFrame(id, sandboxID, stream string, data []byte) (Frame, error) {
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
		SandboxID: sandboxID,
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
func NewExecRequestFrame(id, sandboxID string, payload ExecRequestPayload) (Frame, error) {
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
		SandboxID: sandboxID,
		Method:    "exec",
		Payload:   encoded,
	}, nil
}

// NewExecResultFrame creates the final exec response frame.
func NewExecResultFrame(id, sandboxID string, payload ExecResultPayload) (Frame, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return Frame{}, fmt.Errorf("encode exec result payload: %w", err)
	}
	return Frame{
		Type:      FrameTypeResponse,
		ID:        id,
		SandboxID: sandboxID,
		Method:    "exec",
		Payload:   encoded,
	}, nil
}

// NewFileReadRequestFrame creates a file.read request frame.
func NewFileReadRequestFrame(id, sandboxID, path string) (Frame, error) {
	if path == "" {
		return Frame{}, fmt.Errorf("file path is required")
	}
	payload, err := json.Marshal(FileReadRequestPayload{Path: path})
	if err != nil {
		return Frame{}, fmt.Errorf("encode file read payload: %w", err)
	}
	return Frame{
		Type:      FrameTypeRequest,
		ID:        id,
		SandboxID: sandboxID,
		Method:    "file.read",
		Payload:   payload,
	}, nil
}

// NewFileWriteRequestFrame creates a file.write request frame.
func NewFileWriteRequestFrame(id, sandboxID, path string, data []byte, mode uint32) (Frame, error) {
	if path == "" {
		return Frame{}, fmt.Errorf("file path is required")
	}
	payload, err := json.Marshal(FileWriteRequestPayload{
		Path: path,
		Data: base64.StdEncoding.EncodeToString(data),
		Mode: mode,
	})
	if err != nil {
		return Frame{}, fmt.Errorf("encode file write payload: %w", err)
	}
	return Frame{
		Type:      FrameTypeRequest,
		ID:        id,
		SandboxID: sandboxID,
		Method:    "file.write",
		Payload:   payload,
	}, nil
}

// NewArtifactListRequestFrame creates an artifact.list request frame.
func NewArtifactListRequestFrame(id, sandboxID string) (Frame, error) {
	frame := Frame{
		Type:      FrameTypeRequest,
		ID:        id,
		SandboxID: sandboxID,
		Method:    "artifact.list",
	}
	if err := frame.Validate(); err != nil {
		return Frame{}, err
	}
	return frame, nil
}

// NewArtifactPullRequestFrame creates an artifact.pull request frame.
func NewArtifactPullRequestFrame(id, sandboxID, path string) (Frame, error) {
	if path == "" {
		return Frame{}, fmt.Errorf("artifact path is required")
	}
	payload, err := json.Marshal(ArtifactPullRequestPayload{Path: path})
	if err != nil {
		return Frame{}, fmt.Errorf("encode artifact pull payload: %w", err)
	}
	return Frame{
		Type:      FrameTypeRequest,
		ID:        id,
		SandboxID: sandboxID,
		Method:    "artifact.pull",
		Payload:   payload,
	}, nil
}

// DecodeFileDataPayload decodes base64 file or artifact bytes from a response frame.
func DecodeFileDataPayload(frame Frame) (string, []byte, uint32, error) {
	if frame.Type != FrameTypeResponse {
		return "", nil, 0, fmt.Errorf("frame type %q is not response", frame.Type)
	}
	var payload FileDataPayload
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		return "", nil, 0, fmt.Errorf("decode file data payload: %w", err)
	}
	if payload.Path == "" {
		return "", nil, 0, fmt.Errorf("file path is required")
	}
	data, err := base64.StdEncoding.DecodeString(payload.Data)
	if err != nil {
		return "", nil, 0, fmt.Errorf("decode file data: %w", err)
	}
	return payload.Path, data, payload.Mode, nil
}

// EncodeStreamData encodes data as base64 for a stream frame.
func EncodeStreamData(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeStreamData decodes base64 data from a stream frame.
func DecodeStreamData(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

// NewHelloFrame creates a hello request frame.
func NewHelloFrame(id, sandboxID string) (Frame, error) {
	frame := Frame{
		Type:      FrameTypeRequest,
		ID:        id,
		SandboxID: sandboxID,
		Method:    "hello",
	}
	if err := frame.Validate(); err != nil {
		return Frame{}, err
	}
	return frame, nil
}

// NewReadyFrame creates a ready event frame.
func NewReadyFrame(id, sandboxID string) (Frame, error) {
	frame := Frame{
		Type:      FrameTypeEvent,
		ID:        id,
		SandboxID: sandboxID,
		Method:    "ready",
	}
	if err := frame.Validate(); err != nil {
		return Frame{}, err
	}
	return frame, nil
}

// NewShutdownFrame creates a shutdown event frame.
func NewShutdownFrame(id, sandboxID string) (Frame, error) {
	frame := Frame{
		Type:      FrameTypeEvent,
		ID:        id,
		SandboxID: sandboxID,
		Method:    "shutdown",
	}
	if err := frame.Validate(); err != nil {
		return Frame{}, err
	}
	return frame, nil
}

// ExecFramePayload decodes an exec result frame into its payload.
func ExecFramePayload(frame Frame) (ExecResultPayload, error) {
	var payload ExecResultPayload
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		return ExecResultPayload{}, fmt.Errorf("decode exec result payload: %w", err)
	}
	return payload, nil
}

// StreamFramePayload decodes a stream frame into its stream name and data.
func StreamFramePayload(frame Frame) (string, []byte, error) {
	return DecodeStreamPayload(frame)
}
