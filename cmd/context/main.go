// Package main provides the context command entry point.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/yukihito-jokyu/context-cli/internal/application"
	"github.com/yukihito-jokyu/context-cli/internal/cli"
	localfs "github.com/yukihito-jokyu/context-cli/internal/infrastructure/fs"
	configyaml "github.com/yukihito-jokyu/context-cli/internal/infrastructure/yaml"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	slog.SetDefault(slog.New(cli.NewSlogHandler(os.Stderr)))

	configDir, err := configyaml.ResolveConfigDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to locate the configuration directory.")
		return cli.ExitFailure
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	configRepository := configyaml.NewConfigRepository(configDir)
	console := cli.NewConsoleUI(os.Stdin, os.Stdout)
	fileSystem := localfs.NewLocalFileSystem()
	initUseCase := application.NewInitRepositoryUseCase(configRepository, console, fileSystem)
	initHandler := cli.NewInitHandler(initUseCase, os.Stdout, os.Stderr)
	handler := cli.NewHandler(map[string]cli.Command{
		"init": initHandler,
	}, os.Stderr)

	return handler.Run(ctx, args)
}
