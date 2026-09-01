package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pi-sandbox/pi/pkg/daemon"
	"github.com/pi-sandbox/pi/pkg/events"
	"github.com/pi-sandbox/pi/pkg/sandbox"
	"github.com/pi-sandbox/pi/pkg/system"
)

func main() {
	socketPath := flag.String("socket", "", "Unix socket path (default: $PI_SOCKET_PATH or ~/.pi-box/sandboxd.sock)")
	httpPort := flag.Int("http-port", 0, "HTTP port (0 = disabled; falls back to $PORT)")
	httpAddr := flag.String("http-addr", "", "HTTP bind host (default 127.0.0.1; falls back to $PI_HTTP_ADDR)")
	egressProxyPort := flag.Int("egress-proxy-port", 0, "daemon egress proxy port on 127.0.0.1 (0 = disabled)")
	eventsWebhook := flag.String("events-webhook", "", "POST lifecycle events as JSON to this URL (ADR-007)")
	flag.Parse()

	// Environment fallbacks for PaaS / container deploys (F23 T23.5).
	if *socketPath == "" {
		*socketPath = os.Getenv("PI_SOCKET_PATH")
	}
	if *httpPort == 0 {
		if p, err := strconv.Atoi(os.Getenv("PORT")); err == nil && p > 0 {
			*httpPort = p
		}
	}
	if *httpAddr == "" {
		*httpAddr = os.Getenv("PI_HTTP_ADDR")
	}
	// Bearer token comes from the environment only — never a flag (argv leaks
	// to `ps` and shell history). ADR-003.
	authToken := os.Getenv("PI_DAEMON_TOKEN")

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
	d.SetHTTPHost(*httpAddr)
	d.SetAuthToken(authToken)

	if err := d.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: daemon failed to start: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("pi-sandboxd listening on %s\n", *socketPath)
	if *httpPort > 0 {
		host := *httpAddr
		if host == "" {
			host = "127.0.0.1"
		}
		authState := "no auth (loopback trust)"
		if authToken != "" {
			authState = "bearer-token auth enforced"
		}
		fmt.Printf("pi-sandboxd HTTP on %s:%d — %s\n", host, *httpPort, authState)
	}

	// Block until stopped
	select {}
}
