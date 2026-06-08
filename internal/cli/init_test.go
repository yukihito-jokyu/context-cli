package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/yukihito-jokyu/context-cli/internal/application"
	"github.com/yukihito-jokyu/context-cli/internal/domain"
)

var (
	errLoadConfig       = errors.New("load config")
	errSensitiveDetails = errors.New("failed at /private/secret/path: sensitive detail")
)

type initRunnerFunc func(context.Context, string) error

func (f initRunnerFunc) Run(ctx context.Context, path string) error {
	return f(ctx, path)
}

//nolint:gocognit // Table-driven assertions intentionally cover the complete CLI result mapping.
func TestInitHandlerRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		run        initRunnerFunc
		wantCode   int
		wantStdout []string
		wantStderr []string
	}{
		{
			name:       "configures repository",
			args:       []string{"."},
			run:        func(context.Context, string) error { return nil },
			wantCode:   ExitSuccess,
			wantStdout: []string{"Context repository is configured and up to date:"},
		},
		{
			name:       "requires init path",
			args:       nil,
			run:        func(context.Context, string) error { return nil },
			wantCode:   ExitUsage,
			wantStderr: []string{"Usage: context init <path>"},
		},
		{
			name:       "rejects extra argument",
			args:       []string{".", "extra"},
			run:        func(context.Context, string) error { return nil },
			wantCode:   ExitUsage,
			wantStderr: []string{"Usage: context init <path>"},
		},
		{
			name: "reports validation errors without the repository absolute path",
			args: []string{"/private/repository"},
			run: func(context.Context, string) error {
				return &application.RepositoryValidationError{
					Errors: []domain.ValidationError{
						{Path: "projects", Reason: "projects directory is missing"},
						{Path: "utils/skills", Reason: "utils/skills directory is missing"},
					},
				}
			},
			wantCode: ExitFailure,
			wantStderr: []string{
				"Context repository validation failed:",
				"- projects: projects directory is missing",
				"- utils/skills: utils/skills directory is missing",
			},
		},
		{
			name: "treats explicit rejection as a successful cancellation",
			args: []string{"."},
			run: func(context.Context, string) error {
				return application.ErrChangeAborted
			},
			wantCode:   ExitSuccess,
			wantStdout: []string{"Configuration change canceled. Existing configuration was not changed."},
		},
		{
			name: "reports unsupported existing config",
			args: []string{"."},
			run: func(context.Context, string) error {
				return fmt.Errorf("%w: %w", errLoadConfig, domain.ErrUnsupportedConfigVersion)
			},
			wantCode:   ExitFailure,
			wantStderr: []string{"The existing configuration uses an unsupported version.", "Existing configuration was not changed."},
		},
		{
			name: "does not expose unexpected error details",
			args: []string{"."},
			run: func(context.Context, string) error {
				return errSensitiveDetails
			},
			wantCode:   ExitFailure,
			wantStderr: []string{"Failed to configure the context repository. Review the repository and configuration, then try again."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			handler := NewInitHandler(tt.run, &stdout, &stderr)

			gotCode := handler.Run(context.Background(), tt.args)
			if gotCode != tt.wantCode {
				t.Fatalf("Run() exit code = %d, want %d", gotCode, tt.wantCode)
			}

			for _, want := range tt.wantStdout {
				if !strings.Contains(stdout.String(), want) {
					t.Errorf("stdout = %q, want substring %q", stdout.String(), want)
				}
			}
			for _, want := range tt.wantStderr {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("stderr = %q, want substring %q", stderr.String(), want)
				}
			}
			if strings.Contains(stderr.String(), "/private/secret/path") {
				t.Errorf("stderr exposes a sensitive path: %q", stderr.String())
			}
		})
	}
}

func TestConsoleUIConfirmChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		wantApproved bool
		wantErr      bool
	}{
		{name: "accepts y", input: "y\n", wantApproved: true},
		{name: "accepts y without trailing newline", input: "y", wantApproved: true},
		{name: "accepts yes case insensitively", input: "YES\n", wantApproved: true},
		{name: "rejects n", input: "n\n"},
		{name: "rejects empty input", input: "\n"},
		{name: "reports interrupted input", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			ui := NewConsoleUI(strings.NewReader(tt.input), &stdout)

			approved, err := ui.ConfirmChange(context.Background(), "/current", "/new")
			if (err != nil) != tt.wantErr {
				t.Fatalf("ConfirmChange() error = %v, wantErr %v", err, tt.wantErr)
			}
			if approved != tt.wantApproved {
				t.Errorf("ConfirmChange() approved = %v, want %v", approved, tt.wantApproved)
			}
			if !strings.Contains(stdout.String(), "Change context repository? [y/N]:") {
				t.Errorf("prompt = %q", stdout.String())
			}
			if !strings.Contains(stdout.String(), "Current context repository: /current") ||
				!strings.Contains(stdout.String(), "New context repository: /new") {
				t.Errorf("prompt does not show the requested change: %q", stdout.String())
			}
		})
	}
}

func TestConsoleUIConfirmChangeCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stdout bytes.Buffer
	ui := NewConsoleUI(strings.NewReader("y\n"), &stdout)

	approved, err := ui.ConfirmChange(ctx, "/current", "/new")
	if approved {
		t.Error("ConfirmChange() approved a canceled confirmation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ConfirmChange() error = %v, want context.Canceled", err)
	}
}

func TestConsoleUIConfirmChangeCanceledWhileReading(t *testing.T) {
	t.Parallel()

	reader, writer := io.Pipe()
	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	var stdout bytes.Buffer
	ui := NewConsoleUI(reader, &stdout)

	result := make(chan error, 1)
	go func() {
		_, err := ui.ConfirmChange(ctx, "/current", "/new")
		result <- err
	}()

	cancel()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ConfirmChange() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		_ = reader.Close()
		t.Fatal("ConfirmChange() did not stop after context cancellation")
	}
}

func TestNewSlogHandlerOmitsAttributes(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := slog.New(NewSlogHandler(&output))

	logger.Error("configuration failed", "path", "/private/secret/path")

	got := output.String()
	if !strings.Contains(got, "level=ERROR") || !strings.Contains(got, `msg="configuration failed"`) {
		t.Errorf("log output = %q", got)
	}
	if strings.Contains(got, "path") || strings.Contains(got, "/private/secret/path") || strings.Contains(got, "time=") {
		t.Errorf("log output exposes omitted attributes: %q", got)
	}
}
