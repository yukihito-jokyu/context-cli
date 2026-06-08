// Package cli は、コマンド解析、入出力、終了コードなどのCLI境界を実装します。
package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/yukihito-jokyu/context-cli/internal/application"
	"github.com/yukihito-jokyu/context-cli/internal/domain"
)

type initRunner interface {
	Run(ctx context.Context, path string) error
}

// InitHandler はinitコマンドの引数、ユースケース結果、標準入出力、終了コードを接続します。
type InitHandler struct {
	init   initRunner
	stdout io.Writer
	stderr io.Writer
}

// NewInitHandler はinitコマンドのCLIハンドラを作成します。
func NewInitHandler(init initRunner, stdout, stderr io.Writer) *InitHandler {
	return &InitHandler{
		init:   init,
		stdout: stdout,
		stderr: stderr,
	}
}

// Run はコマンドライン引数を検証し、initユースケースを実行します。
func (h *InitHandler) Run(ctx context.Context, args []string) int {
	if len(args) != 1 || args[0] == "" {
		_, _ = fmt.Fprintln(h.stderr, "Usage: context init <path>")
		return ExitUsage
	}

	if err := h.init.Run(ctx, args[0]); err != nil {
		return h.writeError(err)
	}

	normalizedPath, err := domain.NormalizeRepositoryPath(args[0])
	if err != nil {
		_, _ = fmt.Fprintln(h.stderr, "Failed to identify the configured context repository.")
		return ExitFailure
	}

	_, _ = fmt.Fprintf(h.stdout, "Context repository is configured and up to date: %s\n", normalizedPath)
	return ExitSuccess
}

func (h *InitHandler) writeError(err error) int {
	if errors.Is(err, application.ErrChangeAborted) {
		_, _ = fmt.Fprintln(h.stdout, "Configuration change canceled. Existing configuration was not changed.")
		return ExitSuccess
	}

	var validationErr *application.RepositoryValidationError
	if errors.As(err, &validationErr) {
		_, _ = fmt.Fprintln(h.stderr, "Context repository validation failed:")
		for _, issue := range validationErr.Errors {
			_, _ = fmt.Fprintf(h.stderr, "- %s: %s\n", issue.Path, issue.Reason)
		}
		_, _ = fmt.Fprintln(h.stderr, "Fix the listed repository issues and try again. Existing configuration was not changed.")
		return ExitFailure
	}

	switch {
	case errors.Is(err, domain.ErrUnsupportedConfigVersion):
		_, _ = fmt.Fprintln(h.stderr, "The existing configuration uses an unsupported version.")
		_, _ = fmt.Fprintln(h.stderr, "Existing configuration was not changed. Upgrade context or restore a supported configuration.")
	case errors.Is(err, domain.ErrInvalidConfig):
		_, _ = fmt.Fprintln(h.stderr, "The existing configuration is invalid.")
		_, _ = fmt.Fprintln(h.stderr, "Existing configuration was not changed. Correct the configuration and try again.")
	case errors.Is(err, application.ErrPermissionTooBroad):
		_, _ = fmt.Fprintln(h.stderr, "The configuration permissions are too broad.")
		_, _ = fmt.Fprintln(h.stderr, "Existing configuration was not changed. Restrict the permissions and try again.")
	case errors.Is(err, application.ErrLockFailed):
		_, _ = fmt.Fprintln(h.stderr, "The configuration is being updated by another process.")
		_, _ = fmt.Fprintln(h.stderr, "Existing configuration was not changed. Try again after the other process finishes.")
	case errors.Is(err, application.ErrConfigConflict):
		_, _ = fmt.Fprintln(h.stderr, "The configuration changed during this operation.")
		_, _ = fmt.Fprintln(h.stderr, "Existing configuration was not overwritten. Review it and try again.")
	case errors.Is(err, context.Canceled):
		_, _ = fmt.Fprintln(h.stderr, "Configuration was interrupted. Existing configuration was not changed.")
	default:
		_, _ = fmt.Fprintln(h.stderr, "Failed to configure the context repository. Review the repository and configuration, then try again.")
	}

	return ExitFailure
}

// ConsoleUI は標準入出力を介して設定変更の承認を確認します。
type ConsoleUI struct {
	input  io.Reader
	output io.Writer
}

type confirmationReadResult struct {
	line string
	err  error
}

// NewConsoleUI は指定された入出力を使う対話UIを作成します。
func NewConsoleUI(input io.Reader, output io.Writer) *ConsoleUI {
	return &ConsoleUI{
		input:  input,
		output: output,
	}
}

// NewSlogHandler は時刻と属性を出力しないCLI向けログHandlerを作成します。
func NewSlogHandler(output io.Writer) slog.Handler {
	return slog.NewTextHandler(output, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key != slog.LevelKey && attr.Key != slog.MessageKey {
				return slog.Attr{}
			}
			return attr
		},
	})
}

// ConfirmChange は現在値と変更先を表示し、設定変更の承認を確認します。
func (ui *ConsoleUI) ConfirmChange(
	ctx context.Context,
	currentPath string,
	newPath string,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, fmt.Errorf("confirmation context: %w", err)
	}

	if _, err := fmt.Fprintf(ui.output, "Current context repository: %s\n", currentPath); err != nil {
		return false, fmt.Errorf("failed to write current repository: %w", err)
	}
	if _, err := fmt.Fprintf(ui.output, "New context repository: %s\n", newPath); err != nil {
		return false, fmt.Errorf("failed to write new repository: %w", err)
	}
	if _, err := fmt.Fprint(ui.output, "Change context repository? [y/N]: "); err != nil {
		return false, fmt.Errorf("failed to write confirmation prompt: %w", err)
	}

	result := make(chan confirmationReadResult, 1)
	go func() {
		line, err := bufio.NewReader(ui.input).ReadString('\n')
		result <- confirmationReadResult{line: line, err: err}
	}()

	var readResult confirmationReadResult
	select {
	case <-ctx.Done():
		return false, fmt.Errorf("confirmation context: %w", ctx.Err())
	case readResult = <-result:
	}

	line, err := readResult.line, readResult.err
	if err != nil && (!errors.Is(err, io.EOF) || line == "") {
		if errors.Is(err, io.EOF) {
			return false, application.ErrChangeAborted
		}
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
