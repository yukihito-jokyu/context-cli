package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	errLockCloseTest      = errors.New("lock close failed")
	errCreateTempTest     = errors.New("create temporary file failed")
	errTempChmodTest      = errors.New("temporary file chmod failed")
	errWriteTest          = errors.New("write failed")
	errTempSyncTest       = errors.New("temp sync failed")
	errTempCloseTest      = errors.New("temp close failed")
	errRenameTest         = errors.New("rename failed")
	errOpenDirectoryTest  = errors.New("open directory failed")
	errDirectorySyncTest  = errors.New("directory sync failed")
	errDirectoryCloseTest = errors.New("directory close failed")
	errUnlockTest         = errors.New("unlock failed")
	errLockCreateTest     = errors.New("lock create failed")
	errLockOpenTest       = errors.New("lock open failed")
	errLockChmodTest      = errors.New("lock chmod failed")
	errFlockTest          = errors.New("flock failed")
	errReloadTest         = errors.New("reload failed")
	errRemoveTempTest     = errors.New("remove temporary file failed")
)

type leakageFailureCase struct {
	name          string
	failOperation string
	initial       string
	cause         *os.PathError
	wantErr       error
}

func TestStoreRejectsUnsafeLockFile(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(t *testing.T, path string)
		wantErr error
	}{
		{
			name: "過剰権限",
			prepare: func(t *testing.T, path string) {
				t.Helper()
				mustWriteFile(t, path, nil, 0o644)
			},
			wantErr: ErrPermission,
		},
		{
			name: "シンボリックリンク",
			prepare: func(t *testing.T, path string) {
				t.Helper()
				target := filepath.Join(realTempDir(t), "lock")
				mustWriteFile(t, target, nil, 0o600)
				if err := os.Symlink(target, path); err != nil {
					t.Fatalf("Symlink() error = %v", err)
				}
			},
			wantErr: ErrSymlink,
		},
		{
			name: "ディレクトリ",
			prepare: func(t *testing.T, path string) {
				t.Helper()
				mustMkdir(t, path, 0o700)
			},
			wantErr: ErrFileType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := realTempDir(t)
			configDir := filepath.Join(base, "context")
			mustMkdir(t, configDir, 0o700)
			tt.prepare(t, filepath.Join(configDir, "config.lock"))

			store, err := Open(xdgEnvironment(base), NewOSFileSystem())
			if err != nil {
				t.Fatalf("Open() error = %v", err)
			}
			err = store.SetContextRepository("", "/tmp/context")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("SetContextRepository() error = %v, want errors.Is(_, %v)", err, tt.wantErr)
			}
			if _, statErr := os.Lstat(filepath.Join(configDir, "config.yaml")); !errors.Is(statErr, fs.ErrNotExist) {
				t.Errorf("config.yaml was modified: %v", statErr)
			}
		})
	}
}

func TestStoreUsesNonBlockingLock(t *testing.T) {
	base := realTempDir(t)
	store, err := Open(xdgEnvironment(base), NewOSFileSystem())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.ensureConfigDirectory(); err != nil {
		t.Fatalf("ensureConfigDirectory() error = %v", err)
	}

	lockPath := filepath.Join(base, "context", "config.lock")
	// #nosec G304 -- テスト専用の一時ディレクトリ配下だけを開きます。
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	t.Cleanup(func() { _ = lock.Close() })
	file := &osFile{file: lock}
	if err := NewOSFileSystem().Flock(file); err != nil {
		t.Fatalf("Flock() error = %v", err)
	}
	t.Cleanup(func() { _ = NewOSFileSystem().Funlock(file) })

	err = store.SetContextRepository("", "/tmp/context")
	if !errors.Is(err, ErrLockConflict) {
		t.Fatalf("SetContextRepository() error = %v, want ErrLockConflict", err)
	}
}

func TestStoreClassifiesFailuresAroundCommit(t *testing.T) {
	tests := []struct {
		name          string
		failOperation string
		wantErr       error
		wantCommitted bool
	}{
		{name: "一時ファイル作成失敗", failOperation: "create-temp", wantErr: ErrIO},
		{name: "一時ファイル権限設定失敗", failOperation: "temp-chmod", wantErr: ErrIO},
		{name: "一時ファイル短い書き込み", failOperation: "short-write", wantErr: ErrIO},
		{name: "一時ファイル書き込み失敗", failOperation: "write", wantErr: ErrIO},
		{name: "一時ファイル同期失敗", failOperation: "temp-sync", wantErr: ErrIO},
		{name: "一時ファイルクローズ失敗", failOperation: "temp-close", wantErr: ErrCleanup},
		{name: "置換失敗", failOperation: "rename", wantErr: ErrIO},
		{name: "ディレクトリオープン失敗", failOperation: "dir-open", wantErr: ErrCommitted, wantCommitted: true},
		{name: "ディレクトリ同期失敗", failOperation: "dir-sync", wantErr: ErrCommitted, wantCommitted: true},
		{name: "ディレクトリクローズ失敗", failOperation: "dir-close", wantErr: ErrCommitted, wantCommitted: true},
		{name: "ロック解放失敗", failOperation: "unlock", wantErr: ErrCommitted, wantCommitted: true},
		{name: "ロックファイルクローズ失敗", failOperation: "lock-close", wantErr: ErrCommitted, wantCommitted: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCommitFailureCase(t, tt.failOperation, tt.wantErr, tt.wantCommitted)
		})
	}
}

func TestStoreClassifiesLockAndReloadFailures(t *testing.T) {
	tests := []struct {
		name          string
		failOperation string
		initial       string
		wantCause     error
	}{
		{name: "ロック作成失敗", failOperation: "lock-create", wantCause: errLockCreateTest},
		{name: "既存ロックオープン失敗", failOperation: "lock-open", initial: "/tmp/initial", wantCause: errLockOpenTest},
		{name: "ロック権限設定失敗", failOperation: "lock-chmod", wantCause: errLockChmodTest},
		{name: "Flock失敗", failOperation: "flock", wantCause: errFlockTest},
		{name: "ロック中再読込失敗", failOperation: "reload", initial: "/tmp/initial", wantCause: errReloadTest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := realTempDir(t)
			realFS := NewOSFileSystem()
			if tt.initial != "" {
				initial, err := Open(xdgEnvironment(base), realFS)
				if err != nil {
					t.Fatalf("Open() initial error = %v", err)
				}
				if err := initial.SetContextRepository("", tt.initial); err != nil {
					t.Fatalf("initial.SetContextRepository() error = %v", err)
				}
			}

			failingFS := &failureFileSystem{FileSystem: realFS, operation: tt.failOperation}
			store, err := Open(xdgEnvironment(base), failingFS)
			if err != nil {
				t.Fatalf("Open() error = %v", err)
			}
			err = store.SetContextRepository(tt.initial, "/tmp/updated")
			assertConfigError(t, err, ErrIO, tt.wantCause)
			assertStoredRepository(t, base, tt.initial)
		})
	}
}

func TestStorePreservesTemporaryRemovalFailure(t *testing.T) {
	base := realTempDir(t)
	realFS := NewOSFileSystem()
	initial, err := Open(xdgEnvironment(base), realFS)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := initial.SetContextRepository("", "/tmp/initial"); err != nil {
		t.Fatalf("initial.SetContextRepository() error = %v", err)
	}

	failingFS := &failureFileSystem{FileSystem: realFS, operation: "write+remove-temp"}
	store, err := Open(xdgEnvironment(base), failingFS)
	if err != nil {
		t.Fatalf("Open() with failureFileSystem error = %v", err)
	}
	err = store.SetContextRepository("/tmp/initial", "/tmp/updated")
	assertConfigError(t, err, ErrIO, errWriteTest)
	if !errors.Is(err, errRemoveTempTest) {
		t.Fatalf("SetContextRepository() error = %v, want cleanup cause", err)
	}
	var configErr *Error
	if !errors.As(err, &configErr) || configErr.Cleanup == nil {
		t.Fatalf("SetContextRepository() error = %v, want cleanup detail", err)
	}
	assertStoredRepository(t, base, "/tmp/initial")
}

func TestStorePreservesPrimaryAndCleanupFailures(t *testing.T) {
	base := realTempDir(t)
	realFS := NewOSFileSystem()
	initial, err := Open(xdgEnvironment(base), realFS)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := initial.SetContextRepository("", "/tmp/initial"); err != nil {
		t.Fatalf("initial SetContextRepository() error = %v", err)
	}

	failingFS := &failureFileSystem{FileSystem: realFS, operation: "write+unlock"}
	store, err := Open(xdgEnvironment(base), failingFS)
	if err != nil {
		t.Fatalf("Open() with failureFileSystem error = %v", err)
	}
	err = store.SetContextRepository("/tmp/initial", "/tmp/updated")
	if !errors.Is(err, ErrIO) {
		t.Fatalf("SetContextRepository() error = %v, want ErrIO", err)
	}
	if !errors.Is(err, errUnlockTest) {
		t.Fatalf("SetContextRepository() error = %v, want wrapped unlock error", err)
	}
}

func TestStoreErrorsHideInternalPathsAndPreserveCauses(t *testing.T) {
	secretPath := filepath.Join(string(filepath.Separator), "secret", "configuration", "path")
	tempPath := filepath.Join(string(filepath.Separator), "secret", ".config-private.tmp")
	tests := []leakageFailureCase{
		{
			name:          "ロック作成失敗",
			failOperation: "lock-create",
			cause:         &os.PathError{Op: "open", Path: secretPath, Err: fs.ErrPermission},
			wantErr:       ErrIO,
		},
		{
			name:          "既存ロックオープン失敗",
			failOperation: "lock-open",
			initial:       "/tmp/initial",
			cause:         &os.PathError{Op: "open", Path: secretPath, Err: fs.ErrPermission},
			wantErr:       ErrIO,
		},
		{
			name:          "ロック権限設定失敗",
			failOperation: "lock-chmod",
			cause:         &os.PathError{Op: "chmod", Path: secretPath, Err: fs.ErrPermission},
			wantErr:       ErrIO,
		},
		{
			name:          "Flock失敗",
			failOperation: "flock",
			cause:         &os.PathError{Op: "flock", Path: secretPath, Err: fs.ErrPermission},
			wantErr:       ErrIO,
		},
		{
			name:          "置換失敗",
			failOperation: "rename",
			initial:       "/tmp/initial",
			cause:         &os.PathError{Op: "rename", Path: tempPath, Err: fs.ErrPermission},
			wantErr:       ErrIO,
		},
		{
			name:          "一時ファイル削除失敗",
			failOperation: "write+remove-temp",
			initial:       "/tmp/initial",
			cause:         &os.PathError{Op: "remove", Path: tempPath, Err: fs.ErrPermission},
			wantErr:       ErrIO,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runLeakageFailureCase(t, tt, secretPath, tempPath)
		})
	}
}

func runLeakageFailureCase(t *testing.T, tt leakageFailureCase, secrets ...string) {
	t.Helper()
	base := realTempDir(t)
	realFS := NewOSFileSystem()
	prepareStoredRepository(t, base, tt.initial)

	failingFS := &failureFileSystem{
		FileSystem: realFS,
		operation:  tt.failOperation,
		cause:      tt.cause,
	}
	store, err := Open(xdgEnvironment(base), failingFS)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	err = store.SetContextRepository(tt.initial, "/tmp/updated")
	assertLeakageFailure(t, err, tt, secrets...)
	assertStoredRepository(t, base, tt.initial)
}

func prepareStoredRepository(t *testing.T, base, initialPath string) {
	t.Helper()
	if initialPath == "" {
		return
	}
	initial, err := Open(xdgEnvironment(base), NewOSFileSystem())
	if err != nil {
		t.Fatalf("Open() initial error = %v", err)
	}
	if err := initial.SetContextRepository("", initialPath); err != nil {
		t.Fatalf("initial.SetContextRepository() error = %v", err)
	}
}

func assertLeakageFailure(t *testing.T, err error, tt leakageFailureCase, secrets ...string) {
	t.Helper()
	if !errors.Is(err, tt.wantErr) {
		t.Fatalf("SetContextRepository() error = %v, want errors.Is(_, %v)", err, tt.wantErr)
	}
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) || pathErr != tt.cause {
		t.Fatalf("SetContextRepository() error = %v, want injected PathError", err)
	}
	if tt.failOperation == "write+remove-temp" {
		var configErr *Error
		if !errors.As(err, &configErr) || !errors.Is(configErr.Cleanup, tt.cause) {
			t.Fatalf("SetContextRepository() error = %v, want cleanup PathError", err)
		}
	}
	secrets = append(secrets, tt.cause.Error())
	for _, secret := range secrets {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("公開エラー文字列に内部情報 %q が含まれています: %q", secret, err)
		}
	}
}

func runCommitFailureCase(t *testing.T, operation string, wantErr error, wantCommitted bool) {
	t.Helper()
	base := realTempDir(t)
	realFS := NewOSFileSystem()
	initial, err := Open(xdgEnvironment(base), realFS)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := initial.SetContextRepository("", "/tmp/initial"); err != nil {
		t.Fatalf("initial SetContextRepository() error = %v", err)
	}

	failingFS := &failureFileSystem{FileSystem: realFS, operation: operation}
	store, err := Open(xdgEnvironment(base), failingFS)
	if err != nil {
		t.Fatalf("Open() with failureFileSystem error = %v", err)
	}
	err = store.SetContextRepository("/tmp/initial", "/tmp/updated")
	if !errors.Is(err, wantErr) {
		t.Fatalf("SetContextRepository() error = %v, want errors.Is(_, %v)", err, wantErr)
	}
	if operation == "short-write" && !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("SetContextRepository() error = %v, want io.ErrShortWrite", err)
	}

	reloaded, openErr := Open(xdgEnvironment(base), realFS)
	if openErr != nil {
		t.Fatalf("Open() reload error = %v", openErr)
	}
	want := "/tmp/initial"
	if wantCommitted {
		want = "/tmp/updated"
	}
	if got := reloaded.GetContextRepository(); got != want {
		t.Errorf("GetContextRepository() = %q, want %q", got, want)
	}
}

type failureFileSystem struct {
	FileSystem
	operation string
	cause     error
	readCalls int
}

func (f *failureFileSystem) OpenFile(path string, flag int, perm fs.FileMode) (File, error) {
	if filepath.Base(path) == lockFileName {
		if flag&os.O_CREATE != 0 && f.shouldFail("lock-create") {
			return nil, f.failure(errLockCreateTest)
		}
		if flag&os.O_CREATE == 0 && f.shouldFail("lock-open") {
			return nil, f.failure(errLockOpenTest)
		}
	}
	file, err := f.FileSystem.OpenFile(path, flag, perm)
	if err != nil {
		return nil, fmt.Errorf("テスト用ロックファイルオープン: %w", err)
	}
	if filepath.Base(path) == lockFileName {
		wrapper := &failureFile{File: file}
		if f.shouldFail("lock-close") {
			wrapper.closeErr = f.failure(errLockCloseTest)
		}
		if f.shouldFail("lock-chmod") {
			wrapper.chmodErr = f.failure(errLockChmodTest)
		}
		return wrapper, nil
	}
	return file, nil
}

func (f *failureFileSystem) CreateTemp(dir, pattern string) (File, error) {
	if f.shouldFail("create-temp") {
		return nil, f.failure(errCreateTempTest)
	}
	file, err := f.FileSystem.CreateTemp(dir, pattern)
	if err != nil {
		return nil, fmt.Errorf("テスト用一時ファイル作成: %w", err)
	}
	wrapper := &failureFile{File: file}
	switch {
	case f.shouldFail("temp-chmod"):
		wrapper.chmodErr = f.failure(errTempChmodTest)
	case f.shouldFail("short-write"):
		wrapper.shortWrite = true
	case f.shouldFail("write"):
		wrapper.writeErr = f.failure(errWriteTest)
	case f.shouldFail("temp-sync"):
		wrapper.syncErr = f.failure(errTempSyncTest)
	case f.shouldFail("temp-close"):
		wrapper.closeErr = f.failure(errTempCloseTest)
	}
	return wrapper, nil
}

func (f *failureFileSystem) ReadFile(path string) ([]byte, error) {
	f.readCalls++
	if f.shouldFail("reload") && f.readCalls > 1 {
		return nil, f.failure(errReloadTest)
	}
	data, err := f.FileSystem.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("テスト用設定読み込み: %w", err)
	}
	return data, nil
}

func (f *failureFileSystem) Rename(oldPath, newPath string) error {
	if f.shouldFail("rename") {
		return f.failure(errRenameTest)
	}
	if err := f.FileSystem.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("テスト用ファイル置換: %w", err)
	}
	return nil
}

func (f *failureFileSystem) Remove(path string) error {
	if strings.HasPrefix(filepath.Base(path), ".config-") && f.shouldFail("remove-temp") {
		return f.failure(errRemoveTempTest)
	}
	if err := f.FileSystem.Remove(path); err != nil {
		return fmt.Errorf("テスト用ファイル削除: %w", err)
	}
	return nil
}

func (f *failureFileSystem) Flock(file File) error {
	if f.shouldFail("flock") {
		return f.failure(errFlockTest)
	}
	if err := f.FileSystem.Flock(file); err != nil {
		return fmt.Errorf("テスト用ロック取得: %w", err)
	}
	return nil
}

func (f *failureFileSystem) OpenDir(path string) (File, error) {
	if f.shouldFail("dir-open") {
		return nil, errOpenDirectoryTest
	}
	file, err := f.FileSystem.OpenDir(path)
	if err != nil {
		return nil, fmt.Errorf("テスト用ディレクトリオープン: %w", err)
	}
	if f.shouldFail("dir-sync") {
		return &failureFile{File: file, syncErr: errDirectorySyncTest}, nil
	}
	if f.shouldFail("dir-close") {
		return &failureFile{File: file, closeErr: errDirectoryCloseTest}, nil
	}
	return file, nil
}

func (f *failureFileSystem) Funlock(file File) error {
	err := f.FileSystem.Funlock(file)
	if f.shouldFail("unlock") {
		return errUnlockTest
	}
	if err != nil {
		return fmt.Errorf("テスト用ロック解放: %w", err)
	}
	return nil
}

func (f *failureFileSystem) shouldFail(operation string) bool {
	return strings.Contains(f.operation, operation)
}

func (f *failureFileSystem) failure(defaultErr error) error {
	if f.cause != nil {
		return f.cause
	}
	return defaultErr
}

type failureFile struct {
	File
	writeErr   error
	syncErr    error
	closeErr   error
	chmodErr   error
	shortWrite bool
}

func (f *failureFile) Write(p []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	if f.shortWrite {
		written, err := f.File.Write(p[:len(p)-1])
		if err != nil {
			return written, fmt.Errorf("テスト用短い書き込み: %w", err)
		}
		return written, nil
	}
	written, err := f.File.Write(p)
	if err != nil {
		return written, fmt.Errorf("テスト用ファイル書き込み: %w", err)
	}
	return written, nil
}

func assertConfigError(t *testing.T, err, wantKind, wantCause error) {
	t.Helper()
	if !errors.Is(err, wantKind) {
		t.Fatalf("error = %v, want errors.Is(_, %v)", err, wantKind)
	}
	if !errors.Is(err, wantCause) {
		t.Fatalf("error = %v, want errors.Is(_, %v)", err, wantCause)
	}
	var configErr *Error
	if !errors.As(err, &configErr) {
		t.Fatalf("error = %v, want *Error", err)
	}
}

func assertStoredRepository(t *testing.T, base, want string) {
	t.Helper()
	reloaded, err := Open(xdgEnvironment(base), NewOSFileSystem())
	if err != nil {
		t.Fatalf("Open() reload error = %v", err)
	}
	if got := reloaded.GetContextRepository(); got != want {
		t.Errorf("GetContextRepository() = %q, want %q", got, want)
	}
}

func (f *failureFile) Sync() error {
	if f.syncErr != nil {
		return f.syncErr
	}
	if err := f.File.Sync(); err != nil {
		return fmt.Errorf("テスト用ファイル同期: %w", err)
	}
	return nil
}

func (f *failureFile) Close() error {
	err := f.File.Close()
	if f.closeErr != nil {
		return f.closeErr
	}
	if err != nil {
		return fmt.Errorf("テスト用ファイルクローズ: %w", err)
	}
	return nil
}

func (f *failureFile) Chmod(mode fs.FileMode) error {
	if f.chmodErr != nil {
		return f.chmodErr
	}
	if err := f.File.Chmod(mode); err != nil {
		return fmt.Errorf("テスト用ファイル権限変更: %w", err)
	}
	return nil
}
