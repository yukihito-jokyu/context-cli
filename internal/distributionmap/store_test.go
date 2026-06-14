package distributionmap

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/yukihito-jokyu/context-cli/internal/distribution"
)

//nolint:gocognit,cyclop // 初回作成から再読込までの永続化契約を一つのシナリオで検証します。
func TestStoreLoadsAbsentAndCommitsInitialRecord(t *testing.T) {
	configHome, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store, err := New(&stubEnvironment{xdg: configHome})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	snapshot, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if snapshot.Revision != distribution.EmptyRevision || len(snapshot.Workspaces) != 0 {
		t.Fatalf("snapshot = %#v", snapshot)
	}

	transaction, current, err := store.Begin(snapshot.Revision)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	if current.Revision != snapshot.Revision {
		t.Fatalf("current revision = %q", current.Revision)
	}
	record := validRecord("/workspace")
	result, err := transaction.Commit(record)
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if !result.Committed {
		t.Fatal("Commit() did not report committed")
	}
	if err := transaction.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	mapPath := filepath.Join(configHome, "context", "map.yaml")
	info, err := os.Lstat(mapPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("map.yaml mode = %o, want 600", info.Mode().Perm())
	}
	dirInfo, err := os.Lstat(filepath.Dir(mapPath))
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("config directory mode = %o, want 700", dirInfo.Mode().Perm())
	}
	reloaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Revision == distribution.EmptyRevision || len(reloaded.Workspaces) != 1 {
		t.Fatalf("reloaded = %#v", reloaded)
	}
}

func TestStoreRejectsRevisionConflictAndSymlink(t *testing.T) {
	configHome, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store, err := New(&stubEnvironment{xdg: configHome})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = store.Begin("stale")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("Begin() error = %v, want ErrConflict", err)
	}

	configDir := filepath.Join(configHome, "context")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target", filepath.Join(configDir, "map.yaml")); err != nil {
		t.Fatal(err)
	}
	_, err = store.Load()
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("Load() error = %v, want ErrSymlink", err)
	}
}

func TestStorePreservesOtherWorkspaces(t *testing.T) {
	configHome, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store, err := New(&stubEnvironment{xdg: configHome})
	if err != nil {
		t.Fatal(err)
	}
	first := validRecord("/workspace-a")
	tx, _, err := store.Begin(distribution.EmptyRevision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Commit(first); err != nil {
		t.Fatal(err)
	}
	if err := tx.Close(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	tx, _, err = store.Begin(snapshot.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Commit(validRecord("/workspace-b")); err != nil {
		t.Fatal(err)
	}
	if err := tx.Close(); err != nil {
		t.Fatal(err)
	}
	snapshot, err = store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Workspaces) != 2 {
		t.Fatalf("len(Workspaces) = %d, want 2", len(snapshot.Workspaces))
	}
}

func validRecord(workspace string) distribution.WorkspaceRecord {
	return distribution.WorkspaceRecord{
		WorkspaceRoot: workspace,
		Project:       "project",
		Destinations:  []distribution.Destination{distribution.DestinationCodex},
		Skills: []distribution.SkillRecord{{
			Name: "skill", Source: distribution.SkillSourceProject,
			Destination: distribution.DestinationCodex, RelativePath: ".codex/skills/skill", Hash: hashA,
		}},
	}
}

type stubEnvironment struct {
	xdg  string
	home string
}

func (e *stubEnvironment) LookupEnv(key string) (string, bool) {
	if key == "XDG_CONFIG_HOME" && e.xdg != "" {
		return e.xdg, true
	}
	return "", false
}

func (e *stubEnvironment) UserHomeDir() (string, error) {
	return e.home, nil
}

func TestStoreRemovesWorkspaceWhenSkillsAreEmpty(t *testing.T) {
	// 登録されたSkillが0個の状態でCommitした場合に、そのWorkspaceレコードがmap.yamlから完全に削除されることを検証します。
	configHome, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store, err := New(&stubEnvironment{xdg: configHome})
	if err != nil {
		t.Fatal(err)
	}

	// 1. 最初はSkillが1つある状態でコミットする
	initialRecord := validRecord("/workspace-a")
	tx, _, err := store.Begin(distribution.EmptyRevision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Commit(initialRecord); err != nil {
		t.Fatal(err)
	}
	if err := tx.Close(); err != nil {
		t.Fatal(err)
	}

	// 2. ロードして確認
	snapshot, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(snapshot.Workspaces))
	}

	// 3. 今度はSkillを0個（空）にして同じWorkspaceをコミットする
	emptyRecord := distribution.WorkspaceRecord{
		WorkspaceRoot: "/workspace-a",
		Project:       "project",
		Destinations:  []distribution.Destination{distribution.DestinationCodex},
		Skills:        []distribution.SkillRecord{},
	}
	tx, _, err = store.Begin(snapshot.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Commit(emptyRecord); err != nil {
		t.Fatal(err)
	}
	if err := tx.Close(); err != nil {
		t.Fatal(err)
	}

	// 4. ロードして完全にキーが削除されていることを確認
	finalSnapshot, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := finalSnapshot.Workspaces["/workspace-a"]; exists {
		t.Fatalf("expected workspace-a to be deleted, but it still exists")
	}
	if len(finalSnapshot.Workspaces) != 0 {
		t.Fatalf("expected 0 workspaces, got %d", len(finalSnapshot.Workspaces))
	}
}

//nolint:gocognit,cyclop // 段階的なコミットと検証からなる回帰シナリオを一関数で完結させます。
func TestStoreRemovesOnlyTargetWorkspaceAndPreservesOthersWhenSkillsEmpty(t *testing.T) {
	// 同期で全Skillが消失して空Workspace記録をCommitした場合、
	// 対象Workspaceだけを削除し他Workspaceとスキーマを保持する回帰検証です。
	configHome, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store, err := New(&stubEnvironment{xdg: configHome})
	if err != nil {
		t.Fatal(err)
	}

	// workspace-a と workspace-b をそれぞれ1件ずつコミットして保持する。
	tx, _, err := store.Begin(distribution.EmptyRevision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Commit(validRecord("/workspace-a")); err != nil {
		t.Fatal(err)
	}
	if err := tx.Close(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	tx, _, err = store.Begin(snapshot.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Commit(validRecord("/workspace-b")); err != nil {
		t.Fatal(err)
	}
	if err := tx.Close(); err != nil {
		t.Fatal(err)
	}

	// workspace-a を空記録でCommitし、同期で全Skill消失した状態を再現する。
	snapshot, err = store.Load()
	if err != nil {
		t.Fatal(err)
	}
	tx, _, err = store.Begin(snapshot.Revision)
	if err != nil {
		t.Fatal(err)
	}
	emptyRecord := distribution.WorkspaceRecord{
		WorkspaceRoot: "/workspace-a",
		Project:       "project",
		Destinations:  []distribution.Destination{distribution.DestinationCodex},
		Skills:        []distribution.SkillRecord{},
	}
	if _, err := tx.Commit(emptyRecord); err != nil {
		t.Fatal(err)
	}
	if err := tx.Close(); err != nil {
		t.Fatal(err)
	}

	finalSnapshot, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := finalSnapshot.Workspaces["/workspace-a"]; exists {
		t.Fatal("expected workspace-a to be deleted, but it still exists")
	}
	if _, exists := finalSnapshot.Workspaces["/workspace-b"]; !exists {
		t.Fatal("expected workspace-b to be preserved, but it was removed")
	}
	if len(finalSnapshot.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(finalSnapshot.Workspaces))
	}
}
