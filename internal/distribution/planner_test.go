package distribution

import (
	"os"
	"path/filepath"
	"testing"
)

//nolint:gocognit // 複数Skill・複数配布先の計画全体を一つのシナリオで検証します。
func TestPlannerCreatesInitialPlanForEverySkillAndDestination(t *testing.T) {
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
}

func TestPlannerSupportsManagedWorkspaceAndDetectsConflict(t *testing.T) {
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
	// 一旦、上で配置した alphaDest をそのまま使用
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
}

func TestPlannerGeneratesDeletesWhenSkillsAreRemoved(t *testing.T) {
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
}

func TestPlannerHandlesProjectSwitch(t *testing.T) {
	// 既存管理中のプロジェクトから異なるプロジェクトに切り替える際、
	// 旧プロジェクト由来の全SkillがDeletesに選別され、新プロジェクトのSkillがCreatesとして選別されることを検証します。
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

	// プロジェクトAのSkill "alpha"が配置済みである状態のSnapshot
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

	// プロジェクトBのSkill "beta"を選択するSelection
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

	// 旧プロジェクトのalphaがDeletesに入っていること
	if len(plan.Deletes) != 1 || plan.Deletes[0].Name != "alpha" {
		t.Fatalf("expected 1 Delete for alpha, got: %#v", plan.Deletes)
	}

	// 新プロジェクトのbetaがCreatesに入っていること
	if len(plan.Creates) != 1 || plan.Creates[0].Name != "beta" {
		t.Fatalf("expected 1 Create for beta, got: %#v", plan.Creates)
	}
}

//nolint:cyclop,gocognit // テストの準備、実行、および複数のフラグ検証を一連のシーケンスで行うため、意図的に抑制します。
func TestPlannerDetectsConflictsAndLocalEdits(t *testing.T) {
	// 競合（未管理の同名ファイル存在）、ローカル編集（管理対象のハッシュ不一致）、および欠落がPlan時に検出されることを検証します。
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

	// 1. 未管理競合：配布先に未管理の "alpha" ディレクトリが存在する状態にする
	alphaDest := filepath.Join(workspace, ".codex", "skills", "alpha")
	// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
	if err := os.MkdirAll(alphaDest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(alphaDest, "SKILL.md"), []byte("unmanaged"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 2. ローカル編集：配布先に管理対象の "beta" ディレクトリが存在するが、ハッシュが異なるようにする
	betaDest := filepath.Join(workspace, ".codex", "skills", "beta")
	// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
	if err := os.MkdirAll(betaDest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(betaDest, "SKILL.md"), []byte("modified-locally"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 3. 欠落：管理対象の "gamma" ディレクトリは、ディスク上に存在しない状態（作成しない）にする

	// 前回の管理情報
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

	// 今回の選択
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

	// 競合とローカル編集のフラグ検証
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
}
