package microvm

import (
	"fmt"
	"sync/atomic"
)

// FrameSender sends one control frame and returns the guest response.
type FrameSender interface {
	Send(Frame) (Frame, error)
}

// TransferClient performs file and artifact transfer over the guest control plane.
type TransferClient struct {
	sandboxID string
	sender    FrameSender
	nextID    uint64
}

// NewTransferClient creates a transfer client scoped to one sandbox.
func NewTransferClient(sandboxID string, sender FrameSender) *TransferClient {
	return &TransferClient{sandboxID: sandboxID, sender: sender}
}

// ReadFile reads a file through the guest control plane.
func (c *TransferClient) ReadFile(path string) ([]byte, uint32, error) {
	frame, err := NewFileReadRequestFrame(c.requestID("file-read"), c.sandboxID, path)
	if err != nil {
		return nil, 0, err
	}
	resp, err := c.send(frame)
	if err != nil {
		return nil, 0, err
	}
	gotPath, data, mode, err := DecodeFileDataPayload(resp)
	if err != nil {
		return nil, 0, err
	}
	if gotPath != path {
		return nil, 0, fmt.Errorf("file response path %q does not match request %q", gotPath, path)
	}
	return data, mode, nil
}

// WriteFile writes a file through the guest control plane.
func (c *TransferClient) WriteFile(path string, data []byte, mode uint32) error {
	frame, err := NewFileWriteRequestFrame(c.requestID("file-write"), c.sandboxID, path, data, mode)
	if err != nil {
		return err
	}
	_, err = c.send(frame)
	return err
}

func (c *TransferClient) requestID(prefix string) string {
	id := atomic.AddUint64(&c.nextID, 1)
	return fmt.Sprintf("%s-%d", prefix, id)
}

func (c *TransferClient) send(frame Frame) (Frame, error) {
	if c.sender == nil {
		return Frame{}, fmt.Errorf("guest control transport is required")
	}
	resp, err := c.sender.Send(frame)
	if err != nil {
		return Frame{}, err
	}
	if resp.Error != nil {
		return Frame{}, fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
	}
	if resp.Type != FrameTypeResponse {
		return Frame{}, fmt.Errorf("unexpected transfer response type %q", resp.Type)
	}
	if resp.SandboxID != c.sandboxID {
		return Frame{}, fmt.Errorf("unexpected transfer response sandbox %q", resp.SandboxID)
	}
	return resp, nil
}
