package cmd

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/yukihito-jokyu/context-cli/internal/distribution"
	"github.com/yukihito-jokyu/context-cli/internal/skillcatalog"
)

var (
	errWorkspaceTest  = errors.New("workspace")
	errRepositoryTest = errors.New("repository")
	errPromptTest     = errors.New("prompt")
	errUnexpectedTest = errors.New("unexpected call")
)

func TestAddOptionsComplete(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantName      string
		wantSpecified bool
	}{
		{name: "0引数", args: nil},
		{name: "1引数", args: []string{"project"}, wantName: "project", wantSpecified: true},
		{name: "空文字引数", args: []string{""}, wantSpecified: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &AddOptions{}
			if err := options.Complete(nil, tt.args); err != nil {
				t.Fatalf("Complete() error = %v", err)
			}
			if options.ProjectName != tt.wantName || options.ProjectSpecified != tt.wantSpecified {
				t.Fatalf("Complete() = (%q, %t), want (%q, %t)",
					options.ProjectName, options.ProjectSpecified, tt.wantName, tt.wantSpecified)
			}
		})
	}
}

func TestNewCmdAdd(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "0 arguments",
			args:    nil,
			wantErr: false,
		},
		{
			name:    "1 argument",
			args:    []string{"one"},
			wantErr: false,
		},
		{
			name:    "2 arguments",
			args:    []string{"one", "two"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := NewCmdAdd(&Factory{})
			err := command.Args(command, tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Args() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//nolint:gocognit,cyclop // テーブル駆動テストのため、認知・循環複雑度の上限を無視します。
func TestAddOptionsRun(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "SelectsProjectSkillsAndDestinations",
			run: func(t *testing.T) {
				t.Helper()
				project := skillcatalog.Candidate{Name: "project", Path: "/context/projects/project"}
				projectSkills := []skillcatalog.Candidate{
					{Name: "alpha", Path: "/context/projects/project/skills/alpha"},
					{Name: "shared", Path: "/context/projects/project/skills/shared"},
				}
				commonSkills := []skillcatalog.Candidate{{Name: "common", Path: "/context/utils/skills/common"}}
				catalog := &stubSkillCatalog{
					projects:      []skillcatalog.Candidate{project},
					project:       project,
					projectSkills: projectSkills,
					commonSkills:  commonSkills,
				}
				prompt := &stubPrompt{
					project:             project,
					selectedProject:     projectSkills[:1],
					addCommon:           true,
					selectedCommon:      commonSkills,
					selectedDestination: []distribution.Destination{distribution.DestinationCodex},
				}
				options := newAddOptionsForTest(catalog, prompt)

				if err := options.Run(); err != nil {
					t.Fatalf("Run() error = %v", err)
				}

				want := distribution.Selection{
					WorkspaceRoot: "/workspace",
					Project:       "project",
					Skills: []distribution.SelectedSkill{
						{Name: "alpha", Source: distribution.SkillSourceProject, SourcePath: projectSkills[0].Path},
						{Name: "common", Source: distribution.SkillSourceCommon, SourcePath: commonSkills[0].Path},
					},
					Destinations: []distribution.Destination{distribution.DestinationCodex},
				}
				if !reflect.DeepEqual(options.Selection, want) {
					t.Fatalf("Selection = %#v, want %#v", options.Selection, want)
				}
			},
		},
		{
			name: "SkipsPromptsForMissingCandidates",
			run: func(t *testing.T) {
				t.Helper()
				project := skillcatalog.Candidate{Name: "project", Path: "/context/projects/project"}
				catalog := &stubSkillCatalog{project: project}
				prompt := &stubPrompt{}
				options := newAddOptionsForTest(catalog, prompt)
				options.ProjectName = "project"
				options.ProjectSpecified = true

				if err := options.Run(); err != nil {
					t.Fatalf("Run() error = %v", err)
				}
				if prompt.calls != nil {
					t.Fatalf("Prompt calls = %v, want none", prompt.calls)
				}
				if options.Selection.Project != "project" || len(options.Selection.Skills) != 0 || len(options.Selection.Destinations) != 0 {
					t.Fatalf("Selection = %#v", options.Selection)
				}
			},
		},
		{
			name: "TreatsPromptAbortAsCancellation",
			run: func(t *testing.T) {
				t.Helper()
				project := skillcatalog.Candidate{Name: "project", Path: "/context/projects/project"}
				options := newAddOptionsForTest(
					&stubSkillCatalog{projects: []skillcatalog.Candidate{project}},
					&stubPrompt{errors: map[string]error{"project": huh.ErrUserAborted}},
				)
				options.Selection = distribution.Selection{Project: "unchanged"}

				if err := options.Run(); err != nil {
					t.Fatalf("Run() error = %v", err)
				}
				if options.Selection.Project != "unchanged" {
					t.Fatalf("Selection = %#v, want unchanged", options.Selection)
				}
			},
		},
		{
			name: "ValidatesPreconditionsInOrder",
			run: func(t *testing.T) {
				t.Helper()
				tests := []struct {
					name    string
					mutate  func(*Factory)
					wantErr error
				}{
					{name: "非TTY", mutate: func(f *Factory) {
						f.IsTerminal = func(_ io.Reader, _ io.Writer) bool { return false }
					}, wantErr: ErrNonTTY},
					{name: "Workspace失敗", mutate: func(f *Factory) { f.WorkspaceValidator = &stubWorkspaceValidator{err: errWorkspaceTest} }, wantErr: ErrWorkspace},
					{name: "設定未済", mutate: func(f *Factory) { f.Config = func() (Config, error) { return &recordingConfig{}, nil } }, wantErr: ErrContextRepositoryRequired},
					{name: "Repository再検証失敗", mutate: func(f *Factory) { f.RepositoryValidator = &stubRepositoryValidator{err: errRepositoryTest} }, wantErr: ErrRepository},
				}

				for _, tt := range tests {
					t.Run(tt.name, func(t *testing.T) {
						options := newAddOptionsForTest(&stubSkillCatalog{}, &stubPrompt{})
						tt.mutate(options.Factory)
						err := options.Run()
						if !errors.Is(err, tt.wantErr) {
							t.Fatalf("Run() error = %v, want %v", err, tt.wantErr)
						}
					})
				}
			},
		},
		{
			name: "RejectsExplicitEmptyProjectName",
			run: func(t *testing.T) {
				t.Helper()
				catalog := &stubSkillCatalog{projectErr: skillcatalog.ErrInvalidName}
				options := newAddOptionsForTest(catalog, &stubPrompt{})
				options.ProjectSpecified = true

				err := options.Run()
				if !errors.Is(err, skillcatalog.ErrInvalidName) {
					t.Fatalf("Run() error = %v, want ErrInvalidName", err)
				}
				if catalog.projectName != "" {
					t.Fatalf("Project() input = %q, want empty", catalog.projectName)
				}
			},
		},
		{
			name: "PromptCancellationKeepsSelection",
			run: func(t *testing.T) {
				t.Helper()
				project := skillcatalog.Candidate{Name: "project", Path: "/context/projects/project"}
				projectSkills := []skillcatalog.Candidate{{Name: "project-skill", Path: "/project-skill"}}
				commonSkills := []skillcatalog.Candidate{{Name: "common-skill", Path: "/common-skill"}}
				tests := []struct {
					name  string
					stage string
				}{
					{name: "プロジェクト選択", stage: "project"},
					{name: "プロジェクトSkill選択", stage: string(SkillKindProject)},
					{name: "共通Skill確認", stage: "confirm-common"},
					{name: "共通Skill選択", stage: string(SkillKindCommon)},
					{name: "配布先選択", stage: "destinations"},
				}

				for _, tt := range tests {
					t.Run(tt.name, func(t *testing.T) {
						prompt := &stubPrompt{
							project:             project,
							selectedProject:     projectSkills,
							addCommon:           true,
							selectedCommon:      commonSkills,
							selectedDestination: []distribution.Destination{distribution.DestinationCodex},
							errors:              map[string]error{tt.stage: huh.ErrUserAborted},
						}
						options := newAddOptionsForTest(&stubSkillCatalog{
							projects:      []skillcatalog.Candidate{project},
							projectSkills: projectSkills,
							commonSkills:  commonSkills,
						}, prompt)
						before := distribution.Selection{Project: "unchanged"}
						options.Selection = before

						if err := options.Run(); err != nil {
							t.Fatalf("Run() error = %v", err)
						}
						if !reflect.DeepEqual(options.Selection, before) {
							t.Fatalf("Selection = %#v, want unchanged %#v", options.Selection, before)
						}
					})
				}
			},
		},
		{
			name: "WrapsEveryPromptError",
			run: func(t *testing.T) {
				t.Helper()
				project := skillcatalog.Candidate{Name: "project", Path: "/context/projects/project"}
				projectSkills := []skillcatalog.Candidate{{Name: "project-skill", Path: "/project-skill"}}
				commonSkills := []skillcatalog.Candidate{{Name: "common-skill", Path: "/common-skill"}}
				stages := []string{
					"project",
					string(SkillKindProject),
					"confirm-common",
					string(SkillKindCommon),
					"destinations",
				}

				for _, stage := range stages {
					t.Run(stage, func(t *testing.T) {
						prompt := &stubPrompt{
							project:             project,
							selectedProject:     projectSkills,
							addCommon:           true,
							selectedCommon:      commonSkills,
							selectedDestination: []distribution.Destination{distribution.DestinationCodex},
							errors:              map[string]error{stage: errPromptTest},
						}
						options := newAddOptionsForTest(&stubSkillCatalog{
							projects:      []skillcatalog.Candidate{project},
							projectSkills: projectSkills,
							commonSkills:  commonSkills,
						}, prompt)

						err := options.Run()
						if !errors.Is(err, ErrPrompt) || !errors.Is(err, errPromptTest) {
							t.Fatalf("Run() error = %v, want ErrPrompt wrapping cause", err)
						}
					})
				}
			},
		},
		{
			name: "SelectionBranches",
			run: func(t *testing.T) {
				t.Helper()
				project := skillcatalog.Candidate{Name: "project", Path: "/context/projects/project"}
				projectSkill := skillcatalog.Candidate{Name: "project-skill", Path: "/project-skill"}
				commonSkill := skillcatalog.Candidate{Name: "common-skill", Path: "/common-skill"}

				t.Run("候補不足", func(t *testing.T) {
					options := newAddOptionsForTest(&stubSkillCatalog{}, &stubPrompt{})
					if err := options.Run(); !errors.Is(err, skillcatalog.ErrNoCandidates) {
						t.Fatalf("Run() error = %v, want ErrNoCandidates", err)
					}
				})

				t.Run("共通Skill拒否", func(t *testing.T) {
					prompt := &stubPrompt{
						project:             project,
						selectedProject:     []skillcatalog.Candidate{projectSkill},
						addCommon:           false,
						selectedDestination: []distribution.Destination{distribution.DestinationClaude},
					}
					options := newAddOptionsForTest(&stubSkillCatalog{
						projects:      []skillcatalog.Candidate{project},
						projectSkills: []skillcatalog.Candidate{projectSkill},
						commonSkills:  []skillcatalog.Candidate{commonSkill},
					}, prompt)
					if err := options.Run(); err != nil {
						t.Fatalf("Run() error = %v", err)
					}
					if !reflect.DeepEqual(prompt.calls, []string{"project", string(SkillKindProject), "confirm-common", "destinations"}) {
						t.Fatalf("Prompt calls = %v", prompt.calls)
					}
					if len(options.Selection.Skills) != 1 || options.Selection.Skills[0].Name != projectSkill.Name {
						t.Fatalf("Selection.Skills = %#v", options.Selection.Skills)
					}
				})

				t.Run("配布先0件", func(t *testing.T) {
					prompt := &stubPrompt{
						project:         project,
						selectedProject: []skillcatalog.Candidate{projectSkill},
					}
					options := newAddOptionsForTest(&stubSkillCatalog{
						projects:      []skillcatalog.Candidate{project},
						projectSkills: []skillcatalog.Candidate{projectSkill},
					}, prompt)
					if err := options.Run(); !errors.Is(err, ErrDestinationRequired) {
						t.Fatalf("Run() error = %v, want ErrDestinationRequired", err)
					}
				})
			},
		},
		{
			name: "PassesCandidatesAndCallsInOrder",
			run: func(t *testing.T) {
				t.Helper()
				project := skillcatalog.Candidate{Name: "project", Path: "/context/projects/project"}
				projectSkills := []skillcatalog.Candidate{{Name: "project-skill", Path: "/project-skill"}}
				commonSkills := []skillcatalog.Candidate{{Name: "common-skill", Path: "/common-skill"}}
				trace := []string{}
				catalog := &stubSkillCatalog{
					projects:      []skillcatalog.Candidate{project},
					projectSkills: projectSkills,
					commonSkills:  commonSkills,
					trace:         &trace,
				}
				prompt := &stubPrompt{
					project:             project,
					selectedProject:     projectSkills,
					addCommon:           true,
					selectedCommon:      commonSkills,
					selectedDestination: []distribution.Destination{distribution.DestinationCodex},
					trace:               &trace,
				}
				options := newAddOptionsForTest(catalog, prompt)
				options.Factory.IsTerminal = func(input io.Reader, output io.Writer) bool {
					trace = append(trace, "tty")
					if input != options.Factory.IOIn || output != options.Factory.IOOut {
						t.Fatal("TTY判定へFactoryのIOが渡されていません")
					}
					return true
				}
				options.Factory.WorkspaceValidator = &stubWorkspaceValidator{
					path: "/workspace",
					call: func() { trace = append(trace, "workspace") },
				}
				options.Factory.Config = func() (Config, error) {
					trace = append(trace, "config")
					return &recordingConfig{savedPath: "/context"}, nil
				}
				options.Factory.RepositoryValidator = &stubRepositoryValidator{
					validatedPath: "/context",
					call:          func() { trace = append(trace, "repository") },
				}
				options.Factory.Prompt = func(input io.Reader, output io.Writer) Prompt {
					if input != options.Factory.IOIn || output != options.Factory.IOOut {
						t.Fatal("Prompt生成へFactoryのIOが渡されていません")
					}
					return prompt
				}

				if err := options.Run(); err != nil {
					t.Fatalf("Run() error = %v", err)
				}
				wantTrace := []string{
					"tty", "workspace", "config", "repository", "projects", "prompt-project",
					"project-skills", "prompt-project-skills", "common-skills", "prompt-confirm-common",
					"prompt-common-skills", "prompt-destinations",
				}
				if !reflect.DeepEqual(trace, wantTrace) {
					t.Fatalf("trace = %v, want %v", trace, wantTrace)
				}
				if !reflect.DeepEqual(prompt.projectCandidates, []skillcatalog.Candidate{project}) {
					t.Fatalf("project candidates = %#v", prompt.projectCandidates)
				}
				if !reflect.DeepEqual(prompt.skillCandidates[SkillKindProject], projectSkills) {
					t.Fatalf("project skill candidates = %#v", prompt.skillCandidates[SkillKindProject])
				}
				if !reflect.DeepEqual(prompt.skillCandidates[SkillKindCommon], commonSkills) {
					t.Fatalf("common skill candidates = %#v", prompt.skillCandidates[SkillKindCommon])
				}
				if !reflect.DeepEqual(catalog.commonInput, projectSkills) {
					t.Fatalf("CommonSkills() input = %#v", catalog.commonInput)
				}
			},
		},
		{
			name: "SkipsDistributionDependenciesWhenNoSkillsSelected",
			run: func(t *testing.T) {
				t.Helper()
				project := skillcatalog.Candidate{Name: "project", Path: "/context/projects/project"}
				options := newAddOptionsForTest(&stubSkillCatalog{project: project}, &stubPrompt{})
				options.ProjectSpecified = true
				options.ProjectName = project.Name
				options.Factory.DistributionExecutor = func(distribution.MapStore) DistributionExecutor {
					t.Fatal("DistributionExecutor must not be called")
					return nil
				}

				if err := options.Run(); err != nil {
					t.Fatalf("Run() error = %v", err)
				}
			},
		},
		{
			name: "PlansExecutesAndPrintsResult",
			run: func(t *testing.T) {
				t.Helper()
				project := skillcatalog.Candidate{Name: "project", Path: "/context/projects/project"}
				skill := skillcatalog.Candidate{Name: "skill", Path: "/context/projects/project/skills/skill"}
				planner := &stubDistributionPlanner{
					plan: distribution.Plan{
						ExpectedRevision: "revision",
						Creates:          []distribution.CreateOperation{{}},
					},
				}
				executor := &stubDistributionExecutor{
					result: distribution.Result{
						Created:      2,
						Destinations: []distribution.Destination{distribution.DestinationCodex, distribution.DestinationClaude},
					},
				}
				options := newAddOptionsForTest(
					&stubSkillCatalog{project: project, projectSkills: []skillcatalog.Candidate{skill}},
					&stubPrompt{
						selectedProject:     []skillcatalog.Candidate{skill},
						selectedDestination: []distribution.Destination{distribution.DestinationCodex, distribution.DestinationClaude},
					},
				)
				options.ProjectSpecified = true
				options.ProjectName = project.Name
				store := &stubMapStore{snapshot: distribution.MapSnapshot{Revision: "revision"}}
				options.Factory.MapStore = func() (distribution.MapStore, error) { return store, nil }
				options.Factory.DistributionPlanner = planner
				options.Factory.DistributionExecutor = func(got distribution.MapStore) DistributionExecutor {
					if got != store {
						t.Fatal("executor received unexpected store")
					}
					return executor
				}

				if err := options.Run(); err != nil {
					t.Fatalf("Run() error = %v", err)
				}
				if !reflect.DeepEqual(planner.selection, options.Selection) {
					t.Fatalf("planner selection = %#v, want %#v", planner.selection, options.Selection)
				}
				if executor.plan.ExpectedRevision != "revision" {
					t.Fatalf("executor plan = %#v", executor.plan)
				}
				outputBuffer, ok := options.Factory.IOOut.(*bytes.Buffer)
				if !ok {
					t.Fatalf("IOOut type = %T, want *bytes.Buffer", options.Factory.IOOut)
				}
				output := outputBuffer.String()
				if output != "2件のSkillをcodex, claudeへ配布しました\n" {
					t.Fatalf("output = %q", output)
				}
			},
		},
		{
			name: "ClearsDefaultSkillsOnProjectSwitch",
			run: func(t *testing.T) {
				t.Helper()
				projectA := skillcatalog.Candidate{Name: "projectA", Path: "/context/projects/projectA"}
				projectB := skillcatalog.Candidate{Name: "projectB", Path: "/context/projects/projectB"}
				projectBSkills := []skillcatalog.Candidate{{Name: "beta", Path: "/context/projects/projectB/skills/beta"}}

				catalog := &stubSkillCatalog{
					projects:      []skillcatalog.Candidate{projectA, projectB},
					project:       projectB,
					projectSkills: projectBSkills,
				}

				prompt := &stubPrompt{
					project:             projectB,
					selectedProject:     projectBSkills,
					selectedDestination: []distribution.Destination{distribution.DestinationCodex},
				}

				options := newAddOptionsForTest(catalog, prompt)

				options.Factory.MapStore = func() (distribution.MapStore, error) {
					return &stubMapStore{
						snapshot: distribution.MapSnapshot{
							Revision: "revision-123",
							Workspaces: map[string]distribution.WorkspaceRecord{
								"/workspace": {
									WorkspaceRoot: "/workspace",
									Project:       "projectA",
									Destinations:  []distribution.Destination{distribution.DestinationCodex},
									Skills: []distribution.SkillRecord{{
										Name:         "alpha",
										Source:       distribution.SkillSourceProject,
										Destination:  distribution.DestinationCodex,
										RelativePath: ".codex/skills/alpha",
										Hash:         "some-hash",
									}},
								},
							},
						},
					}, nil
				}

				if err := options.Run(); err != nil {
					t.Fatalf("Run() error = %v", err)
				}

				if prompt.defaultProject != "projectA" {
					t.Fatalf("expected default project to be projectA, got %q", prompt.defaultProject)
				}

				defaultProjectSkills := prompt.defaultSkills[SkillKindProject]
				if len(defaultProjectSkills) != 0 {
					t.Fatalf("expected project skills default to be cleared, got %#v", defaultProjectSkills)
				}

				defaultCommonSkills := prompt.defaultSkills[SkillKindCommon]
				if len(defaultCommonSkills) != 0 {
					t.Fatalf("expected common skills default to be cleared, got %#v", defaultCommonSkills)
				}
			},
		},
		{
			name: "PromptsForConflictsAndLocalEdits",
			run: func(t *testing.T) {
				t.Helper()
				project := skillcatalog.Candidate{Name: "project", Path: "/context/projects/project"}
				skill := skillcatalog.Candidate{Name: "skill", Path: "/context/projects/project/skills/skill"}

				t.Run("ユーザーが上書きを承認した場合", func(t *testing.T) {
					planner := &stubDistributionPlanner{
						plan: distribution.Plan{
							ExpectedRevision: "revision",
							Creates: []distribution.CreateOperation{
								{Name: "skill", RelativePath: "skills/skill", IsConflict: true},
							},
						},
					}
					executorCalled := false
					executor := &stubDistributionExecutor{
						result: distribution.Result{Created: 1, Destinations: []distribution.Destination{distribution.DestinationCodex}},
					}
					prompt := &stubPrompt{
						selectedProject:     []skillcatalog.Candidate{skill},
						selectedDestination: []distribution.Destination{distribution.DestinationCodex},
						confirmOverwrite:    true, // 承認
					}
					options := newAddOptionsForTest(
						&stubSkillCatalog{project: project, projectSkills: []skillcatalog.Candidate{skill}},
						prompt,
					)
					options.ProjectSpecified = true
					options.ProjectName = project.Name
					options.Factory.DistributionPlanner = planner
					options.Factory.DistributionExecutor = func(_ distribution.MapStore) DistributionExecutor {
						executorCalled = true
						return executor
					}

					if err := options.Run(); err != nil {
						t.Fatalf("Run() error = %v", err)
					}

					if !reflect.DeepEqual(prompt.conflictsSeen, []string{"skills/skill"}) {
						t.Fatalf("expected conflictsSeen to be ['skills/skill'], got %#v", prompt.conflictsSeen)
					}
					if !executorCalled {
						t.Fatal("expected executor to be called but it was not")
					}
				})

				t.Run("ユーザーが上書きを拒否した場合", func(t *testing.T) {
					planner := &stubDistributionPlanner{
						plan: distribution.Plan{
							ExpectedRevision: "revision",
							Creates: []distribution.CreateOperation{
								{Name: "skill", RelativePath: "skills/skill", IsConflict: true},
							},
						},
					}
					executorCalled := false
					prompt := &stubPrompt{
						selectedProject:     []skillcatalog.Candidate{skill},
						selectedDestination: []distribution.Destination{distribution.DestinationCodex},
						confirmOverwrite:    false, // 拒否
					}
					options := newAddOptionsForTest(
						&stubSkillCatalog{project: project, projectSkills: []skillcatalog.Candidate{skill}},
						prompt,
					)
					options.ProjectSpecified = true
					options.ProjectName = project.Name
					options.Factory.DistributionPlanner = planner
					options.Factory.DistributionExecutor = func(_ distribution.MapStore) DistributionExecutor {
						executorCalled = true
						return &stubDistributionExecutor{}
					}

					if err := options.Run(); err != nil {
						t.Fatalf("Run() error = %v, want nil", err)
					}

					if executorCalled {
						t.Fatal("expected executor NOT to be called, but it was")
					}
				})

				t.Run("ユーザーが対話をキャンセル（アボート）した場合", func(t *testing.T) {
					planner := &stubDistributionPlanner{
						plan: distribution.Plan{
							ExpectedRevision: "revision",
							Creates: []distribution.CreateOperation{
								{Name: "skill", RelativePath: "skills/skill", IsConflict: true},
							},
						},
					}
					executorCalled := false
					prompt := &stubPrompt{
						selectedProject:     []skillcatalog.Candidate{skill},
						selectedDestination: []distribution.Destination{distribution.DestinationCodex},
						errors:              map[string]error{"confirm-overwrite": huh.ErrUserAborted}, // アボート
					}
					options := newAddOptionsForTest(
						&stubSkillCatalog{project: project, projectSkills: []skillcatalog.Candidate{skill}},
						prompt,
					)
					options.ProjectSpecified = true
					options.ProjectName = project.Name
					options.Factory.DistributionPlanner = planner
					options.Factory.DistributionExecutor = func(_ distribution.MapStore) DistributionExecutor {
						executorCalled = true
						return &stubDistributionExecutor{}
					}

					if err := options.Run(); err != nil {
						t.Fatalf("Run() error = %v, want nil", err)
					}

					if executorCalled {
						t.Fatal("expected executor NOT to be called, but it was")
					}
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func newAddOptionsForTest(catalog SkillCatalog, prompt Prompt) *AddOptions {
	input := &bytes.Buffer{}
	output := &bytes.Buffer{}
	return &AddOptions{Factory: &Factory{
		IOIn:               input,
		IOOut:              output,
		IsTerminal:         func(io.Reader, io.Writer) bool { return true },
		WorkspaceValidator: &stubWorkspaceValidator{path: "/workspace"},
		RepositoryValidator: &stubRepositoryValidator{
			validatedPath: "/context",
		},
		Config: func() (Config, error) {
			return &recordingConfig{savedPath: "/context"}, nil
		},
		SkillCatalog: func(string) SkillCatalog { return catalog },
		Prompt: func(io.Reader, io.Writer) Prompt {
			return prompt
		},
		MapStore: func() (distribution.MapStore, error) {
			return &stubMapStore{
				snapshot: distribution.MapSnapshot{
					Revision:   distribution.EmptyRevision,
					Workspaces: map[string]distribution.WorkspaceRecord{},
				},
			}, nil
		},
		DistributionPlanner: &stubDistributionPlanner{},
		DistributionExecutor: func(distribution.MapStore) DistributionExecutor {
			return &stubDistributionExecutor{}
		},
	}}
}

type stubMapStore struct {
	snapshot distribution.MapSnapshot
	err      error
}

func (s *stubMapStore) Load() (distribution.MapSnapshot, error) {
	return s.snapshot, s.err
}

func (s *stubMapStore) Begin(distribution.Revision) (
	distribution.MapTransaction,
	distribution.MapSnapshot,
	error,
) {
	return nil, distribution.MapSnapshot{}, errUnexpectedTest
}

type stubDistributionPlanner struct {
	snapshot  distribution.MapSnapshot
	selection distribution.Selection
	plan      distribution.Plan
	err       error
}

func (p *stubDistributionPlanner) Plan(
	snapshot distribution.MapSnapshot,
	selection distribution.Selection,
) (distribution.Plan, error) {
	p.snapshot = snapshot
	p.selection = selection
	return p.plan, p.err
}

type stubDistributionExecutor struct {
	plan   distribution.Plan
	result distribution.Result
	err    error
}

func (e *stubDistributionExecutor) Execute(plan distribution.Plan) (distribution.Result, error) {
	e.plan = plan
	return e.result, e.err
}

type stubWorkspaceValidator struct {
	path string
	err  error
	call func()
}

func (v *stubWorkspaceValidator) Validate() (string, error) {
	if v.call != nil {
		v.call()
	}
	return v.path, v.err
}

type stubSkillCatalog struct {
	projects      []skillcatalog.Candidate
	project       skillcatalog.Candidate
	projectSkills []skillcatalog.Candidate
	commonSkills  []skillcatalog.Candidate
	err           error
	projectErr    error
	projectName   string
	commonInput   []skillcatalog.Candidate
	trace         *[]string
}

func (c *stubSkillCatalog) Projects() ([]skillcatalog.Candidate, error) {
	c.record("projects")
	return c.projects, c.err
}

func (c *stubSkillCatalog) Project(name string) (skillcatalog.Candidate, error) {
	c.record("project")
	c.projectName = name
	if c.projectErr != nil {
		return skillcatalog.Candidate{}, c.projectErr
	}
	return c.project, c.err
}

func (c *stubSkillCatalog) ProjectSkills(skillcatalog.Candidate) ([]skillcatalog.Candidate, error) {
	c.record("project-skills")
	return c.projectSkills, c.err
}

func (c *stubSkillCatalog) CommonSkills(candidates []skillcatalog.Candidate) ([]skillcatalog.Candidate, error) {
	c.record("common-skills")
	c.commonInput = candidates
	return c.commonSkills, c.err
}

func (c *stubSkillCatalog) ResolveRecordedSources(_ string, _ []skillcatalog.RecordedSkillRef) ([]skillcatalog.ResolvedSkillSource, error) {
	c.record("resolve-recorded-sources")
	return nil, c.err
}

func (c *stubSkillCatalog) record(call string) {
	if c.trace != nil {
		*c.trace = append(*c.trace, call)
	}
}

type stubPrompt struct {
	project             skillcatalog.Candidate
	selectedProject     []skillcatalog.Candidate
	addCommon           bool
	selectedCommon      []skillcatalog.Candidate
	selectedDestination []distribution.Destination
	errors              map[string]error
	calls               []string
	projectCandidates   []skillcatalog.Candidate
	skillCandidates     map[SkillKind][]skillcatalog.Candidate
	trace               *[]string
	defaultProject      string
	defaultSkills       map[SkillKind][]string
	defaultConfirmed    bool
	defaultDestinations []distribution.Destination
	confirmOverwrite    bool
	conflictsSeen       []string
	localEditsSeen      []string
	confirmSync         bool
	updatesSeen         []string
	deletesSeen         []string
}

func (p *stubPrompt) SelectProject(candidates []skillcatalog.Candidate, defaultProject string) (skillcatalog.Candidate, error) {
	p.calls = append(p.calls, "project")
	p.record("prompt-project")
	p.projectCandidates = candidates
	p.defaultProject = defaultProject
	return p.project, p.errors["project"]
}

func (p *stubPrompt) SelectSkills(kind SkillKind, candidates []skillcatalog.Candidate, defaultNames []string) ([]skillcatalog.Candidate, error) {
	p.calls = append(p.calls, string(kind))
	p.record("prompt-" + string(kind))
	if p.skillCandidates == nil {
		p.skillCandidates = make(map[SkillKind][]skillcatalog.Candidate)
	}
	p.skillCandidates[kind] = candidates
	if p.defaultSkills == nil {
		p.defaultSkills = make(map[SkillKind][]string)
	}
	p.defaultSkills[kind] = defaultNames
	if kind == SkillKindProject {
		return p.selectedProject, p.errors[string(kind)]
	}
	return p.selectedCommon, p.errors[string(kind)]
}

func (p *stubPrompt) ConfirmCommonSkills(defaultConfirmed bool) (bool, error) {
	p.calls = append(p.calls, "confirm-common")
	p.record("prompt-confirm-common")
	p.defaultConfirmed = defaultConfirmed
	return p.addCommon, p.errors["confirm-common"]
}

func (p *stubPrompt) SelectDestinations(defaultDestinations []distribution.Destination) ([]distribution.Destination, error) {
	p.calls = append(p.calls, "destinations")
	p.record("prompt-destinations")
	p.defaultDestinations = defaultDestinations
	return p.selectedDestination, p.errors["destinations"]
}

func (p *stubPrompt) ConfirmOverwrite(conflicts []string, localEdits []string) (bool, error) {
	p.calls = append(p.calls, "confirm-overwrite")
	p.record("prompt-confirm-overwrite")
	p.conflictsSeen = conflicts
	p.localEditsSeen = localEdits
	return p.confirmOverwrite, p.errors["confirm-overwrite"]
}

func (p *stubPrompt) ConfirmSync(updates []string, deletes []string) (bool, error) {
	p.calls = append(p.calls, "confirm-sync")
	p.record("prompt-confirm-sync")
	p.updatesSeen = updates
	p.deletesSeen = deletes
	return p.confirmSync, p.errors["confirm-sync"]
}

func (p *stubPrompt) record(call string) {
	if p.trace != nil {
		*p.trace = append(*p.trace, call)
	}
}
