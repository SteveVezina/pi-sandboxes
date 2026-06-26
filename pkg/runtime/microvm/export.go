package microvm

import (
	"fmt"
	"os"
	"path/filepath"
)

// ArtifactExporter exports artifacts from a microVM sandbox through the
// guest control plane (ADR-001).
type ArtifactExporter struct {
	transfer *TransferClient
}

// NewArtifactExporter creates an exporter for the given session bound to the
// given control-plane sender (vsock client).
func NewArtifactExporter(sessionID string, sender FrameSender) *ArtifactExporter {
	return &ArtifactExporter{
		transfer: NewTransferClient(sessionID, sender),
	}
}

// ExportAll lists every artifact in the guest and writes each one to dest.
// Returns the slice of guest paths that were exported.
func (e *ArtifactExporter) ExportAll(dest string) ([]string, error) {
	if e == nil || e.transfer == nil {
		return nil, fmt.Errorf("artifact exporter not initialized")
	}
	if dest == "" {
		return nil, fmt.Errorf("artifact export destination is required")
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, fmt.Errorf("create export dir: %w", err)
	}

	paths, err := e.transfer.ListArtifacts()
	if err != nil {
		return nil, fmt.Errorf("artifact list: %w", err)
	}

	exported := make([]string, 0, len(paths))
	for _, guestPath := range paths {
		data, mode, err := e.transfer.PullArtifact(guestPath)
		if err != nil {
			return exported, fmt.Errorf("artifact pull %s: %w", guestPath, err)
		}
		hostPath := filepath.Join(dest, filepath.Base(guestPath))
		fileMode := os.FileMode(mode)
		if fileMode == 0 {
			fileMode = 0o644
		}
		if err := os.WriteFile(hostPath, data, fileMode); err != nil {
			return exported, fmt.Errorf("write artifact %s: %w", hostPath, err)
		}
		exported = append(exported, guestPath)
	}
	return exported, nil
}
