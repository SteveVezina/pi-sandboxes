package git

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// PatchResult holds the result of a git patch export.
type PatchResult struct {
	Patch      string
	TimedOut   bool
	DurationMs int64
}

// Patch exports workspace changes as a git patch.
func Patch(ctx context.Context, workspaceDir string) (*PatchResult, error) {
	start := time.Now()

	// Run git diff --cached (staged changes)
	cmd := exec.CommandContext(ctx, "git", "-C", workspaceDir, "diff", "--cached", "--unified=3")
	output, err := cmd.CombinedOutput()
	if err != nil {
		timedOut := ctx.Err() == context.DeadlineExceeded
		return nil, fmt.Errorf("git patch failed (timedOut=%v): %w",
			timedOut, err)
	}

	return &PatchResult{
		Patch:      string(output),
		TimedOut:   false,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}
