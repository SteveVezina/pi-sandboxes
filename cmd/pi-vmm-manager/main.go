package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pi-sandbox/pi/pkg/runtime/microvm"
)

func main() {
	checkOnly := flag.Bool("check", false, "Check host microVM capability and exit")
	flag.Parse()

	availability := microvm.CheckAvailability(microvm.DefaultCapabilityChecker())
	if *checkOnly {
		if err := availability.Error(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("microvm available")
		return
	}

	fmt.Println("pi-vmm-manager")
	if err := availability.Error(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
