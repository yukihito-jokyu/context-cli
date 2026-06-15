package cmd

import (
	"errors"
	"fmt"
	"slices"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/yukihito-jokyu/context-cli/internal/distribution"
)

// ErrSkillNotDistributed は指定されたSkillが配布されていない場合のエラーを表します。
var ErrSkillNotDistributed = errors.New("skill is not distributed in this workspace")

// DeleteOptions は delete コマンドの実行オプションと依存関係を表します。
type DeleteOptions struct {
	Factory    *Factory
	All        bool
	SkillNames []string
}

// NewCmdDelete は delete コマンドを生成して返します。
func NewCmdDelete(f *Factory) *cobra.Command {
	options := &DeleteOptions{Factory: f}
	cmd := &cobra.Command{
		Use:   "delete [skill-name...]",
		Short: "配布済みのSkillを削除します",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.Complete(cmd, args); err != nil {
				return err
			}
			if err := options.Validate(); err != nil {
				return err
			}
			return options.Run()
		},
	}
	cmd.Flags().BoolVarP(&options.All, "all", "a", false, "すべてのSkillを削除します")
	return cmd
}

// Complete は引数やファクトリから必要な情報の補完を行います。
func (o *DeleteOptions) Complete(_ *cobra.Command, args []string) error {
	o.SkillNames = args
	return nil
}

// Validate は実行パラメータや依存関係の検証を行います。
func (o *DeleteOptions) Validate() error {
	if o.Factory == nil {
		return ErrFactoryRequired
	}
	return nil
}

// Run は削除処理のメインロジックを実行します。
//
//nolint:gocognit,cyclop // 削除計画、対話確認、削除実行を一つの制御フローにまとめます。
func (o *DeleteOptions) Run() error {
	workspaceRoot, err := o.Factory.WorkspaceValidator.Validate()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrWorkspace, err)
	}

	store, err := o.Factory.MapStore()
	if err != nil {
		return fmt.Errorf("failed to open distribution map: %w", err)
	}
	snapshot, err := store.Load()
	if err != nil {
		return fmt.Errorf("failed to load distribution map: %w", err)
	}

	record, isManaged := snapshot.Workspaces[workspaceRoot]
	if !isManaged {
		return fmt.Errorf("%w", distribution.ErrUnmanagedWorkspace)
	}

	// 削除対象の決定
	var targetSkills []string

	switch {
	case o.All:
		// すべてのSkillを削除対象に設定
		for _, s := range record.Skills {
			targetSkills = append(targetSkills, s.Name)
		}
	case len(o.SkillNames) > 0:
		// 指定されたSkillを削除対象に設定
		seen := make(map[string]bool)
		for _, name := range o.SkillNames {
			if !seen[name] {
				seen[name] = true
				targetSkills = append(targetSkills, name)
			}
		}

		// 管理情報にないSkill名があれば即座にエラー終了
		managedSkills := make(map[string]bool)
		for _, s := range record.Skills {
			managedSkills[s.Name] = true
		}
		for _, name := range targetSkills {
			if !managedSkills[name] {
				return fmt.Errorf("skill %q is not distributed in this workspace: %w", name, ErrSkillNotDistributed)
			}
		}
	default:
		// 引数も --all もない場合
		if !o.Factory.IsTerminal(o.Factory.IOIn, o.Factory.IOOut) {
			return ErrNonTTY
		}

		// 管理中のSkill名を名前順で重複排除して取得
		var candidates []string
		seen := make(map[string]bool)
		for _, s := range record.Skills {
			if !seen[s.Name] {
				seen[s.Name] = true
				candidates = append(candidates, s.Name)
			}
		}
		slices.Sort(candidates)

		selected, err := o.prompt().SelectSkillsToDelete(candidates)
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil
			}
			return fmt.Errorf("%w: %w", ErrPrompt, err)
		}
		targetSkills = selected
	}

	// 削除対象がない場合は正常終了
	if len(targetSkills) == 0 {
		_, err = fmt.Fprint(o.Factory.IOOut, "削除対象のSkillはありません\n")
		if err != nil {
			return fmt.Errorf("failed to write result: %w", err)
		}
		return nil
	}

	// 削除計画の作成
	plan, err := o.Factory.DistributionPlanner.PlanDelete(snapshot, workspaceRoot, targetSkills)
	if err != nil {
		return fmt.Errorf("failed to plan skill deletion: %w", err)
	}

	// ローカル編集の検証
	var localEdits []string
	for _, del := range plan.Deletes {
		if del.IsLocalEdit {
			localEdits = append(localEdits, del.RelativePath)
		}
	}

	if len(localEdits) > 0 {
		if !o.Factory.IsTerminal(o.Factory.IOIn, o.Factory.IOOut) {
			return distribution.ErrLocalChange
		}

		// 上書き/削除の確認
		confirmed, err := o.prompt().ConfirmOverwrite(nil, localEdits)
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil
			}
			return fmt.Errorf("%w: %w", ErrPrompt, err)
		}
		if !confirmed {
			return nil
		}
	}

	// 計画の実行
	_, err = o.Factory.DistributionExecutor(store).Execute(plan)
	if err != nil {
		return fmt.Errorf("failed to execute skill deletion: %w", err)
	}

	// 削除された一意なSkill名の数を算出
	deletedSkills := make(map[string]struct{})
	for _, del := range plan.Deletes {
		deletedSkills[del.Name] = struct{}{}
	}

	_, err = fmt.Fprintf(o.Factory.IOOut, "%d件のSkillを削除しました\n", len(deletedSkills))
	if err != nil {
		return fmt.Errorf("failed to write result: %w", err)
	}

	return nil
}

func (o *DeleteOptions) prompt() Prompt {
	return o.Factory.Prompt(o.Factory.IOIn, o.Factory.IOOut)
}
