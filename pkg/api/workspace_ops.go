package api

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/pi-sandbox/pi/pkg/runtime/compat"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// workspaceRoot is the workspace mount point inside every sandbox.
const workspaceRoot = "/workspace"

// compatContainerHandle returns a handle to the sandbox's compat container.
// The container must already exist (created at sandbox creation).
func compatContainerHandle(id string) (*compat.Container, error) {
	if err := compat.EnsureRuntimeAvailable(); err != nil {
		return nil, fmt.Errorf("compat backend: %w", err)
	}
	name := compatContainerName(id)
	exists, err := compat.ContainerExists(name)
	if err != nil {
		return nil, fmt.Errorf("check container: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("sandbox container %s not found", name)
	}
	return &compat.Container{
		ID:    id,
		Spec:  &compat.ContainerSpec{ID: id, Name: name},
		Ready: true,
	}, nil
}

// resolveWorkspacePath normalizes a user-supplied path to an absolute path
// inside /workspace. Accepts "src/index.ts" or "/workspace/src/index.ts".
// Rejects traversal outside the workspace.
func resolveWorkspacePath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return workspaceRoot, nil
	}
	if !strings.HasPrefix(p, "/") {
		p = workspaceRoot + "/" + p
	}
	cleaned := path.Clean(p)
	if cleaned != workspaceRoot && !strings.HasPrefix(cleaned, workspaceRoot+"/") {
		return "", fmt.Errorf("path %q is outside %s", p, workspaceRoot)
	}
	return cleaned, nil
}

// shellQuote single-quotes a string for safe interpolation into sh -c.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// workspaceExec runs a shell command inside the sandbox container and
// returns stdout. A non-zero exit code is returned as an error carrying
// stderr for context.
func workspaceExec(ctx context.Context, id, command string) (string, error) {
	c, err := compatContainerHandle(id)
	if err != nil {
		return "", err
	}
	result, err := c.Exec(ctx, command)
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		msg := strings.TrimSpace(result.Stderr)
		if msg == "" {
			msg = strings.TrimSpace(result.Stdout)
		}
		return "", fmt.Errorf("exit %d: %s", result.ExitCode, msg)
	}
	return result.Stdout, nil
}

// requireCompat rejects modes whose workspace ops are not container-backed yet.
func requireCompat(meta *sandbox.Meta) error {
	if meta.Mode != "compat" && meta.Mode != "secure" {
		return fmt.Errorf("workspace operations require an OCI-backed sandbox (mode %q not supported yet)", meta.Mode)
	}
	return nil
}
