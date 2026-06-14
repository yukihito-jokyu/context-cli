//go:build darwin || linux

package distribution

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const temporaryDirectoryPermission = 0o700

type stageCleanup struct {
	parent       *os.File
	staging      *os.File
	name         string
	path         string
	closeStaging bool
}

type removalContext struct {
	hooks fileSystemHooks
}

func (f osFileSystem) Mkdir(path string, parent PathExpectation, perm fs.FileMode) error {
	if f.hooks.beforeMkdir != nil {
		f.hooks.beforeMkdir(path)
	}
	parentFile, err := openExpectedDirectory(parent)
	if err != nil {
		return err
	}
	defer func() { _ = parentFile.Close() }()
	if err := unix.Mkdirat(int(parentFile.Fd()), filepath.Base(path), uint32(perm.Perm())); err != nil {
		return classifyMutationError("create directory", err)
	}
	return nil
}

func (f osFileSystem) Stage(source string, parent PathExpectation) (string, error) {
	if f.hooks.beforeStage != nil {
		f.hooks.beforeStage(parent.Path)
	}
	parentFile, err := openExpectedDirectory(parent)
	if err != nil {
		return "", err
	}
	defer func() { _ = parentFile.Close() }()
	name, staging, err := createTemporaryDirectory(parentFile, parent.Path, ".context-stage-")
	if err != nil {
		return "", err
	}
	stagingPath := filepath.Join(parent.Path, name)
	if err := newSafeTree().copyToDirectory(source, staging); err != nil {
		return "", f.cleanupFailedStage(stageCleanup{
			parent: parentFile, staging: staging, name: name, path: stagingPath, closeStaging: true,
		}, err)
	}
	if err := staging.Sync(); err != nil {
		primaryErr := newError("stage sync directory", ErrIO, err)
		return "", f.cleanupFailedStage(stageCleanup{
			parent: parentFile, staging: staging, name: name, path: stagingPath, closeStaging: true,
		}, primaryErr)
	}
	if err := staging.Close(); err != nil {
		primaryErr := newError("stage close directory", ErrIO, err)
		return "", f.cleanupFailedStage(stageCleanup{
			parent: parentFile, name: name, path: stagingPath,
		}, primaryErr)
	}
	return stagingPath, nil
}

func (f osFileSystem) cleanupFailedStage(cleanup stageCleanup, primaryErr error) error {
	var cleanupErr error
	if cleanup.closeStaging {
		if err := cleanup.staging.Close(); err != nil {
			cleanupErr = errors.Join(cleanupErr, newError("stage cleanup close", ErrIO, err))
		}
	}
	removeErr := removalContext(f).removeEntry(cleanup.parent, cleanup.name, cleanup.path, nil)
	if removeErr != nil {
		cleanupErr = errors.Join(cleanupErr, removeErr)
	}
	if cleanupErr == nil {
		return primaryErr
	}
	unrestored := []string{}
	if removeErr != nil {
		unrestored = append(unrestored, cleanup.path)
	}
	return &Error{
		Operation:  "stage cleanup",
		Kind:       ErrRollback,
		Err:        primaryErr,
		Cleanup:    cleanupErr,
		Unrestored: unrestored,
	}
}

func (f osFileSystem) Backup(path string, parent PathExpectation, expected PathExpectation) (string, error) {
	if f.hooks.beforeBackup != nil {
		f.hooks.beforeBackup(path)
	}
	parentFile, err := openExpectedDirectory(parent)
	if err != nil {
		return "", err
	}
	defer func() { _ = parentFile.Close() }()
	if err := verifyExpectedEntryAt(parentFile, filepath.Base(path), expected); err != nil {
		return "", err
	}
	name, err := temporaryName(".context-backup-")
	if err != nil {
		return "", err
	}
	if err := unix.Renameat(int(parentFile.Fd()), filepath.Base(path), int(parentFile.Fd()), name); err != nil {
		return "", classifyMutationError("backup rename", err)
	}
	return filepath.Join(parent.Path, name), nil
}

func (f osFileSystem) Rename(operation RenameOperation) error {
	if f.hooks.beforeRename != nil {
		f.hooks.beforeRename(operation.OldPath, operation.NewPath)
	}
	oldDirectory, err := openExpectedDirectory(operation.OldParent)
	if err != nil {
		return err
	}
	defer func() { _ = oldDirectory.Close() }()
	newDirectory, err := openExpectedDirectory(operation.NewParent)
	if err != nil {
		return err
	}
	defer func() { _ = newDirectory.Close() }()
	if err := verifyExpectedEntryAt(
		oldDirectory, filepath.Base(operation.OldPath), operation.OldExpected,
	); err != nil {
		return err
	}
	if err := verifyExpectedEntryAt(
		newDirectory, filepath.Base(operation.NewPath), operation.NewExpected,
	); err != nil {
		return err
	}
	if err := unix.Renameat(
		int(oldDirectory.Fd()), filepath.Base(operation.OldPath),
		int(newDirectory.Fd()), filepath.Base(operation.NewPath),
	); err != nil {
		return classifyMutationError("rename", err)
	}
	return nil
}

func (f osFileSystem) RemoveAll(path string, parent PathExpectation, expected PathExpectation) error {
	if f.hooks.beforeRemove != nil {
		f.hooks.beforeRemove(path)
	}
	parentFile, err := openExpectedDirectory(parent)
	if err != nil {
		return err
	}
	defer func() { _ = parentFile.Close() }()
	if err := verifyExpectedEntryAt(parentFile, filepath.Base(path), expected); err != nil {
		return err
	}
	if err := removalContext(f).removeEntry(
		parentFile, filepath.Base(path), path, &expected,
	); err != nil {
		return err
	}
	return nil
}

func (f osFileSystem) Remove(path string, parent PathExpectation, expected PathExpectation) error {
	if f.hooks.beforeRemove != nil {
		f.hooks.beforeRemove(path)
	}
	parentFile, err := openExpectedDirectory(parent)
	if err != nil {
		return err
	}
	defer func() { _ = parentFile.Close() }()
	if err := verifyEntryAt(parentFile, filepath.Base(path), expected); err != nil {
		return err
	}
	if err := unix.Unlinkat(int(parentFile.Fd()), filepath.Base(path), unix.AT_REMOVEDIR); err != nil {
		return classifyMutationError("remove directory", err)
	}
	return nil
}

func verifyEntryAt(parent *os.File, name string, expected PathExpectation) error {
	var current unix.Stat_t
	if err := unix.Fstatat(int(parent.Fd()), name, &current, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return classifyMutationError("verify entry", err)
	}
	device, inode := statIdentity(&current)
	if device != expected.Device || inode != expected.Inode ||
		pathKindFromUnixMode(uint64(current.Mode)) != expected.Kind {
		return newError("verify entry", ErrConflict, nil)
	}
	return nil
}

func verifyExpectedEntryAt(parent *os.File, name string, expected PathExpectation) error {
	var current unix.Stat_t
	err := unix.Fstatat(int(parent.Fd()), name, &current, unix.AT_SYMLINK_NOFOLLOW)
	if !expected.Exists {
		if errors.Is(err, unix.ENOENT) {
			return nil
		}
		if err != nil {
			return classifyMutationError("verify absent entry", err)
		}
		return newError("verify absent entry", ErrConflict, nil)
	}
	if err != nil {
		return classifyMutationError("verify entry", err)
	}
	device, inode := statIdentity(&current)
	if device != expected.Device || inode != expected.Inode ||
		pathKindFromUnixMode(uint64(current.Mode)) != expected.Kind {
		return newError("verify entry", ErrConflict, nil)
	}
	return nil
}

func pathKindFromUnixMode(mode uint64) PathKind {
	switch mode & unix.S_IFMT {
	case unix.S_IFDIR:
		return PathKindDirectory
	case unix.S_IFREG:
		return PathKindRegularFile
	case unix.S_IFIFO:
		return PathKindFIFO
	case unix.S_IFSOCK:
		return PathKindSocket
	case unix.S_IFCHR, unix.S_IFBLK:
		return PathKindDevice
	default:
		return PathKindOther
	}
}

func openExpectedDirectory(expected PathExpectation) (*os.File, error) {
	if !expected.Exists || expected.Kind != PathKindDirectory {
		return nil, newError("open expected directory", ErrConflict, nil)
	}
	directory, err := openDirectoryNoFollow(expected.Path)
	if err != nil {
		return nil, err
	}
	info, err := directory.Stat()
	if err != nil {
		_ = directory.Close()
		return nil, newError("stat expected directory", ErrIO, err)
	}
	device, inode := fileIdentity(info)
	if device != expected.Device || inode != expected.Inode || info.Mode().Perm() != expected.Perm {
		_ = directory.Close()
		return nil, newError("verify expected directory", ErrConflict, nil)
	}
	return directory, nil
}

func openDirectoryNoFollow(path string) (*os.File, error) {
	if !filepath.IsAbs(path) {
		return nil, newError("open directory", ErrUnsafePath, nil)
	}
	fd, err := unix.Open(string(filepath.Separator), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, newError("open root directory", ErrIO, err)
	}
	current := os.NewFile(uintptr(fd), string(filepath.Separator))
	if current == nil {
		_ = unix.Close(fd)
		return nil, newError("open root directory", ErrIO, nil)
	}
	components := strings.Split(strings.TrimPrefix(filepath.Clean(path), string(filepath.Separator)), string(filepath.Separator))
	if len(components) == 1 && components[0] == "" {
		return current, nil
	}
	for _, component := range components {
		nextFD, openErr := unix.Openat(
			int(current.Fd()), component,
			unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0,
		)
		if openErr != nil {
			_ = current.Close()
			return nil, classifyOpenError("open directory component", openErr)
		}
		next := os.NewFile(uintptr(nextFD), component)
		if next == nil {
			_ = unix.Close(nextFD)
			_ = current.Close()
			return nil, newError("open directory component", ErrIO, nil)
		}
		_ = current.Close()
		current = next
	}
	return current, nil
}

func createTemporaryDirectory(parent *os.File, parentPath, prefix string) (string, *os.File, error) {
	for range 100 {
		name, err := temporaryName(prefix)
		if err != nil {
			return "", nil, err
		}
		if err := unix.Mkdirat(int(parent.Fd()), name, temporaryDirectoryPermission); err != nil {
			if errors.Is(err, unix.EEXIST) {
				continue
			}
			return "", nil, classifyMutationError("create temporary directory", err)
		}
		fd, err := unix.Openat(
			int(parent.Fd()), name,
			unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0,
		)
		if err != nil {
			_ = unix.Unlinkat(int(parent.Fd()), name, unix.AT_REMOVEDIR)
			return "", nil, classifyOpenError("open temporary directory", err)
		}
		directory := os.NewFile(uintptr(fd), filepath.Join(parentPath, name))
		if directory == nil {
			_ = unix.Close(fd)
			_ = unix.Unlinkat(int(parent.Fd()), name, unix.AT_REMOVEDIR)
			return "", nil, newError("open temporary directory", ErrIO, nil)
		}
		return name, directory, nil
	}
	return "", nil, newError("create temporary directory", ErrIO, fs.ErrExist)
}

func temporaryName(prefix string) (string, error) {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", newError("create temporary name", ErrIO, err)
	}
	return prefix + hex.EncodeToString(random[:]), nil
}

func (context removalContext) removeEntry(
	parent *os.File,
	name string,
	path string,
	expectedState *PathExpectation,
) error {
	var expected unix.Stat_t
	if err := unix.Fstatat(int(parent.Fd()), name, &expected, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return classifyMutationError("inspect removal target", err)
	}
	if expectedState != nil && !statMatchesExpectation(expected, *expectedState) {
		return newError("verify removal target", ErrConflict, nil)
	}
	if context.hooks.beforeRemoveOpen != nil {
		context.hooks.beforeRemoveOpen(path)
	}
	if expected.Mode&unix.S_IFMT != unix.S_IFDIR {
		return context.removeNonDirectory(parent, name, path, expected)
	}
	return context.removeDirectory(parent, name, path, expected)
}

func (context removalContext) removeDirectory(
	parent *os.File,
	name string,
	path string,
	expected unix.Stat_t,
) error {
	fd, err := unix.Openat(
		int(parent.Fd()), name,
		unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0,
	)
	if err != nil {
		return classifyOpenError("open removal target", err)
	}
	directory := os.NewFile(uintptr(fd), name)
	if directory == nil {
		_ = unix.Close(fd)
		return newError("open removal target", ErrIO, nil)
	}
	var opened unix.Stat_t
	if err := unix.Fstat(fd, &opened); err != nil {
		_ = directory.Close()
		return newError("stat removal target", ErrIO, err)
	}
	if !sameStatIdentityAndType(expected, opened) {
		_ = directory.Close()
		return newError("verify opened removal target", ErrConflict, nil)
	}
	entries, err := directory.ReadDir(-1)
	if err != nil {
		_ = directory.Close()
		return newError("read removal target", ErrIO, err)
	}
	for _, entry := range entries {
		childPath := filepath.Join(path, entry.Name())
		if err := context.removeEntry(directory, entry.Name(), childPath, nil); err != nil {
			_ = directory.Close()
			return err
		}
	}
	if context.hooks.beforeRemoveUnlink != nil {
		if err := context.hooks.beforeRemoveUnlink(path); err != nil {
			_ = directory.Close()
			return newError("before remove directory", ErrIO, err)
		}
	}
	if err := verifyRemovalEntryAt(parent, name, opened); err != nil {
		_ = directory.Close()
		return err
	}
	if err := unix.Unlinkat(int(parent.Fd()), name, unix.AT_REMOVEDIR); err != nil {
		_ = directory.Close()
		return classifyMutationError("remove directory", err)
	}
	if err := directory.Close(); err != nil {
		return newError("close removal target", ErrIO, err)
	}
	return nil
}

func (context removalContext) removeNonDirectory(
	parent *os.File,
	name string,
	path string,
	expected unix.Stat_t,
) error {
	openedFD, opened, err := openRemovalEntry(parent, name, expected)
	if err != nil {
		return err
	}
	if context.hooks.beforeRemoveUnlink != nil {
		if err := context.hooks.beforeRemoveUnlink(path); err != nil {
			closeRemovalEntry(openedFD)
			return newError("before remove entry", ErrIO, err)
		}
	}
	if err := verifyRemovalEntryAt(parent, name, opened); err != nil {
		closeRemovalEntry(openedFD)
		return err
	}
	if err := unix.Unlinkat(int(parent.Fd()), name, 0); err != nil {
		closeRemovalEntry(openedFD)
		return classifyMutationError("remove entry", err)
	}
	if openedFD >= 0 {
		if err := unix.Close(openedFD); err != nil {
			return newError("close removal entry", ErrIO, err)
		}
	}
	return nil
}

func openRemovalEntry(parent *os.File, name string, expected unix.Stat_t) (int, unix.Stat_t, error) {
	if expected.Mode&unix.S_IFMT != unix.S_IFREG {
		return -1, expected, nil
	}
	fd, err := unix.Openat(
		int(parent.Fd()), name,
		unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0,
	)
	if err != nil {
		return -1, unix.Stat_t{}, classifyOpenError("open removal entry", err)
	}
	var opened unix.Stat_t
	if err := unix.Fstat(fd, &opened); err != nil {
		_ = unix.Close(fd)
		return -1, unix.Stat_t{}, newError("stat removal entry", ErrIO, err)
	}
	if !sameStatIdentityAndType(expected, opened) {
		_ = unix.Close(fd)
		return -1, unix.Stat_t{}, newError("verify opened removal entry", ErrConflict, nil)
	}
	return fd, opened, nil
}

func closeRemovalEntry(fd int) {
	if fd >= 0 {
		_ = unix.Close(fd)
	}
}

func verifyRemovalEntryAt(parent *os.File, name string, opened unix.Stat_t) error {
	var current unix.Stat_t
	if err := unix.Fstatat(int(parent.Fd()), name, &current, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return classifyMutationError("verify removal entry", err)
	}
	if !sameStatIdentityAndType(opened, current) {
		return newError("verify removal entry", ErrConflict, nil)
	}
	return nil
}

func sameStatIdentityAndType(left, right unix.Stat_t) bool {
	leftDevice, leftInode := statIdentity(&left)
	rightDevice, rightInode := statIdentity(&right)
	return leftDevice == rightDevice &&
		leftInode == rightInode &&
		left.Mode&unix.S_IFMT == right.Mode&unix.S_IFMT
}

func statMatchesExpectation(stat unix.Stat_t, expected PathExpectation) bool {
	device, inode := statIdentity(&stat)
	return expected.Exists &&
		device == expected.Device &&
		inode == expected.Inode &&
		pathKindFromUnixMode(uint64(stat.Mode)) == expected.Kind
}

func classifyMutationError(operation string, err error) error {
	if errors.Is(err, unix.ENOENT) || errors.Is(err, unix.ESTALE) ||
		errors.Is(err, unix.ENOTDIR) || errors.Is(err, unix.ELOOP) {
		return newError(operation, ErrConflict, err)
	}
	return newError(operation, ErrIO, err)
}
