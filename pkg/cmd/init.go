package cmd

import (
	"fmt"

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
func (o *InitOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

// Validate は提供されたオプションが有効であるか検証します。
func (o *InitOptions) Validate() error {
	if o.RepoPath == "" {
		return fmt.Errorf("--repo flag is required")
	}
	return nil
}

// Run はコマンドの実際のビジネスロジックを実行します。
func (o *InitOptions) Run() error {
	cfg, err := o.Factory.Config()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	err = cfg.SetContextRepository(o.RepoPath)
	if err != nil {
		return fmt.Errorf("failed to set context repository: %w", err)
	}

	fmt.Fprintf(o.Factory.IOOut, "Successfully initialized context repository at: %s\n", o.RepoPath)
	return nil
}
