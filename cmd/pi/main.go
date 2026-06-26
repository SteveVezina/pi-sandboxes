package main

import (
	"os"

	_ "github.com/pi-sandbox/pi/cmd/pi/box"
	_ "github.com/pi-sandbox/pi/cmd/pi/bench"
	"github.com/pi-sandbox/pi/cmd/pi/cli"
	_ "github.com/pi-sandbox/pi/cmd/pi/system"
	_ "github.com/pi-sandbox/pi/cmd/pi/template"
)

func main() {
	if err := cli.Root.Execute(); err != nil {
		os.Exit(1)
	}
}
