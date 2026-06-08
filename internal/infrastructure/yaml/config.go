// Package yaml は、YAMLファイルを使用した設定リポジトリの永続化を実装します。
package yaml

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"github.com/yukihito-jokyu/context-cli/internal/application"
	"github.com/yukihito-jokyu/context-cli/internal/domain"
	"go.yaml.in/yaml/v3"
)

const (
	dirPerm  fs.FileMode = 0700
	filePerm fs.FileMode = 0600
)

var (
	// ErrNotDirectory は、解決されたパスは存在するがディレクトリではないことを示します。
	ErrNotDirectory = errors.New("config path is not a directory")

	// ErrNotRegularFile は、解決された設定パスは存在するが通常のファイルではないことを示します。
	ErrNotRegularFile = errors.New("config file is not a regular file")
)

// ConfigRepository は application.ConfigRepository を実装します。
type ConfigRepository struct {
	configDir string
}

// NewConfigRepository は新しい ConfigRepository を作成します。
// テスト用にカスタム設定ディレクトリを渡すことができます。
func NewConfigRepository(configDir string) *ConfigRepository {
	return &ConfigRepository{configDir: configDir}
}

// ResolveConfigDir はデフォルトの設定ディレクトリを解決します。
// 最初に XDG_CONFIG_HOME をチェックし、設定されていなければ ~/.config/context にフォールバックします。
func ResolveConfigDir() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "context"), nil
}

// checkPermissions は、設定ディレクトリとファイルのパーミッションが
// 広すぎないことを検証します。
func checkPermissions(dir, configPath string) error {
	// ディレクトリのパーミッションをチェック
	dirInfo, err := os.Lstat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // ディレクトリがまだ存在しない場合は問題ありません。
		}
		return fmt.Errorf("failed to lstat directory: %w", err)
	}
	if !dirInfo.IsDir() {
		return fmt.Errorf("%w: %s", ErrNotDirectory, dir)
	}
	// グループまたは他のユーザーに権限がある場合はパーミッションが広すぎます
	if dirInfo.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("directory perms %o: %w", dirInfo.Mode().Perm(), application.ErrPermissionTooBroad)
	}

	// ファイルのパーミッションをチェック
	fileInfo, err := os.Lstat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // ファイルがまだ存在しない場合は問題ありません。
		}
		return fmt.Errorf("failed to lstat file: %w", err)
	}
	if !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("%w: %s", ErrNotRegularFile, configPath)
	}
	// グループまたは他のユーザーに権限があるか、所有者の実行ビットが立っている場合はパーミッションが広すぎます
	if fileInfo.Mode().Perm()&0177 != 0 {
		return fmt.Errorf("file perms %o: %w", fileInfo.Mode().Perm(), application.ErrPermissionTooBroad)
	}
	return nil
}

// Load は設定ディレクトリから設定をロードします。
func (r *ConfigRepository) Load(ctx context.Context) (domain.Config, error) {
	if err := ctx.Err(); err != nil {
		return domain.Config{}, fmt.Errorf("load config context: %w", err)
	}

	configPath := filepath.Join(r.configDir, "config.yaml")

	if err := checkPermissions(r.configDir, configPath); err != nil {
		return domain.Config{}, fmt.Errorf("permission check failed: %w", err)
	}

	data, err := os.ReadFile(configPath) // #nosec G304
	if err != nil {
		return domain.Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	config, err := domain.ParseConfig(data)
	if err != nil {
		return domain.Config{}, fmt.Errorf("failed to parse config: %w", err)
	}

	return config, nil
}

// Save は設定ディレクトリに設定を保存します。
func (r *ConfigRepository) Save(ctx context.Context, config domain.Config, expectedOld *domain.Config) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("save config context: %w", err)
	}

	configPath := filepath.Join(r.configDir, "config.yaml")

	if err := checkPermissions(r.configDir, configPath); err != nil {
		return fmt.Errorf("initial permission check failed: %w", err)
	}
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config for save: %w", err)
	}

	if err := r.ensureDirExists(); err != nil {
		return fmt.Errorf("failed to ensure config directory exists: %w", err)
	}

	lockFile, err := r.acquireLock()
	if err != nil {
		return err
	}
	defer func() {
		_ = lockFile.Close()
	}()

	// ロック後にパーミッションを再チェック
	if err := checkPermissions(r.configDir, configPath); err != nil {
		return fmt.Errorf("post-lock permission check failed: %w", err)
	}

	if err := r.verifyExpectedConfig(configPath, expectedOld); err != nil {
		return fmt.Errorf("optimistic lock check failed: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("save config context after conflict check: %w", err)
	}

	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := r.writeConfigAtomically(configPath, yamlData); err != nil {
		return fmt.Errorf("failed to write config atomically: %w", err)
	}

	return nil
}

func (r *ConfigRepository) ensureDirExists() error {
	if _, err := os.Stat(r.configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(r.configDir, dirPerm); err != nil {
			return fmt.Errorf("failed to mkdir: %w", err)
		}
		// #nosec G302
		if err := os.Chmod(r.configDir, dirPerm); err != nil {
			return fmt.Errorf("failed to chmod directory: %w", err)
		}
	}
	return nil
}

func (r *ConfigRepository) acquireLock() (*os.File, error) {
	lockPath := filepath.Join(r.configDir, "config.yaml.lock")
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, filePerm) // #nosec G304
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	// #nosec G115
	fd := int(lockFile.Fd())
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = lockFile.Close()
		return nil, fmt.Errorf("%w: %w", application.ErrLockFailed, err)
	}

	return lockFile, nil
}

func (r *ConfigRepository) verifyExpectedConfig(configPath string, expectedOld *domain.Config) error {
	if expectedOld == nil {
		if _, err := os.Stat(configPath); err == nil {
			return application.ErrConfigConflict
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat config file: %w", err)
		}
	} else {
		data, err := os.ReadFile(configPath) // #nosec G304
		if err != nil {
			if os.IsNotExist(err) {
				return application.ErrConfigConflict
			}
			return fmt.Errorf("failed to read old config: %w", err)
		}
		oldConfig, err := domain.ParseConfig(data)
		if err != nil {
			return fmt.Errorf("failed to parse old config: %w (original error: %w)", application.ErrConfigConflict, err)
		}
		if oldConfig.Version != expectedOld.Version || oldConfig.RepositoryPath != expectedOld.RepositoryPath {
			return application.ErrConfigConflict
		}
	}
	return nil
}

func (r *ConfigRepository) writeConfigAtomically(configPath string, yamlData []byte) error {
	tmpFile, err := os.CreateTemp(r.configDir, "config.yaml.tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create tmp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}()

	if err := tmpFile.Chmod(filePerm); err != nil {
		return fmt.Errorf("failed to chmod tmp file: %w", err)
	}

	if _, err := tmpFile.Write(yamlData); err != nil {
		return fmt.Errorf("failed to write tmp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync tmp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close tmp file: %w", err)
	}

	if err := os.Rename(tmpPath, configPath); err != nil {
		return fmt.Errorf("failed to rename tmp file: %w", err)
	}

	return nil
}
