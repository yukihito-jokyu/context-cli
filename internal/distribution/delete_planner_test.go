package distribution

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

//nolint:gocognit,cyclop // テーブル駆動テストのため、認知・循環複雑度の上限を無視します。
func TestPlanDelete(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "DeletesSpecificSkillsAndReconstitutesWorkspace",
			run: func(t *testing.T) {
				t.Helper()
				base, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				workspace := filepath.Join(base, "workspace")

				// 擬似的なWorkspace Skillを作成
				alphaDest := filepath.Join(workspace, ".codex", "skills", "alpha")
				betaDest := filepath.Join(workspace, ".claude", "skills", "beta")
				for _, path := range []string{alphaDest, betaDest} {
					if err := os.MkdirAll(path, 0o755); err != nil { // #nosec G301 -- 配布先権限要件を再現します。
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte("skill"), 0o600); err != nil {
						t.Fatal(err)
					}
				}

				snapshot := MapSnapshot{
					Revision: "rev-1",
					Workspaces: map[string]WorkspaceRecord{
						workspace: {
							WorkspaceRoot: workspace,
							Project:       "myproject",
							Destinations:  []Destination{DestinationCodex, DestinationClaude},
							Skills: []SkillRecord{
								{
									Name:         "alpha",
									Source:       SkillSourceProject,
									Destination:  DestinationCodex,
									RelativePath: ".codex/skills/alpha",
									Hash:         "alpha-hash",
								},
								{
									Name:         "beta",
									Source:       SkillSourceProject,
									Destination:  DestinationClaude,
									RelativePath: ".claude/skills/beta",
									Hash:         "beta-hash",
								},
							},
						},
					},
				}

				// "alpha"を削除し、"beta"を維持する計画
				planner := NewPlanner(NewOSFileSystem())
				
				// この正常系のテストケースでローカル編集扱いになるのを防ぐため、ハッシュをディスク内容と一致させる
				alphaHash, err := NewOSFileSystem().HashSkill(alphaDest)
				if err != nil {
					t.Fatal(err)
				}
				snapshot.Workspaces[workspace].Skills[0].Hash = alphaHash

				plan, err := planner.PlanDelete(snapshot, workspace, []string{"alpha"})
				if err != nil {
					t.Fatalf("PlanDelete() error = %v", err)
				}

				// 削除計画（Deletes）を検証
				if len(plan.Deletes) != 1 {
					t.Fatalf("len(Deletes) = %d, want 1", len(plan.Deletes))
				}
				if plan.Deletes[0].Name != "alpha" {
					t.Errorf("Deletes[0].Name = %q, want alpha", plan.Deletes[0].Name)
				}
				if plan.Deletes[0].IsLocalEdit {
					t.Error("expected alpha not to have local edits")
				}

				// 削除後のWorkspaceレコードを検証
				if len(plan.Workspace.Skills) != 1 {
					t.Fatalf("len(Workspace.Skills) = %d, want 1", len(plan.Workspace.Skills))
				}
				if plan.Workspace.Skills[0].Name != "beta" {
					t.Errorf("Workspace.Skills[0].Name = %q, want beta", plan.Workspace.Skills[0].Name)
				}

				// 宛先（Destinations）にはClaudeのみが含まれることを検証
				if len(plan.Workspace.Destinations) != 1 || plan.Workspace.Destinations[0] != DestinationClaude {
					t.Errorf("Workspace.Destinations = %v, want [claude]", plan.Workspace.Destinations)
				}
			},
		},
		{
			name: "DeletesAllSkillsAndCleansWorkspaceRecord",
			run: func(t *testing.T) {
				t.Helper()
				base, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				workspace := filepath.Join(base, "workspace")

				alphaDest := filepath.Join(workspace, ".codex", "skills", "alpha")
				if err := os.MkdirAll(alphaDest, 0o755); err != nil { // #nosec G301 -- 配布先権限要件を再現します。
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(alphaDest, "SKILL.md"), []byte("skill"), 0o600); err != nil {
					t.Fatal(err)
				}

				alphaHash, err := NewOSFileSystem().HashSkill(alphaDest)
				if err != nil {
					t.Fatal(err)
				}

				snapshot := MapSnapshot{
					Revision: "rev-1",
					Workspaces: map[string]WorkspaceRecord{
						workspace: {
							WorkspaceRoot: workspace,
							Project:       "myproject",
							Destinations:  []Destination{DestinationCodex},
							Skills: []SkillRecord{
								{
									Name:         "alpha",
									Source:       SkillSourceProject,
									Destination:  DestinationCodex,
									RelativePath: ".codex/skills/alpha",
									Hash:         alphaHash,
								},
							},
						},
					},
				}

				planner := NewPlanner(NewOSFileSystem())
				plan, err := planner.PlanDelete(snapshot, workspace, []string{"alpha"})
				if err != nil {
					t.Fatalf("PlanDelete() error = %v", err)
				}

				if len(plan.Deletes) != 1 || plan.Deletes[0].Name != "alpha" {
					t.Fatalf("Deletes = %v, want alpha", plan.Deletes)
				}

				if len(plan.Workspace.Skills) != 0 {
					t.Errorf("Workspace.Skills = %v, want empty", plan.Workspace.Skills)
				}
				if len(plan.Workspace.Destinations) != 0 {
					t.Errorf("Workspace.Destinations = %v, want empty", plan.Workspace.Destinations)
				}
			},
		},
		{
			name: "FailsOnUnmanagedWorkspace",
			run: func(t *testing.T) {
				t.Helper()
				planner := NewPlanner(NewOSFileSystem())
				snapshot := MapSnapshot{
					Revision:   "rev-1",
					Workspaces: map[string]WorkspaceRecord{},
				}
				_, err := planner.PlanDelete(snapshot, "/unmanaged/workspace", []string{"alpha"})
				if !errors.Is(err, ErrUnmanagedWorkspace) {
					t.Fatalf("expected ErrUnmanagedWorkspace, got: %v", err)
				}
			},
		},
		{
			name: "FailsOnNonexistentSkill",
			run: func(t *testing.T) {
				t.Helper()
				base, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				workspace := filepath.Join(base, "workspace")

				snapshot := MapSnapshot{
					Revision: "rev-1",
					Workspaces: map[string]WorkspaceRecord{
						workspace: {
							WorkspaceRoot: workspace,
							Project:       "myproject",
							Destinations:  []Destination{DestinationCodex},
							Skills: []SkillRecord{
								{
									Name:         "alpha",
									Source:       SkillSourceProject,
									Destination:  DestinationCodex,
									RelativePath: ".codex/skills/alpha",
									Hash:         "alpha-hash",
								},
							},
						},
					},
				}

				planner := NewPlanner(NewOSFileSystem())
				_, err = planner.PlanDelete(snapshot, workspace, []string{"nonexistent"})
				if !errors.Is(err, ErrPrecondition) {
					t.Fatalf("expected ErrPrecondition, got: %v", err)
				}
				if err.Error() != "distribution plan delete failed" {
					t.Errorf("unexpected error string: %v", err.Error())
				}
			},
		},
		{
			name: "DetectsLocalEditsOnDeletion",
			run: func(t *testing.T) {
				t.Helper()
				base, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				workspace := filepath.Join(base, "workspace")

				alphaDest := filepath.Join(workspace, ".codex", "skills", "alpha")
				if err := os.MkdirAll(alphaDest, 0o755); err != nil { // #nosec G301 -- 配布先権限要件を再現します。
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(alphaDest, "SKILL.md"), []byte("modified-locally"), 0o600); err != nil {
					t.Fatal(err)
				}

				snapshot := MapSnapshot{
					Revision: "rev-1",
					Workspaces: map[string]WorkspaceRecord{
						workspace: {
							WorkspaceRoot: workspace,
							Project:       "myproject",
							Destinations:  []Destination{DestinationCodex},
							Skills: []SkillRecord{
								{
									Name:         "alpha",
									Source:       SkillSourceProject,
									Destination:  DestinationCodex,
									RelativePath: ".codex/skills/alpha",
									Hash:         "different-hash", // ハッシュ不一致
								},
							},
						},
					},
				}

				planner := NewPlanner(NewOSFileSystem())
				plan, err := planner.PlanDelete(snapshot, workspace, []string{"alpha"})
				if err != nil {
					t.Fatalf("PlanDelete() error = %v", err)
				}

				if len(plan.Deletes) != 1 {
					t.Fatalf("len(Deletes) = %d, want 1", len(plan.Deletes))
				}
				if !plan.Deletes[0].IsLocalEdit {
					t.Error("expected alpha to be detected as local edit")
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
