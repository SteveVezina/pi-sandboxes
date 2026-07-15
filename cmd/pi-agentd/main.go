package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/pi-sandbox/pi/pkg/runtime/microvm"
)

func main() {
	sandboxID := flag.String("sandbox", "", "Sandbox identifier")
	flag.Parse()

	if *sandboxID == "" {
		fmt.Fprintln(os.Stderr, "sandbox is required")
		os.Exit(2)
	}
	if err := run(*sandboxID, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(sandboxID string, stdin io.Reader, stdout io.Writer) error {
	if err := microvm.WriteReady(stdout, sandboxID); err != nil {
		return fmt.Errorf("write ready: %w", err)
	}

	reader := bufio.NewReader(stdin)
	for {
		frame, err := microvm.DecodeFrame(reader)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("decode control frame: %w", err)
		}
		if frame.Method == "shutdown" && frame.SandboxID == sandboxID {
			return nil
		}
	}
}
