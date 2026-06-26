package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CloneResult holds the result of a git clone operation.
type CloneResult struct {
	RepoURL   string
	LocalPath string
	TimedOut  bool
	DurationMs int64
}

// Clone clones a repository into the specified directory.
func Clone(ctx context.Context, url, localPath string) (*CloneResult, error) {
	start := time.Now()

	// Validate URL
	if err := validateURL(url); err != nil {
		return nil, err
	}

	// Create parent directory
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Run git clone
	cmd := exec.CommandContext(ctx, "git", "clone", url, localPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		timedOut := ctx.Err() == context.DeadlineExceeded
		return nil, fmt.Errorf("git clone failed (timedOut=%v): %s: %w",
			timedOut, string(output), err)
	}

	return &CloneResult{
		RepoURL:    url,
		LocalPath:  localPath,
		TimedOut:   false,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

// validateURL checks that the URL is safe to clone.
func validateURL(url string) error {
	// Reject dangerous protocols
	lower := strings.ToLower(url)
	if strings.HasPrefix(lower, "file://") {
		return fmt.Errorf("file:// protocol not allowed for security reasons")
	}
	if strings.HasPrefix(lower, "git://") {
		return fmt.Errorf("git:// protocol not allowed (use https://)")
	}
	if strings.HasPrefix(lower, "ssh://") || strings.HasPrefix(lower, "git@") {
		// SSH is allowed but requires brokered credentials
		// For now, allow it — credentials are handled by git credential helper
		return nil
	}
	if strings.HasPrefix(lower, "https://") {
		return nil
	}
	return fmt.Errorf("unsupported protocol: %s", url)
}
