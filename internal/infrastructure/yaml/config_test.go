package yaml_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/yukihito-jokyu/context-cli/internal/application"
	"github.com/yukihito-jokyu/context-cli/internal/domain"
	infra "github.com/yukihito-jokyu/context-cli/internal/infrastructure/yaml"
)

var errAnyFormat = errors.New("format error")

type resolveConfigDirTestCase struct {
	name       string
	envSetup   func(t *testing.T)
	wantSuffix string
	wantPath   string
}

// TestResolveConfigDir は ResolveConfigDir の動作を検証するテーブル駆動テストです。
func TestResolveConfigDir(t *testing.T) {
	tests := []resolveConfigDirTestCase{
		{
			name: "XDG_CONFIG_HOME が設定されている場合",
			envSetup: func(t *testing.T) {
				t.Helper()
				t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")
			},
			wantPath: "/custom/xdg/context",
		},
		{
			name: "XDG_CONFIG_HOME が未設定（空）の場合",
			envSetup: func(t *testing.T) {
				t.Helper()
				t.Setenv("XDG_CONFIG_HOME", "")
			},
			wantSuffix: filepath.Join(".config", "context"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runResolveConfigDirTestCase(t, tt)
		})
	}
}

func runResolveConfigDirTestCase(t *testing.T, tt resolveConfigDirTestCase) {
	t.Helper()
	tt.envSetup(t)
	path, err := infra.ResolveConfigDir()
	if err != nil {
		t.Fatalf("予期しないエラーが発生しました: %v", err)
	}
	if tt.wantPath != "" && path != tt.wantPath {
		t.Errorf("想定されたパス: %q, 実際: %q", tt.wantPath, path)
	}
	if tt.wantSuffix != "" && !strings.HasSuffix(path, tt.wantSuffix) {
		t.Errorf("想定されたサフィックス: %q, 実際のパス: %q", tt.wantSuffix, path)
	}
}

type loadTestCase struct {
	name             string
	configDir        string // 空の場合は t.TempDir() から生成したディレクトリを使用
	writeParentFile  bool
	writeDir         bool
	dirPerm          fs.FileMode
	writeConfigAsDir bool
	writeFile        bool
	filePerm         fs.FileMode
	fileContent      string
	wantConfig       domain.Config
	wantErr          error
}

// TestConfigRepository_Load は Load メソッドの動作を検証するテーブル駆動テストです。
func TestConfigRepository_Load(t *testing.T) {
	tests := []loadTestCase{
		{
			name:        "正常系 - 有効な設定ファイルの読み込み",
			writeDir:    true,
			dirPerm:     0700,
			writeFile:   true,
			filePerm:    0600,
			fileContent: "version: 1\nrepository_path: /absolute/path\n",
			wantConfig: domain.Config{
				Version:        1,
				RepositoryPath: "/absolute/path",
			},
			wantErr: nil,
		},
		{
			name:     "異常系 - 設定ファイルが存在しない場合",
			writeDir: true,
			dirPerm:  0700,
			wantErr:  fs.ErrNotExist,
		},
		{
			name:    "異常系 - ディレクトリが存在しない場合",
			wantErr: fs.ErrNotExist,
		},
		{
			name:            "異常系 - 設定ディレクトリ自体がファイルの場合",
			writeParentFile: true,
			wantErr:         infra.ErrNotDirectory,
		},
		{
			name:     "異常系 - 設定ディレクトリのパーミッションが広すぎる場合",
			writeDir: true,
			dirPerm:  0755,
			wantErr:  application.ErrPermissionTooBroad,
		},
		{
			name:             "異常系 - 設定ファイル自体がディレクトリの場合",
			writeDir:         true,
			dirPerm:          0700,
			writeConfigAsDir: true,
			wantErr:          infra.ErrNotRegularFile,
		},
		{
			name:        "異常系 - 設定ファイルのパーミッションが広すぎる場合",
			writeDir:    true,
			dirPerm:     0700,
			writeFile:   true,
			filePerm:    0644,
			fileContent: "version: 1\nrepository_path: /absolute/path\n",
			wantErr:     application.ErrPermissionTooBroad,
		},
		{
			name:        "異常系 - 設定ファイルの内容が不正なYAML形式の場合",
			writeDir:    true,
			dirPerm:     0700,
			writeFile:   true,
			filePerm:    0600,
			fileContent: "version: [invalid yaml",
			wantErr:     errAnyFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runLoadTestCase(t, tt)
		})
	}
}

func runLoadTestCase(t *testing.T, tt loadTestCase) {
	t.Helper()
	tmpDir := t.TempDir()
	configDir := tt.configDir
	if configDir == "" {
		configDir = filepath.Join(tmpDir, "context")
	}

	prepareLoadFileSystem(t, configDir, tt)

	repo := infra.NewConfigRepository(configDir)
	cfg, err := repo.Load(context.Background())

	assertLoadResult(t, cfg, err, tt)
}

func prepareLoadFileSystem(t *testing.T, configDir string, tt loadTestCase) {
	t.Helper()
	if tt.writeParentFile {
		if err := os.WriteFile(configDir, []byte("plain file"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	if tt.writeDir {
		// #nosec G301
		if err := os.MkdirAll(configDir, tt.dirPerm); err != nil {
			t.Fatal(err)
		}
	}

	configPath := filepath.Join(configDir, "config.yaml")

	if tt.writeConfigAsDir {
		if err := os.MkdirAll(configPath, 0700); err != nil {
			t.Fatal(err)
		}
	}

	if tt.writeFile {
		// #nosec G306
		if err := os.WriteFile(configPath, []byte(tt.fileContent), tt.filePerm); err != nil {
			t.Fatal(err)
		}
	}
}

func assertLoadResult(t *testing.T, cfg domain.Config, err error, tt loadTestCase) {
	t.Helper()
	if tt.wantErr != nil {
		if err == nil {
			t.Fatal("エラーが発生すると予想されましたが、発生しませんでした")
		}
		if errors.Is(tt.wantErr, errAnyFormat) {
			return
		}
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("想定されたエラー: %v, 実際のエラー: %v", tt.wantErr, err)
		}
		return
	}

	if err != nil {
		t.Fatalf("Load に失敗しました: %v", err)
	}

	if cfg.Version != tt.wantConfig.Version {
		t.Errorf("Version = %d, 想定: %d", cfg.Version, tt.wantConfig.Version)
	}
	if cfg.RepositoryPath != tt.wantConfig.RepositoryPath {
		t.Errorf("RepositoryPath = %q, 想定: %q", cfg.RepositoryPath, tt.wantConfig.RepositoryPath)
	}
}

type saveTestCase struct {
	name              string
	parentPathIsFile  bool
	writeDirAsFile    bool
	writeDir          bool
	dirPerm           fs.FileMode
	writeFile         bool
	filePerm          fs.FileMode
	fileContent       string
	writeConfigAsDir  bool
	writeTempAsDir    bool
	writeTempSymlink  bool
	acquireLockBefore bool
	config            domain.Config
	expectedOld       *domain.Config
	wantErr           error
	verify            func(t *testing.T, configDir string)
}

// TestConfigRepository_Save は Save メソッドの動作を検証するテーブル駆動テストです。
//
//nolint:gocognit // Table-driven cases cover the complete persistence failure matrix.
func TestConfigRepository_Save(t *testing.T) {
	tests := []saveTestCase{
		{
			name: "正常系 - 新規設定ファイルの作成と適切なパーミッション付与",
			config: domain.Config{
				Version:        1,
				RepositoryPath: "/absolute/path",
			},
			expectedOld: nil,
			wantErr:     nil,
			verify: func(t *testing.T, configDir string) {
				t.Helper()
				dirInfo, err := os.Stat(configDir)
				if err != nil {
					t.Fatalf("設定ディレクトリのステータス取得に失敗しました: %v", err)
				}
				if dirInfo.Mode().Perm() != 0700 {
					t.Errorf("想定されたディレクトリパーミッション 0700, 実際: %o", dirInfo.Mode().Perm())
				}

				configPath := filepath.Join(configDir, "config.yaml")
				fileInfo, err := os.Stat(configPath)
				if err != nil {
					t.Fatalf("設定ファイルのステータス取得に失敗しました: %v", err)
				}
				if fileInfo.Mode().Perm() != 0600 {
					t.Errorf("想定されたファイルパーミッション 0600, 実際: %o", fileInfo.Mode().Perm())
				}

				tmpPath := filepath.Join(configDir, "config.yaml.tmp")
				if _, err := os.Stat(tmpPath); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("一時ファイル config.yaml.tmp が存在しないことを想定しましたが、存在しています (エラー: %v)", err)
				}
			},
		},
		{
			name:        "正常系 - 楽観的ロック成功（既存設定と期待値が一致）",
			writeDir:    true,
			dirPerm:     0700,
			writeFile:   true,
			filePerm:    0600,
			fileContent: "version: 1\nrepository_path: /expected/old/path\n",
			config: domain.Config{
				Version:        1,
				RepositoryPath: "/new/path",
			},
			expectedOld: &domain.Config{
				Version:        1,
				RepositoryPath: "/expected/old/path",
			},
			wantErr: nil,
			verify: func(t *testing.T, configDir string) {
				t.Helper()
				configPath := filepath.Join(configDir, "config.yaml")
				// #nosec G304
				data, err := os.ReadFile(configPath)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(data), "/new/path") {
					t.Errorf("設定ファイルに新しいパスが反映されていません: %s", string(data))
				}
			},
		},
		{
			name:           "異常系 - 設定ディレクトリ自体がファイルの場合",
			writeDirAsFile: true,
			config: domain.Config{
				Version:        1,
				RepositoryPath: "/absolute/path",
			},
			expectedOld: nil,
			wantErr:     infra.ErrNotDirectory,
		},
		{
			name:     "異常系 - 設定ディレクトリのパーミッションが広すぎる場合",
			writeDir: true,
			dirPerm:  0755,
			config: domain.Config{
				Version:        1,
				RepositoryPath: "/absolute/path",
			},
			expectedOld: nil,
			wantErr:     application.ErrPermissionTooBroad,
		},
		{
			name:             "異常系 - 設定ファイルがディレクトリの場合",
			writeDir:         true,
			dirPerm:          0700,
			writeConfigAsDir: true,
			config: domain.Config{
				Version:        1,
				RepositoryPath: "/absolute/path",
			},
			expectedOld: nil,
			wantErr:     infra.ErrNotRegularFile,
		},
		{
			name:        "異常系 - 設定ファイルのパーミッションが広すぎる場合",
			writeDir:    true,
			dirPerm:     0700,
			writeFile:   true,
			filePerm:    0644,
			fileContent: "version: 1\nrepository_path: /absolute/path\n",
			config: domain.Config{
				Version:        2,
				RepositoryPath: "/absolute/path",
			},
			expectedOld: nil,
			wantErr:     application.ErrPermissionTooBroad,
		},
		{
			name:              "異常系 - 他のプロセスによるロックの取得に失敗した場合",
			writeDir:          true,
			dirPerm:           0700,
			acquireLockBefore: true,
			config: domain.Config{
				Version:        1,
				RepositoryPath: "/absolute/path",
			},
			expectedOld: nil,
			wantErr:     application.ErrLockFailed,
		},
		{
			name:        "異常系 - 楽観的ロック競合（期待値はnilだが既にファイルが存在する）",
			writeDir:    true,
			dirPerm:     0700,
			writeFile:   true,
			filePerm:    0600,
			fileContent: "version: 1\nrepository_path: /absolute/path\n",
			config: domain.Config{
				Version:        1,
				RepositoryPath: "/absolute/path2",
			},
			expectedOld: nil,
			wantErr:     application.ErrConfigConflict,
		},
		{
			name:     "異常系 - 楽観的ロック競合（期待値は存在するがファイルが存在しない）",
			writeDir: true,
			dirPerm:  0700,
			config: domain.Config{
				Version:        1,
				RepositoryPath: "/absolute/path",
			},
			expectedOld: &domain.Config{
				Version:        1,
				RepositoryPath: "/old/path",
			},
			wantErr: application.ErrConfigConflict,
		},
		{
			name:        "異常系 - 楽観的ロック競合（既存設定の値が期待値と異なる）",
			writeDir:    true,
			dirPerm:     0700,
			writeFile:   true,
			filePerm:    0600,
			fileContent: "version: 1\nrepository_path: /different/path\n",
			config: domain.Config{
				Version:        1,
				RepositoryPath: "/new/path",
			},
			expectedOld: &domain.Config{
				Version:        1,
				RepositoryPath: "/expected/old/path",
			},
			wantErr: application.ErrConfigConflict,
		},
		{
			name:        "異常系 - 楽観的ロック競合（既存設定ファイルが不正なYAML形式の場合）",
			writeDir:    true,
			dirPerm:     0700,
			writeFile:   true,
			filePerm:    0600,
			fileContent: "invalid: [yaml",
			config: domain.Config{
				Version:        1,
				RepositoryPath: "/new/path",
			},
			expectedOld: &domain.Config{
				Version:        1,
				RepositoryPath: "/expected/old/path",
			},
			wantErr: application.ErrConfigConflict,
		},
		{
			name:             "異常系 - 設定ディレクトリの作成に失敗した場合（親パスがファイル）",
			parentPathIsFile: true,
			config: domain.Config{
				Version:        1,
				RepositoryPath: "/absolute/path",
			},
			expectedOld: nil,
			wantErr:     errAnyFormat,
		},
		{
			name:           "正常系 - 固定名の一時ファイルが存在しても使用しない",
			writeDir:       true,
			dirPerm:        0700,
			writeTempAsDir: true,
			config: domain.Config{
				Version:        1,
				RepositoryPath: "/absolute/path",
			},
			expectedOld: nil,
			wantErr:     nil,
		},
		{
			name:             "正常系 - 固定名の一時ファイルsymlinkを追跡しない",
			writeDir:         true,
			dirPerm:          0700,
			writeTempSymlink: true,
			config: domain.Config{
				Version:        1,
				RepositoryPath: "/absolute/path",
			},
			expectedOld: nil,
			wantErr:     nil,
			verify: func(t *testing.T, configDir string) {
				t.Helper()
				targetPath := filepath.Join(configDir, "symlink-target")
				data, err := os.ReadFile(targetPath) // #nosec G304 -- Test path is created under t.TempDir.
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != "do not overwrite" {
					t.Errorf("symlink target was overwritten: %q", data)
				}
			},
		},
		{
			name:     "異常系 - 保存する設定値が不正",
			writeDir: true,
			dirPerm:  0700,
			config: domain.Config{
				Version:        2,
				RepositoryPath: "/absolute/path",
			},
			expectedOld: nil,
			wantErr:     domain.ErrUnsupportedConfigVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runSaveTestCase(t, tt)
		})
	}
}

func runSaveTestCase(t *testing.T, tt saveTestCase) {
	t.Helper()
	tmpDir := t.TempDir()
	configDir := resolveSaveConfigDir(t, tmpDir, tt)

	prepareSaveDir(t, configDir, tt)
	prepareSaveFiles(t, configDir, tt)
	prepareSaveLock(t, configDir, tt)

	repo := infra.NewConfigRepository(configDir)
	err := repo.Save(context.Background(), tt.config, tt.expectedOld)

	assertSaveResult(t, err, configDir, tt)
}

func resolveSaveConfigDir(t *testing.T, tmpDir string, tt saveTestCase) string {
	t.Helper()
	configDir := filepath.Join(tmpDir, "context")
	if tt.parentPathIsFile {
		parentFile := filepath.Join(tmpDir, "parent_is_file")
		if err := os.WriteFile(parentFile, []byte("regular file"), 0600); err != nil {
			t.Fatal(err)
		}
		configDir = filepath.Join(parentFile, "context")
	}
	return configDir
}

func prepareSaveDir(t *testing.T, configDir string, tt saveTestCase) {
	t.Helper()
	if tt.writeDirAsFile {
		if err := os.WriteFile(configDir, []byte("plain file"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	if tt.writeDir {
		// #nosec G301
		if err := os.MkdirAll(configDir, tt.dirPerm); err != nil {
			t.Fatal(err)
		}
	}
}

func prepareSaveFiles(t *testing.T, configDir string, tt saveTestCase) {
	t.Helper()
	configPath := filepath.Join(configDir, "config.yaml")

	if tt.writeConfigAsDir {
		if err := os.MkdirAll(configPath, 0700); err != nil {
			t.Fatal(err)
		}
	}

	if tt.writeFile {
		// #nosec G306
		if err := os.WriteFile(configPath, []byte(tt.fileContent), tt.filePerm); err != nil {
			t.Fatal(err)
		}
	}

	if tt.writeTempAsDir {
		tmpPath := filepath.Join(configDir, "config.yaml.tmp")
		if err := os.MkdirAll(tmpPath, 0700); err != nil {
			t.Fatal(err)
		}
	}

	if tt.writeTempSymlink {
		targetPath := filepath.Join(configDir, "symlink-target")
		if err := os.WriteFile(targetPath, []byte("do not overwrite"), 0600); err != nil {
			t.Fatal(err)
		}
		tmpPath := filepath.Join(configDir, "config.yaml.tmp")
		if err := os.Symlink(targetPath, tmpPath); err != nil {
			t.Fatal(err)
		}
	}
}

func prepareSaveLock(t *testing.T, configDir string, tt saveTestCase) {
	t.Helper()
	if tt.acquireLockBefore {
		lockPath := filepath.Join(configDir, "config.yaml.lock")
		// #nosec G304
		lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			t.Fatal(err)
		}
		// #nosec G115
		fd := int(lockFile.Fd())
		err = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
		if err != nil {
			t.Fatalf("テストでの排他ロック取得に失敗しました: %v", err)
		}
		t.Cleanup(func() {
			_ = syscall.Flock(fd, syscall.LOCK_UN)
			_ = lockFile.Close()
		})
	}
}

func assertSaveResult(t *testing.T, err error, configDir string, tt saveTestCase) {
	t.Helper()
	if tt.wantErr != nil {
		if err == nil {
			t.Fatal("エラーが発生すると予想されましたが、発生しませんでした")
		}
		if errors.Is(tt.wantErr, errAnyFormat) {
			return
		}
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("想定されたエラー: %v, 実際のエラー: %v", tt.wantErr, err)
		}
		return
	}

	if err != nil {
		t.Fatalf("Save に失敗しました: %v", err)
	}

	if tt.verify != nil {
		tt.verify(t, configDir)
	}
}
