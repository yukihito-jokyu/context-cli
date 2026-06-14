package cmd

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/yukihito-jokyu/context-cli/internal/distribution"
)

type stubSyncPlanner struct {
	plan distribution.SyncPlan
	err  error
}

func (p *stubSyncPlanner) Plan(
	_ distribution.MapSnapshot,
	_ distribution.SyncInput,
	_ []distribution.ResolvedSource,
) (distribution.SyncPlan, error) {
	return p.plan, p.err
}

type stubSyncExecutor struct {
	plan   distribution.Plan
	result distribution.Result
	err    error
}

func (e *stubSyncExecutor) Execute(plan distribution.Plan) (distribution.Result, error) {
	e.plan = plan
	return e.result, e.err
}

//nolint:revive // テスト用ヘルパーのため、引数の上限超過を許容します。
func newSyncOptionsForTest(
	catalog SkillCatalog,
	prompt Prompt,
	store distribution.MapStore,
	planner SyncPlanner,
	executor DistributionExecutor,
) *SyncOptions {
	input := &bytes.Buffer{}
	output := &bytes.Buffer{}
	return &SyncOptions{Factory: &Factory{
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
			return store, nil
		},
		SyncPlanner: planner,
		DistributionExecutor: func(distribution.MapStore) DistributionExecutor {
			return executor
		},
	}}
}

//nolint:gocognit,cyclop // テーブル駆動テストのため、認知・循環複雑度の上限を無視します。
func TestSyncOptionsRun(t *testing.T) {
	tests := []struct {
		name                string
		workspaceValidator  *stubWorkspaceValidator
		repositoryValidator *stubRepositoryValidator
		configPath          string
		store               *stubMapStore
		planner             *stubSyncPlanner
		prompt              *stubPrompt
		executor            *stubSyncExecutor
		isTerminal          bool
		wantErr             error
		wantOutput          string
		wantExecutorCalled  bool
		verify              func(t *testing.T, prompt *stubPrompt, executor *stubSyncExecutor)
	}{
		{
			name:               "Workspace validation failure",
			workspaceValidator: &stubWorkspaceValidator{err: errWorkspaceTest},
			wantErr:            ErrWorkspace,
		},
		{
			name:       "Context repository not configured",
			configPath: "",
			wantErr:    ErrContextRepositoryRequired,
		},
		{
			name:                "Repository validation failure",
			repositoryValidator: &stubRepositoryValidator{err: errRepositoryTest},
			wantErr:             ErrRepository,
		},
		{
			name: "Unmanaged workspace",
			store: &stubMapStore{
				snapshot: distribution.MapSnapshot{
					Revision:   "revision-1",
					Workspaces: map[string]distribution.WorkspaceRecord{}, // 空マップのため/workspaceは未登録
				},
			},
			wantErr: distribution.ErrUnmanagedWorkspace,
		},
		{
			name: "No change",
			store: &stubMapStore{
				snapshot: distribution.MapSnapshot{
					Revision: "revision-1",
					Workspaces: map[string]distribution.WorkspaceRecord{
						"/workspace": {
							WorkspaceRoot: "/workspace",
							Project:       "projectA",
							Destinations:  []distribution.Destination{distribution.DestinationCodex},
							Skills: []distribution.SkillRecord{
								{Name: "skill-1", Source: distribution.SkillSourceProject, Destination: distribution.DestinationCodex, Hash: "hash-1"},
							},
						},
					},
				},
			},
			planner: &stubSyncPlanner{
				plan: distribution.SyncPlan{
					ExpectedRevision: "revision-1",
					WorkspaceRoot:    "/workspace",
					Project:          "projectA",
					Updates:          nil,
					Deletes:          nil,
				},
			},
			wantOutput: "同期対象に変更はありません\n",
		},
		{
			name: "Success with update",
			store: &stubMapStore{
				snapshot: distribution.MapSnapshot{
					Revision: "revision-1",
					Workspaces: map[string]distribution.WorkspaceRecord{
						"/workspace": {
							WorkspaceRoot: "/workspace",
							Project:       "projectA",
							Destinations:  []distribution.Destination{distribution.DestinationCodex},
							Skills: []distribution.SkillRecord{
								{Name: "skill-1", Source: distribution.SkillSourceProject, Destination: distribution.DestinationCodex, Hash: "hash-1"},
							},
						},
					},
				},
			},
			planner: &stubSyncPlanner{
				plan: distribution.SyncPlan{
					ExpectedRevision: "revision-1",
					WorkspaceRoot:    "/workspace",
					Project:          "projectA",
					Updates: []distribution.SyncOperation{
						{Kind: distribution.SyncOperationUpdate, Name: "skill-1", Destination: distribution.DestinationCodex, RelativePath: ".codex/skills/skill-1"},
					},
				},
			},
			executor: &stubSyncExecutor{
				result: distribution.Result{
					UniqueUpdated: 1,
					UniqueDeleted: 0,
				},
			},
			wantExecutorCalled: true,
			wantOutput:         "1件のSkillを更新し、0件を削除しました\n",
			verify: func(t *testing.T, _ *stubPrompt, executor *stubSyncExecutor) {
				t.Helper()
				if executor.plan.ExpectedRevision != "revision-1" || len(executor.plan.Creates) != 1 {
					t.Fatalf("executor received unexpected plan: %#v", executor.plan)
				}
			},
		},
		{
			name: "Local changes - Non-TTY",
			store: &stubMapStore{
				snapshot: distribution.MapSnapshot{
					Revision: "revision-1",
					Workspaces: map[string]distribution.WorkspaceRecord{
						"/workspace": {
							WorkspaceRoot: "/workspace",
							Project:       "projectA",
							Destinations:  []distribution.Destination{distribution.DestinationCodex},
							Skills: []distribution.SkillRecord{
								{Name: "skill-1", Source: distribution.SkillSourceProject, Destination: distribution.DestinationCodex, Hash: "hash-1"},
							},
						},
					},
				},
			},
			planner: &stubSyncPlanner{
				plan: distribution.SyncPlan{
					ExpectedRevision: "revision-1",
					WorkspaceRoot:    "/workspace",
					Project:          "projectA",
					Updates: []distribution.SyncOperation{
						{Kind: distribution.SyncOperationUpdate, Name: "skill-1", Destination: distribution.DestinationCodex, RelativePath: ".codex/skills/skill-1", IsLocalEdit: true},
					},
					LocalChanges: []distribution.SyncOperation{
						{Kind: distribution.SyncOperationUpdate, Name: "skill-1", Destination: distribution.DestinationCodex, RelativePath: ".codex/skills/skill-1", IsLocalEdit: true},
					},
				},
			},
			isTerminal: false,
			wantErr:    distribution.ErrLocalChange,
		},
		{
			name: "Local changes - TTY Approve",
			store: &stubMapStore{
				snapshot: distribution.MapSnapshot{
					Revision: "revision-1",
					Workspaces: map[string]distribution.WorkspaceRecord{
						"/workspace": {
							WorkspaceRoot: "/workspace",
							Project:       "projectA",
							Destinations:  []distribution.Destination{distribution.DestinationCodex, distribution.DestinationClaude},
							Skills: []distribution.SkillRecord{
								{Name: "skill-1", Source: distribution.SkillSourceProject, Destination: distribution.DestinationCodex, Hash: "hash-1"},
								{Name: "skill-2", Source: distribution.SkillSourceProject, Destination: distribution.DestinationClaude, Hash: "hash-2"},
							},
						},
					},
				},
			},
			planner: &stubSyncPlanner{
				plan: distribution.SyncPlan{
					ExpectedRevision: "revision-1",
					WorkspaceRoot:    "/workspace",
					Project:          "projectA",
					Updates: []distribution.SyncOperation{
						{Kind: distribution.SyncOperationUpdate, Name: "skill-1", Destination: distribution.DestinationCodex, RelativePath: ".codex/skills/skill-1", IsLocalEdit: true},
					},
					Deletes: []distribution.SyncOperation{
						{Kind: distribution.SyncOperationDelete, Name: "skill-2", Destination: distribution.DestinationClaude, RelativePath: ".claude/skills/skill-2", IsLocalEdit: true},
					},
					LocalChanges: []distribution.SyncOperation{
						{Kind: distribution.SyncOperationUpdate, Name: "skill-1", Destination: distribution.DestinationCodex, RelativePath: ".codex/skills/skill-1", IsLocalEdit: true},
						{Kind: distribution.SyncOperationDelete, Name: "skill-2", Destination: distribution.DestinationClaude, RelativePath: ".claude/skills/skill-2", IsLocalEdit: true},
					},
				},
			},
			prompt: &stubPrompt{
				confirmSync: true,
			},
			executor: &stubSyncExecutor{
				result: distribution.Result{
					UniqueUpdated: 1,
					UniqueDeleted: 1,
				},
			},
			isTerminal:         true,
			wantExecutorCalled: true,
			wantOutput:         "1件のSkillを更新し、1件を削除しました\n",
			verify: func(t *testing.T, prompt *stubPrompt, _ *stubSyncExecutor) {
				t.Helper()
				if !reflect.DeepEqual(prompt.updatesSeen, []string{".codex/skills/skill-1"}) {
					t.Fatalf("expected updatesSeen to be ['.codex/skills/skill-1'], got %v", prompt.updatesSeen)
				}
				if !reflect.DeepEqual(prompt.deletesSeen, []string{".claude/skills/skill-2"}) {
					t.Fatalf("expected deletesSeen to be ['.claude/skills/skill-2'], got %v", prompt.deletesSeen)
				}
			},
		},
		{
			name: "Local changes - TTY Reject",
			store: &stubMapStore{
				snapshot: distribution.MapSnapshot{
					Revision: "revision-1",
					Workspaces: map[string]distribution.WorkspaceRecord{
						"/workspace": {
							WorkspaceRoot: "/workspace",
							Project:       "projectA",
							Destinations:  []distribution.Destination{distribution.DestinationCodex},
							Skills: []distribution.SkillRecord{
								{Name: "skill-1", Source: distribution.SkillSourceProject, Destination: distribution.DestinationCodex, Hash: "hash-1"},
							},
						},
					},
				},
			},
			planner: &stubSyncPlanner{
				plan: distribution.SyncPlan{
					ExpectedRevision: "revision-1",
					WorkspaceRoot:    "/workspace",
					Project:          "projectA",
					Updates: []distribution.SyncOperation{
						{Kind: distribution.SyncOperationUpdate, Name: "skill-1", Destination: distribution.DestinationCodex, RelativePath: ".codex/skills/skill-1", IsLocalEdit: true},
					},
					LocalChanges: []distribution.SyncOperation{
						{Kind: distribution.SyncOperationUpdate, Name: "skill-1", Destination: distribution.DestinationCodex, RelativePath: ".codex/skills/skill-1", IsLocalEdit: true},
					},
				},
			},
			prompt: &stubPrompt{
				confirmSync: false,
			},
			isTerminal: true,
		},
		{
			name: "Local changes - TTY Cancel",
			store: &stubMapStore{
				snapshot: distribution.MapSnapshot{
					Revision: "revision-1",
					Workspaces: map[string]distribution.WorkspaceRecord{
						"/workspace": {
							WorkspaceRoot: "/workspace",
							Project:       "projectA",
							Destinations:  []distribution.Destination{distribution.DestinationCodex},
							Skills: []distribution.SkillRecord{
								{Name: "skill-1", Source: distribution.SkillSourceProject, Destination: distribution.DestinationCodex, Hash: "hash-1"},
							},
						},
					},
				},
			},
			planner: &stubSyncPlanner{
				plan: distribution.SyncPlan{
					ExpectedRevision: "revision-1",
					WorkspaceRoot:    "/workspace",
					Project:          "projectA",
					Updates: []distribution.SyncOperation{
						{Kind: distribution.SyncOperationUpdate, Name: "skill-1", Destination: distribution.DestinationCodex, RelativePath: ".codex/skills/skill-1", IsLocalEdit: true},
					},
					LocalChanges: []distribution.SyncOperation{
						{Kind: distribution.SyncOperationUpdate, Name: "skill-1", Destination: distribution.DestinationCodex, RelativePath: ".codex/skills/skill-1", IsLocalEdit: true},
					},
				},
			},
			prompt: &stubPrompt{
				errors: map[string]error{"confirm-sync": huh.ErrUserAborted},
			},
			isTerminal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// デフォルト値の設定
			catalog := &stubSkillCatalog{}
			prompt := &stubPrompt{}
			if tt.prompt != nil {
				prompt = tt.prompt
			}
			store := &stubMapStore{}
			if tt.store != nil {
				store = tt.store
			}
			planner := &stubSyncPlanner{}
			if tt.planner != nil {
				planner = tt.planner
			}
			executor := &stubSyncExecutor{}
			if tt.executor != nil {
				executor = tt.executor
			}

			options := newSyncOptionsForTest(catalog, prompt, store, planner, executor)

			// カスタマイズされたフックやモックの注入
			if tt.workspaceValidator != nil {
				options.Factory.WorkspaceValidator = tt.workspaceValidator
			}
			if tt.repositoryValidator != nil {
				options.Factory.RepositoryValidator = tt.repositoryValidator
			}
			if tt.configPath != "" || tt.name == "Context repository not configured" {
				options.Factory.Config = func() (Config, error) {
					return &recordingConfig{savedPath: tt.configPath}, nil
				}
			}
			options.Factory.IsTerminal = func(io.Reader, io.Writer) bool { return tt.isTerminal }

			executorCalled := false
			options.Factory.DistributionExecutor = func(distribution.MapStore) DistributionExecutor {
				executorCalled = true
				return executor
			}

			// 実行
			err := options.Run()

			// 結果の検証
			if tt.wantErr != nil {
				if err == nil || !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if executorCalled != tt.wantExecutorCalled {
				t.Fatalf("executorCalled = %t, want %t", executorCalled, tt.wantExecutorCalled)
			}

			buf, ok := options.Factory.IOOut.(*bytes.Buffer)
			if !ok {
				t.Fatal("IOOut is not a bytes.Buffer")
			}
			output := buf.String()
			if output != tt.wantOutput {
				t.Fatalf("unexpected output: %q, want %q", output, tt.wantOutput)
			}

			if tt.verify != nil {
				tt.verify(t, prompt, executor)
			}
		})
	}
}
