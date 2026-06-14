package distribution

import (
	"errors"
	"fmt"
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
	Mkdir(path string, parent PathExpectation, perm fs.FileMode) error
	Stage(source string, parent PathExpectation) (string, error)
	Backup(path string, parent PathExpectation, expected PathExpectation) (string, error)
	Rename(operation RenameOperation) error
	RemoveAll(path string, parent PathExpectation, expected PathExpectation) error
	Remove(path string, parent PathExpectation, expected PathExpectation) error
}

type fileSystemHooks struct {
	beforeMkdir        func(path string)
	beforeStage        func(parent string)
	beforeBackup       func(path string)
	beforeRename       func(oldPath, newPath string)
	beforeRemove       func(path string)
	beforeRemoveOpen   func(path string)
	beforeRemoveUnlink func(path string) error
}

type osFileSystem struct {
	hooks fileSystemHooks
}

// NewOSFileSystem はローカルファイルシステムを使う配布実装を返します。
func NewOSFileSystem() FileSystem {
	return osFileSystem{}
}

func (osFileSystem) Inspect(path string, kind PathKind, allowMissing bool) (PathExpectation, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) && allowMissing {
		return PathExpectation{Path: path, Kind: kind}, nil
	}
	if err != nil {
		return PathExpectation{}, newError("inspect", ErrIO, err)
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return PathExpectation{}, newError("inspect", ErrSymlink, nil)
	}
	actualKind := pathKind(info.Mode())
	switch kind {
	case PathKindDirectory:
		if !info.IsDir() {
			return PathExpectation{}, newError("inspect", ErrFileType, nil)
		}
	case PathKindRegularFile:
		if !info.Mode().IsRegular() {
			return PathExpectation{}, newError("inspect", ErrFileType, nil)
		}
	case PathKindFIFO, PathKindSocket, PathKindDevice, PathKindOther:
		if actualKind != kind {
			return PathExpectation{}, newError("inspect", ErrFileType, nil)
		}
	case PathKindAny:
	default:
		return PathExpectation{}, newError("inspect", ErrStructure, nil)
	}
	device, inode := fileIdentity(info)
	return PathExpectation{
		Path: path, Exists: true, Kind: actualKind, Perm: info.Mode().Perm(), Device: device, Inode: inode,
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

func pathKind(mode fs.FileMode) PathKind {
	switch {
	case mode.IsDir():
		return PathKindDirectory
	case mode.IsRegular():
		return PathKindRegularFile
	case mode&fs.ModeNamedPipe != 0:
		return PathKindFIFO
	case mode&fs.ModeSocket != 0:
		return PathKindSocket
	case mode&fs.ModeDevice != 0:
		return PathKindDevice
	default:
		return PathKindOther
	}
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
