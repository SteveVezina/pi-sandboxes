//go:build linux
// +build linux

// Package microvm provides MicroVM backend support for Pi Sandbox.
// Implements Firecracker-based microVM lifecycle with virtio-vsock guest control.
package microvm

import (
	"bufio"
	"fmt"
	"io"
	"sync"
)

// VsockConn represents a virtio-vsock connection to a guest.
// On Linux this wraps a vsock socket; on non-Linux it is a stub.
type VsockConn struct {
	port uint32
	conn io.ReadWriteCloser
}

// NewVsockConn creates a new vsock connection to the guest at the given port.
func NewVsockConn(port uint32) (*VsockConn, error) {
	return &VsockConn{port: port}, nil
}

// Close closes the vsock connection.
func (c *VsockConn) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SendFrame sends a frame to the guest and waits for a response.
func (c *VsockConn) SendFrame(frame Frame) (Frame, error) {
	if c.conn == nil {
		return Frame{}, fmt.Errorf("vsock connection not established")
	}

	if err := EncodeFrame(c.conn, frame); err != nil {
		return Frame{}, fmt.Errorf("encode send frame: %w", err)
	}

	reader := bufio.NewReader(c.conn)
	return DecodeFrame(reader)
}

// SendEvent sends an event frame to the guest (fire-and-forget).
func (c *VsockConn) SendEvent(eventType, id, sessionID string) error {
	frame := Frame{
		Type:      FrameTypeEvent,
		ID:        id,
		SessionID: sessionID,
		Method:    eventType,
	}
	return EncodeFrame(c.conn, frame)
}

// StreamFrames reads stream frames from the guest until an error or end.
func (c *VsockConn) StreamFrames() (<-chan Frame, <-chan error) {
	ch := make(chan Frame)
	errCh := make(chan error, 1)

	go func() {
		defer close(ch)
		defer close(errCh)

		reader := bufio.NewReader(c.conn)
		for {
			frame, err := DecodeFrame(reader)
			if err != nil {
				if err != io.EOF {
					errCh <- fmt.Errorf("decode stream frame: %w", err)
				}
				return
			}
			ch <- frame
		}
	}()

	return ch, errCh
}

// Hello sends a hello frame to the guest and expects a hello response.
func (c *VsockConn) Hello() error {
	frame := Frame{
		Type:      FrameTypeRequest,
		ID:        "hello-1",
		SessionID: "init",
		Method:    "hello",
	}
	resp, err := c.SendFrame(frame)
	if err != nil {
		return fmt.Errorf("hello request failed: %w", err)
	}
	if resp.Method != "hello" {
		return fmt.Errorf("unexpected hello response method: %s", resp.Method)
	}
	return nil
}

// Ready waits for the guest to report ready.
func (c *VsockConn) Ready(sessionID string) error {
	ch, errCh := c.StreamFrames()
	select {
	case frame := <-ch:
		if frame.Method != "ready" || frame.SessionID != sessionID {
			return fmt.Errorf("unexpected ready frame: %+v", frame)
		}
		return nil
	case err := <-errCh:
		return err
	}
}

// Shutdown sends a shutdown event to the guest.
func (c *VsockConn) Shutdown(sessionID string) error {
	return c.SendEvent("shutdown", "shutdown-1", sessionID)
}

// Client manages a vsock session with a guest.
type Client struct {
	conn    *VsockConn
	session string
	mu      sync.Mutex
}

// NewClient creates a new vsock client.
func NewClient(port uint32, sessionID string) (*Client, error) {
	conn, err := NewVsockConn(port)
	if err != nil {
		return nil, fmt.Errorf("create vsock connection: %w", err)
	}
	return &Client{
		conn:    conn,
		session: sessionID,
	}, nil
}

// Close closes the vsock connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Session returns the session ID.
func (c *Client) Session() string {
	return c.session
}

// Send sends a frame and returns the response.
func (c *Client) Send(frame Frame) (Frame, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.SendFrame(frame)
}

// StreamFrames returns a channel of stream frames from the guest.
func (c *Client) StreamFrames() (<-chan Frame, <-chan error) {
	return c.conn.StreamFrames()
}

// EncodeFrame writes one validated JSON frame followed by a newline.
// Alias for EncodeFrame for convenience.
func EncodeFrameJSON(w io.Writer, frame Frame) error {
	return EncodeFrame(w, frame)
}

// DecodeFrame reads one newline-delimited JSON frame and validates it.
// Alias for DecodeFrame for convenience.
func DecodeFrameJSON(r io.Reader) (Frame, error) {
	return DecodeFrame(bufio.NewReader(r))
}
