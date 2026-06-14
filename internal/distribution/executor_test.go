package distribution

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

var errCommitTest = errors.New("commit failed")

func TestExecutorCreatesSkillsAndCommitsMap(t *testing.T) {
	plan, workspace := createExecutorPlan(t)
	store := &fakeMapStore{
		snapshot: MapSnapshot{Revision: plan.ExpectedRevision, Workspaces: map[string]WorkspaceRecord{}},
		tx:       &fakeMapTransaction{},
	}

	result, err := NewExecutor(NewOSFileSystem(), store).Execute(plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Created != len(plan.Creates) {
		t.Fatalf("Created = %d, want %d", result.Created, len(plan.Creates))
	}
	for _, operation := range plan.Creates {
		data, err := os.ReadFile(filepath.Join(operation.FinalPath, "SKILL.md"))
		if err != nil {
			t.Fatalf("read distributed skill: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("distributed skill is empty")
		}
	}
	if store.expected != plan.ExpectedRevision {
		t.Fatalf("Begin() expected = %q", store.expected)
	}
	if store.tx.committed.WorkspaceRoot != workspace {
		t.Fatalf("committed workspace = %q, want %q", store.tx.committed.WorkspaceRoot, workspace)
	}
	if store.tx.closeCalls != 1 {
		t.Fatalf("Close() calls = %d, want 1", store.tx.closeCalls)
	}
}

func TestExecutorRollsBackWhenCommitFailsBeforeCommitPoint(t *testing.T) {
	plan, _ := createExecutorPlan(t)
	store := &fakeMapStore{
		snapshot: MapSnapshot{Revision: plan.ExpectedRevision, Workspaces: map[string]WorkspaceRecord{}},
		tx: &fakeMapTransaction{
			commitErr: errCommitTest,
		},
	}

	_, err := NewExecutor(NewOSFileSystem(), store).Execute(plan)
	if !errors.Is(err, ErrIO) {
		t.Fatalf("Execute() error = %v, want ErrIO", err)
	}
	for _, operation := range plan.Creates {
		if _, statErr := os.Lstat(operation.FinalPath); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("final path remains after rollback: %s (%v)", operation.FinalPath, statErr)
		}
	}
}

func TestExecutorDoesNotRollbackAfterCommitPoint(t *testing.T) {
	plan, _ := createExecutorPlan(t)
	store := &fakeMapStore{
		snapshot: MapSnapshot{Revision: plan.ExpectedRevision, Workspaces: map[string]WorkspaceRecord{}},
		tx: &fakeMapTransaction{
			commitResult: CommitResult{Committed: true},
			commitErr:    ErrCommitted,
		},
	}

	_, err := NewExecutor(NewOSFileSystem(), store).Execute(plan)
	if !errors.Is(err, ErrCommitted) {
		t.Fatalf("Execute() error = %v, want ErrCommitted", err)
	}
	for _, operation := range plan.Creates {
		if _, statErr := os.Lstat(operation.FinalPath); statErr != nil {
			t.Fatalf("committed final path was rolled back: %v", statErr)
		}
	}
}

func createExecutorPlan(t *testing.T) (Plan, string) {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(base, "workspace")
	source := filepath.Join(base, "context", "projects", "project", "skills", "alpha")
	if err := os.MkdirAll(workspace, 0o755); err != nil { // #nosec G301 -- 配布先ディレクトリ要件を再現します。
		t.Fatal(err)
	}
	if err := os.MkdirAll(source, 0o755); err != nil { // #nosec G301 -- Skillディレクトリの権限維持を検証します。
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("skill\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := NewPlanner(NewOSFileSystem()).Plan(
		MapSnapshot{Revision: EmptyRevision, Workspaces: map[string]WorkspaceRecord{}},
		Selection{
			WorkspaceRoot: workspace,
			Project:       "project",
			Skills:        []SelectedSkill{{Name: "alpha", Source: SkillSourceProject, SourcePath: source}},
			Destinations:  []Destination{DestinationCodex, DestinationClaude},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return plan, workspace
}

type fakeMapStore struct {
	expected Revision
	snapshot MapSnapshot
	tx       *fakeMapTransaction
	beginErr error
}

func (s *fakeMapStore) Load() (MapSnapshot, error) {
	return s.snapshot, nil
}

func (s *fakeMapStore) Begin(expected Revision) (MapTransaction, MapSnapshot, error) {
	s.expected = expected
	if s.beginErr != nil {
		return nil, MapSnapshot{}, s.beginErr
	}
	return s.tx, s.snapshot, nil
}

type fakeMapTransaction struct {
	committed    WorkspaceRecord
	commitResult CommitResult
	commitErr    error
	closeErr     error
	closeCalls   int
}

func (t *fakeMapTransaction) Commit(workspace WorkspaceRecord) (CommitResult, error) {
	t.committed = workspace
	if t.commitResult == (CommitResult{}) && t.commitErr == nil {
		return CommitResult{Committed: true}, nil
	}
	return t.commitResult, t.commitErr
}

func (t *fakeMapTransaction) Close() error {
	t.closeCalls++
	return t.closeErr
}

type deleteTestSetup struct {
	plan      Plan
	workspace string
	betaPath  string
	alphaPath string
}

func TestExecutorDeletesAndRollsBack(t *testing.T) {
	// 既存ファイルの削除と、コミット失敗時のロールバック（既存ファイル復元、新規ファイル削除）を検証します。
	setup := createExecutorPlanWithDeletes(t)
	store := &fakeMapStore{
		snapshot: MapSnapshot{Revision: setup.plan.ExpectedRevision, Workspaces: map[string]WorkspaceRecord{}},
		tx: &fakeMapTransaction{
			commitErr: errCommitTest,
		},
	}

	_, err := NewExecutor(NewOSFileSystem(), store).Execute(setup.plan)
	if !errors.Is(err, ErrIO) {
		t.Fatalf("Execute() error = %v, want ErrIO", err)
	}

	// ロールバックにより削除予定だったbetaPathが復元されていることを検証します。
	if _, err := os.Lstat(filepath.Join(setup.betaPath, "SKILL.md")); err != nil {
		t.Fatalf("deleted skill should be restored but got error: %v", err)
	}

	// ロールバックにより新規配置予定だったパスが削除されていることを検証します。
	for _, operation := range setup.plan.Creates {
		if _, err := os.Lstat(operation.FinalPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("newly created path should be removed but it still exists: %s", operation.FinalPath)
		}
	}
}

func TestExecutorDeletesAndCommits(t *testing.T) {
	// 既存ファイルの削除と、コミット成功時の完全削除・バックアップクリーンアップを検証します。
	setup := createExecutorPlanWithDeletes(t)
	store := &fakeMapStore{
		snapshot: MapSnapshot{Revision: setup.plan.ExpectedRevision, Workspaces: map[string]WorkspaceRecord{}},
		tx:       &fakeMapTransaction{},
	}

	result, err := NewExecutor(NewOSFileSystem(), store).Execute(setup.plan)
	if err != nil {
		t.Fatalf("Execute() error = %v, want ErrIO", err)
	}
	if result.Created != len(setup.plan.Creates) {
		t.Fatalf("Created = %d, want %d", result.Created, len(setup.plan.Creates))
	}

	// コミット成功により、削除予定だったbetaPathが完全に削除されていることを検証します。
	if _, err := os.Lstat(setup.betaPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("deleted skill path should not exist: %v", err)
	}

	// 新規配置予定だったファイルが配置されていることを検証します。
	for _, operation := range setup.plan.Creates {
		if _, err := os.Lstat(filepath.Join(operation.FinalPath, "SKILL.md")); err != nil {
			t.Fatalf("newly created skill does not exist: %v", err)
		}
	}

	// 一時バックアップフォルダ（.context-backup-*）が削除されていることを検証します。
	files, err := os.ReadDir(filepath.Dir(setup.betaPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if file.IsDir() && len(file.Name()) > 15 && file.Name()[:16] == ".context-backup-" {
			t.Fatalf("backup directory remains: %s", file.Name())
		}
	}
}

func createExecutorPlanWithDeletes(t *testing.T) deleteTestSetup {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(base, "workspace")

	// 供給元ディレクトリの作成
	alphaSource := filepath.Join(base, "context", "projects", "project", "skills", "alpha")
	betaSource := filepath.Join(base, "context", "projects", "project", "skills", "beta")

	// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
	if err := os.MkdirAll(alphaSource, 0o755); err != nil {
		t.Fatal(err)
	}
	// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
	if err := os.MkdirAll(betaSource, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(alphaSource, "SKILL.md"), []byte("alpha\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(betaSource, "SKILL.md"), []byte("beta\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 配布先ディレクトリと既存のSkillを配置
	alphaDest := filepath.Join(workspace, ".codex", "skills", "alpha")
	betaDest := filepath.Join(workspace, ".codex", "skills", "beta")
	// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
	if err := os.MkdirAll(filepath.Dir(betaDest), 0o755); err != nil {
		t.Fatal(err)
	}
	// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
	if err := os.MkdirAll(betaDest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(betaDest, "SKILL.md"), []byte("beta\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 前回の管理情報（Snapshot）の作成（betaが配置済み）
	snapshot := MapSnapshot{
		Revision: "rev-1234",
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
						Hash:         "some-hash",
					},
				},
			},
		},
	}

	// 今回の選択（alphaのみを選択、betaは解除）
	selection := Selection{
		WorkspaceRoot: workspace,
		Project:       "project",
		Skills: []SelectedSkill{
			{
				Name:       "alpha",
				Source:     SkillSourceProject,
				SourcePath: alphaSource,
			},
		},
		Destinations: []Destination{DestinationCodex},
	}

	plan, err := NewPlanner(NewOSFileSystem()).Plan(snapshot, selection)
	if err != nil {
		t.Fatal(err)
	}

	return deleteTestSetup{
		plan:      plan,
		workspace: workspace,
		betaPath:  betaDest,
		alphaPath: alphaDest,
	}
}
