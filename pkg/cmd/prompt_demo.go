package cmd

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// PromptDemoOptions は prompt-demo コマンドの実行に必要なすべての入力と依存関係を保持します。
type PromptDemoOptions struct {
	Factory *Factory

	SelectedModel  string
	SelectedSkills []string
}

// NewCmdPromptDemo は prompt-demo コブラーコマンドを作成して返します。
func NewCmdPromptDemo(f *Factory) *cobra.Command {
	opts := &PromptDemoOptions{
		Factory: f,
	}

	cmd := &cobra.Command{
		Use:   "prompt-demo",
		Short: "huh プロンプトのデモを実行します",
		Long:  `単一選択（Select）と複数選択（MultiSelect）の動作を検証するためのデモです。`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return opts.Run()
		},
	}

	return cmd
}

var errNoSkillsSelected = errors.New("少なくとも1つのスキルを選択する必要があります")

// Run はコマンドの実際のビジネスロジックを実行します。
func (o *PromptDemoOptions) Run() error {
	// フォームを作成して実行します
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("利用するAIモデルを選択してください").
				Options(
					huh.NewOption("Gemini 2.5 Flash", "Gemini 2.5 Flash"),
					huh.NewOption("Claude 3.5 Sonnet", "Claude 3.5 Sonnet"),
					huh.NewOption("GPT-4o", "GPT-4o"),
				).
				Value(&o.SelectedModel),
		),
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("有効化するスキルパッケージを選択してください").
				Options(
					huh.NewOption("Git Helper", "Git Helper"),
					huh.NewOption("Code Reviewer", "Code Reviewer"),
					huh.NewOption("Linter Rules", "Linter Rules"),
					huh.NewOption("Doc Generator", "Doc Generator"),
				).
				Validate(func(val []string) error {
					if len(val) == 0 {
						return errNoSkillsSelected
					}
					return nil
				}).
				Value(&o.SelectedSkills),
		),
	)

	// Factory から標準入出力を取り出して設定（可能な場合）
	// huh の WithInput/WithOutput は通常、特定のインターフェースを要求するため、
	// 標準入出力が指定されている場合はそちらを優先します。
	// ここでは、デフォルトの入出力（os.Stdin / os.Stdout）を使用します。

	err := form.Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			_, _ = fmt.Fprintln(o.Factory.IOErr, "\n操作がキャンセルされました。")
			return nil
		}
		return fmt.Errorf("プロンプトの実行に失敗しました: %w", err)
	}

	_, _ = fmt.Fprintln(o.Factory.IOOut, "\n--- 選択結果 ---")
	_, _ = fmt.Fprintf(o.Factory.IOOut, "選択されたAIモデル: %s\n", o.SelectedModel)
	_, _ = fmt.Fprintf(o.Factory.IOOut, "選択されたスキル: %v\n", o.SelectedSkills)

	return nil
}
