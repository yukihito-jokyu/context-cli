package cmd

import (
	"io"
	"os"

	"github.com/charmbracelet/x/term"
	"github.com/yukihito-jokyu/context-cli/internal/config"
	"github.com/yukihito-jokyu/context-cli/internal/distribution"
	"github.com/yukihito-jokyu/context-cli/internal/distributionmap"
	"github.com/yukihito-jokyu/context-cli/internal/repository"
	"github.com/yukihito-jokyu/context-cli/internal/skillcatalog"
	"github.com/yukihito-jokyu/context-cli/internal/workspace"
)

// Config は CLI 設定のインターフェースを表します。
type Config interface {
	GetContextRepository() string
	SetContextRepository(expected, newPath string) error
}

// RepositoryValidator はContext Repositoryの検証境界を表します。
type RepositoryValidator interface {
	Validate(path string) (string, error)
}

// WorkspaceValidator は配布先Workspaceの検証境界を表します。
type WorkspaceValidator interface {
	Validate() (string, error)
}

// SkillCatalog はプロジェクトとSkillの列挙境界を表します。
type SkillCatalog interface {
	Projects() ([]skillcatalog.Candidate, error)
	Project(string) (skillcatalog.Candidate, error)
	ProjectSkills(skillcatalog.Candidate) ([]skillcatalog.Candidate, error)
	CommonSkills([]skillcatalog.Candidate) ([]skillcatalog.Candidate, error)
	ResolveRecordedSources(project string, skills []skillcatalog.RecordedSkillRef) ([]skillcatalog.ResolvedSkillSource, error)
}

// SyncPlanner は同期計画の構築境界を表します。
type SyncPlanner interface {
	Plan(distribution.MapSnapshot, distribution.SyncInput, []distribution.ResolvedSource) (distribution.SyncPlan, error)
}

// DistributionPlanner は初回配布計画の構築境界を表します。
type DistributionPlanner interface {
	Plan(distribution.MapSnapshot, distribution.Selection) (distribution.Plan, error)
}

// DistributionExecutor は初回配布計画の実行境界を表します。
type DistributionExecutor interface {
	Execute(distribution.Plan) (distribution.Result, error)
}

// Factory は CLI の依存関係を管理し、注入します。
type Factory struct {
	IOOut io.Writer
	IOErr io.Writer
	IOIn  io.Reader

	RepositoryValidator  RepositoryValidator
	WorkspaceValidator   WorkspaceValidator
	IsTerminal           func(io.Reader, io.Writer) bool
	SkillCatalog         func(string) SkillCatalog
	Prompt               func(io.Reader, io.Writer) Prompt
	MapStore             func() (distribution.MapStore, error)
	DistributionPlanner  DistributionPlanner
	SyncPlanner          SyncPlanner
	DistributionExecutor func(distribution.MapStore) DistributionExecutor

	// Config は Config インスタンスを返す関数です（遅延ロードされます）。
	Config func() (Config, error)
}

// NewFactory は標準の入出力（os.Stdout/Stderr/Stdin）を使用して新しい Factory を作成します。
func NewFactory() *Factory {
	environment := config.NewOSEnvironment()
	fileSystem := config.NewOSFileSystem()
	distributionFileSystem := distribution.NewOSFileSystem()
	return &Factory{
		IOOut:               os.Stdout,
		IOErr:               os.Stderr,
		IOIn:                os.Stdin,
		RepositoryValidator: repository.NewValidator(repository.NewFileSystem()),
		WorkspaceValidator:  workspace.NewValidator(os.Getwd),
		IsTerminal: func(input io.Reader, output io.Writer) bool {
			inputFile, inputOK := input.(interface{ Fd() uintptr })
			outputFile, outputOK := output.(interface{ Fd() uintptr })
			return inputOK && outputOK &&
				term.IsTerminal(inputFile.Fd()) &&
				term.IsTerminal(outputFile.Fd())
		},
		SkillCatalog: func(root string) SkillCatalog {
			return skillcatalog.New(root)
		},
		Prompt: newHuhPrompt,
		MapStore: func() (distribution.MapStore, error) {
			return distributionmap.New(environment)
		},
		DistributionPlanner: distribution.NewPlanner(distributionFileSystem),
		SyncPlanner:         distribution.NewSyncPlanner(distributionFileSystem),
		DistributionExecutor: func(store distribution.MapStore) DistributionExecutor {
			return distribution.NewExecutor(distributionFileSystem, store)
		},
		Config: func() (Config, error) {
			return config.Open(environment, fileSystem)
		},
	}
}
