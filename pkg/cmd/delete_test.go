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

//nolint:gocognit,cyclop // テーブル駆動テストのため、認知・循環複雑度の上限を無視します。
func TestCmdDelete(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "引数指定による特定のSkill削除",
			run: func(t *testing.T) {
				t.Helper()
				prompt := &stubPrompt{}
				options := newDeleteOptionsForTest(prompt)
				options.SkillNames = []string{"alpha"}

				planner := &stubDistributionPlanner{
					plan: distribution.Plan{
						Deletes: []distribution.DeleteOperation{
							{Name: "alpha", Destination: distribution.DestinationCodex, RelativePath: ".codex/skills/alpha"},
						},
					},
				}
				options.Factory.DistributionPlanner = planner

				executor := &stubDistributionExecutor{}
				options.Factory.DistributionExecutor = func(_ distribution.MapStore) DistributionExecutor {
					return executor
				}

				if err := options.Run(); err != nil {
					t.Fatalf("Run() error = %v", err)
				}

				buf, ok := options.Factory.IOOut.(*bytes.Buffer)
				if !ok {
					t.Fatal("IOOutが*bytes.Bufferではない")
				}
				output := buf.String()
				if output != "1件のSkillを削除しました\n" {
					t.Fatalf("output = %q, want '1件のSkillを削除しました\\n'", output)
				}
			},
		},
		{
			name: "--all 指定による全Skillの削除",
			run: func(t *testing.T) {
				t.Helper()
				prompt := &stubPrompt{}
				options := newDeleteOptionsForTest(prompt)
				options.All = true

				planner := &stubDistributionPlanner{
					plan: distribution.Plan{
						Deletes: []distribution.DeleteOperation{
							{Name: "alpha", Destination: distribution.DestinationCodex, RelativePath: ".codex/skills/alpha"},
							{Name: "beta", Destination: distribution.DestinationClaude, RelativePath: ".claude/skills/beta"},
						},
					},
				}
				options.Factory.DistributionPlanner = planner

				if err := options.Run(); err != nil {
					t.Fatalf("Run() error = %v", err)
				}

				buf, ok := options.Factory.IOOut.(*bytes.Buffer)
				if !ok {
					t.Fatal("IOOutが*bytes.Bufferではない")
				}
				output := buf.String()
				if output != "2件のSkillを削除しました\n" {
					t.Fatalf("output = %q, want '2件のSkillを削除しました\\n'", output)
				}
			},
		},
		{
			name: "対話UI経由でのSkill削除",
			run: func(t *testing.T) {
				t.Helper()
				prompt := &stubPrompt{
					selectedSkillsToDelete: []string{"beta"},
				}
				options := newDeleteOptionsForTest(prompt)

				planner := &stubDistributionPlanner{
					plan: distribution.Plan{
						Deletes: []distribution.DeleteOperation{
							{Name: "beta", Destination: distribution.DestinationClaude, RelativePath: ".claude/skills/beta"},
						},
					},
				}
				options.Factory.DistributionPlanner = planner

				if err := options.Run(); err != nil {
					t.Fatalf("Run() error = %v", err)
				}

				if !reflect.DeepEqual(prompt.skillsToDeleteCandidates, []string{"alpha", "beta"}) {
					t.Fatalf("candidates = %v, want [alpha, beta]", prompt.skillsToDeleteCandidates)
				}

				buf, ok := options.Factory.IOOut.(*bytes.Buffer)
				if !ok {
					t.Fatal("IOOutが*bytes.Bufferではない")
				}
				output := buf.String()
				if output != "1件のSkillを削除しました\n" {
					t.Fatalf("output = %q, want '1件のSkillを削除しました\\n'", output)
				}
			},
		},
		{
			name: "対話UIでキャンセルされた場合の無変更終了",
			run: func(t *testing.T) {
				t.Helper()
				prompt := &stubPrompt{
					errors: map[string]error{"select-skills-to-delete": huh.ErrUserAborted},
				}
				options := newDeleteOptionsForTest(prompt)

				executorCalled := false
				options.Factory.DistributionExecutor = func(_ distribution.MapStore) DistributionExecutor {
					executorCalled = true
					return &stubDistributionExecutor{}
				}

				if err := options.Run(); err != nil {
					t.Fatalf("Run() error = %v, want nil", err)
				}

				if executorCalled {
					t.Fatal("expected executor not to be called")
				}
			},
		},
		{
			name: "未管理Workspaceでのエラー",
			run: func(t *testing.T) {
				t.Helper()
				prompt := &stubPrompt{}
				options := newDeleteOptionsForTest(prompt)
				options.Factory.MapStore = func() (distribution.MapStore, error) {
					return &stubMapStore{
						snapshot: distribution.MapSnapshot{
							Revision:   distribution.EmptyRevision,
							Workspaces: map[string]distribution.WorkspaceRecord{},
						},
					}, nil
				}

				err := options.Run()
				if !errors.Is(err, distribution.ErrUnmanagedWorkspace) {
					t.Fatalf("Run() error = %v, want ErrUnmanagedWorkspace", err)
				}
			},
		},
		{
			name: "存在しないSkill名を引数に含めた際のエラー",
			run: func(t *testing.T) {
				t.Helper()
				prompt := &stubPrompt{}
				options := newDeleteOptionsForTest(prompt)
				options.SkillNames = []string{"nonexistent"}

				err := options.Run()
				if !errors.Is(err, ErrSkillNotDistributed) {
					t.Fatalf("Run() error = %v, want ErrSkillNotDistributed", err)
				}
			},
		},
		{
			name: "ローカル編集検知時の確認プロンプト（承認による削除）",
			run: func(t *testing.T) {
				t.Helper()
				prompt := &stubPrompt{
					confirmOverwrite: true, // 承認
				}
				options := newDeleteOptionsForTest(prompt)
				options.SkillNames = []string{"alpha"}

				planner := &stubDistributionPlanner{
					plan: distribution.Plan{
						Deletes: []distribution.DeleteOperation{
							{Name: "alpha", Destination: distribution.DestinationCodex, RelativePath: ".codex/skills/alpha", IsLocalEdit: true},
						},
					},
				}
				options.Factory.DistributionPlanner = planner

				executorCalled := false
				options.Factory.DistributionExecutor = func(_ distribution.MapStore) DistributionExecutor {
					executorCalled = true
					return &stubDistributionExecutor{}
				}

				if err := options.Run(); err != nil {
					t.Fatalf("Run() error = %v", err)
				}

				if !executorCalled {
					t.Fatal("expected executor to be called")
				}
				if !reflect.DeepEqual(prompt.localEditsSeen, []string{".codex/skills/alpha"}) {
					t.Fatalf("expected localEditsSeen = [.codex/skills/alpha], got %v", prompt.localEditsSeen)
				}
			},
		},
		{
			name: "ローカル編集検知時の確認プロンプト（拒否による無変更終了）",
			run: func(t *testing.T) {
				t.Helper()
				prompt := &stubPrompt{
					confirmOverwrite: false, // 拒否
				}
				options := newDeleteOptionsForTest(prompt)
				options.SkillNames = []string{"alpha"}

				planner := &stubDistributionPlanner{
					plan: distribution.Plan{
						Deletes: []distribution.DeleteOperation{
							{Name: "alpha", Destination: distribution.DestinationCodex, RelativePath: ".codex/skills/alpha", IsLocalEdit: true},
						},
					},
				}
				options.Factory.DistributionPlanner = planner

				executorCalled := false
				options.Factory.DistributionExecutor = func(_ distribution.MapStore) DistributionExecutor {
					executorCalled = true
					return &stubDistributionExecutor{}
				}

				if err := options.Run(); err != nil {
					t.Fatalf("Run() error = %v", err)
				}

				if executorCalled {
					t.Fatal("expected executor NOT to be called")
				}
			},
		},
		{
			name: "非TTY環境におけるローカル編集検知時のエラー終了",
			run: func(t *testing.T) {
				t.Helper()
				prompt := &stubPrompt{}
				options := newDeleteOptionsForTest(prompt)
				options.SkillNames = []string{"alpha"}
				options.Factory.IsTerminal = func(io.Reader, io.Writer) bool { return false } // 非TTY

				planner := &stubDistributionPlanner{
					plan: distribution.Plan{
						Deletes: []distribution.DeleteOperation{
							{Name: "alpha", Destination: distribution.DestinationCodex, RelativePath: ".codex/skills/alpha", IsLocalEdit: true},
						},
					},
				}
				options.Factory.DistributionPlanner = planner

				err := options.Run()
				if !errors.Is(err, distribution.ErrLocalChange) {
					t.Fatalf("Run() error = %v, want ErrLocalChange", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func newDeleteOptionsForTest(prompt Prompt) *DeleteOptions {
	input := &bytes.Buffer{}
	output := &bytes.Buffer{}
	return &DeleteOptions{
		Factory: &Factory{
			IOIn:               input,
			IOOut:              output,
			IsTerminal:         func(io.Reader, io.Writer) bool { return true },
			WorkspaceValidator: &stubWorkspaceValidator{path: "/workspace"},
			Prompt: func(io.Reader, io.Writer) Prompt {
				return prompt
			},
			MapStore: func() (distribution.MapStore, error) {
				return &stubMapStore{
					snapshot: distribution.MapSnapshot{
						Revision: "rev-123",
						Workspaces: map[string]distribution.WorkspaceRecord{
							"/workspace": {
								WorkspaceRoot: "/workspace",
								Project:       "myproject",
								Destinations:  []distribution.Destination{distribution.DestinationCodex, distribution.DestinationClaude},
								Skills: []distribution.SkillRecord{
									{
										Name:         "alpha",
										Source:       distribution.SkillSourceProject,
										Destination:  distribution.DestinationCodex,
										RelativePath: ".codex/skills/alpha",
										Hash:         "alpha-hash",
									},
									{
										Name:         "beta",
										Source:       distribution.SkillSourceProject,
										Destination:  distribution.DestinationClaude,
										RelativePath: ".claude/skills/beta",
										Hash:         "beta-hash",
									},
								},
							},
						},
					},
				}, nil
			},
			DistributionPlanner: &stubDistributionPlanner{},
			DistributionExecutor: func(distribution.MapStore) DistributionExecutor {
				return &stubDistributionExecutor{}
			},
		},
	}
}
