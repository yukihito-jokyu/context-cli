//go:build darwin || linux

package distribution

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

//nolint:gocognit,cyclop // テーブル駆動テストのため、認知・循環複雑度の上限を無視します。
func TestSafeTree(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "RejectsEntryReplacedWithSymlinkBeforeOpen",
			run: func(t *testing.T) {
				t.Helper()
				base := canonicalTempDir(t)
				root := filepath.Join(base, "skill")
				outside := filepath.Join(base, "outside")
				if err := os.Mkdir(root, 0o755); err != nil { // #nosec G301 -- Skillディレクトリ要件を再現します。
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("inside\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(outside, []byte("outside\n"), 0o600); err != nil {
					t.Fatal(err)
				}

				tree := newSafeTree()
				tree.hooks.beforeOpen = func(relative string) {
					if relative != "SKILL.md" {
						return
					}
					if err := os.Remove(filepath.Join(root, relative)); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(outside, filepath.Join(root, relative)); err != nil {
						t.Fatal(err)
					}
				}

				_, err := tree.hash(root)
				if !errors.Is(err, ErrSymlink) && !errors.Is(err, ErrConflict) {
					t.Fatalf("hash() error = %v, want ErrSymlink or ErrConflict", err)
				}
				data, err := os.ReadFile(outside) // #nosec G304 -- テスト専用一時ディレクトリ内の固定パスです。
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != "outside\n" {
					t.Fatalf("outside content = %q", data)
				}
			},
		},
		{
			name: "DetectsEntryReplacementBeforeRead",
			run: func(t *testing.T) {
				t.Helper()
				base := canonicalTempDir(t)
				root := filepath.Join(base, "skill")
				path := filepath.Join(root, "SKILL.md")
				if err := os.Mkdir(root, 0o755); err != nil { // #nosec G301 -- Skillディレクトリ要件を再現します。
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("before\n"), 0o600); err != nil {
					t.Fatal(err)
				}

				tree := newSafeTree()
				tree.hooks.beforeRead = func(relative string) {
					if relative != "SKILL.md" {
						return
					}
					replacement := filepath.Join(root, "replacement")
					if err := os.WriteFile(replacement, []byte("after\n"), 0o600); err != nil {
						t.Fatal(err)
					}
					if err := os.Rename(replacement, path); err != nil {
						t.Fatal(err)
					}
				}

				_, err := tree.hash(root)
				if !errors.Is(err, ErrConflict) {
					t.Fatalf("hash() error = %v, want ErrConflict", err)
				}
			},
		},
		{
			name: "DetectsEntryReplacementAfterOpenBeforeStat",
			run: func(t *testing.T) {
				t.Helper()
				base := canonicalTempDir(t)
				root := filepath.Join(base, "skill")
				path := filepath.Join(root, "SKILL.md")
				outside := filepath.Join(base, "outside")
				if err := os.Mkdir(root, 0o755); err != nil { // #nosec G301 -- Skillディレクトリ要件を再現します。
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("inside\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(outside, []byte("outside\n"), 0o600); err != nil {
					t.Fatal(err)
				}

				tree := newSafeTree()
				tree.hooks.afterOpen = func(relative string) {
					if relative != "SKILL.md" {
						return
					}
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(outside, path); err != nil {
						t.Fatal(err)
					}
				}

				_, err := tree.hash(root)
				if !errors.Is(err, ErrSymlink) && !errors.Is(err, ErrConflict) {
					t.Fatalf("hash() error = %v, want ErrSymlink or ErrConflict", err)
				}
			},
		},
		{
			name: "ClassifiesRemovedEnumeratedEntryAsConflict",
			run: func(t *testing.T) {
				t.Helper()
				root := filepath.Join(canonicalTempDir(t), "skill")
				if err := os.Mkdir(root, 0o755); err != nil { // #nosec G301 -- Skillディレクトリ要件を再現します。
					t.Fatal(err)
				}
				path := filepath.Join(root, "SKILL.md")
				if err := os.WriteFile(path, []byte("skill"), 0o600); err != nil {
					t.Fatal(err)
				}
				tree := newSafeTree()
				tree.hooks.beforeOpen = func(relative string) {
					if relative == "SKILL.md" {
						if err := os.Remove(path); err != nil {
							t.Fatal(err)
						}
					}
				}

				_, err := tree.hash(root)
				if !errors.Is(err, ErrConflict) {
					t.Fatalf("hash() error = %v, want ErrConflict", err)
				}
			},
		},
		{
			name: "RejectsSourceAncestorReplacedWithSymlink",
			run: func(t *testing.T) {
				t.Helper()
				base := canonicalTempDir(t)
				ancestor := filepath.Join(base, "context")
				root := filepath.Join(ancestor, "skill")
				outside := filepath.Join(base, "outside")
				if err := os.MkdirAll(root, 0o755); err != nil { // #nosec G301 -- Skillディレクトリ要件を再現します。
					t.Fatal(err)
				}
				if err := os.Mkdir(outside, 0o755); err != nil { // #nosec G301 -- 差し替え先ディレクトリを作成します。
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("inside"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(outside, "SKILL.md"), []byte("outside"), 0o600); err != nil {
					t.Fatal(err)
				}
				tree := newSafeTree()
				tree.hooks.beforeRootOpen = func(string) {
					if err := os.Rename(ancestor, ancestor+"-moved"); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(outside, ancestor); err != nil {
						t.Fatal(err)
					}
				}

				_, err := tree.hash(root)
				if !errors.Is(err, ErrConflict) && !errors.Is(err, ErrSymlink) {
					t.Fatalf("hash() error = %v, want ErrConflict or ErrSymlink", err)
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
