package fs_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	infra "github.com/yukihito-jokyu/context-cli/internal/infrastructure/fs"
)

type lstatTestCase struct {
	name          string
	path          string
	wantIsDir     bool
	wantIsRegular bool
	wantIsSymlink bool
	wantPerm      os.FileMode
	wantErr       error
}

// TestLocalFileSystem_LStat は LStat メソッドの動作を検証するテーブル駆動テストです。
func TestLocalFileSystem_LStat(t *testing.T) {
	lfs := infra.NewLocalFileSystem()
	tmpDir := t.TempDir()

	// テスト用のファイル、ディレクトリ、シンボリックリンクを作成
	testFile := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0600); err != nil {
		t.Fatalf("ファイルの作成に失敗しました: %v", err)
	}

	subDir := filepath.Join(tmpDir, "dir")
	if err := os.Mkdir(subDir, 0700); err != nil {
		t.Fatalf("ディレクトリの作成に失敗しました: %v", err)
	}

	symlinkPath := filepath.Join(tmpDir, "link.txt")
	if err := os.Symlink(testFile, symlinkPath); err != nil {
		t.Fatalf("シンボリックリンクの作成に失敗しました: %v", err)
	}

	tests := []lstatTestCase{
		{
			name:          "通常のファイル",
			path:          testFile,
			wantIsDir:     false,
			wantIsRegular: true,
			wantIsSymlink: false,
			wantPerm:      0600,
			wantErr:       nil,
		},
		{
			name:          "ディレクトリ",
			path:          subDir,
			wantIsDir:     true,
			wantIsRegular: false,
			wantIsSymlink: false,
			wantPerm:      0700,
			wantErr:       nil,
		},
		{
			name:          "シンボリックリンク",
			path:          symlinkPath,
			wantIsDir:     false,
			wantIsRegular: false,
			wantIsSymlink: true,
			wantErr:       nil,
		},
		{
			name:    "存在しないパス",
			path:    filepath.Join(tmpDir, "does-not-exist"),
			wantErr: os.ErrNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runLStatTestCase(t, lfs, tt)
		})
	}
}

func runLStatTestCase(t *testing.T, lfs *infra.LocalFileSystem, tt lstatTestCase) {
	t.Helper()
	status, err := lfs.LStat(context.Background(), tt.path)

	if tt.wantErr != nil {
		if err == nil {
			t.Fatal("エラーが発生すると予想されましたが、発生しませんでした")
		}
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("想定されたエラー: %v, 実際のエラー: %v", tt.wantErr, err)
		}
		return
	}

	if err != nil {
		t.Fatalf("LStat に失敗しました: %v", err)
	}

	if status.IsDir() != tt.wantIsDir {
		t.Errorf("IsDir() = %v, 想定: %v", status.IsDir(), tt.wantIsDir)
	}
	if status.IsRegular() != tt.wantIsRegular {
		t.Errorf("IsRegular() = %v, 想定: %v", status.IsRegular(), tt.wantIsRegular)
	}
	if status.IsSymlink() != tt.wantIsSymlink {
		t.Errorf("IsSymlink() = %v, 想定: %v", status.IsSymlink(), tt.wantIsSymlink)
	}
	if tt.wantPerm != 0 && status.Mode().Perm() != tt.wantPerm {
		t.Errorf("パーミッション = %o, 想定: %o", status.Mode().Perm(), tt.wantPerm)
	}
}

type readDirTestCase struct {
	name        string
	path        string
	wantErr     error
	wantEntries map[string]bool
}

// TestLocalFileSystem_ReadDir は ReadDir メソッドの動作を検証するテーブル駆動テストです。
func TestLocalFileSystem_ReadDir(t *testing.T) {
	lfs := infra.NewLocalFileSystem()
	tmpDir := t.TempDir()

	// テスト用のディレクトリ構造をセットアップ
	subDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(subDir, 0700); err != nil {
		t.Fatalf("ディレクトリの作成に失敗しました: %v", err)
	}

	testFile := filepath.Join(subDir, "file.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		t.Fatalf("ファイルの作成に失敗しました: %v", err)
	}

	symlinkPath := filepath.Join(subDir, "link.txt")
	if err := os.Symlink(testFile, symlinkPath); err != nil {
		t.Fatalf("シンボリックリンクの作成に失敗しました: %v", err)
	}

	nestedDir := filepath.Join(subDir, "nested")
	if err := os.Mkdir(nestedDir, 0700); err != nil {
		t.Fatalf("サブディレクトリの作成に失敗しました: %v", err)
	}

	tests := []readDirTestCase{
		{
			name: "正常なディレクトリ読み込み",
			path: subDir,
			wantEntries: map[string]bool{
				"file.txt": false,
				"link.txt": false,
				"nested":   true,
			},
			wantErr: nil,
		},
		{
			name:    "存在しないディレクトリ",
			path:    filepath.Join(tmpDir, "does-not-exist"),
			wantErr: os.ErrNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runReadDirTestCase(t, lfs, tt)
		})
	}
}

func runReadDirTestCase(t *testing.T, lfs *infra.LocalFileSystem, tt readDirTestCase) {
	t.Helper()
	entries, err := lfs.ReadDir(context.Background(), tt.path)

	if tt.wantErr != nil {
		if err == nil {
			t.Fatal("エラーが発生すると予想されましたが、発生しませんでした")
		}
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("想定されたエラー: %v, 実際のエラー: %v", tt.wantErr, err)
		}
		return
	}

	if err != nil {
		t.Fatalf("ReadDir に失敗しました: %v", err)
	}

	if len(entries) != len(tt.wantEntries) {
		t.Fatalf("エントリ数 = %d, 想定: %d", len(entries), len(tt.wantEntries))
	}

	for _, entry := range entries {
		wantIsDir, ok := tt.wantEntries[entry.Name()]
		if !ok {
			t.Errorf("予期しないエントリが見つかりました: %s", entry.Name())
			continue
		}
		if entry.IsDir() != wantIsDir {
			t.Errorf("エントリ %s の IsDir() = %v, 想定: %v", entry.Name(), entry.IsDir(), wantIsDir)
		}
	}
}

type readFileTestCase struct {
	name     string
	path     string
	wantData []byte
	wantErr  error
}

// TestLocalFileSystem_ReadFile は ReadFile メソッドの動作を検証するテーブル駆動テストです。
func TestLocalFileSystem_ReadFile(t *testing.T) {
	lfs := infra.NewLocalFileSystem()
	tmpDir := t.TempDir()

	// テスト用のファイルを作成
	testFile := filepath.Join(tmpDir, "file.txt")
	content := []byte("hello world")
	if err := os.WriteFile(testFile, content, 0600); err != nil {
		t.Fatalf("ファイルの作成に失敗しました: %v", err)
	}

	tests := []readFileTestCase{
		{
			name:     "正常なファイル読み込み",
			path:     testFile,
			wantData: content,
			wantErr:  nil,
		},
		{
			name:    "存在しないファイル",
			path:    filepath.Join(tmpDir, "does-not-exist"),
			wantErr: os.ErrNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runReadFileTestCase(t, lfs, tt)
		})
	}
}

func runReadFileTestCase(t *testing.T, lfs *infra.LocalFileSystem, tt readFileTestCase) {
	t.Helper()
	data, err := lfs.ReadFile(context.Background(), tt.path)

	if tt.wantErr != nil {
		if err == nil {
			t.Fatal("エラーが発生すると予想されましたが、発生しませんでした")
		}
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("想定されたエラー: %v, 実際のエラー: %v", tt.wantErr, err)
		}
		return
	}

	if err != nil {
		t.Fatalf("ReadFile に失敗しました: %v", err)
	}

	if string(data) != string(tt.wantData) {
		t.Errorf("データ = %q, 想定: %q", string(data), string(tt.wantData))
	}
}
