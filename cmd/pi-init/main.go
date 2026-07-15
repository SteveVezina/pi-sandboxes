package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
)

func main() {
	sandboxID := flag.String("sandbox", "", "Sandbox identifier")
	agentd := flag.String("agentd", "pi-agentd", "Path to pi-agentd")
	flag.Parse()

	if *sandboxID == "" {
		fmt.Fprintln(os.Stderr, "sandbox is required")
		os.Exit(2)
	}
	if err := run(context.Background(), *agentd, *sandboxID); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, agentd, sandboxID string) error {
	cmd := exec.CommandContext(ctx, agentd, "--sandbox", sandboxID)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("start pi-agentd: %w", err)
	}
	return nil
}
