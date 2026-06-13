package repository

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	repositoryTarget = "."
	parentTarget     = "repository parent"
)

// FileSystem はリポジトリ検証に必要なファイルシステム操作を表します。
type FileSystem interface {
	Abs(path string) (string, error)
	Lstat(path string) (os.FileInfo, error)
}

type standardFileSystem struct{}

// NewFileSystem は標準ファイルシステムを使用する実装を返します。
func NewFileSystem() FileSystem {
	return standardFileSystem{}
}

func (standardFileSystem) Abs(path string) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to convert repository path to absolute path: %w", err)
	}
	return absolutePath, nil
}

func (standardFileSystem) Lstat(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect repository path: %w", err)
	}
	return info, nil
}

// Validator はContext Repositoryのパスと必須構造を検証します。
type Validator struct {
	fs FileSystem
}

// NewValidator は指定されたファイルシステムを使うValidatorを返します。
func NewValidator(fileSystem FileSystem) *Validator {
	return &Validator{fs: fileSystem}
}

// Validate はリンクを解決せずに検証し、字句的に正規化した絶対パスを返します。
func (v *Validator) Validate(path string) (string, error) {
	cleanedInput := filepath.Clean(path)
	absolutePath, err := v.fs.Abs(cleanedInput)
	if err != nil {
		return "", validationError(ErrIO, repositoryTarget, err)
	}
	absolutePath = filepath.Clean(absolutePath)

	for _, parent := range inspectionParents(cleanedInput, absolutePath) {
		info, lstatErr := v.fs.Lstat(parent)
		if lstatErr != nil {
			return "", validationError(ErrIO, parentTarget, lstatErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", validationError(ErrSymlink, parentTarget, nil)
		}
		if !info.IsDir() {
			return "", validationError(ErrIO, parentTarget, fs.ErrInvalid)
		}
	}

	if err := v.validateRepository(absolutePath); err != nil {
		return "", err
	}

	required := []struct {
		path   string
		target string
	}{
		{path: filepath.Join(absolutePath, "projects"), target: "projects"},
		{path: filepath.Join(absolutePath, "utils"), target: "utils"},
		{path: filepath.Join(absolutePath, "utils", "skills"), target: "utils/skills"},
	}
	for _, item := range required {
		if err := v.validateRequiredDirectory(item.path, item.target); err != nil {
			return "", err
		}
	}

	return absolutePath, nil
}

func (v *Validator) validateRepository(path string) error {
	info, err := v.fs.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return validationError(ErrPathNotExist, repositoryTarget, err)
		}
		return validationError(ErrIO, repositoryTarget, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return validationError(ErrSymlink, repositoryTarget, nil)
	}
	if !info.IsDir() {
		return validationError(ErrNotDirectory, repositoryTarget, nil)
	}
	return nil
}

func (v *Validator) validateRequiredDirectory(path, target string) error {
	info, err := v.fs.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return validationError(ErrRequiredStructure, target, err)
		}
		return validationError(ErrIO, target, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return validationError(ErrSymlink, target, nil)
	}
	if !info.IsDir() {
		return validationError(ErrRequiredStructure, target, nil)
	}
	return nil
}

func validationError(kind error, target string, cause error) *ValidationError {
	return &ValidationError{Kind: kind, Target: target, Err: cause}
}

func inspectionParents(cleanedInput, absolutePath string) []string {
	if filepath.IsAbs(cleanedInput) {
		return absoluteParents(absolutePath)
	}

	parent := filepath.Dir(cleanedInput)
	if parent == "." {
		return nil
	}
	count := relativeParentComponentCount(parent)
	parents := make([]string, count)
	current := filepath.Dir(absolutePath)
	for i := count - 1; i >= 0; i-- {
		parents[i] = current
		current = filepath.Dir(current)
	}
	return parents
}

func absoluteParents(path string) []string {
	volume := filepath.VolumeName(path)
	root := volume + string(filepath.Separator)
	relative := strings.TrimPrefix(path, root)
	components := splitPath(relative)
	if len(components) <= 1 {
		return nil
	}

	parents := make([]string, 0, len(components)-1)
	current := root
	for _, component := range components[:len(components)-1] {
		current = filepath.Join(current, component)
		parents = append(parents, current)
	}
	return parents
}

func relativeParentComponentCount(path string) int {
	count := 0
	for _, component := range splitPath(path) {
		if component != ".." {
			count++
		}
	}
	return count
}

func splitPath(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool {
		return r == rune(filepath.Separator)
	})
}
