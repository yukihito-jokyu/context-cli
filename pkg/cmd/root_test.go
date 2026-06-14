package cmd

import "testing"

func TestNewCmdRoot(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "DelegatesErrorOutputToMain",
			run: func(t *testing.T) {
				t.Helper()
				root := NewCmdRoot(NewFactory())
				if !root.SilenceErrors {
					t.Error("SilenceErrors = false, want true")
				}
				if !root.SilenceUsage {
					t.Error("SilenceUsage = false, want true")
				}
			},
		},
		{
			name: "RegistersAddCommand",
			run: func(t *testing.T) {
				t.Helper()
				root := NewCmdRoot(NewFactory())
				command, _, err := root.Find([]string{"add"})
				if err != nil {
					t.Fatalf("Find(add) error = %v", err)
				}
				if command.Name() != "add" {
					t.Fatalf("command.Name() = %q, want add", command.Name())
				}
			},
		},
		{
			name: "RegistersSyncCommand",
			run: func(t *testing.T) {
				t.Helper()
				root := NewCmdRoot(NewFactory())
				command, _, err := root.Find([]string{"sync"})
				if err != nil {
					t.Fatalf("Find(sync) error = %v", err)
				}
				if command.Name() != "sync" {
					t.Fatalf("command.Name() = %q, want sync", command.Name())
				}
				if err := command.Args(command, []string{"extra"}); err == nil {
					t.Fatal("Args(1) error = nil, want error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}
