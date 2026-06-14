//go:build darwin || linux

package distribution

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"

	"golang.org/x/sys/unix"
)

type safeTreeHooks struct {
	beforeRootOpen func(path string)
	beforeOpen     func(relative string)
	afterOpen      func(relative string)
	beforeRead     func(relative string)
}

type safeTree struct {
	hooks safeTreeHooks
}

type safeTreeEntry struct {
	relativePath string
	mode         fs.FileMode
	isDirectory  bool
	content      []byte
}

func newSafeTree() *safeTree {
	return &safeTree{}
}

func (tree *safeTree) hash(root string) (string, error) {
	entries, err := tree.read(root)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	if _, err := io.WriteString(hash, skillHashHeader); err != nil {
		return "", newError("hash header", ErrIO, err)
	}
	for _, entry := range entries {
		if err := writeSafeTreeHashEntry(hash, entry); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

//nolint:gocognit,cyclop // コピー、権限維持、同期、クローズ失敗を一つの処理境界で扱います。
func (tree *safeTree) copyToDirectory(source string, target *os.File) error {
	entries, err := tree.read(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.relativePath == "." {
			if err := unix.Fchmod(int(target.Fd()), uint32(entry.mode.Perm())); err != nil {
				return newError("stage chmod", ErrIO, err)
			}
			continue
		}
		parent, name, err := openRelativeParent(target, entry.relativePath)
		if err != nil {
			return err
		}
		if entry.isDirectory {
			if err := unix.Mkdirat(int(parent.Fd()), name, uint32(entry.mode.Perm())); err != nil {
				_ = parent.Close()
				return newError("stage create directory", ErrIO, err)
			}
			_ = parent.Close()
			continue
		}
		fd, err := unix.Openat(
			int(parent.Fd()), name,
			unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW,
			uint32(entry.mode.Perm()),
		)
		_ = parent.Close()
		if err != nil {
			return newError("stage create file", ErrIO, err)
		}
		file := os.NewFile(uintptr(fd), entry.relativePath)
		operationErr := error(nil)
		if _, err := file.Write(entry.content); err != nil {
			operationErr = newError("stage copy file", ErrIO, err)
		} else if err := file.Sync(); err != nil {
			operationErr = newError("stage sync file", ErrIO, err)
		}
		if err := file.Close(); err != nil {
			operationErr = errors.Join(operationErr, newError("stage close file", ErrIO, err))
		}
		if operationErr != nil {
			return operationErr
		}
	}
	for _, entry := range slices.Backward(entries) {
		if !entry.isDirectory {
			continue
		}
		directory := target
		if entry.relativePath != "." {
			directory, err = openRelativeDirectory(target, entry.relativePath)
			if err != nil {
				return err
			}
		}
		if err := directory.Sync(); err != nil {
			if directory != target {
				_ = directory.Close()
			}
			return newError("stage sync directory", ErrIO, err)
		}
		if directory != target {
			if err := directory.Close(); err != nil {
				return newError("stage close directory", ErrIO, err)
			}
		}
	}
	return nil
}

func (tree *safeTree) read(root string) ([]safeTreeEntry, error) {
	if tree.hooks.beforeRootOpen != nil {
		tree.hooks.beforeRootOpen(root)
	}
	rootFile, err := openDirectoryNoFollow(root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rootFile.Close() }()
	info, err := rootFile.Stat()
	if err != nil {
		return nil, newError("tree stat root", ErrIO, err)
	}
	if !info.IsDir() {
		return nil, newError("tree inspect root", ErrFileType, nil)
	}
	entries := []safeTreeEntry{{relativePath: ".", mode: info.Mode(), isDirectory: true}}
	if err := tree.readDirectory(rootFile, ".", &entries); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].relativePath < entries[j].relativePath
	})
	return entries, nil
}

//nolint:gocognit,cyclop // 各競合点の検証とFDの確実なクローズを同じ走査境界で扱います。
func (tree *safeTree) readDirectory(directory *os.File, relativeDirectory string, output *[]safeTreeEntry) error {
	entries, err := directory.ReadDir(-1)
	if err != nil {
		return newError("tree read directory", ErrIO, err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	for _, entry := range entries {
		relative := entry.Name()
		if relativeDirectory != "." {
			relative = filepath.ToSlash(filepath.Join(relativeDirectory, entry.Name()))
		}
		var expected unix.Stat_t
		if err := unix.Fstatat(int(directory.Fd()), entry.Name(), &expected, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			return classifyTreeStatError("tree inspect entry", err)
		}
		expectedType := expected.Mode & unix.S_IFMT
		listedType := entry.Type().Type()
		if listedType != unixModeToFileMode(uint64(expectedType)).Type() {
			return newError("tree inspect entry", ErrConflict, nil)
		}
		if expectedType == unix.S_IFLNK {
			return newError("tree inspect entry", ErrSymlink, nil)
		}
		if expectedType != unix.S_IFDIR && expectedType != unix.S_IFREG {
			return newError("tree inspect entry", ErrFileType, nil)
		}
		if tree.hooks.beforeOpen != nil {
			tree.hooks.beforeOpen(relative)
		}
		flags := unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW
		if expectedType == unix.S_IFDIR {
			flags |= unix.O_DIRECTORY
		}
		childFD, err := unix.Openat(int(directory.Fd()), entry.Name(), flags, 0)
		if err != nil {
			return classifyOpenError("tree open entry", err)
		}
		child := os.NewFile(uintptr(childFD), relative)
		if child == nil {
			_ = unix.Close(childFD)
			return newError("tree open entry", ErrIO, nil)
		}
		if tree.hooks.afterOpen != nil {
			tree.hooks.afterOpen(relative)
		}
		actual, statErr := child.Stat()
		if statErr != nil {
			_ = child.Close()
			return classifyTreeStatError("tree stat entry", statErr)
		}
		expectedDevice, expectedInode := statIdentity(&expected)
		actualDevice, actualInode := fileIdentity(actual)
		if expectedDevice != actualDevice || expectedInode != actualInode ||
			(expectedType == unix.S_IFDIR) != actual.IsDir() {
			_ = child.Close()
			return newError("tree verify entry", ErrConflict, nil)
		}
		*output = append(*output, safeTreeEntry{
			relativePath: relative,
			mode:         actual.Mode(),
			isDirectory:  actual.IsDir(),
		})
		if err = verifyOpenedEntry(directory, entry.Name(), actual); err != nil {
			_ = child.Close()
			return err
		}
		if actual.IsDir() {
			err = tree.readDirectory(child, relative, output)
		} else {
			if tree.hooks.beforeRead != nil {
				tree.hooks.beforeRead(relative)
			}
			if err = verifyOpenedEntry(directory, entry.Name(), actual); err == nil {
				var content []byte
				content, err = io.ReadAll(child)
				if err == nil {
					(*output)[len(*output)-1].content = content
				} else {
					err = newError("tree read file", ErrIO, err)
				}
			}
		}
		closeErr := child.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return newError("tree close entry", ErrIO, closeErr)
		}
	}
	return nil
}

func classifyTreeStatError(operation string, err error) error {
	if errors.Is(err, unix.ENOENT) || errors.Is(err, unix.ESTALE) || errors.Is(err, unix.ENOTDIR) {
		return newError(operation, ErrConflict, err)
	}
	return newError(operation, ErrIO, err)
}

func unixModeToFileMode(mode uint64) fs.FileMode {
	switch mode {
	case unix.S_IFDIR:
		return fs.ModeDir
	case unix.S_IFLNK:
		return fs.ModeSymlink
	case unix.S_IFIFO:
		return fs.ModeNamedPipe
	case unix.S_IFSOCK:
		return fs.ModeSocket
	case unix.S_IFCHR, unix.S_IFBLK:
		return fs.ModeDevice
	default:
		return 0
	}
}

func verifyOpenedEntry(parent *os.File, name string, opened fs.FileInfo) error {
	var current unix.Stat_t
	if err := unix.Fstatat(int(parent.Fd()), name, &current, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return newError("tree verify entry", ErrConflict, err)
	}
	device, inode := fileIdentity(opened)
	currentDevice, currentInode := statIdentity(&current)
	if currentDevice != device || currentInode != inode {
		return newError("tree verify entry", ErrConflict, nil)
	}
	if current.Mode&unix.S_IFMT == unix.S_IFLNK {
		return newError("tree verify entry", ErrSymlink, nil)
	}
	return nil
}

func statIdentity(stat *unix.Stat_t) (uint64, uint64) {
	value := reflect.ValueOf(stat).Elem()
	return unsignedField(value, "Dev"), unsignedField(value, "Ino")
}

func classifyOpenError(operation string, err error) error {
	if errors.Is(err, unix.ELOOP) {
		return newError(operation, ErrSymlink, err)
	}
	if errors.Is(err, unix.ENOENT) || errors.Is(err, unix.ESTALE) || errors.Is(err, unix.ENOTDIR) {
		return newError(operation, ErrConflict, err)
	}
	return newError(operation, ErrIO, err)
}

func openRelativeParent(root *os.File, relative string) (*os.File, string, error) {
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, "", newError("open relative parent", ErrUnsafePath, nil)
	}
	parentPath, name := filepath.Split(clean)
	parentPath = filepath.Clean(parentPath)
	if parentPath == "." {
		duplicate, err := unix.Dup(int(root.Fd()))
		if err != nil {
			return nil, "", newError("duplicate target directory", ErrIO, err)
		}
		return os.NewFile(uintptr(duplicate), "."), name, nil
	}
	parent, err := openRelativeDirectory(root, parentPath)
	return parent, name, err
}

func openRelativeDirectory(root *os.File, relative string) (*os.File, error) {
	duplicate, err := unix.Dup(int(root.Fd()))
	if err != nil {
		return nil, newError("duplicate target directory", ErrIO, err)
	}
	current := os.NewFile(uintptr(duplicate), ".")
	for component := range strings.SplitSeq(filepath.Clean(relative), string(filepath.Separator)) {
		fd, openErr := unix.Openat(
			int(current.Fd()), component,
			unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0,
		)
		if openErr != nil {
			_ = current.Close()
			return nil, classifyOpenError("open target directory", openErr)
		}
		next := os.NewFile(uintptr(fd), component)
		_ = current.Close()
		current = next
	}
	return current, nil
}

func writeSafeTreeHashEntry(writer io.Writer, entry safeTreeEntry) error {
	kind := hashFileRecord
	if entry.isDirectory {
		kind = hashDirectoryRecord
	}
	if err := binary.Write(writer, binary.BigEndian, kind); err != nil {
		return newError("hash encode", ErrIO, err)
	}
	pathBytes := []byte(entry.relativePath)
	if err := binary.Write(writer, binary.BigEndian, uint64(len(pathBytes))); err != nil {
		return newError("hash encode", ErrIO, err)
	}
	if _, err := writer.Write(pathBytes); err != nil {
		return newError("hash encode", ErrIO, err)
	}
	if err := binary.Write(writer, binary.BigEndian, uint32(entry.mode.Perm())); err != nil {
		return newError("hash encode", ErrIO, err)
	}
	if err := binary.Write(writer, binary.BigEndian, uint64(len(entry.content))); err != nil {
		return newError("hash encode", ErrIO, err)
	}
	if _, err := writer.Write(entry.content); err != nil {
		return newError("hash content", ErrIO, err)
	}
	return nil
}
