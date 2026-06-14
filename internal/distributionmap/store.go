package distributionmap

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"syscall"

	"github.com/yukihito-jokyu/context-cli/internal/distribution"
	"golang.org/x/sys/unix"
)

const (
	configDirectoryName = "context"
	mapFileName         = "map.yaml"
	lockFileName        = "map.lock"
	directoryPermission = 0o700
	filePermission      = 0o600
)

// Environment は設定保存先の探索に必要な環境情報を表します。
type Environment interface {
	LookupEnv(key string) (string, bool)
	UserHomeDir() (string, error)
}

// Store はmap.yamlの安全な読込と比較更新を提供します。
type Store struct {
	configDirectory string
	mapPath         string
	lockPath        string
}

// New は環境からmap.yamlの保存先を解決したStoreを返します。
func New(environment Environment) (*Store, error) {
	configDirectory, err := discoverConfigDirectory(environment)
	if err != nil {
		return nil, err
	}
	return &Store{
		configDirectory: configDirectory,
		mapPath:         filepath.Join(configDirectory, mapFileName),
		lockPath:        filepath.Join(configDirectory, lockFileName),
	}, nil
}

// Load は現在の管理情報と正規化リビジョンを返します。
func (s *Store) Load() (distribution.MapSnapshot, error) {
	exists, err := validateDirectoryChain(s.configDirectory, true)
	if err != nil || !exists {
		if err != nil {
			return distribution.MapSnapshot{}, err
		}
		return emptySnapshot(), nil
	}
	exists, err = validateRegularFile(s.mapPath, true)
	if err != nil || !exists {
		if err != nil {
			return distribution.MapSnapshot{}, err
		}
		return emptySnapshot(), nil
	}
	data, err := os.ReadFile(s.mapPath) // #nosec G304 -- 検証済みの固定map.yamlだけを読みます。
	if err != nil {
		return distribution.MapSnapshot{}, newError("read", ErrIO, err)
	}
	records, err := decode(data)
	if err != nil {
		return distribution.MapSnapshot{}, err
	}
	normalized, err := encode(records)
	if err != nil {
		return distribution.MapSnapshot{}, err
	}
	return distribution.MapSnapshot{
		Revision:   revision(normalized),
		Workspaces: records,
	}, nil
}

// Begin は待機なしでmap.lockを取得し、期待リビジョンを比較します。
func (s *Store) Begin(expected distribution.Revision) (distribution.MapTransaction, distribution.MapSnapshot, error) {
	if err := s.ensureConfigDirectory(); err != nil {
		return nil, distribution.MapSnapshot{}, err
	}
	lock, err := s.openLock()
	if err != nil {
		return nil, distribution.MapSnapshot{}, err
	}
	// #nosec G115 -- Unixシステムにおいてファイル記述子は十分にintの範囲に収まり、オーバーフローのリスクがないため警告を抑制します。
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = lock.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, distribution.MapSnapshot{}, newError("lock", ErrLock, err)
		}
		return nil, distribution.MapSnapshot{}, newError("lock", ErrIO, err)
	}
	snapshot, err := s.Load()
	if err != nil {
		// #nosec G115 -- Unixシステムにおいてファイル記述子は十分にintの範囲に収まり、オーバーフローのリスクがないため警告を抑制します。
		_ = unix.Flock(int(lock.Fd()), unix.LOCK_UN)
		_ = lock.Close()
		return nil, distribution.MapSnapshot{}, err
	}
	if snapshot.Revision != expected {
		// #nosec G115 -- Unixシステムにおいてファイル記述子は十分にintの範囲に収まり、オーバーフローのリスクがないため警告を抑制します。
		_ = unix.Flock(int(lock.Fd()), unix.LOCK_UN)
		_ = lock.Close()
		return nil, distribution.MapSnapshot{}, newError("compare", ErrConflict, nil)
	}
	return &transaction{store: s, lock: lock, snapshot: snapshot}, snapshot, nil
}

type transaction struct {
	store    *Store
	lock     *os.File
	snapshot distribution.MapSnapshot
}

//nolint:gocognit // renameの前後でコミット状態と後処理失敗を分けて保持します。
func (t *transaction) Commit(workspace distribution.WorkspaceRecord) (distribution.CommitResult, error) {
	records := make(map[string]distribution.WorkspaceRecord, len(t.snapshot.Workspaces)+1)
	maps.Copy(records, t.snapshot.Workspaces)
	if len(workspace.Skills) == 0 {
		delete(records, workspace.WorkspaceRoot)
	} else {
		records[workspace.WorkspaceRoot] = workspace
	}
	data, err := encode(records)
	if err != nil {
		return distribution.CommitResult{}, err
	}
	temp, err := os.CreateTemp(t.store.configDirectory, ".map-*.tmp")
	if err != nil {
		return distribution.CommitResult{}, newError("create temporary file", ErrIO, err)
	}
	tempName := temp.Name()
	committed := false
	var primaryErr error
	var cleanupErr error
	if err := temp.Chmod(filePermission); err != nil {
		primaryErr = newError("chmod temporary file", ErrIO, err)
	} else if _, err := temp.Write(data); err != nil {
		primaryErr = newError("write temporary file", ErrIO, err)
	} else if err := temp.Sync(); err != nil {
		primaryErr = newError("sync temporary file", ErrIO, err)
	}
	if err := temp.Close(); err != nil {
		cleanupErr = errors.Join(cleanupErr, err)
	}
	if primaryErr == nil && cleanupErr == nil {
		if err := os.Rename(tempName, t.store.mapPath); err != nil {
			primaryErr = newError("replace", ErrIO, err)
		} else {
			committed = true
		}
	}
	if !committed {
		if err := os.Remove(tempName); err != nil && !errors.Is(err, fs.ErrNotExist) {
			cleanupErr = errors.Join(cleanupErr, err)
		}
		return distribution.CommitResult{}, errors.Join(primaryErr, cleanupErr)
	}
	if err := syncDirectory(t.store.configDirectory); err != nil {
		return distribution.CommitResult{Committed: true}, &Error{
			Operation: "sync directory",
			Kind:      ErrCommitted,
			Err:       err,
		}
	}
	return distribution.CommitResult{Committed: true}, nil
}

func (t *transaction) Close() error {
	var closeErr error
	// #nosec G115 -- Unixシステムにおいてファイル記述子は十分にintの範囲に収まり、オーバーフローのリスクがないため警告を抑制します。
	if err := unix.Flock(int(t.lock.Fd()), unix.LOCK_UN); err != nil {
		closeErr = errors.Join(closeErr, fmt.Errorf("ロック解放: %w", err))
	}
	if err := t.lock.Close(); err != nil {
		closeErr = errors.Join(closeErr, fmt.Errorf("ロックファイルクローズ: %w", err))
	}
	return closeErr
}

func (s *Store) ensureConfigDirectory() error {
	exists, err := validateDirectoryChain(s.configDirectory, true)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	parent := filepath.Dir(s.configDirectory)
	parentExists, err := validateDirectoryChain(parent, false)
	if err != nil {
		return err
	}
	if !parentExists {
		return newError("create directory", ErrDiscovery, nil)
	}
	if err := os.Mkdir(s.configDirectory, directoryPermission); err != nil {
		return newError("create directory", ErrIO, err)
	}
	info, err := os.Lstat(s.configDirectory)
	if err != nil || !info.IsDir() || info.Mode().Perm() != directoryPermission {
		_ = os.Remove(s.configDirectory)
		return newError("verify directory", ErrPermission, err)
	}
	return nil
}

func (s *Store) openLock() (*os.File, error) {
	exists, err := validateRegularFile(s.lockPath, true)
	if err != nil {
		return nil, err
	}
	flags := os.O_RDWR
	if !exists {
		flags |= os.O_CREATE
	}
	lock, err := os.OpenFile(s.lockPath, flags, filePermission) // #nosec G304 -- 検証済み設定ディレクトリの固定ロックファイルです。
	if err != nil {
		return nil, newError("open lock", ErrIO, err)
	}
	if err := lock.Chmod(filePermission); err != nil {
		_ = lock.Close()
		return nil, newError("chmod lock", ErrIO, err)
	}
	return lock, nil
}

func discoverConfigDirectory(environment Environment) (string, error) {
	if xdg, ok := environment.LookupEnv("XDG_CONFIG_HOME"); ok && xdg != "" {
		if !filepath.IsAbs(xdg) {
			return "", newError("discover", ErrDiscovery, nil)
		}
		return filepath.Join(filepath.Clean(xdg), configDirectoryName), nil
	}
	home, err := environment.UserHomeDir()
	if err != nil || !filepath.IsAbs(home) {
		return "", newError("discover", ErrDiscovery, err)
	}
	return filepath.Join(filepath.Clean(home), ".config", configDirectoryName), nil
}

func validateDirectoryChain(path string, validateFinalPermission bool) (bool, error) {
	for _, component := range pathComponents(path) {
		info, err := os.Lstat(component)
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
		if validateFinalPermission && component == path && info.Mode().Perm()&0o077 != 0 {
			return false, newError("inspect", ErrPermission, nil)
		}
	}
	return true, nil
}

func validateRegularFile(path string, allowMissing bool) (bool, error) {
	info, err := os.Lstat(path)
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

func emptySnapshot() distribution.MapSnapshot {
	return distribution.MapSnapshot{
		Revision:   distribution.EmptyRevision,
		Workspaces: map[string]distribution.WorkspaceRecord{},
	}
}

func revision(data []byte) distribution.Revision {
	sum := sha256.Sum256(data)
	return distribution.Revision(hex.EncodeToString(sum[:]))
}

func syncDirectory(path string) error {
	directory, err := os.Open(path) // #nosec G304 -- 検証済み設定ディレクトリだけを開きます。
	if err != nil {
		return fmt.Errorf("設定ディレクトリオープン: %w", err)
	}
	var syncErr error
	if err := directory.Sync(); err != nil {
		syncErr = err
	}
	if err := directory.Close(); err != nil {
		syncErr = errors.Join(syncErr, err)
	}
	return syncErr
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
	result := make([]string, len(reversed))
	for index := range reversed {
		result[len(reversed)-1-index] = reversed[index]
	}
	return result
}
