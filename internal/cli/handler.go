package cli

import (
	"context"
	"fmt"
	"io"
)

const (
	// ExitSuccess はコマンドが正常に完了したことを示します。
	ExitSuccess = 0
	// ExitFailure はコマンドの実行に失敗したことを示します。
	ExitFailure = 1
	// ExitUsage はコマンドライン引数が不正であることを示します。
	ExitUsage = 2
)

// Command はサブコマンドのCLI境界を表します。
type Command interface {
	Run(ctx context.Context, args []string) int
}

// Handler はコマンドライン引数を対応するサブコマンドへ振り分けます。
type Handler struct {
	commands map[string]Command
	stderr   io.Writer
}

// NewHandler はルートCLIハンドラを作成します。
func NewHandler(commands map[string]Command, stderr io.Writer) *Handler {
	return &Handler{
		commands: commands,
		stderr:   stderr,
	}
}

// Run はサブコマンドを解決して実行します。
func (h *Handler) Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return h.writeUsage()
	}

	command, ok := h.commands[args[0]]
	if !ok {
		return h.writeUsage()
	}

	return command.Run(ctx, args[1:])
}

func (h *Handler) writeUsage() int {
	_, _ = fmt.Fprintln(h.stderr, "Usage: context <command> [arguments]")
	return ExitUsage
}
