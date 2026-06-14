package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Validator はカレントディレクトリを安全なWorkspaceとして検証します。
type Validator struct {
	getwd func() (string, error)
	fs    FileSystem
}

// FileSystem はWorkspace検証に必要なファイルシステム操作を表します。
type FileSystem interface {
	Abs(string) (string, error)
	Lstat(string) (os.FileInfo, error)
}

type standardFileSystem struct{}

func (standardFileSystem) Abs(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to convert workspace path to absolute path: %w", err)
	}
	return absolute, nil
}

func (standardFileSystem) Lstat(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect workspace path: %w", err)
	}
	return info, nil
}

// NewValidator は指定したカレントディレクトリ取得関数を使うValidatorを返します。
func NewValidator(getwd func() (string, error)) *Validator {
	return NewValidatorWithFileSystem(getwd, standardFileSystem{})
}

// NewValidatorWithFileSystem は指定したファイルシステムを使うValidatorを返します。
func NewValidatorWithFileSystem(getwd func() (string, error), fileSystem FileSystem) *Validator {
	return &Validator{getwd: getwd, fs: fileSystem}
}

// Validate は字句的に正規化した絶対パスを返します。
func (v *Validator) Validate() (string, error) {
	current, err := v.getwd()
	if err != nil {
		return "", &Error{Kind: ErrCurrentDirectory, Err: err}
	}
	absolute, err := v.fs.Abs(filepath.Clean(current))
	if err != nil {
		return "", &Error{Kind: ErrAbsolutePath, Err: err}
	}
	absolute = filepath.Clean(absolute)

	for _, path := range pathComponents(absolute) {
		if inspectErr := v.inspectDirectory(path, path == absolute); inspectErr != nil {
			return "", inspectErr
		}
	}
	return absolute, nil
}

func (v *Validator) inspectDirectory(path string, workspaceRoot bool) error {
	info, err := v.fs.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) && workspaceRoot {
			return &Error{Kind: ErrNotExist, Err: err}
		}
		return &Error{Kind: ErrIO, Err: err}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return &Error{Kind: ErrSymlink}
	}
	if info.IsDir() {
		return nil
	}
	if workspaceRoot {
		return &Error{Kind: ErrNotDirectory}
	}
	return &Error{Kind: ErrIO, Err: fs.ErrInvalid}
}

func pathComponents(path string) []string {
	volume := filepath.VolumeName(path)
	root := volume + string(filepath.Separator)
	relative := strings.TrimPrefix(path, root)
	parts := strings.Split(relative, string(filepath.Separator))
	paths := make([]string, 0, len(parts))
	current := root
	for _, part := range parts {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		paths = append(paths, current)
	}
	return paths
}
