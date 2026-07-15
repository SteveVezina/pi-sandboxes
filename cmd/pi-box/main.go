package main

import (
	"os"

	_ "github.com/pi-sandbox/pi/cmd/pi-box/bench"
	_ "github.com/pi-sandbox/pi/cmd/pi-box/box"
	"github.com/pi-sandbox/pi/cmd/pi-box/cli"
	_ "github.com/pi-sandbox/pi/cmd/pi-box/context"
	_ "github.com/pi-sandbox/pi/cmd/pi-box/system"
	_ "github.com/pi-sandbox/pi/cmd/pi-box/template"
)

func main() {
	if err := cli.Root.Execute(); err != nil {
		os.Exit(1)
	}
}
