package cmd

import (
	"github.com/spf13/cobra"
)

// NewCmdRoot は context-cli のルートコマンドを作成して返します。
func NewCmdRoot(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context <command> [flags]",
		Short: "context-cli は AI 開発のコンテキストファイルを管理する",
		Long:  `context-cli は、複数リポジトリにまたがる AI の指示（instructions）とスキル（skills）を初期化・同期するツールです。`,
	}

	// サブコマンドの登録
	cmd.AddCommand(NewCmdInit(f))

	return cmd
}
