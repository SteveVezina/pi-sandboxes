package workspace

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Pull copies files from the workspace to a host destination.
func (m *Manager) Pull(relPath, hostDest string) error {
	srcAbs, err := m.ValidatePath(relPath)
	if err != nil {
		return err
	}

	return copyRecursive(srcAbs, hostDest)
}

// copyRecursive copies a file or directory from src to dst.
func copyRecursive(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat src: %w", err)
	}

	// If dst exists and is a directory, append src filename
	if srcInfo.IsDir() {
		return copyDirectory(src, dst, srcInfo)
	}
	// If dst is a directory, append the source filename
	dstInfo, dstErr := os.Stat(dst)
	if dstErr == nil && dstInfo.IsDir() {
		dst = filepath.Join(dst, srcInfo.Name())
	}
	return copyFile(src, dst)
}

// copyDirectory copies a directory recursively.
func copyDirectory(srcDir, dstDir string, srcInfo os.FileInfo) error {
	if err := os.MkdirAll(dstDir, srcInfo.Mode()); err != nil {
		return fmt.Errorf("mkdir dst: %w", err)
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		if err := copyRecursive(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

// copyFile copies a single file.
func copyFile(src, dst string) error {
	// Ensure destination parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("mkdir dst parent: %w", err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	return dstFile.Sync()
}
