package distribution

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

//nolint:gocognit,cyclop // テーブル駆動テストのため、認知・循環複雑度の上限を無視します。
func TestOSFileSystem(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "StageCopiesContentsAndPermissions",
			run: func(t *testing.T) {
				t.Helper()
				base := canonicalTempDir(t)
				source := filepath.Join(base, "source")
				parent := filepath.Join(base, "target")
				if err := os.MkdirAll(filepath.Join(source, "scripts"), 0o750); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(parent, 0o755); err != nil { // #nosec G301 -- 配布先要件の0755を再現します。
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("skill\n"), 0o640); err != nil { // #nosec G306 -- 権限維持を検証します。
					t.Fatal(err)
				}

				parentState, err := NewOSFileSystem().Inspect(parent, PathKindDirectory, false)
				if err != nil {
					t.Fatal(err)
				}
				staging, err := NewOSFileSystem().Stage(source, parentState)
				if err != nil {
					t.Fatalf("Stage() error = %v", err)
				}
				t.Cleanup(func() { _ = os.RemoveAll(staging) })

				sourceHash, err := HashSkill(source)
				if err != nil {
					t.Fatal(err)
				}
				stagingHash, err := HashSkill(staging)
				if err != nil {
					t.Fatal(err)
				}
				if stagingHash != sourceHash {
					t.Fatalf("staging hash = %q, want %q", stagingHash, sourceHash)
				}
				info, err := os.Stat(filepath.Join(staging, "SKILL.md"))
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode().Perm() != 0o640 {
					t.Fatalf("staged permission = %o, want 640", info.Mode().Perm())
				}
			},
		},
		{
			name: "StageRejectsUnsupportedEntryAndCleansStaging",
			run: func(t *testing.T) {
				t.Helper()
				base := canonicalTempDir(t)
				source := filepath.Join(base, "source")
				parent := filepath.Join(base, "target")
				if err := os.Mkdir(source, 0o755); err != nil { // #nosec G301 -- Skillディレクトリ要件を再現します。
					t.Fatal(err)
				}
				if err := os.Mkdir(parent, 0o755); err != nil { // #nosec G301 -- 配布先要件の0755を再現します。
					t.Fatal(err)
				}
				if err := unix.Mkfifo(filepath.Join(source, "pipe"), 0o600); err != nil {
					t.Fatal(err)
				}

				parentState, err := NewOSFileSystem().Inspect(parent, PathKindDirectory, false)
				if err != nil {
					t.Fatal(err)
				}
				_, err = NewOSFileSystem().Stage(source, parentState)
				if !errors.Is(err, ErrFileType) {
					t.Fatalf("Stage() error = %v, want ErrFileType", err)
				}
				entries, err := os.ReadDir(parent)
				if err != nil {
					t.Fatal(err)
				}
				if len(entries) != 0 {
					t.Fatalf("staging remains after failure: %v", entries)
				}
			},
		},
		{
			name: "StageReportsCleanupFailureAndRemainingPath",
			run: func(t *testing.T) {
				t.Helper()
				base := canonicalTempDir(t)
				source := filepath.Join(base, "source")
				parent := filepath.Join(base, "target")
				if err := os.Mkdir(source, 0o755); err != nil { // #nosec G301 -- Skillディレクトリ要件を再現します。
					t.Fatal(err)
				}
				if err := os.Mkdir(parent, 0o755); err != nil { // #nosec G301 -- 配布先要件の0755を再現します。
					t.Fatal(err)
				}
				if err := unix.Mkfifo(filepath.Join(source, "pipe"), 0o600); err != nil {
					t.Fatal(err)
				}
				parentState, err := NewOSFileSystem().Inspect(parent, PathKindDirectory, false)
				if err != nil {
					t.Fatal(err)
				}
				fileSystem := osFileSystem{hooks: fileSystemHooks{
					beforeRemoveUnlink: func(path string) error {
						if strings.HasPrefix(filepath.Base(path), ".context-stage-") {
							return fs.ErrPermission
						}
						return nil
					},
				}}

				_, err = fileSystem.Stage(source, parentState)
				if !errors.Is(err, ErrFileType) || !errors.Is(err, ErrRollback) {
					t.Fatalf("Stage() error = %v, want ErrFileType and ErrRollback", err)
				}
				var distributionErr *Error
				if !errors.As(err, &distributionErr) {
					t.Fatalf("Stage() error type = %T, want *Error", err)
				}
				if distributionErr.Cleanup == nil || len(distributionErr.Unrestored) != 1 {
					t.Fatalf("Cleanup = %v, Unrestored = %v", distributionErr.Cleanup, distributionErr.Unrestored)
				}
				remaining := distributionErr.Unrestored[0]
				if !strings.HasPrefix(filepath.Base(remaining), ".context-stage-") {
					t.Fatalf("Unrestored = %v, want staging path", distributionErr.Unrestored)
				}
				if _, statErr := os.Stat(remaining); statErr != nil {
					t.Fatalf("remaining staging path = %q: %v", remaining, statErr)
				}
			},
		},
		{
			name: "InspectRecordsNonSymlinkLeafType",
			run: func(t *testing.T) {
				t.Helper()
				path := filepath.Join(canonicalTempDir(t), "pipe")
				if err := unix.Mkfifo(path, 0o600); err != nil {
					t.Fatal(err)
				}

				state, err := NewOSFileSystem().Inspect(path, PathKindAny, false)
				if err != nil {
					t.Fatalf("Inspect() error = %v", err)
				}
				if state.Kind != PathKindFIFO {
					t.Fatalf("Kind = %v, want PathKindFIFO", state.Kind)
				}
			},
		},
		{
			name: "StageRejectsReplacedParentWithoutWritingOutside",
			run: func(t *testing.T) {
				t.Helper()
				base := canonicalTempDir(t)
				source := filepath.Join(base, "source")
				parent := filepath.Join(base, "parent")
				outside := filepath.Join(base, "outside")
				if err := os.Mkdir(source, 0o755); err != nil { // #nosec G301 -- Skillディレクトリ要件を再現します。
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("skill"), 0o600); err != nil {
					t.Fatal(err)
				}
				for _, path := range []string{parent, outside} {
					if err := os.Mkdir(path, 0o755); err != nil { // #nosec G301 -- 配布先ディレクトリ要件を再現します。
						t.Fatal(err)
					}
				}
				parentState, err := NewOSFileSystem().Inspect(parent, PathKindDirectory, false)
				if err != nil {
					t.Fatal(err)
				}
				moved := parent + "-moved"
				fileSystem := osFileSystem{hooks: fileSystemHooks{
					beforeStage: func(string) {
						if err := os.Rename(parent, moved); err != nil {
							t.Fatal(err)
						}
						if err := os.Symlink(outside, parent); err != nil {
							t.Fatal(err)
						}
					},
				}}

				_, err = fileSystem.Stage(source, parentState)
				if !errors.Is(err, ErrConflict) && !errors.Is(err, ErrSymlink) {
					t.Fatalf("Stage() error = %v, want ErrConflict or ErrSymlink", err)
				}
				entries, err := os.ReadDir(outside)
				if err != nil {
					t.Fatal(err)
				}
				if len(entries) != 0 {
					t.Fatalf("outside entries = %v, want empty", entries)
				}
			},
		},
		{
			name: "MkdirRejectsReplacedParentWithoutWritingOutside",
			run: func(t *testing.T) {
				t.Helper()
				base := canonicalTempDir(t)
				parent := filepath.Join(base, "parent")
				outside := filepath.Join(base, "outside")
				for _, path := range []string{parent, outside} {
					if err := os.Mkdir(path, 0o755); err != nil { // #nosec G301 -- 配布先ディレクトリ要件を再現します。
						t.Fatal(err)
					}
				}
				parentState, err := NewOSFileSystem().Inspect(parent, PathKindDirectory, false)
				if err != nil {
					t.Fatal(err)
				}
				fileSystem := osFileSystem{hooks: fileSystemHooks{
					beforeMkdir: func(string) {
						if err := os.Rename(parent, parent+"-moved"); err != nil {
							t.Fatal(err)
						}
						if err := os.Symlink(outside, parent); err != nil {
							t.Fatal(err)
						}
					},
				}}

				err = fileSystem.Mkdir(filepath.Join(parent, "child"), parentState, 0o755)
				if !errors.Is(err, ErrConflict) && !errors.Is(err, ErrSymlink) {
					t.Fatalf("Mkdir() error = %v, want ErrConflict or ErrSymlink", err)
				}
				if _, err := os.Lstat(filepath.Join(outside, "child")); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("outside child exists: %v", err)
				}
			},
		},
		{
			name: "BackupRejectsReplacedTarget",
			run: func(t *testing.T) {
				t.Helper()
				base := canonicalTempDir(t)
				parent := filepath.Join(base, "parent")
				target := filepath.Join(parent, "target")
				if err := os.Mkdir(parent, 0o755); err != nil { // #nosec G301 -- 配布先ディレクトリ要件を再現します。
					t.Fatal(err)
				}
				if err := os.WriteFile(target, []byte("original"), 0o600); err != nil {
					t.Fatal(err)
				}
				parentState, err := NewOSFileSystem().Inspect(parent, PathKindDirectory, false)
				if err != nil {
					t.Fatal(err)
				}
				targetState, err := NewOSFileSystem().Inspect(target, PathKindRegularFile, false)
				if err != nil {
					t.Fatal(err)
				}
				fileSystem := osFileSystem{hooks: fileSystemHooks{
					beforeBackup: func(string) {
						if err := os.Remove(target); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(target, []byte("replacement"), 0o600); err != nil {
							t.Fatal(err)
						}
					},
				}}

				_, err = fileSystem.Backup(target, parentState, targetState)
				if !errors.Is(err, ErrConflict) {
					t.Fatalf("Backup() error = %v, want ErrConflict", err)
				}
				data, err := os.ReadFile(target) // #nosec G304 -- テスト専用一時ディレクトリ内の対象です。
				if err != nil || string(data) != "replacement" {
					t.Fatalf("target = %q, %v", data, err)
				}
			},
		},
		{
			name: "RemoveAllRejectsReplacedTarget",
			run: func(t *testing.T) {
				t.Helper()
				base := canonicalTempDir(t)
				parent := filepath.Join(base, "parent")
				target := filepath.Join(parent, "target")
				if err := os.MkdirAll(target, 0o755); err != nil { // #nosec G301 -- 配布先ディレクトリ要件を再現します。
					t.Fatal(err)
				}
				parentState, err := NewOSFileSystem().Inspect(parent, PathKindDirectory, false)
				if err != nil {
					t.Fatal(err)
				}
				targetState, err := NewOSFileSystem().Inspect(target, PathKindDirectory, false)
				if err != nil {
					t.Fatal(err)
				}
				fileSystem := osFileSystem{hooks: fileSystemHooks{
					beforeRemove: func(string) {
						if err := os.Rename(target, target+"-original"); err != nil {
							t.Fatal(err)
						}
						if err := os.Mkdir(target, 0o755); err != nil { // #nosec G301 -- 差し替え対象を作成します。
							t.Fatal(err)
						}
					},
				}}

				err = fileSystem.RemoveAll(target, parentState, targetState)
				if !errors.Is(err, ErrConflict) {
					t.Fatalf("RemoveAll() error = %v, want ErrConflict", err)
				}
				if _, err := os.Stat(target); err != nil {
					t.Fatalf("replacement target was removed: %v", err)
				}
			},
		},
		{
			name: "RemoveAllRejectsRecursiveDirectoryReplacement",
			run: func(t *testing.T) {
				t.Helper()
				base := canonicalTempDir(t)
				parent := filepath.Join(base, "parent")
				target := filepath.Join(parent, "target")
				child := filepath.Join(target, "child")
				replacementMarker := filepath.Join(child, "replacement")
				if err := os.MkdirAll(child, 0o755); err != nil { // #nosec G301 -- 再帰削除対象を作成します。
					t.Fatal(err)
				}
				parentState, err := NewOSFileSystem().Inspect(parent, PathKindDirectory, false)
				if err != nil {
					t.Fatal(err)
				}
				targetState, err := NewOSFileSystem().Inspect(target, PathKindDirectory, false)
				if err != nil {
					t.Fatal(err)
				}
				replaced := false
				fileSystem := osFileSystem{hooks: fileSystemHooks{
					beforeRemoveOpen: func(path string) {
						if path != child || replaced {
							return
						}
						replaced = true
						if err := os.Rename(child, child+"-original"); err != nil {
							t.Fatal(err)
						}
						if err := os.Mkdir(child, 0o755); err != nil { // #nosec G301 -- 差し替え対象を作成します。
							t.Fatal(err)
						}
						if err := os.WriteFile(replacementMarker, []byte("replacement"), 0o600); err != nil {
							t.Fatal(err)
						}
					},
				}}

				err = fileSystem.RemoveAll(target, parentState, targetState)
				if !errors.Is(err, ErrConflict) {
					t.Fatalf("RemoveAll() error = %v, want ErrConflict", err)
				}
				if _, err := os.Stat(replacementMarker); err != nil {
					t.Fatalf("replacement directory was modified: %v", err)
				}
			},
		},
		{
			name: "RemoveAllRejectsRegularFileReplacementBeforeUnlink",
			run: func(t *testing.T) {
				t.Helper()
				base := canonicalTempDir(t)
				parent := filepath.Join(base, "parent")
				target := filepath.Join(parent, "target")
				file := filepath.Join(target, "file")
				if err := os.MkdirAll(target, 0o755); err != nil { // #nosec G301 -- 再帰削除対象を作成します。
					t.Fatal(err)
				}
				if err := os.WriteFile(file, []byte("original"), 0o600); err != nil {
					t.Fatal(err)
				}
				parentState, err := NewOSFileSystem().Inspect(parent, PathKindDirectory, false)
				if err != nil {
					t.Fatal(err)
				}
				targetState, err := NewOSFileSystem().Inspect(target, PathKindDirectory, false)
				if err != nil {
					t.Fatal(err)
				}
				replaced := false
				fileSystem := osFileSystem{hooks: fileSystemHooks{
					beforeRemoveUnlink: func(path string) error {
						if path != file || replaced {
							return nil
						}
						replaced = true
						if err := os.Rename(file, file+"-original"); err != nil {
							t.Fatal(err)
						}
						return os.WriteFile(file, []byte("replacement"), 0o600)
					},
				}}

				err = fileSystem.RemoveAll(target, parentState, targetState)
				if !errors.Is(err, ErrConflict) {
					t.Fatalf("RemoveAll() error = %v, want ErrConflict", err)
				}
				data, err := os.ReadFile(file) // #nosec G304 -- テスト専用一時ディレクトリ内の対象です。
				if err != nil || string(data) != "replacement" {
					t.Fatalf("replacement file = %q, %v", data, err)
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

func canonicalTempDir(t *testing.T) string {
	t.Helper()
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return path
}
