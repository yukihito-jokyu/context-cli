package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	errHomeTest             = errors.New("home failed")
	errMkdirTest            = errors.New("mkdir failed")
	errDirectoryCleanupTest = errors.New("directory cleanup failed")
)

type testEnvironment struct {
	values  map[string]string
	present map[string]bool
	home    string
	homeErr error
}

func (e testEnvironment) LookupEnv(key string) (string, bool) {
	return e.values[key], e.present[key]
}

func (e testEnvironment) UserHomeDir() (string, error) {
	return e.home, e.homeErr
}

func TestOpenExploresConfigurationLocation(t *testing.T) {
	tests := []struct {
		name    string
		env     testEnvironment
		wantErr error
	}{
		{
			name: "絶対XDGパスを使用する",
			env: testEnvironment{
				values:  map[string]string{"XDG_CONFIG_HOME": realTempDir(t)},
				present: map[string]bool{"XDG_CONFIG_HOME": true},
			},
		},
		{
			name: "未設定時はホームへフォールバックする",
			env: testEnvironment{
				home: realTempDir(t),
			},
		},
		{
			name: "空文字はホームへフォールバックする",
			env: testEnvironment{
				values:  map[string]string{"XDG_CONFIG_HOME": ""},
				present: map[string]bool{"XDG_CONFIG_HOME": true},
				home:    realTempDir(t),
			},
		},
		{
			name: "相対XDGパスを拒否する",
			env: testEnvironment{
				values:  map[string]string{"XDG_CONFIG_HOME": "relative/config"},
				present: map[string]bool{"XDG_CONFIG_HOME": true},
			},
			wantErr: ErrDiscovery,
		},
		{
			name: "ホーム取得失敗を分類する",
			env: testEnvironment{
				homeErr: errHomeTest,
			},
			wantErr: ErrDiscovery,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := Open(tt.env, NewOSFileSystem())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Open() error = %v, want errors.Is(_, %v)", err, tt.wantErr)
			}
			if tt.wantErr == nil && store.GetContextRepository() != "" {
				t.Errorf("GetContextRepository() = %q, want empty", store.GetContextRepository())
			}
			if tt.env.homeErr != nil && !errors.Is(err, errHomeTest) {
				t.Errorf("Open() error = %v, want wrapped home error", err)
			}
		})
	}
}

func TestOpenDoesNotCreateMissingConfiguration(t *testing.T) {
	base := filepath.Join(realTempDir(t), "missing", "base")
	store, err := Open(xdgEnvironment(base), NewOSFileSystem())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if store.GetContextRepository() != "" {
		t.Fatalf("GetContextRepository() = %q, want empty", store.GetContextRepository())
	}
	if _, err := os.Lstat(base); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("読み込みだけで基底ディレクトリが作成されました: %v", err)
	}
}

func TestStorePersistsAndReloadsContextRepository(t *testing.T) {
	base := filepath.Join(realTempDir(t), "nested", "config")
	first, err := Open(xdgEnvironment(base), NewOSFileSystem())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := first.SetContextRepository("", "/tmp/first-context"); err != nil {
		t.Fatalf("SetContextRepository() error = %v", err)
	}

	configDir := filepath.Join(base, "context")
	configPath := filepath.Join(configDir, "config.yaml")
	assertPermission(t, configDir, 0o700)
	assertPermission(t, configPath, 0o600)

	second, err := Open(xdgEnvironment(base), NewOSFileSystem())
	if err != nil {
		t.Fatalf("Open() second error = %v", err)
	}
	if got := second.GetContextRepository(); got != "/tmp/first-context" {
		t.Fatalf("GetContextRepository() = %q, want /tmp/first-context", got)
	}
	if err := second.SetContextRepository("/tmp/first-context", "/tmp/second-context"); err != nil {
		t.Fatalf("SetContextRepository() update error = %v", err)
	}

	third, err := Open(xdgEnvironment(base), NewOSFileSystem())
	if err != nil {
		t.Fatalf("Open() third error = %v", err)
	}
	if got := third.GetContextRepository(); got != "/tmp/second-context" {
		t.Fatalf("GetContextRepository() = %q, want /tmp/second-context", got)
	}
}

func TestStoreLoadsLegacyConfigurationAndWritesCurrentSchema(t *testing.T) {
	base := realTempDir(t)
	configDir := filepath.Join(base, "context")
	configPath := filepath.Join(configDir, "config.yaml")
	mustMkdir(t, configDir, 0o700)
	mustWriteFile(
		t,
		configPath,
		[]byte("version: 1\nrepository_path: /tmp/legacy-context\n"),
		0o600,
	)

	store, err := Open(xdgEnvironment(base), NewOSFileSystem())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if got := store.GetContextRepository(); got != "/tmp/legacy-context" {
		t.Fatalf("GetContextRepository() = %q, want /tmp/legacy-context", got)
	}
	if err := store.SetContextRepository("/tmp/legacy-context", "/tmp/current-context"); err != nil {
		t.Fatalf("SetContextRepository() error = %v", err)
	}

	// #nosec G304 -- テスト専用の一時ディレクトリ配下だけを読み込みます。
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	want := "schema_version: 1\ncontext_repository: /tmp/current-context\n"
	if string(data) != want {
		t.Errorf("config.yaml = %q, want %q", data, want)
	}
}

func TestStoreDetectsCompareAndSetConflict(t *testing.T) {
	base := realTempDir(t)
	first, err := Open(xdgEnvironment(base), NewOSFileSystem())
	if err != nil {
		t.Fatalf("Open() first error = %v", err)
	}
	second, err := Open(xdgEnvironment(base), NewOSFileSystem())
	if err != nil {
		t.Fatalf("Open() second error = %v", err)
	}
	if err := first.SetContextRepository("", "/tmp/first"); err != nil {
		t.Fatalf("first.SetContextRepository() error = %v", err)
	}
	err = second.SetContextRepository("", "/tmp/second")
	if !errors.Is(err, ErrUpdateConflict) {
		t.Fatalf("second.SetContextRepository() error = %v, want ErrUpdateConflict", err)
	}

	reloaded, openErr := Open(xdgEnvironment(base), NewOSFileSystem())
	if openErr != nil {
		t.Fatalf("Open() reload error = %v", openErr)
	}
	if got := reloaded.GetContextRepository(); got != "/tmp/first" {
		t.Errorf("GetContextRepository() = %q, want /tmp/first", got)
	}
}

func TestOpenRejectsUnsafeConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(t *testing.T, base string)
		wantErr error
	}{
		{
			name: "親経路がシンボリックリンク",
			prepare: func(t *testing.T, base string) {
				t.Helper()
				target := realTempDir(t)
				if err := os.Symlink(target, filepath.Join(base, "linked")); err != nil {
					t.Fatalf("Symlink() error = %v", err)
				}
				mustMkdir(t, filepath.Join(target, "context"), 0o700)
			},
			wantErr: ErrSymlink,
		},
		{
			name: "設定ディレクトリの過剰権限",
			prepare: func(t *testing.T, base string) {
				t.Helper()
				mustMkdir(t, filepath.Join(base, "context"), 0o755)
			},
			wantErr: ErrPermission,
		},
		{
			name: "設定ファイルの過剰権限",
			prepare: func(t *testing.T, base string) {
				t.Helper()
				dir := filepath.Join(base, "context")
				mustMkdir(t, dir, 0o700)
				mustWriteFile(t, filepath.Join(dir, "config.yaml"), validConfig("/tmp/context"), 0o644)
			},
			wantErr: ErrPermission,
		},
		{
			name: "設定ファイルがディレクトリ",
			prepare: func(t *testing.T, base string) {
				t.Helper()
				dir := filepath.Join(base, "context")
				mustMkdir(t, dir, 0o700)
				mustMkdir(t, filepath.Join(dir, "config.yaml"), 0o700)
			},
			wantErr: ErrFileType,
		},
		{
			name: "設定ディレクトリがシンボリックリンク",
			prepare: func(t *testing.T, base string) {
				t.Helper()
				target := realTempDir(t)
				if err := os.Symlink(target, filepath.Join(base, "context")); err != nil {
					t.Fatalf("Symlink() error = %v", err)
				}
			},
			wantErr: ErrSymlink,
		},
		{
			name: "設定ファイルがシンボリックリンク",
			prepare: func(t *testing.T, base string) {
				t.Helper()
				dir := filepath.Join(base, "context")
				mustMkdir(t, dir, 0o700)
				target := filepath.Join(realTempDir(t), "config.yaml")
				mustWriteFile(t, target, validConfig("/tmp/context"), 0o600)
				if err := os.Symlink(target, filepath.Join(dir, "config.yaml")); err != nil {
					t.Fatalf("Symlink() error = %v", err)
				}
			},
			wantErr: ErrSymlink,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := realTempDir(t)
			tt.prepare(t, base)
			configHome := base
			if tt.name == "親経路がシンボリックリンク" {
				configHome = filepath.Join(base, "linked")
			}
			_, err := Open(xdgEnvironment(configHome), NewOSFileSystem())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Open() error = %v, want errors.Is(_, %v)", err, tt.wantErr)
			}
		})
	}
}

func TestStoreRemovesDirectoriesCreatedBeforeFailure(t *testing.T) {
	root := realTempDir(t)
	base := filepath.Join(root, "first", "second")
	fileSystem := &mkdirFailureFileSystem{
		FileSystem: NewOSFileSystem(),
		failPath:   filepath.Join(root, "first", "second"),
	}
	store, err := Open(xdgEnvironment(base), fileSystem)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	err = store.SetContextRepository("", "/tmp/context")
	if !errors.Is(err, ErrIO) {
		t.Fatalf("SetContextRepository() error = %v, want ErrIO", err)
	}
	if _, statErr := os.Lstat(filepath.Join(root, "first")); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("作成途中のディレクトリが残っています: %v", statErr)
	}
}

func TestStorePreservesDirectoryCreationCleanupFailure(t *testing.T) {
	root := realTempDir(t)
	base := filepath.Join(root, "first", "second")
	secretCleanupPath := filepath.Join(string(filepath.Separator), "secret", "created-directory")
	cleanupCause := &os.PathError{
		Op:   "remove",
		Path: secretCleanupPath,
		Err:  errDirectoryCleanupTest,
	}
	fileSystem := &directoryCreationFailureFileSystem{
		FileSystem: NewOSFileSystem(),
		failMkdir:  filepath.Join(root, "first", "second"),
		failRemove: filepath.Join(root, "first"),
		removeErr:  cleanupCause,
	}
	store, err := Open(xdgEnvironment(base), fileSystem)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	err = store.SetContextRepository("", "/tmp/context")
	if !errors.Is(err, ErrIO) || !errors.Is(err, errMkdirTest) {
		t.Fatalf("SetContextRepository() error = %v, want ErrIO and mkdir cause", err)
	}
	if !errors.Is(err, errDirectoryCleanupTest) {
		t.Fatalf("SetContextRepository() error = %v, want cleanup cause", err)
	}
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) || pathErr != cleanupCause {
		t.Fatalf("SetContextRepository() error = %v, want cleanup PathError", err)
	}
	var configErr *Error
	if !errors.As(err, &configErr) || configErr.Cleanup == nil {
		t.Fatalf("SetContextRepository() error = %v, want cleanup detail", err)
	}
	if strings.Contains(err.Error(), secretCleanupPath) || strings.Contains(err.Error(), cleanupCause.Error()) {
		t.Errorf("公開エラー文字列にcleanup内部情報が含まれています: %q", err)
	}
	if _, statErr := os.Lstat(filepath.Join(root, "first")); statErr != nil {
		t.Fatalf("cleanup失敗対象のディレクトリが残っていません: %v", statErr)
	}
	if _, statErr := os.Lstat(filepath.Join(base, "context", "config.yaml")); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("config.yaml was modified: %v", statErr)
	}
}

func TestConfigurationErrorsDoNotLeakPathsOrContent(t *testing.T) {
	base := realTempDir(t)
	configDir := filepath.Join(base, "context")
	configPath := filepath.Join(configDir, "config.yaml")
	mustMkdir(t, configDir, 0o700)
	secret := "secret-repository-value"
	mustWriteFile(t, configPath, []byte("schema_version: 1\ncontext_repository: "+secret+"\n"), 0o600)

	_, err := Open(xdgEnvironment(base), NewOSFileSystem())
	if !errors.Is(err, ErrSchema) {
		t.Fatalf("Open() error = %v, want ErrSchema", err)
	}
	for _, value := range []string{base, configPath, secret} {
		if strings.Contains(err.Error(), value) {
			t.Errorf("エラー文字列に内部情報 %q が含まれています: %q", value, err)
		}
	}
}

func xdgEnvironment(base string) testEnvironment {
	return testEnvironment{
		values:  map[string]string{"XDG_CONFIG_HOME": base},
		present: map[string]bool{"XDG_CONFIG_HOME": true},
	}
}

func realTempDir(t *testing.T) string {
	t.Helper()
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks() error = %v", err)
	}
	return path
}

func mustMkdir(t *testing.T, path string, perm fs.FileMode) {
	t.Helper()
	if err := os.Mkdir(path, perm); err != nil {
		t.Fatalf("Mkdir(%q) error = %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte, perm fs.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, data, perm); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func validConfig(path string) []byte {
	return []byte("schema_version: 1\ncontext_repository: " + path + "\n")
}

func assertPermission(t *testing.T, path string, want fs.FileMode) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%s permission = %#o, want %#o", path, got, want)
	}
}

type mkdirFailureFileSystem struct {
	FileSystem
	failPath string
}

type directoryCreationFailureFileSystem struct {
	FileSystem
	failMkdir  string
	failRemove string
	removeErr  error
}

func (f *directoryCreationFailureFileSystem) Mkdir(path string, perm fs.FileMode) error {
	if path == f.failMkdir {
		return errMkdirTest
	}
	if err := f.FileSystem.Mkdir(path, perm); err != nil {
		return fmt.Errorf("テスト用ディレクトリ作成: %w", err)
	}
	return nil
}

func (f *directoryCreationFailureFileSystem) Remove(path string) error {
	if path == f.failRemove {
		return f.removeErr
	}
	if err := f.FileSystem.Remove(path); err != nil {
		return fmt.Errorf("テスト用ディレクトリ削除: %w", err)
	}
	return nil
}

func (f *mkdirFailureFileSystem) Mkdir(path string, perm fs.FileMode) error {
	if path == f.failPath {
		return errMkdirTest
	}
	if err := f.FileSystem.Mkdir(path, perm); err != nil {
		return fmt.Errorf("テスト用ディレクトリ作成: %w", err)
	}
	return nil
}
