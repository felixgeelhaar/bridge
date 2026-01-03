package cli_test

import (
	"testing"

	"github.com/felixgeelhaar/bridge/internal/interfaces/cli"
)

func TestNewApp(t *testing.T) {
	app := cli.NewApp()
	if app == nil {
		t.Fatal("expected non-nil app")
	}

	if app.Name != "bridge" {
		t.Errorf("expected name 'bridge', got %s", app.Name)
	}

	if app.Version != "0.1.0" {
		t.Errorf("expected version '0.1.0', got %s", app.Version)
	}
}

func TestNewApp_HasCommands(t *testing.T) {
	app := cli.NewApp()

	expectedCommands := []string{"init", "validate", "run", "status", "approve"}

	if len(app.Commands) != len(expectedCommands) {
		t.Errorf("expected %d commands, got %d", len(expectedCommands), len(app.Commands))
	}

	commandNames := make(map[string]bool)
	for _, cmd := range app.Commands {
		commandNames[cmd.Name] = true
	}

	for _, expected := range expectedCommands {
		if !commandNames[expected] {
			t.Errorf("missing command: %s", expected)
		}
	}
}

func TestNewApp_HasFlags(t *testing.T) {
	app := cli.NewApp()

	expectedFlags := []string{"config", "log-level", "output"}

	if len(app.Flags) != len(expectedFlags) {
		t.Errorf("expected %d flags, got %d", len(expectedFlags), len(app.Flags))
	}
}

func TestNewApp_Usage(t *testing.T) {
	app := cli.NewApp()

	if app.Usage == "" {
		t.Error("expected non-empty usage")
	}
}
