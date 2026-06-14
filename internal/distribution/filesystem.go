package distribution

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
)

// FileSystem は配布計画と実行に必要なファイル操作を表します。
//
//nolint:interfacebloat // 失敗地点ごとのロールバックを検証するため操作単位を分離します。
type FileSystem interface {
	Inspect(path string, kind PathKind, allowMissing bool) (PathExpectation, error)
	Revalidate(expectations []PathExpectation) error
	HashSkill(path string) (string, error)
	Mkdir(path string, perm fs.FileMode) error
	Stage(source, parent string) (string, error)
	Backup(path string) (string, error)
	Rename(oldPath, newPath string) error
	RemoveAll(path string) error
	Remove(path string) error
}

type osFileSystem struct{}

// NewOSFileSystem はローカルファイルシステムを使う配布実装を返します。
func NewOSFileSystem() FileSystem {
	return osFileSystem{}
}

func (osFileSystem) Inspect(path string, kind PathKind, allowMissing bool) (PathExpectation, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) && allowMissing {
		return PathExpectation{Path: path}, nil
	}
	if err != nil {
		return PathExpectation{}, newError("inspect", ErrIO, err)
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return PathExpectation{}, newError("inspect", ErrSymlink, nil)
	}
	switch kind {
	case PathKindDirectory:
		if !info.IsDir() {
			return PathExpectation{}, newError("inspect", ErrFileType, nil)
		}
	case PathKindRegularFile:
		if !info.Mode().IsRegular() {
			return PathExpectation{}, newError("inspect", ErrFileType, nil)
		}
	default:
		return PathExpectation{}, newError("inspect", ErrStructure, nil)
	}
	device, inode := fileIdentity(info)
	return PathExpectation{
		Path: path, Exists: true, Kind: kind, Perm: info.Mode().Perm(), Device: device, Inode: inode,
	}, nil
}

func (f osFileSystem) Revalidate(expectations []PathExpectation) error {
	for _, expected := range expectations {
		current, err := f.Inspect(expected.Path, expected.Kind, true)
		if err != nil {
			return newError("revalidate", ErrConflict, err)
		}
		if current.Exists != expected.Exists ||
			current.Kind != expected.Kind ||
			current.Perm != expected.Perm ||
			current.Device != expected.Device ||
			current.Inode != expected.Inode {
			return newError("revalidate", ErrConflict, nil)
		}
	}
	return nil
}

func (osFileSystem) HashSkill(path string) (string, error) {
	return HashSkill(path)
}

func (osFileSystem) Mkdir(path string, perm fs.FileMode) error {
	if err := os.Mkdir(path, perm); err != nil {
		return newError("create directory", ErrIO, err)
	}
	return nil
}

func (osFileSystem) Stage(source, parent string) (string, error) {
	sourceInfo, err := os.Lstat(source)
	if err != nil {
		return "", newError("stage inspect", ErrIO, err)
	}
	if sourceInfo.Mode()&fs.ModeSymlink != 0 || !sourceInfo.IsDir() {
		return "", newError("stage inspect", ErrStructure, nil)
	}
	staging, err := os.MkdirTemp(parent, ".context-stage-*")
	if err != nil {
		return "", newError("stage create", ErrIO, err)
	}
	if err := os.Chmod(staging, sourceInfo.Mode().Perm()); err != nil {
		_ = os.RemoveAll(staging)
		return "", newError("stage chmod", ErrIO, err)
	}
	if err := copyDirectoryContents(source, staging); err != nil {
		_ = os.RemoveAll(staging)
		return "", err
	}
	if err := syncDirectory(staging); err != nil {
		_ = os.RemoveAll(staging)
		return "", err
	}
	return staging, nil
}

func (osFileSystem) Backup(path string) (string, error) {
	parent := filepath.Dir(path)
	backup, err := os.MkdirTemp(parent, ".context-backup-*")
	if err != nil {
		return "", newError("backup create", ErrIO, err)
	}
	if err := os.Remove(backup); err != nil {
		return "", newError("backup prepare", ErrIO, err)
	}
	if err := os.Rename(path, backup); err != nil {
		return "", newError("backup rename", ErrIO, err)
	}
	return backup, nil
}

func (osFileSystem) Rename(oldPath, newPath string) error {
	if err := os.Rename(oldPath, newPath); err != nil {
		return newError("rename", ErrIO, err)
	}
	return nil
}

func (osFileSystem) RemoveAll(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return newError("remove tree", ErrIO, err)
	}
	return nil
}

func (osFileSystem) Remove(path string) error {
	if err := os.Remove(path); err != nil {
		return newError("remove directory", ErrIO, err)
	}
	return nil
}

//nolint:gocognit // 再帰コピーでは種別ごとの安全検証と同期を同じ走査で行います。
func copyDirectoryContents(source, target string) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return newError("stage read directory", ErrIO, err)
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(source, entry.Name())
		targetPath := filepath.Join(target, entry.Name())
		info, err := os.Lstat(sourcePath)
		if err != nil {
			return newError("stage inspect", ErrIO, err)
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return newError("stage inspect", ErrSymlink, nil)
		}
		if info.IsDir() {
			if err := os.Mkdir(targetPath, info.Mode().Perm()); err != nil {
				return newError("stage create directory", ErrIO, err)
			}
			if err := copyDirectoryContents(sourcePath, targetPath); err != nil {
				return err
			}
			if err := syncDirectory(targetPath); err != nil {
				return err
			}
			continue
		}
		if !info.Mode().IsRegular() {
			return newError("stage inspect", ErrFileType, nil)
		}
		if err := copyRegularFile(sourcePath, targetPath, info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}

func copyRegularFile(source, target string, perm fs.FileMode) error {
	input, err := os.Open(source) // #nosec G304 -- Lstatで検証したSkill配下通常ファイルだけを開きます。
	if err != nil {
		return newError("stage open source", ErrIO, err)
	}
	defer func() { _ = input.Close() }()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm) // #nosec G304 -- 検証済み配布先の一時領域だけへ作成します。
	if err != nil {
		return newError("stage create file", ErrIO, err)
	}
	var operationErr error
	if _, err := io.Copy(output, input); err != nil {
		operationErr = newError("stage copy file", ErrIO, err)
	} else if err := output.Sync(); err != nil {
		operationErr = newError("stage sync file", ErrIO, err)
	}
	if closeErr := output.Close(); closeErr != nil {
		operationErr = errors.Join(operationErr, newError("stage close file", ErrIO, closeErr))
	}
	return operationErr
}

func syncDirectory(path string) error {
	directory, err := os.Open(path) // #nosec G304 -- 検証済み配布先の一時ディレクトリだけを開きます。
	if err != nil {
		return newError("stage open directory", ErrIO, err)
	}
	var operationErr error
	if err := directory.Sync(); err != nil {
		operationErr = newError("stage sync directory", ErrIO, err)
	}
	if err := directory.Close(); err != nil {
		operationErr = errors.Join(operationErr, newError("stage close directory", ErrIO, err))
	}
	return operationErr
}

func fileIdentity(info fs.FileInfo) (uint64, uint64) {
	value := reflect.ValueOf(info.Sys())
	if !value.IsValid() {
		return 0, 0
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return 0, 0
		}
		value = value.Elem()
	}
	device := unsignedField(value, "Dev")
	inode := unsignedField(value, "Ino")
	return device, inode
}

func unsignedField(value reflect.Value, name string) uint64 {
	field := value.FieldByName(name)
	if !field.IsValid() {
		return 0
	}
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value := field.Int()
		if value < 0 {
			return 0
		}
		return uint64(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return field.Uint()
	case reflect.Invalid, reflect.Bool, reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128, reflect.Array, reflect.Chan,
		reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer,
		reflect.Slice, reflect.String, reflect.Struct, reflect.UnsafePointer:
		return 0
	}
	return 0
}

func destinationRelativePath(destination Destination, skillName string) (string, error) {
	switch destination {
	case DestinationCodex:
		return filepath.Join(".codex", "skills", skillName), nil
	case DestinationClaude:
		return filepath.Join(".claude", "skills", skillName), nil
	default:
		return "", fmt.Errorf("%w: unknown destination", ErrStructure)
	}
}
