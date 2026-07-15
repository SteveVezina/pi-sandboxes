package cli_test

import (
	"bytes"
	"strings"
	"testing"

	_ "github.com/pi-sandbox/pi/cmd/pi-box/bench"
	_ "github.com/pi-sandbox/pi/cmd/pi-box/box"
	"github.com/pi-sandbox/pi/cmd/pi-box/cli"
	_ "github.com/pi-sandbox/pi/cmd/pi-box/system"
	_ "github.com/pi-sandbox/pi/cmd/pi-box/template"
)

func TestRootHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	cli.Root.SetOut(buf)
	cli.Root.SetArgs([]string{"--help"})
	cli.Root.SetErr(buf)

	if err := cli.Root.Execute(); err != nil {
		t.Fatalf("Root command failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("Expected help output, got empty string")
	}
}

func TestBoxSubcommand(t *testing.T) {
	found := false
	for _, cmd := range cli.Root.Commands() {
		if cmd.Use == "box" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Expected 'box' subcommand to be registered")
	}
}

func TestSystemSubcommand(t *testing.T) {
	found := false
	for _, cmd := range cli.Root.Commands() {
		if cmd.Use == "system" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Expected 'system' subcommand to be registered")
	}
}

func TestBenchSubcommand(t *testing.T) {
	found := false
	for _, cmd := range cli.Root.Commands() {
		if cmd.Use == "bench" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Expected 'bench' subcommand to be registered")
	}
}

func TestTemplateSubcommand(t *testing.T) {
	found := false
	for _, cmd := range cli.Root.Commands() {
		if cmd.Use == "template" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Expected 'template' subcommand to be registered")
	}
}

func TestBoxHasAllSubcommands(t *testing.T) {
	boxCmd, _, _ := cli.Root.Find([]string{"box"})
	if boxCmd == nil {
		t.Fatal("box command not found")
	}

	t.Logf("boxCmd.Commands() count: %d", len(boxCmd.Commands()))
	for _, c := range boxCmd.Commands() {
		t.Logf("  - %s", c.Use)
	}

	expected := []string{"create", "list", "inspect", "destroy", "clone", "exec", "shell", "files", "diff", "patch", "artifacts", "snapshot", "logs"}
	for _, name := range expected {
		found := false
		for _, cmd := range boxCmd.Commands() {
			if strings.HasPrefix(cmd.Use, name) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected box subcommand '%s' to be registered", name)
		}
	}
}
