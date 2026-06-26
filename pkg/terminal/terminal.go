package terminal

import (
	"io"
	"os"

	"golang.org/x/term"
)

// MakeRaw puts the terminal attached to fd into raw mode.
// Returns the prior state so the caller can defer Restore.
func MakeRaw(fd int) (*term.State, error) {
	return term.MakeRaw(fd)
}

// Restore restores a terminal to a prior state.
func Restore(fd int, state *term.State) error {
	return term.Restore(fd, state)
}

// IsTerminal reports whether fd is connected to a terminal.
func IsTerminal(fd int) bool {
	return term.IsTerminal(fd)
}

// GetSize returns the terminal dimensions for fd.
func GetSize(fd int) (width, height int, err error) {
	return term.GetSize(fd)
}

// RawCopy copies between a WebSocket-style io.ReadWriter and the terminal in
// raw mode. It connects:
//
//	os.Stdin → dst (send user input to remote)
//	src      → os.Stdout (display remote output locally)
//
// The function blocks until either side closes or returns an error.
// The caller is responsible for restoring the terminal after RawCopy returns.
func RawCopy(src io.Reader, dst io.Writer) error {
	errc := make(chan error, 2)

	go func() {
		_, err := io.Copy(dst, os.Stdin)
		errc <- err
	}()
	go func() {
		_, err := io.Copy(os.Stdout, src)
		errc <- err
	}()

	// Return when either direction finishes (remote closed or user Ctrl-D).
	err := <-errc
	return err
}
