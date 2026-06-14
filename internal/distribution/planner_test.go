package distribution

import (
	"os"
	"path/filepath"
	"testing"
)

//nolint:gocognit,cyclop // テーブル駆動テストのため、認知・循環複雑度の上限を無視します。
func TestPlanner(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "CreatesInitialPlanForEverySkillAndDestination",
			run: func(t *testing.T) {
				t.Helper()
				base, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				workspace := filepath.Join(base, "workspace")
				sourceA := filepath.Join(base, "context", "projects", "project", "skills", "alpha")
				sourceB := filepath.Join(base, "context", "utils", "skills", "common")
				for _, path := range []string{workspace, sourceA, sourceB} {
					if err := os.MkdirAll(path, 0o755); err != nil { // #nosec G301 -- 配布元と配布先の権限要件を再現します。
						t.Fatal(err)
					}
				}
				for _, path := range []string{sourceA, sourceB} {
					if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte(path), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				selection := Selection{
					WorkspaceRoot: workspace,
					Project:       "project",
					Skills: []SelectedSkill{
						{Name: "alpha", Source: SkillSourceProject, SourcePath: sourceA},
						{Name: "common", Source: SkillSourceCommon, SourcePath: sourceB},
					},
					Destinations: []Destination{DestinationCodex, DestinationClaude},
				}

				plan, err := NewPlanner(NewOSFileSystem()).Plan(
					MapSnapshot{Revision: EmptyRevision, Workspaces: map[string]WorkspaceRecord{}},
					selection,
				)
				if err != nil {
					t.Fatalf("Plan() error = %v", err)
				}
				if len(plan.Creates) != 4 {
					t.Fatalf("len(Creates) = %d, want 4", len(plan.Creates))
				}
				if plan.ExpectedRevision != EmptyRevision {
					t.Fatalf("ExpectedRevision = %q", plan.ExpectedRevision)
				}
				if len(plan.Workspace.Skills) != 4 {
					t.Fatalf("len(Workspace.Skills) = %d, want 4", len(plan.Workspace.Skills))
				}
				for _, operation := range plan.Creates {
					if len(operation.Hash) != 64 {
						t.Fatalf("Hash = %q", operation.Hash)
					}
					if len(operation.SourcePathStates) == 0 || len(operation.TargetPathStates) == 0 {
						t.Fatalf("path states are empty: %#v", operation)
					}
					if operation.TargetPathStates[len(operation.TargetPathStates)-1].Exists {
						t.Fatalf("final target must be absent: %#v", operation.TargetPathStates)
					}
				}
			},
		},
		{
			name: "SupportsManagedWorkspaceAndDetectsConflict",
			run: func(t *testing.T) {
				t.Helper()
				base, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				workspace := filepath.Join(base, "workspace")
				source := filepath.Join(base, "context", "projects", "project", "skills", "alpha")
				// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
				if err := os.MkdirAll(source, 0o755); err != nil {
					t.Fatal(err)
				}
				// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
				if err := os.MkdirAll(workspace, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("skill"), 0o600); err != nil {
					t.Fatal(err)
				}
				selection := Selection{
					WorkspaceRoot: workspace,
					Project:       "project",
					Skills:        []SelectedSkill{{Name: "alpha", Source: SkillSourceProject, SourcePath: source}},
					Destinations:  []Destination{DestinationCodex},
				}
				planner := NewPlanner(NewOSFileSystem())

				hash, err := NewOSFileSystem().HashSkill(source)
				if err != nil {
					t.Fatal(err)
				}

				// 配布先にファイルを事前に配置して、欠落と判定されないようにする
				alphaDest := filepath.Join(workspace, ".codex", "skills", "alpha")
				// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
				if err := os.MkdirAll(alphaDest, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(alphaDest, "SKILL.md"), []byte("skill"), 0o600); err != nil {
					t.Fatal(err)
				}

				// 管理済みのWorkspaceで、かつ選択に変更がない場合はCreatesもDeletesも空になることを検証
				plan, err := planner.Plan(MapSnapshot{
					Revision: "revision",
					Workspaces: map[string]WorkspaceRecord{
						workspace: {
							WorkspaceRoot: workspace,
							Project:       "project",
							Destinations:  []Destination{DestinationCodex},
							Skills: []SkillRecord{{
								Name:         "alpha",
								Source:       SkillSourceProject,
								Destination:  DestinationCodex,
								RelativePath: ".codex/skills/alpha",
								Hash:         hash,
							}},
						},
					},
				}, selection)
				if err != nil {
					t.Fatalf("Plan() error = %v", err)
				}
				if len(plan.Creates) != 0 || len(plan.Deletes) != 0 {
					t.Fatalf("Plan.Creates = %d, Deletes = %d, want both 0", len(plan.Creates), len(plan.Deletes))
				}

				// 未管理の対象が既にディスク上に存在する場合は衝突（IsConflict = true）とすることを検証
				conflictPlan, err := planner.Plan(
					MapSnapshot{Revision: EmptyRevision, Workspaces: map[string]WorkspaceRecord{}},
					selection,
				)
				if err != nil {
					t.Fatalf("Plan() error = %v", err)
				}
				if len(conflictPlan.Creates) != 1 || !conflictPlan.Creates[0].IsConflict {
					t.Fatalf("expected conflict for alpha, got: %#v", conflictPlan.Creates)
				}
			},
		},
		{
			name: "GeneratesDeletesWhenSkillsAreRemoved",
			run: func(t *testing.T) {
				t.Helper()
				base, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				workspace := filepath.Join(base, "workspace")
				sourceA := filepath.Join(base, "context", "projects", "project", "skills", "alpha")
				sourceB := filepath.Join(base, "context", "projects", "project", "skills", "beta")
				for _, path := range []string{workspace, sourceA, sourceB} {
					if err := os.MkdirAll(path, 0o755); err != nil { // #nosec G301 -- ディレクトリの権限要件を再現します。
						t.Fatal(err)
					}
				}
				for _, path := range []string{sourceA, sourceB} {
					if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte("skill"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				selection := Selection{
					WorkspaceRoot: workspace,
					Project:       "project",
					Skills:        []SelectedSkill{{Name: "beta", Source: SkillSourceProject, SourcePath: sourceB}},
					Destinations:  []Destination{DestinationCodex},
				}
				planner := NewPlanner(NewOSFileSystem())

				plan, err := planner.Plan(MapSnapshot{
					Revision: "revision",
					Workspaces: map[string]WorkspaceRecord{
						workspace: {
							WorkspaceRoot: workspace,
							Project:       "project",
							Destinations:  []Destination{DestinationCodex},
							Skills: []SkillRecord{{
								Name:         "alpha",
								Source:       SkillSourceProject,
								Destination:  DestinationCodex,
								RelativePath: ".codex/skills/alpha",
								Hash:         "hash",
							}},
						},
					},
				}, selection)
				if err != nil {
					t.Fatalf("Plan() error = %v", err)
				}
				if len(plan.Creates) != 1 || plan.Creates[0].Name != "beta" {
					t.Fatalf("Creates = %v, want beta", plan.Creates)
				}
				if len(plan.Deletes) != 1 || plan.Deletes[0].Name != "alpha" {
					t.Fatalf("Deletes = %v, want alpha", plan.Deletes)
				}
			},
		},
		{
			name: "HandlesProjectSwitch",
			run: func(t *testing.T) {
				t.Helper()
				base, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				workspace := filepath.Join(base, "workspace")
				sourceA := filepath.Join(base, "context", "projects", "projectA", "skills", "alpha")
				sourceB := filepath.Join(base, "context", "projects", "projectB", "skills", "beta")
				for _, path := range []string{workspace, sourceA, sourceB} {
					// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
					if err := os.MkdirAll(path, 0o755); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(filepath.Join(sourceA, "SKILL.md"), []byte("alpha"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(sourceB, "SKILL.md"), []byte("beta"), 0o600); err != nil {
					t.Fatal(err)
				}

				snapshot := MapSnapshot{
					Revision: "revision-123",
					Workspaces: map[string]WorkspaceRecord{
						workspace: {
							WorkspaceRoot: workspace,
							Project:       "projectA",
							Destinations:  []Destination{DestinationCodex},
							Skills: []SkillRecord{{
								Name:         "alpha",
								Source:       SkillSourceProject,
								Destination:  DestinationCodex,
								RelativePath: ".codex/skills/alpha",
								Hash:         "old-hash",
							}},
						},
					},
				}

				selection := Selection{
					WorkspaceRoot: workspace,
					Project:       "projectB",
					Skills: []SelectedSkill{
						{Name: "beta", Source: SkillSourceProject, SourcePath: sourceB},
					},
					Destinations: []Destination{DestinationCodex},
				}

				planner := NewPlanner(NewOSFileSystem())
				plan, err := planner.Plan(snapshot, selection)
				if err != nil {
					t.Fatalf("Plan() error = %v", err)
				}

				if len(plan.Deletes) != 1 || plan.Deletes[0].Name != "alpha" {
					t.Fatalf("expected 1 Delete for alpha, got: %#v", plan.Deletes)
				}

				if len(plan.Creates) != 1 || plan.Creates[0].Name != "beta" {
					t.Fatalf("expected 1 Create for beta, got: %#v", plan.Creates)
				}
			},
		},
		{
			name: "DetectsConflictsAndLocalEdits",
			run: func(t *testing.T) {
				t.Helper()
				base, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				workspace := filepath.Join(base, "workspace")
				sourceAlpha := filepath.Join(base, "context", "projects", "project", "skills", "alpha")
				sourceBeta := filepath.Join(base, "context", "projects", "project", "skills", "beta")
				sourceGamma := filepath.Join(base, "context", "projects", "project", "skills", "gamma")

				for _, path := range []string{workspace, sourceAlpha, sourceBeta, sourceGamma} {
					// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
					if err := os.MkdirAll(path, 0o755); err != nil {
						t.Fatal(err)
					}
				}
				for _, path := range []string{sourceAlpha, sourceBeta, sourceGamma} {
					if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte(filepath.Base(path)), 0o600); err != nil {
						t.Fatal(err)
					}
				}

				alphaDest := filepath.Join(workspace, ".codex", "skills", "alpha")
				// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
				if err := os.MkdirAll(alphaDest, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(alphaDest, "SKILL.md"), []byte("unmanaged"), 0o600); err != nil {
					t.Fatal(err)
				}

				betaDest := filepath.Join(workspace, ".codex", "skills", "beta")
				// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
				if err := os.MkdirAll(betaDest, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(betaDest, "SKILL.md"), []byte("modified-locally"), 0o600); err != nil {
					t.Fatal(err)
				}

				snapshot := MapSnapshot{
					Revision: "rev-1",
					Workspaces: map[string]WorkspaceRecord{
						workspace: {
							WorkspaceRoot: workspace,
							Project:       "project",
							Destinations:  []Destination{DestinationCodex},
							Skills: []SkillRecord{
								{
									Name:         "beta",
									Source:       SkillSourceProject,
									Destination:  DestinationCodex,
									RelativePath: ".codex/skills/beta",
									Hash:         "original-beta-hash",
								},
								{
									Name:         "gamma",
									Source:       SkillSourceProject,
									Destination:  DestinationCodex,
									RelativePath: ".codex/skills/gamma",
									Hash:         "original-gamma-hash",
								},
							},
						},
					},
				}

				selection := Selection{
					WorkspaceRoot: workspace,
					Project:       "project",
					Skills: []SelectedSkill{
						{Name: "alpha", Source: SkillSourceProject, SourcePath: sourceAlpha},
						{Name: "beta", Source: SkillSourceProject, SourcePath: sourceBeta},
						{Name: "gamma", Source: SkillSourceProject, SourcePath: sourceGamma},
					},
					Destinations: []Destination{DestinationCodex},
				}

				planner := NewPlanner(NewOSFileSystem())
				plan, err := planner.Plan(snapshot, selection)
				if err != nil {
					t.Fatalf("Plan() error = %v", err)
				}

				foundAlpha, foundBeta, foundGamma := false, false, false
				for _, op := range plan.Creates {
					switch op.Name {
					case "alpha":
						foundAlpha = true
						if !op.IsConflict {
							t.Error("expected alpha to be detected as conflict")
						}
						if op.IsLocalEdit {
							t.Error("alpha should not be marked as local edit")
						}
					case "beta":
						foundBeta = true
						if op.IsConflict {
							t.Error("beta should not be marked as conflict")
						}
						if !op.IsLocalEdit {
							t.Error("expected beta to be detected as local edit")
						}
					case "gamma":
						foundGamma = true
						if op.IsConflict {
							t.Error("gamma should not be marked as conflict")
						}
						if !op.IsLocalEdit {
							t.Error("expected gamma to be detected as local edit (missing)")
						}
					}
				}

				if !foundAlpha || !foundBeta || !foundGamma {
					t.Fatalf("missing plans: alpha=%t, beta=%t, gamma=%t", foundAlpha, foundBeta, foundGamma)
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
