package git

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// DiffResult holds the result of a git diff operation.
type DiffResult struct {
	Diff       string
	TimedOut   bool
	DurationMs int64
}

// Diff computes the git diff of the workspace.
func Diff(ctx context.Context, workspaceDir string) (*DiffResult, error) {
	start := time.Now()

	// Run git diff (unstaged + staged)
	cmd := exec.CommandContext(ctx, "git", "-C", workspaceDir, "diff", "--cached", "--unified=3")
	output, err := cmd.CombinedOutput()
	if err != nil {
		timedOut := ctx.Err() == context.DeadlineExceeded
		return nil, fmt.Errorf("git diff failed (timedOut=%v): %w",
			timedOut, err)
	}

	return &DiffResult{
		Diff:       string(output),
		TimedOut:   false,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}
