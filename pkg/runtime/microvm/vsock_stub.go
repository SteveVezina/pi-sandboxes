//go:build !linux
// +build !linux

package microvm

import "fmt"

// VsockConn is a stub for non-Linux platforms.
type VsockConn struct{}

// NewVsockConn returns an error on non-Linux.
func NewVsockConn(port uint32) (*VsockConn, error) {
	return nil, fmt.Errorf("vsock not available on non-Linux platforms")
}

// Close is a no-op on non-Linux.
func (c *VsockConn) Close() error {
	return nil
}

// SendFrame returns an error on non-Linux.
func (c *VsockConn) SendFrame(frame Frame) (Frame, error) {
	return Frame{}, fmt.Errorf("vsock not available on non-Linux platforms")
}

// SendEvent returns an error on non-Linux.
func (c *VsockConn) SendEvent(eventType, id, sessionID string) error {
	return fmt.Errorf("vsock not available on non-Linux platforms")
}

// StreamFrames returns empty channels on non-Linux.
func (c *VsockConn) StreamFrames() (<-chan Frame, <-chan error) {
	ch := make(chan Frame)
	errCh := make(chan error, 1)
	close(ch)
	errCh <- fmt.Errorf("vsock not available on non-Linux platforms")
	return ch, errCh
}

// Hello returns an error on non-Linux.
func (c *VsockConn) Hello() error {
	return fmt.Errorf("vsock not available on non-Linux platforms")
}

// Ready returns an error on non-Linux.
func (c *VsockConn) Ready(sessionID string) error {
	return fmt.Errorf("vsock not available on non-Linux platforms")
}

// Shutdown returns an error on non-Linux.
func (c *VsockConn) Shutdown(sessionID string) error {
	return fmt.Errorf("vsock not available on non-Linux platforms")
}

// Client is a stub for non-Linux platforms.
type Client struct {
	session string
}

// NewClient returns an error on non-Linux.
func NewClient(port uint32, sessionID string) (*Client, error) {
	return nil, fmt.Errorf("vsock not available on non-Linux platforms")
}

// Close is a no-op on non-Linux.
func (c *Client) Close() error {
	return nil
}

// Session returns the session ID.
func (c *Client) Session() string {
	return c.session
}

// Send returns an error on non-Linux.
func (c *Client) Send(frame Frame) (Frame, error) {
	return Frame{}, fmt.Errorf("vsock not available on non-Linux platforms")
}

// StreamFrames returns empty channels on non-Linux.
func (c *Client) StreamFrames() (<-chan Frame, <-chan error) {
	ch := make(chan Frame)
	errCh := make(chan error, 1)
	close(ch)
	errCh <- fmt.Errorf("vsock not available on non-Linux platforms")
	return ch, errCh
}
