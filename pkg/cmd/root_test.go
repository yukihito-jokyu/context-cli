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
