package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"syscall"
)

const (
	configDirectoryName = "context"
	configFileName      = "config.yaml"
	lockFileName        = "config.lock"
	directoryPermission = 0o700
	filePermission      = 0o600
)

// Store はContext Repository設定の参照と比較更新を提供します。
type Store struct {
	fileSystem        FileSystem
	configDirectory   string
	configPath        string
	contextRepository string
}

// Open は環境から設定保存先を探索し、安全な既存設定を読み込みます。
func Open(environment Environment, fileSystem FileSystem) (*Store, error) {
	configDirectory, err := discoverConfigDirectory(environment)
	if err != nil {
		return nil, err
	}
	store := &Store{
		fileSystem:      fileSystem,
		configDirectory: configDirectory,
		configPath:      filepath.Join(configDirectory, configFileName),
	}
	current, err := store.loadContextRepository()
	if err != nil {
		return nil, err
	}
	store.contextRepository = current
	return store, nil
}

// GetContextRepository は読み込み済みのContext Repositoryを返します。
func (s *Store) GetContextRepository() string {
	return s.contextRepository
}

// SetContextRepository は期待値が現在値と一致する場合だけ設定を更新します。
func (s *Store) SetContextRepository(expected, newPath string) error {
	if _, err := encodeSchema(newPath); err != nil {
		return err
	}
	if err := s.ensureConfigDirectory(); err != nil {
		return err
	}

	lock, err := s.openLockFile()
	if err != nil {
		return err
	}

	committed := false
	var primaryErr error
	var cleanupErr error

	if err := s.fileSystem.Flock(lock); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			primaryErr = newError("lock", ErrLockConflict, err)
		} else {
			primaryErr = newError("lock", ErrIO, err)
		}
	} else {
		current, loadErr := s.loadContextRepository()
		switch {
		case loadErr != nil:
			primaryErr = loadErr
		case current != expected:
			primaryErr = newError("compare", ErrUpdateConflict, nil)
		default:
			committed, primaryErr, cleanupErr = s.writeConfiguration(newPath)
		}

		if unlockErr := s.fileSystem.Funlock(lock); unlockErr != nil {
			cleanupErr = errors.Join(cleanupErr, unlockErr)
		}
	}
	if closeErr := lock.Close(); closeErr != nil {
		cleanupErr = errors.Join(cleanupErr, closeErr)
	}

	if committed {
		s.contextRepository = newPath
	}
	return classifyUpdateError(committed, primaryErr, cleanupErr)
}

func discoverConfigDirectory(environment Environment) (string, error) {
	if xdgHome, ok := environment.LookupEnv("XDG_CONFIG_HOME"); ok && xdgHome != "" {
		if !filepath.IsAbs(xdgHome) {
			return "", newError("discovery", ErrDiscovery, nil)
		}
		return filepath.Join(filepath.Clean(xdgHome), configDirectoryName), nil
	}

	home, err := environment.UserHomeDir()
	if err != nil {
		return "", newError("discovery", ErrDiscovery, err)
	}
	if !filepath.IsAbs(home) {
		return "", newError("discovery", ErrDiscovery, nil)
	}
	return filepath.Join(filepath.Clean(home), ".config", configDirectoryName), nil
}

func (s *Store) loadContextRepository() (string, error) {
	exists, err := s.validateDirectoryChain()
	if err != nil || !exists {
		return "", err
	}

	exists, err = s.validateRegularFile(s.configPath, true)
	if err != nil || !exists {
		return "", err
	}
	data, err := s.fileSystem.ReadFile(s.configPath)
	if err != nil {
		return "", newError("read", ErrIO, err)
	}
	value, err := decodeSchema(data)
	if err != nil {
		return "", err
	}
	return value.ContextRepository, nil
}

func (s *Store) validateDirectoryChain() (bool, error) {
	paths := pathComponents(s.configDirectory)
	for index, path := range paths {
		info, err := s.fileSystem.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		if err != nil {
			return false, newError("inspect", ErrIO, err)
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return false, newError("inspect", ErrSymlink, nil)
		}
		if !info.IsDir() {
			return false, newError("inspect", ErrFileType, nil)
		}
		if index == len(paths)-1 && info.Mode().Perm()&0o077 != 0 {
			return false, newError("inspect", ErrPermission, nil)
		}
	}
	return true, nil
}

func (s *Store) validateRegularFile(path string, allowMissing bool) (bool, error) {
	info, err := s.fileSystem.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) && allowMissing {
		return false, nil
	}
	if err != nil {
		return false, newError("inspect", ErrIO, err)
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return false, newError("inspect", ErrSymlink, nil)
	}
	if !info.Mode().IsRegular() {
		return false, newError("inspect", ErrFileType, nil)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return false, newError("inspect", ErrPermission, nil)
	}
	return true, nil
}

func (s *Store) ensureConfigDirectory() error {
	paths := pathComponents(s.configDirectory)
	firstMissing, err := s.firstMissingDirectory(paths)
	if err != nil {
		return err
	}
	if firstMissing == -1 {
		return nil
	}
	return s.createDirectories(paths[firstMissing:])
}

func (s *Store) firstMissingDirectory(paths []string) (int, error) {
	for index, path := range paths {
		info, err := s.fileSystem.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			return index, nil
		}
		if err != nil {
			return 0, newError("inspect", ErrIO, err)
		}
		if err := validateDirectoryInfo(info, index == len(paths)-1); err != nil {
			return 0, err
		}
	}
	return -1, nil
}

func (s *Store) createDirectories(paths []string) error {
	var created []string
	for _, path := range paths {
		if err := s.fileSystem.Mkdir(path, directoryPermission); err != nil {
			return s.directoryCreationError(created, err)
		}
		created = append(created, path)
		info, err := s.fileSystem.Lstat(path)
		if err != nil {
			return s.directoryCreationError(created, err)
		}
		if err := validateCreatedDirectoryInfo(info); err != nil {
			return s.directoryCreationError(created, err)
		}
	}
	return nil
}

func validateDirectoryInfo(info fs.FileInfo, validatePermission bool) error {
	if info.Mode()&fs.ModeSymlink != 0 {
		return newError("inspect", ErrSymlink, nil)
	}
	if !info.IsDir() {
		return newError("inspect", ErrFileType, nil)
	}
	if validatePermission && info.Mode().Perm()&0o077 != 0 {
		return newError("inspect", ErrPermission, nil)
	}
	return nil
}

func validateCreatedDirectoryInfo(info fs.FileInfo) error {
	if info.Mode()&fs.ModeSymlink != 0 {
		return ErrSymlink
	}
	if !info.IsDir() {
		return ErrFileType
	}
	if info.Mode().Perm() != directoryPermission {
		return ErrPermission
	}
	return nil
}

func (s *Store) directoryCreationError(created []string, cause error) error {
	var cleanupErr error
	for _, path := range slices.Backward(created) {
		if err := s.fileSystem.Remove(path); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	kind := ErrIO
	switch {
	case errors.Is(cause, ErrSymlink):
		kind = ErrSymlink
	case errors.Is(cause, ErrFileType):
		kind = ErrFileType
	case errors.Is(cause, ErrPermission):
		kind = ErrPermission
	}
	return &Error{
		Operation: "create directory",
		Kind:      kind,
		Err:       cause,
		Cleanup:   cleanupErr,
	}
}

func (s *Store) openLockFile() (File, error) {
	lockPath := filepath.Join(s.configDirectory, lockFileName)
	exists, err := s.validateRegularFile(lockPath, true)
	if err != nil {
		return nil, err
	}
	if exists {
		lock, openErr := s.fileSystem.OpenFile(lockPath, os.O_RDWR, filePermission)
		if openErr != nil {
			return nil, newError("open lock", ErrIO, openErr)
		}
		return lock, nil
	}

	lock, err := s.fileSystem.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, filePermission)
	if errors.Is(err, fs.ErrExist) {
		exists, validateErr := s.validateRegularFile(lockPath, false)
		if validateErr != nil {
			return nil, validateErr
		}
		if !exists {
			return nil, newError("open lock", ErrIO, err)
		}
		lock, err = s.fileSystem.OpenFile(lockPath, os.O_RDWR, filePermission)
	}
	if err != nil {
		return nil, newError("open lock", ErrIO, err)
	}
	if err := lock.Chmod(filePermission); err != nil {
		closeErr := lock.Close()
		return nil, &Error{
			Operation: "prepare lock",
			Kind:      ErrIO,
			Err:       err,
			Cleanup:   closeErr,
		}
	}
	return lock, nil
}

func (s *Store) writeConfiguration(newPath string) (bool, error, error) {
	data, err := encodeSchema(newPath)
	if err != nil {
		return false, err, nil
	}
	tempName, primaryErr, cleanupErr := s.prepareTemporaryFile(data)
	if primaryErr != nil || cleanupErr != nil {
		cleanupErr = errors.Join(cleanupErr, s.removeTemporaryFile(tempName))
		return false, primaryErr, cleanupErr
	}
	if renameErr := s.fileSystem.Rename(tempName, s.configPath); renameErr != nil {
		cleanupErr = s.removeTemporaryFile(tempName)
		return false, newError("replace", ErrIO, renameErr), cleanupErr
	}
	dirErr, dirCleanupErr := s.syncConfigDirectory()
	return true, dirErr, dirCleanupErr
}

func (s *Store) prepareTemporaryFile(data []byte) (string, error, error) {
	temp, err := s.fileSystem.CreateTemp(s.configDirectory, ".config-*.tmp")
	if err != nil {
		return "", newError("create temporary file", ErrIO, err), nil
	}
	tempName := temp.Name()
	var primaryErr error
	var cleanupErr error

	if err := temp.Chmod(filePermission); err != nil {
		primaryErr = newError("prepare temporary file", ErrIO, err)
	} else if written, writeErr := temp.Write(data); writeErr != nil {
		primaryErr = newError("write temporary file", ErrIO, writeErr)
	} else if written != len(data) {
		primaryErr = newError("write temporary file", ErrIO, io.ErrShortWrite)
	} else if syncErr := temp.Sync(); syncErr != nil {
		primaryErr = newError("sync temporary file", ErrIO, syncErr)
	}

	if closeErr := temp.Close(); closeErr != nil {
		cleanupErr = errors.Join(cleanupErr, closeErr)
	}
	return tempName, primaryErr, cleanupErr
}

func (s *Store) removeTemporaryFile(path string) error {
	if path == "" {
		return nil
	}
	if err := s.fileSystem.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("一時ファイル削除: %w", err)
	}
	return nil
}

func (s *Store) syncConfigDirectory() (error, error) {
	directory, err := s.fileSystem.OpenDir(s.configDirectory)
	if err != nil {
		return newError("open directory", ErrIO, err), nil
	}
	var primaryErr error
	if err := directory.Sync(); err != nil {
		primaryErr = newError("sync directory", ErrIO, err)
	}
	if err := directory.Close(); err != nil {
		return primaryErr, fmt.Errorf("設定ディレクトリクローズ: %w", err)
	}
	return primaryErr, nil
}

func pathComponents(path string) []string {
	cleaned := filepath.Clean(path)
	var reversed []string
	for current := cleaned; ; current = filepath.Dir(current) {
		reversed = append(reversed, current)
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	paths := make([]string, len(reversed))
	for index := range reversed {
		paths[len(reversed)-1-index] = reversed[index]
	}
	return paths
}
