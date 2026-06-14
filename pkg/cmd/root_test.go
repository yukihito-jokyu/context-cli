package cmd

import "testing"

func TestNewCmdRootDelegatesErrorOutputToMain(t *testing.T) {
	root := NewCmdRoot(NewFactory())

	if !root.SilenceErrors {
		t.Error("SilenceErrors = false, want true")
	}
	if !root.SilenceUsage {
		t.Error("SilenceUsage = false, want true")
	}
}

func TestNewCmdRootRegistersAddCommand(t *testing.T) {
	root := NewCmdRoot(NewFactory())
	command, _, err := root.Find([]string{"add"})
	if err != nil {
		t.Fatalf("Find(add) error = %v", err)
	}
	if command.Name() != "add" {
		t.Fatalf("command.Name() = %q, want add", command.Name())
	}
}
