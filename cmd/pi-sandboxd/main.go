package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pi-sandbox/pi/pkg/daemon"
	"github.com/pi-sandbox/pi/pkg/session"
	"github.com/pi-sandbox/pi/pkg/system"
)

func main() {
	socketPath := flag.String("socket", "", "Unix socket path (default: ~/.pi-box/sandboxd.sock)")
	httpPort := flag.Int("http-port", 0, "HTTP port for development (0 = disabled)")
	flag.Parse()

	if *socketPath == "" {
		*socketPath = filepath.Join(system.PiHome(), "sandboxd.sock")
	}

	// Create session store
	storeDir := filepath.Join(system.PiHome(), "sandboxes")
	store := session.NewStore(storeDir)

	d := daemon.New(*socketPath, *httpPort, store)

	if err := d.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: daemon failed to start: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("pi-sandboxd listening on %s\n", *socketPath)

	// Block until stopped
	select {}
}
