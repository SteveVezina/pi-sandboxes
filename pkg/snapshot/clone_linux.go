//go:build linux

package snapshot

import (
	"os"

	"golang.org/x/sys/unix"
)

// cloneFile makes dst a reflink (copy-on-write clone) of src using the
// Linux FICLONE ioctl. Returns an error on filesystems that don't support
// it (ext4, overlayfs) — the caller falls back to a plain copy.
func cloneFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	fi, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fi.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()

	return unix.IoctlFileClone(int(out.Fd()), int(in.Fd()))
}

func reflinkSupported() bool { return true }
