package cmd

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/yukihito-jokyu/context-cli/internal/distribution"
	"github.com/yukihito-jokyu/context-cli/internal/skillcatalog"
)

// SyncOptions は sync コマンドの実行オプションと依存関係を表します。
type SyncOptions struct {
	Factory *Factory
}

// NewCmdSync は sync コマンドを生成して返します。
func NewCmdSync(f *Factory) *cobra.Command {
	options := &SyncOptions{Factory: f}
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "配布済みのSkillを同期します",
		Args:  cobra.NoArgs,
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
	return cmd
}

// Complete は引数やファクトリから必要な情報の補完を行います。
func (o *SyncOptions) Complete(_ *cobra.Command, _ []string) error {
	return nil
}

// Validate は実行パラメータや依存関係の検証を行います。
func (o *SyncOptions) Validate() error {
	if o.Factory == nil {
		return ErrFactoryRequired
	}
	return nil
}

// Run は同期処理を実行します。
//
//nolint:gocognit,cyclop // 同期計画、対話確認、同期実行を一つの制御フローにまとめます。
func (o *SyncOptions) Run() error {
	workspaceRoot, err := o.Factory.WorkspaceValidator.Validate()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrWorkspace, err)
	}

	configuration, err := o.Factory.Config()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	configuredRepository := configuration.GetContextRepository()
	if configuredRepository == "" {
		return ErrContextRepositoryRequired
	}
	repositoryRoot, err := o.Factory.RepositoryValidator.Validate(configuredRepository)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRepository, err)
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

	catalog := o.Factory.SkillCatalog(repositoryRoot)

	// SkillRecord を RecordedSkillRef へ変換
	refs := make([]skillcatalog.RecordedSkillRef, len(record.Skills))
	for i, s := range record.Skills {
		refs[i] = skillcatalog.RecordedSkillRef{
			Name:   s.Name,
			Source: skillcatalog.SkillSource(s.Source),
		}
	}

	catalogResolved, err := catalog.ResolveRecordedSources(record.Project, refs)
	if err != nil {
		return fmt.Errorf("failed to resolve skill sources: %w", err)
	}

	// skillcatalog.ResolvedSkillSource を distribution.ResolvedSource へ変換
	resolvedSources := make([]distribution.ResolvedSource, len(catalogResolved))
	for i, r := range catalogResolved {
		resolvedSources[i] = distribution.ResolvedSource{
			Name:   r.Name,
			Source: distribution.SkillSource(r.Source),
			State:  distribution.SourceState(r.State),
			Path:   r.Path,
		}
	}

	// record.Skills を distribution.RecordedSkill へ変換
	recordedSkills := make([]distribution.RecordedSkill, len(record.Skills))
	for i, s := range record.Skills {
		recordedSkills[i] = distribution.RecordedSkill{
			Name:         s.Name,
			Source:       s.Source,
			Destination:  s.Destination,
			RelativePath: s.RelativePath,
			RecordedHash: s.Hash,
		}
	}

	input := distribution.SyncInput{
		WorkspaceRoot: record.WorkspaceRoot,
		Project:       record.Project,
		Destinations:  record.Destinations,
		Skills:        recordedSkills,
	}

	plan, err := o.Factory.SyncPlanner.Plan(snapshot, input, resolvedSources)
	if err != nil {
		return fmt.Errorf("failed to plan skill sync: %w", err)
	}

	// ローカル変更がある場合は、非TTYならエラー、TTYなら対話確認を行う
	if len(plan.LocalChanges) > 0 {
		if !o.Factory.IsTerminal(o.Factory.IOIn, o.Factory.IOOut) {
			return distribution.ErrLocalChange
		}

		var updates []string
		var deletes []string
		for _, op := range plan.LocalChanges {
			switch op.Kind {
			case distribution.SyncOperationUpdate:
				updates = append(updates, op.RelativePath)
			case distribution.SyncOperationDelete:
				deletes = append(deletes, op.RelativePath)
			case distribution.SyncOperationKeep:
				// Keepの場合は何もしない
			default:
				// 未知の操作種別の場合は何もしない
			}
		}

		confirmed, err := o.prompt().ConfirmSync(updates, deletes)
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

	// 更新・削除対象がない場合
	if len(plan.Updates) == 0 && len(plan.Deletes) == 0 {
		_, err = fmt.Fprint(o.Factory.IOOut, "同期対象に変更はありません\n")
		if err != nil {
			return fmt.Errorf("failed to write result: %w", err)
		}
		return nil
	}

	toPlan, err := plan.ToPlan()
	if err != nil {
		return fmt.Errorf("failed to convert sync plan: %w", err)
	}

	result, err := o.Factory.DistributionExecutor(store).Execute(toPlan)
	if err != nil {
		return fmt.Errorf("failed to execute sync: %w", err)
	}

	_, err = fmt.Fprintf(
		o.Factory.IOOut,
		"%d件のSkillを更新し、%d件を削除しました\n",
		result.UniqueUpdated,
		result.UniqueDeleted,
	)
	if err != nil {
		return fmt.Errorf("failed to write sync result: %w", err)
	}

	return nil
}

func (o *SyncOptions) prompt() Prompt {
	return o.Factory.Prompt(o.Factory.IOIn, o.Factory.IOOut)
}
