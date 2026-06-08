package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

type commandFunc func(context.Context, []string) int

func (f commandFunc) Run(ctx context.Context, args []string) int {
	return f(ctx, args)
}

func equalSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestHandlerRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantArgs   []string
		wantStderr string
	}{
		{
			name:     "dispatches command",
			args:     []string{"init", "."},
			wantCode: ExitSuccess,
			wantArgs: []string{"."},
		},
		{
			name:       "requires command",
			wantCode:   ExitUsage,
			wantStderr: "Usage: context <command> [arguments]",
		},
		{
			name:       "rejects unknown command",
			args:       []string{"unknown"},
			wantCode:   ExitUsage,
			wantStderr: "Usage: context <command> [arguments]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			var gotArgs []string
			handler := NewHandler(map[string]Command{
				"init": commandFunc(func(_ context.Context, args []string) int {
					gotArgs = args
					return ExitSuccess
				}),
			}, &stderr)

			gotCode := handler.Run(context.Background(), tt.args)
			if gotCode != tt.wantCode {
				t.Fatalf("Run() exit code = %d, want %d", gotCode, tt.wantCode)
			}
			if !equalSlice(gotArgs, tt.wantArgs) {
				t.Fatalf("command args = %v, want %v", gotArgs, tt.wantArgs)
			}
			if got := stderr.String(); !strings.Contains(got, tt.wantStderr) {
				t.Errorf("stderr = %q, want substring %q", got, tt.wantStderr)
			}
		})
	}
}
