package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/yukihito-jokyu/context-cli/internal/distribution"
	"github.com/yukihito-jokyu/context-cli/internal/skillcatalog"
)

var (
	// ErrNonTTY は対話端末で実行されていないことを表します。
	ErrNonTTY = errors.New("context add requires an interactive terminal")
	// ErrWorkspace はWorkspaceの検証失敗を表します。
	ErrWorkspace = errors.New("failed to validate workspace")
	// ErrContextRepositoryRequired はContext Repositoryが未設定であることを表します。
	ErrContextRepositoryRequired = errors.New("context repository is not configured; run context init first")
	// ErrRepository は保存済みContext Repositoryの再検証失敗を表します。
	ErrRepository = errors.New("failed to validate configured context repository")
	// ErrPrompt は対話処理の失敗を表します。
	ErrPrompt = errors.New("interactive prompt failed")
	// ErrDestinationRequired はSkill選択時に配布先が未選択であることを表します。
	ErrDestinationRequired = errors.New("at least one destination must be selected")
	// ErrFactoryRequired はFactoryが指定されていないことを表します。
	ErrFactoryRequired = errors.New("factory is required")
)

// AddOptions はaddコマンドの入力、依存関係、選択結果を保持します。
type AddOptions struct {
	Factory *Factory

	ProjectName string
	// ProjectSpecified は位置引数が明示されたことを表します。
	ProjectSpecified bool
	Selection        distribution.Selection
}

// NewCmdAdd はaddコマンドを作成して返します。
func NewCmdAdd(f *Factory) *cobra.Command {
	options := &AddOptions{Factory: f}
	command := &cobra.Command{
		Use:   "add [project-name]",
		Short: "配布するプロジェクトとSkillを選択します",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			if err := options.Complete(command, args); err != nil {
				return err
			}
			if err := options.Validate(); err != nil {
				return err
			}
			return options.Run()
		},
	}
	return command
}

// Complete は位置引数からプロジェクト名を補完します。
func (o *AddOptions) Complete(_ *cobra.Command, args []string) error {
	if len(args) == 1 {
		o.ProjectName = args[0]
		o.ProjectSpecified = true
	}
	return nil
}

// Validate はAddOptionsに必要な依存関係を検証します。
func (o *AddOptions) Validate() error {
	if o.Factory == nil {
		return ErrFactoryRequired
	}
	return nil
}

// Run はプロジェクト、Skill、配布先の選択結果を確定します。
//
//nolint:gocognit,cyclop // 収集、対話選択、計画作成、配布実行の一連のフローを一箇所で制御します。
func (o *AddOptions) Run() error {
	workspaceRoot, catalog, err := o.prepare()
	if err != nil {
		return err
	}
	store, err := o.Factory.MapStore()
	if err != nil {
		return fmt.Errorf("failed to open distribution map: %w", err)
	}
	snapshot, err := store.Load()
	if err != nil {
		return fmt.Errorf("failed to load distribution map: %w", err)
	}

	oldRecord, isManaged := snapshot.Workspaces[workspaceRoot]

	defaultProject := ""
	if isManaged {
		defaultProject = oldRecord.Project
	}
	project, cancelled, err := o.selectProject(catalog, defaultProject)
	if err != nil || cancelled {
		return err
	}

	selectedSkills, cancelled, err := o.selectAllSkills(catalog, project, oldRecord, isManaged)
	if err != nil || cancelled {
		return err
	}

	defaultDestinations := []distribution.Destination{}
	if isManaged && len(selectedSkills) > 0 {
		defaultDestinations = oldRecord.Destinations
	}
	destinations, cancelled, err := o.selectDestinations(selectedSkills, defaultDestinations)
	if err != nil || cancelled {
		return err
	}

	o.Selection = distribution.Selection{
		WorkspaceRoot: workspaceRoot,
		Project:       project.Name,
		Skills:        selectedSkills,
		Destinations:  destinations,
	}

	plan, err := o.Factory.DistributionPlanner.Plan(snapshot, o.Selection)
	if err != nil {
		return fmt.Errorf("failed to plan skill distribution: %w", err)
	}

	var conflicts []string
	var localEdits []string
	for _, op := range plan.Creates {
		if op.IsConflict {
			conflicts = append(conflicts, op.RelativePath)
		}
		if op.IsLocalEdit {
			localEdits = append(localEdits, op.RelativePath)
		}
	}
	for _, op := range plan.Deletes {
		if op.IsLocalEdit {
			localEdits = append(localEdits, op.RelativePath)
		}
	}

	if len(conflicts) > 0 || len(localEdits) > 0 {
		confirmed, err := o.prompt().ConfirmOverwrite(conflicts, localEdits)
		if errors.Is(err, huh.ErrUserAborted) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to confirm overwrite: %w", err)
		}
		if !confirmed {
			return nil
		}
	}

	if len(plan.Creates) == 0 && len(plan.Deletes) == 0 {
		return nil
	}

	result, err := o.Factory.DistributionExecutor(store).Execute(plan)
	if err != nil {
		return fmt.Errorf("failed to distribute skills: %w", err)
	}
	_, err = fmt.Fprintf(
		o.Factory.IOOut,
		"%d件のSkillを%sへ配布しました\n",
		result.Created,
		formatDestinations(result.Destinations),
	)
	if err != nil {
		return fmt.Errorf("failed to write distribution result: %w", err)
	}
	return nil
}

func (o *AddOptions) prepare() (string, SkillCatalog, error) {
	if !o.Factory.IsTerminal(o.Factory.IOIn, o.Factory.IOOut) {
		return "", nil, ErrNonTTY
	}
	workspaceRoot, err := o.Factory.WorkspaceValidator.Validate()
	if err != nil {
		return "", nil, fmt.Errorf("%w: %w", ErrWorkspace, err)
	}
	configuration, err := o.Factory.Config()
	if err != nil {
		return "", nil, fmt.Errorf("failed to load config: %w", err)
	}
	configuredRepository := configuration.GetContextRepository()
	if configuredRepository == "" {
		return "", nil, ErrContextRepositoryRequired
	}
	repositoryRoot, err := o.Factory.RepositoryValidator.Validate(configuredRepository)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %w", ErrRepository, err)
	}
	return workspaceRoot, o.Factory.SkillCatalog(repositoryRoot), nil
}

// selectAllSkills はプロジェクト固有Skillと共通Skillを順に収集・選択します。
//
//nolint:gocognit // プロジェクト固有と共通Skillのそれぞれのデフォルト値の抽出と選択を統合して行います。
func (o *AddOptions) selectAllSkills(
	catalog SkillCatalog,
	project skillcatalog.Candidate,
	oldRecord distribution.WorkspaceRecord,
	isManaged bool,
) ([]distribution.SelectedSkill, bool, error) {
	projectSkills, err := catalog.ProjectSkills(project)
	if err != nil {
		return nil, false, fmt.Errorf("failed to list project skills: %w", err)
	}

	defaultProjectSkills := []string{}
	if isManaged && project.Name == oldRecord.Project {
		for _, s := range oldRecord.Skills {
			if s.Source == distribution.SkillSourceProject {
				defaultProjectSkills = append(defaultProjectSkills, s.Name)
			}
		}
	}
	selectedProject, cancelled, err := o.selectSkills(SkillKindProject, projectSkills, defaultProjectSkills)
	if err != nil || cancelled {
		return nil, cancelled, err
	}

	commonSkills, err := catalog.CommonSkills(projectSkills)
	if err != nil {
		return nil, false, fmt.Errorf("failed to list common skills: %w", err)
	}

	defaultCommonSkills := []string{}
	if isManaged && project.Name == oldRecord.Project {
		for _, s := range oldRecord.Skills {
			if s.Source == distribution.SkillSourceCommon {
				defaultCommonSkills = append(defaultCommonSkills, s.Name)
			}
		}
	}

	defaultConfirmed := len(defaultCommonSkills) > 0
	selectedCommon, cancelled, err := o.selectCommonSkills(commonSkills, defaultCommonSkills, defaultConfirmed)
	if err != nil || cancelled {
		return nil, cancelled, err
	}
	selectedSkills := append(
		toSelectedSkills(selectedProject, distribution.SkillSourceProject),
		toSelectedSkills(selectedCommon, distribution.SkillSourceCommon)...,
	)
	return selectedSkills, false, nil
}

func (o *AddOptions) selectDestinations(
	selectedSkills []distribution.SelectedSkill,
	defaultDestinations []distribution.Destination,
) ([]distribution.Destination, bool, error) {
	if len(selectedSkills) == 0 {
		return []distribution.Destination{}, false, nil
	}
	destinations, err := o.prompt().SelectDestinations(defaultDestinations)
	if errors.Is(err, huh.ErrUserAborted) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, wrapPromptError("failed to select destinations", err)
	}
	if len(destinations) == 0 {
		return nil, false, ErrDestinationRequired
	}
	return destinations, false, nil
}

func (o *AddOptions) selectProject(catalog SkillCatalog, defaultProject string) (skillcatalog.Candidate, bool, error) {
	if o.ProjectSpecified {
		project, err := catalog.Project(o.ProjectName)
		if err != nil {
			return skillcatalog.Candidate{}, false, fmt.Errorf("failed to validate project: %w", err)
		}
		return project, false, nil
	}
	projects, err := catalog.Projects()
	if err != nil {
		return skillcatalog.Candidate{}, false, fmt.Errorf("failed to list projects: %w", err)
	}
	if len(projects) == 0 {
		return skillcatalog.Candidate{}, false, skillcatalog.ErrNoCandidates
	}
	project, err := o.prompt().SelectProject(projects, defaultProject)
	if errors.Is(err, huh.ErrUserAborted) {
		return skillcatalog.Candidate{}, true, nil
	}
	if err != nil {
		return skillcatalog.Candidate{}, false, wrapPromptError("failed to select project", err)
	}
	return project, false, nil
}

func (o *AddOptions) selectSkills(
	kind SkillKind,
	candidates []skillcatalog.Candidate,
	defaultNames []string,
) ([]skillcatalog.Candidate, bool, error) {
	if len(candidates) == 0 {
		return []skillcatalog.Candidate{}, false, nil
	}
	var filteredDefaults []string
	for _, def := range defaultNames {
		for _, cand := range candidates {
			if cand.Name == def {
				filteredDefaults = append(filteredDefaults, def)
				break
			}
		}
	}
	selected, err := o.prompt().SelectSkills(kind, candidates, filteredDefaults)
	if errors.Is(err, huh.ErrUserAborted) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, wrapPromptError("failed to select skills", err)
	}
	return selected, false, nil
}

func (o *AddOptions) selectCommonSkills(
	candidates []skillcatalog.Candidate,
	defaultNames []string,
	defaultConfirmed bool,
) ([]skillcatalog.Candidate, bool, error) {
	if len(candidates) == 0 {
		return []skillcatalog.Candidate{}, false, nil
	}
	confirmed, err := o.prompt().ConfirmCommonSkills(defaultConfirmed)
	if errors.Is(err, huh.ErrUserAborted) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, wrapPromptError("failed to confirm common skills", err)
	}
	if !confirmed {
		return []skillcatalog.Candidate{}, false, nil
	}
	return o.selectSkills(SkillKindCommon, candidates, defaultNames)
}

func (o *AddOptions) prompt() Prompt {
	return o.Factory.Prompt(o.Factory.IOIn, o.Factory.IOOut)
}

func wrapPromptError(operation string, err error) error {
	return fmt.Errorf("%s: %w: %w", operation, ErrPrompt, err)
}

func toSelectedSkills(
	candidates []skillcatalog.Candidate,
	source distribution.SkillSource,
) []distribution.SelectedSkill {
	selected := make([]distribution.SelectedSkill, len(candidates))
	for i, candidate := range candidates {
		selected[i] = distribution.SelectedSkill{
			Name:       candidate.Name,
			Source:     source,
			SourcePath: candidate.Path,
		}
	}
	return selected
}

func formatDestinations(destinations []distribution.Destination) string {
	values := make([]string, len(destinations))
	for index, destination := range destinations {
		values[index] = string(destination)
	}
	return strings.Join(values, ", ")
}
