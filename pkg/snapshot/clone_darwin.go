//go:build darwin

package snapshot

import "golang.org/x/sys/unix"

// cloneFile makes dst a copy-on-write clone of src using APFS clonefile(2).
func cloneFile(src, dst string) error {
	return unix.Clonefile(src, dst, 0)
}

func reflinkSupported() bool { return true }
