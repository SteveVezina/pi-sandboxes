package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pi-sandbox/pi/pkg/daemon"
	"github.com/pi-sandbox/pi/pkg/events"
	"github.com/pi-sandbox/pi/pkg/sandbox"
	"github.com/pi-sandbox/pi/pkg/system"
)

func main() {
	socketPath := flag.String("socket", "", "Unix socket path (default: ~/.pi-box/sandboxd.sock)")
	httpPort := flag.Int("http-port", 0, "HTTP port for development (0 = disabled)")
	egressProxyPort := flag.Int("egress-proxy-port", 0, "daemon egress proxy port on 127.0.0.1 (0 = disabled)")
	eventsWebhook := flag.String("events-webhook", "", "POST lifecycle events as JSON to this URL (ADR-007)")
	flag.Parse()

	if *eventsWebhook != "" {
		events.SetDefault(events.New(events.SlogSink{}, events.NewWebhookSink(*eventsWebhook)))
	}

	if *socketPath == "" {
		*socketPath = filepath.Join(system.PiHome(), "sandboxd.sock")
	}

	// Create sandbox store
	storeDir := filepath.Join(system.PiHome(), "sandboxes")
	store := sandbox.NewStore(storeDir)
	runStore := sandbox.NewAgentRunStore()

	d := daemon.New(*socketPath, *httpPort, store, runStore)
	d.SetEgressProxyPort(*egressProxyPort)

	if err := d.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: daemon failed to start: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("pi-sandboxd listening on %s\n", *socketPath)

	// Block until stopped
	select {}
}
