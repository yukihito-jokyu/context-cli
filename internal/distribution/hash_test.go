package distribution

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestHashSkillIsDeterministicAndDetectsChanges(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skill")
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("skill\n"), 0o640); err != nil { // #nosec G306 -- 権限差分をハッシュへ含めるテストです。
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "run.sh"), []byte("#!/bin/sh\n"), 0o750); err != nil { // #nosec G306 -- 実行権限をハッシュへ含めるテストです。
		t.Fatal(err)
	}

	first, err := HashSkill(root)
	if err != nil {
		t.Fatalf("HashSkill() error = %v", err)
	}
	second, err := HashSkill(root)
	if err != nil {
		t.Fatalf("HashSkill() error = %v", err)
	}
	if first != second {
		t.Fatalf("hash is not deterministic: %q != %q", first, second)
	}

	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("changed\n"), 0o640); err != nil { // #nosec G306 -- 権限差分をハッシュへ含めるテストです。
		t.Fatal(err)
	}
	changedContent, err := HashSkill(root)
	if err != nil {
		t.Fatalf("HashSkill() error = %v", err)
	}
	if changedContent == first {
		t.Fatal("content change did not change hash")
	}

	if err := os.Chmod(filepath.Join(root, "SKILL.md"), 0o600); err != nil {
		t.Fatal(err)
	}
	changedMode, err := HashSkill(root)
	if err != nil {
		t.Fatalf("HashSkill() error = %v", err)
	}
	if changedMode == changedContent {
		t.Fatal("permission change did not change hash")
	}
}

func TestHashSkillIncludesEmptyDirectoriesAndEntryBoundaries(t *testing.T) {
	base := t.TempDir()
	left := filepath.Join(base, "left")
	right := filepath.Join(base, "right")
	for _, root := range []string{left, right} {
		if err := os.Mkdir(root, 0o755); err != nil { // #nosec G301 -- ディレクトリ権限をハッシュへ含めるテストです。
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(left, "empty"), 0o755); err != nil { // #nosec G301 -- ディレクトリ権限をハッシュへ含めるテストです。
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(left, "ab"), []byte("c"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(right, "a"), []byte("bc"), 0o600); err != nil {
		t.Fatal(err)
	}

	leftHash, err := HashSkill(left)
	if err != nil {
		t.Fatal(err)
	}
	rightHash, err := HashSkill(right)
	if err != nil {
		t.Fatal(err)
	}
	if leftHash == rightHash {
		t.Fatal("different directory structures produced the same hash")
	}
}

func TestHashSkillRejectsUnsafeEntries(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skill")
	if err := os.Mkdir(root, 0o755); err != nil { // #nosec G301 -- シンボリックリンク拒否用のSkillルートです。
		t.Fatal(err)
	}
	if err := os.Symlink("missing", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}

	_, err := HashSkill(root)
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("HashSkill() error = %v, want ErrSymlink", err)
	}
}
