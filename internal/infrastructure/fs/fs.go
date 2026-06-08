// Package fs は、OSのファイルシステムコールを使用して domain.FileSystem インターフェースを実装します。
package fs

import (
	"context"
	"fmt"
	"io/fs"
	"os"

	"github.com/yukihito-jokyu/context-cli/internal/domain"
)

// LocalFileSystem は、実際のOSファイルシステム関数を使用して domain.FileSystem を実装します。
type LocalFileSystem struct{}

// NewLocalFileSystem は、新しい LocalFileSystem を作成します。
func NewLocalFileSystem() *LocalFileSystem {
	return &LocalFileSystem{}
}

type localFileStatus struct {
	info os.FileInfo
}

func (s *localFileStatus) IsDir() bool {
	return s.info.IsDir()
}

func (s *localFileStatus) IsRegular() bool {
	return s.info.Mode().IsRegular()
}

func (s *localFileStatus) IsSymlink() bool {
	return (s.info.Mode() & os.ModeSymlink) != 0
}

func (s *localFileStatus) Mode() fs.FileMode {
	return s.info.Mode()
}

// LStat は os.Lstat を呼び出し、結果をラップします。
func (fs *LocalFileSystem) LStat(_ context.Context, path string) (domain.FileStatus, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to lstat path %s: %w", path, err)
	}
	return &localFileStatus{info: info}, nil
}

type localFileEntry struct {
	entry os.DirEntry
}

func (e *localFileEntry) Name() string {
	return e.entry.Name()
}

func (e *localFileEntry) IsDir() bool {
	return e.entry.IsDir()
}

// ReadDir は os.ReadDir を呼び出し、結果をラップします。
func (fs *LocalFileSystem) ReadDir(_ context.Context, path string) ([]domain.FileEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", path, err)
	}
	var fileEntries []domain.FileEntry
	for _, entry := range entries {
		fileEntries = append(fileEntries, &localFileEntry{entry: entry})
	}
	return fileEntries, nil
}

// ReadFile は os.ReadFile を呼び出します。
func (fs *LocalFileSystem) ReadFile(_ context.Context, path string) ([]byte, error) {
	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return data, nil
}
