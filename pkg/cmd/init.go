package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// InitOptions は init コマンドの実行に必要なすべての入力と依存関係を保持します。
type InitOptions struct {
	Factory *Factory

	RepoPath string
}

// NewCmdInit は init コブラーコマンドを作成して返します。
func NewCmdInit(f *Factory) *cobra.Command {
	opts := &InitOptions{
		Factory: f,
	}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Context Repositoryの設定を初期化します",
		Long:  `設定ファイルにContext Repositoryの場所を設定します。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Complete(cmd, args); err != nil {
				return err
			}
			if err := opts.Validate(); err != nil {
				return err
			}
			return opts.Run()
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "Context Repositoryのパスを指定します")

	return cmd
}

// Complete は位置引数をパースし、オプションを設定します。
func (o *InitOptions) Complete(_ *cobra.Command, _ []string) error {
	return nil
}

var errRepoRequired = errors.New("--repo flag is required")

// ErrRepositoryChangeCanceled はContext Repositoryの設定変更が承認されなかったことを表します。
var ErrRepositoryChangeCanceled = errors.New("context repository change canceled")

type initOutputError struct {
	message string
	err     error
}

func (e *initOutputError) Error() string {
	return e.message
}

func (e *initOutputError) Unwrap() error {
	return e.err
}

// Validate は提供されたオプションが有効であるか検証します。
func (o *InitOptions) Validate() error {
	if o.RepoPath == "" {
		return errRepoRequired
	}
	return nil
}

// Run はコマンドの実際のビジネスロジックを実行します。
func (o *InitOptions) Run() error {
	validatedPath, err := o.Factory.RepositoryValidator.Validate(o.RepoPath)
	if err != nil {
		return fmt.Errorf("failed to validate context repository: %w", err)
	}

	cfg, err := o.Factory.Config()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	currentPath := cfg.GetContextRepository()
	if currentPath != "" && currentPath != validatedPath {
		if err := o.confirmRepositoryChange(currentPath, validatedPath); err != nil {
			return err
		}
	}

	if currentPath != validatedPath {
		if err := cfg.SetContextRepository(validatedPath); err != nil {
			return fmt.Errorf("failed to set context repository: %w", err)
		}
	}

	if _, err := fmt.Fprintf(
		o.Factory.IOOut,
		"Successfully initialized context repository at: %s\n",
		validatedPath,
	); err != nil {
		return &initOutputError{
			message: "failed to write initialization success message",
			err:     err,
		}
	}
	return nil
}

func (o *InitOptions) confirmRepositoryChange(currentPath, validatedPath string) error {
	if _, err := fmt.Fprintf(
		o.Factory.IOOut,
		"Current context repository: %s\nNew context repository: %s\n変更しますか? [y/N] ",
		currentPath,
		validatedPath,
	); err != nil {
		return &initOutputError{
			message: "failed to write repository change confirmation",
			err:     err,
		}
	}

	answer, err := bufio.NewReader(o.Factory.IOIn).ReadString('\n')
	if err != nil || strings.TrimSpace(answer) != "y" {
		return ErrRepositoryChangeCanceled
	}
	return nil
}
