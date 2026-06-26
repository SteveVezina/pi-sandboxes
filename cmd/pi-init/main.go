package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
)

func main() {
	sessionID := flag.String("session", "", "Sandbox session identifier")
	agentd := flag.String("agentd", "pi-agentd", "Path to pi-agentd")
	flag.Parse()

	if *sessionID == "" {
		fmt.Fprintln(os.Stderr, "session is required")
		os.Exit(2)
	}
	if err := run(context.Background(), *agentd, *sessionID); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, agentd, sessionID string) error {
	cmd := exec.CommandContext(ctx, agentd, "--session", sessionID)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("start pi-agentd: %w", err)
	}
	return nil
}
