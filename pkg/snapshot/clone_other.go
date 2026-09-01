//go:build !linux && !darwin

package snapshot

import "fmt"

func cloneFile(src, dst string) error { return fmt.Errorf("reflink not supported on this platform") }

func reflinkSupported() bool { return false }
