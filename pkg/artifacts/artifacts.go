package artifacts

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ArtifactInfo holds metadata about a single artifact file.
type ArtifactInfo struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	IsDir   bool      `json:"isDir"`
}

// ArtifactManager manages artifact operations for a sandbox session.
type ArtifactManager struct {
	workspaceDir string
}

// Known artifact locations relative to workspace.
var knownLocations = []string{
	"artifacts",
	"workspace/dist",
	"workspace/build",
	"workspace/coverage",
	"workspace/test-results",
	"workspace/target/release",
}

// NewManager creates an artifact manager for the given sandbox workspace.
func NewManager(workspaceDir string) *ArtifactManager {
	return &ArtifactManager{workspaceDir: workspaceDir}
}

// List returns all artifact files from known locations.
func (m *ArtifactManager) List() ([]ArtifactInfo, error) {
	var results []ArtifactInfo

	for _, loc := range knownLocations {
		absPath := filepath.Join(m.workspaceDir, loc)
		info, err := os.Stat(absPath)
		if err != nil {
			continue // Location doesn't exist, skip
		}

		if info.IsDir() {
			files, err := scanDir(absPath, loc)
			if err != nil {
				return nil, fmt.Errorf("scan %s: %w", loc, err)
			}
			// Only include files, exclude directories
			for _, f := range files {
				if !f.IsDir {
					results = append(results, f)
				}
			}
		} else {
			results = append(results, ArtifactInfo{
				Path:    loc,
				Size:    info.Size(),
				ModTime: info.ModTime(),
				IsDir:   false,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})

	return results, nil
}

// Pull copies artifacts from known locations to a host destination.
func (m *ArtifactManager) Pull(hostDest string) error {
	for _, loc := range knownLocations {
		src := filepath.Join(m.workspaceDir, loc)
		dst := filepath.Join(hostDest, loc)

		info, err := os.Stat(src)
		if err != nil {
			continue // Skip non-existent locations
		}

		if info.IsDir() {
			if err := copyDir(src, dst); err != nil {
				return fmt.Errorf("copy %s: %w", loc, err)
			}
		}
	}
	return nil
}

// Pack creates a tar archive of all artifacts.
func (m *ArtifactManager) Pack(outputPath string) error {
	// Collect all artifact files
	var files []string
	for _, loc := range knownLocations {
		src := filepath.Join(m.workspaceDir, loc)
		info, err := os.Stat(src)
		if err != nil {
			continue
		}
		if info.IsDir() {
			entries, err := os.ReadDir(src)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				files = append(files, filepath.Join(src, entry.Name()))
			}
		}
	}

	if len(files) == 0 {
		return createEmptyFile(outputPath)
	}

	return createTarArchive(outputPath, files)
}

// scanDir recursively scans a directory and returns ArtifactInfo entries.
func scanDir(dir, relPrefix string) ([]ArtifactInfo, error) {
	var results []ArtifactInfo

	// Strip trailing separator for consistent path handling
	dir = filepath.Clean(dir)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		var relPath string
		if path == dir {
			relPath = relPrefix
		} else {
			// Compute relative path from dir
			rel := path[len(dir):]
			if len(rel) > 0 && rel[0] == filepath.Separator {
				rel = rel[1:]
			}
			relPath = filepath.Join(relPrefix, rel)
		}

		results = append(results, ArtifactInfo{
			Path:    relPath,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
		})
		return nil
	})

	return results, err
}

// copyDir copies a directory recursively.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat src: %w", err)
	}

	if srcInfo.IsDir() {
		if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
			return fmt.Errorf("mkdir dst: %w", err)
		}

		entries, err := os.ReadDir(src)
		if err != nil {
			return fmt.Errorf("read dir: %w", err)
		}

		for _, entry := range entries {
			srcPath := filepath.Join(src, entry.Name())
			dstPath := filepath.Join(dst, entry.Name())
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		}
		return nil
	}

	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
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

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// createEmptyFile creates an empty file.
func createEmptyFile(path string) error {
	return os.WriteFile(path, []byte{}, 0644)
}

// createTarArchive creates a tar archive of the given files.
func createTarArchive(path string, files []string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	tw := tar.NewWriter(f)
	defer tw.Close()

	for _, file := range files {
		if err := addToTar(tw, file); err != nil {
			return err
		}
	}

	return tw.Close()
}

// addToTar adds a single file to the tar writer.
func addToTar(tw *tar.Writer, file string) error {
	info, err := os.Stat(file)
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name:    filepath.Base(file),
		Mode:    int64(info.Mode().Perm()),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	src, err := os.Open(file)
	if err != nil {
		return err
	}
	defer src.Close()

	_, err = io.Copy(tw, src)
	return err
}
