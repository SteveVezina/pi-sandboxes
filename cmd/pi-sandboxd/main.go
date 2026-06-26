package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pi-sandbox/pi/pkg/daemon"
)

func main() {
	socketPath := flag.String("socket", "", "Unix socket path (default: ~/.pi/sandboxd.sock)")
	httpPort := flag.Int("http-port", 0, "HTTP port for development (0 = disabled)")
	flag.Parse()

	if *socketPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: could not determine home directory: %v\n", err)
			os.Exit(1)
		}
		*socketPath = filepath.Join(home, ".pi", "sandboxd.sock")
	}

	d := daemon.New(*socketPath, *httpPort)

	if err := d.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: daemon failed to start: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("pi-sandboxd listening on %s\n", *socketPath)

	// Block until stopped
	select {}
}
