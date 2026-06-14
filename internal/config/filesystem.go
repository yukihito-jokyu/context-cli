package config

import (
	"fmt"
	"io"
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
)

// Environment は設定保存先の探索に必要な環境情報を提供します。
type Environment interface {
	LookupEnv(key string) (string, bool)
	UserHomeDir() (string, error)
}

// FileSystem は設定永続化に必要なファイルシステム操作を提供します。
//
//nolint:interfacebloat // 安全な永続化処理をテスト可能にするため操作単位を明示します。
type FileSystem interface {
	Lstat(path string) (fs.FileInfo, error)
	Mkdir(path string, perm fs.FileMode) error
	OpenFile(path string, flag int, perm fs.FileMode) (File, error)
	CreateTemp(dir, pattern string) (File, error)
	ReadFile(path string) ([]byte, error)
	Rename(oldPath, newPath string) error
	Remove(path string) error
	OpenDir(path string) (File, error)
	Flock(file File) error
	Funlock(file File) error
}

// File は同期とロックを含む設定ファイル操作を提供します。
//
//nolint:interfacebloat // 一時ファイルとロックファイルの失敗地点を個別に検証します。
type File interface {
	io.Writer
	Sync() error
	Close() error
	Chmod(mode fs.FileMode) error
	Name() string
	Fd() uintptr
}

type osEnvironment struct{}

// NewOSEnvironment はOSの環境情報を使用する実装を返します。
func NewOSEnvironment() Environment {
	return osEnvironment{}
}

func (osEnvironment) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

func (osEnvironment) UserHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("ホームディレクトリ取得: %w", err)
	}
	return home, nil
}

type osFileSystem struct{}

// NewOSFileSystem はOSのファイルシステムを使用する実装を返します。
func NewOSFileSystem() FileSystem {
	return osFileSystem{}
}

func (osFileSystem) Lstat(path string) (fs.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("ファイル情報取得: %w", err)
	}
	return info, nil
}

func (osFileSystem) Mkdir(path string, perm fs.FileMode) error {
	if err := os.Mkdir(path, perm); err != nil {
		return fmt.Errorf("ディレクトリ作成: %w", err)
	}
	return nil
}

func (osFileSystem) OpenFile(path string, flag int, perm fs.FileMode) (File, error) {
	// #nosec G304 -- Storeが検証した設定ディレクトリ配下だけを開きます。
	file, err := os.OpenFile(path, flag, perm)
	if err != nil {
		return nil, fmt.Errorf("ファイルオープン: %w", err)
	}
	return &osFile{file: file}, nil
}

func (osFileSystem) CreateTemp(dir, pattern string) (File, error) {
	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, fmt.Errorf("一時ファイル作成: %w", err)
	}
	return &osFile{file: file}, nil
}

func (osFileSystem) ReadFile(path string) ([]byte, error) {
	// #nosec G304 -- Storeが種別、権限、シンボリックリンクを検証した設定ファイルだけを読みます。
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ファイル読み込み: %w", err)
	}
	return data, nil
}

func (osFileSystem) Rename(oldPath, newPath string) error {
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("ファイル置換: %w", err)
	}
	return nil
}

func (osFileSystem) Remove(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("ファイル削除: %w", err)
	}
	return nil
}

func (osFileSystem) OpenDir(path string) (File, error) {
	// #nosec G304 -- Storeが検証した設定ディレクトリだけを開きます。
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ディレクトリオープン: %w", err)
	}
	return &osFile{file: file}, nil
}

func (osFileSystem) Flock(file File) error {
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return fmt.Errorf("排他ロック取得: %w", err)
	}
	return nil
}

func (osFileSystem) Funlock(file File) error {
	if err := unix.Flock(int(file.Fd()), unix.LOCK_UN); err != nil {
		return fmt.Errorf("排他ロック解放: %w", err)
	}
	return nil
}

type osFile struct {
	file *os.File
}

func (f *osFile) Write(p []byte) (int, error) {
	written, err := f.file.Write(p)
	if err != nil {
		return written, fmt.Errorf("ファイル書き込み: %w", err)
	}
	return written, nil
}

func (f *osFile) Sync() error {
	if err := f.file.Sync(); err != nil {
		return fmt.Errorf("ファイル同期: %w", err)
	}
	return nil
}

func (f *osFile) Close() error {
	if err := f.file.Close(); err != nil {
		return fmt.Errorf("ファイルクローズ: %w", err)
	}
	return nil
}

func (f *osFile) Chmod(mode fs.FileMode) error {
	if err := f.file.Chmod(mode); err != nil {
		return fmt.Errorf("ファイル権限変更: %w", err)
	}
	return nil
}

func (f *osFile) Name() string {
	return f.file.Name()
}

func (f *osFile) Fd() uintptr {
	return f.file.Fd()
}
