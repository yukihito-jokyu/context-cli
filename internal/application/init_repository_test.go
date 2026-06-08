package application_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/yukihito-jokyu/context-cli/internal/application"
	"github.com/yukihito-jokyu/context-cli/internal/domain"
)

// errInterrupted はI/O中断を表すテスト用のセンチネルエラーです。
var errInterrupted = errors.New("interrupted")

// --- モック定義 ---

// mockConfigRepository は ConfigRepository のモック実装です。
type mockConfigRepository struct {
	loadFn func(ctx context.Context) (domain.Config, error)
	saveFn func(ctx context.Context, config domain.Config, expectedOld *domain.Config) error
}

func (m *mockConfigRepository) Load(ctx context.Context) (domain.Config, error) {
	return m.loadFn(ctx)
}

func (m *mockConfigRepository) Save(ctx context.Context, config domain.Config, expectedOld *domain.Config) error {
	return m.saveFn(ctx, config, expectedOld)
}

// mockUIPort は UIPort のモック実装です。
type mockUIPort struct {
	confirmFn func(ctx context.Context, currentPath, newPath string) (bool, error)
}

func (m *mockUIPort) ConfirmChange(ctx context.Context, currentPath, newPath string) (bool, error) {
	return m.confirmFn(ctx, currentPath, newPath)
}

// mockFileSystem は domain.FileSystem のモック実装です。
type mockFileSystem struct {
	lstatFn    func(ctx context.Context, path string) (domain.FileStatus, error)
	readDirFn  func(ctx context.Context, path string) ([]domain.FileEntry, error)
	readFileFn func(ctx context.Context, path string) ([]byte, error)
}

func (m *mockFileSystem) LStat(ctx context.Context, path string) (domain.FileStatus, error) {
	return m.lstatFn(ctx, path)
}

func (m *mockFileSystem) ReadDir(ctx context.Context, path string) ([]domain.FileEntry, error) {
	return m.readDirFn(ctx, path)
}

func (m *mockFileSystem) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return m.readFileFn(ctx, path)
}

// --- テストヘルパー ---

// validFileStatus は検証成功に使う正常なファイルステータスです。
type validFileStatus struct {
	isDir     bool
	isRegular bool
	isSymlink bool
	mode      fs.FileMode
}

func (s *validFileStatus) IsDir() bool {
	return s.isDir
}

func (s *validFileStatus) IsRegular() bool {
	return s.isRegular
}

func (s *validFileStatus) IsSymlink() bool {
	return s.isSymlink
}

func (s *validFileStatus) Mode() fs.FileMode {
	return s.mode
}

// validFileEntry は ReadDir から返すディレクトリエントリです。
type validFileEntry struct {
	name  string
	isDir bool
}

func (e *validFileEntry) Name() string {
	return e.name
}

func (e *validFileEntry) IsDir() bool {
	return e.isDir
}

// newPassingFileSystem は全ての検証を通過する FileSystem モックを返します。
func newPassingFileSystem() *mockFileSystem {
	return &mockFileSystem{
		lstatFn: func(_ context.Context, path string) (domain.FileStatus, error) {
			if len(path) > 9 && path[len(path)-8:] == "SKILL.md" {
				return &validFileStatus{isDir: false, isRegular: true, mode: 0o644 & ^fs.FileMode(0o022)}, nil
			}
			return &validFileStatus{isDir: true, mode: 0o755 & ^fs.FileMode(0o022)}, nil
		},
		readDirFn: func(_ context.Context, path string) ([]domain.FileEntry, error) {
			if len(path) >= 8 && path[len(path)-8:] == "projects" {
				return []domain.FileEntry{
					&validFileEntry{name: "proj1", isDir: true},
				}, nil
			}
			if len(path) >= 6 && path[len(path)-6:] == "skills" {
				return []domain.FileEntry{}, nil
			}
			return []domain.FileEntry{
				&validFileEntry{name: "projects", isDir: true},
				&validFileEntry{name: "utils", isDir: true},
			}, nil
		},
		readFileFn: func(_ context.Context, _ string) ([]byte, error) {
			return []byte("# SKILL"), nil
		},
	}
}

// newFailingFileSystem はリポジトリ検証が失敗する FileSystem モックを返します。
func newFailingFileSystem() *mockFileSystem {
	return &mockFileSystem{
		lstatFn: func(_ context.Context, _ string) (domain.FileStatus, error) {
			return &validFileStatus{isDir: true, mode: 0o700}, nil
		},
		readDirFn: func(_ context.Context, _ string) ([]domain.FileEntry, error) {
			return []domain.FileEntry{}, nil
		},
		readFileFn: func(_ context.Context, _ string) ([]byte, error) {
			return nil, fs.ErrNotExist
		},
	}
}

// noopConfirmFn は何もしない ConfirmChange 関数です。
func noopConfirmFn(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

// --- 共通テスト実行ヘルパー ---

// useCaseDeps はユースケースのテスト依存をまとめます。
type useCaseDeps struct {
	repo      *mockConfigRepository
	ui        *mockUIPort
	fsys      *mockFileSystem
	inputPath string
}

func runUseCase(t *testing.T, deps useCaseDeps) error {
	t.Helper()

	uc := application.NewInitRepositoryUseCase(deps.repo, deps.ui, deps.fsys)
	//nolint:wrapcheck // テストヘルパーからの直接返却
	return uc.Run(context.Background(), deps.inputPath)
}

func assertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("予期しないエラーが発生しました: %v", err)
	}
}

func assertError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("エラーが発生すると予想されましたが、発生しませんでした")
	}
}

func assertErrorIs(t *testing.T, err, target error) {
	t.Helper()

	assertError(t, err)

	if !errors.Is(err, target) {
		t.Errorf("想定されたエラー: %v, 実際のエラー: %v", target, err)
	}
}

func assertValidationError(t *testing.T, err error) {
	t.Helper()

	assertError(t, err)

	var valErr *application.RepositoryValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("RepositoryValidationError として判定できるエラーを期待しましたが: %v", err)
	}
}

// --- TT-013: 初回設定テスト ---

func TestInitRepository_FirstTimeSetup(t *testing.T) {
	t.Run("初回設定 - config.yaml未存在時、プロンプトなしで永続化が呼ばれる", func(t *testing.T) {
		saveCalled := false
		confirmCalled := false
		var savedConfig domain.Config
		var savedExpectedOld *domain.Config

		repo := &mockConfigRepository{
			loadFn: func(_ context.Context) (domain.Config, error) {
				return domain.Config{}, fs.ErrNotExist
			},
			saveFn: func(_ context.Context, config domain.Config, expectedOld *domain.Config) error {
				saveCalled = true
				savedConfig = config
				savedExpectedOld = expectedOld
				return nil
			},
		}
		ui := &mockUIPort{
			confirmFn: func(_ context.Context, _, _ string) (bool, error) {
				confirmCalled = true
				return false, nil
			},
		}

		err := runUseCase(t, useCaseDeps{
			repo: repo, ui: ui, fsys: newPassingFileSystem(), inputPath: "/valid/repo/path",
		})
		assertNoError(t, err)

		if !saveCalled {
			t.Error("Save が呼び出されませんでした")
		}
		if confirmCalled {
			t.Error("初回設定時に ConfirmChange が呼び出されました")
		}
		if savedExpectedOld != nil {
			t.Error("初回設定時の expectedOld は nil であるべきです")
		}
		if savedConfig.RepositoryPath != "/valid/repo/path" {
			t.Errorf("保存された RepositoryPath = %q, 想定: %q", savedConfig.RepositoryPath, "/valid/repo/path")
		}
		if savedConfig.Version != domain.CurrentConfigVersion {
			t.Errorf("保存された Version = %d, 想定: %d", savedConfig.Version, domain.CurrentConfigVersion)
		}
	})
}

// --- TT-014: 同一リポジトリ再実行テスト ---

func TestInitRepository_SameRepositoryReRun(t *testing.T) {
	existingConfig := domain.Config{
		Version:        domain.CurrentConfigVersion,
		RepositoryPath: "/same/repo/path",
	}

	t.Run("検証成功時、プロンプトとファイル書き込みがスキップされ正常終了", func(t *testing.T) {
		repo := &mockConfigRepository{
			loadFn: func(_ context.Context) (domain.Config, error) {
				return existingConfig, nil
			},
			saveFn: func(_ context.Context, _ domain.Config, _ *domain.Config) error {
				t.Error("同一リポジトリ再実行時に Save が呼び出されました")
				return nil
			},
		}
		ui := &mockUIPort{
			confirmFn: func(_ context.Context, _, _ string) (bool, error) {
				t.Error("同一リポジトリ再実行時に ConfirmChange が呼び出されました")
				return false, nil
			},
		}

		err := runUseCase(t, useCaseDeps{
			repo: repo, ui: ui, fsys: newPassingFileSystem(), inputPath: "/same/repo/path",
		})
		assertNoError(t, err)
	})

	t.Run("検証失敗時、エラー終了し既存設定は維持", func(t *testing.T) {
		repo := &mockConfigRepository{
			loadFn: func(_ context.Context) (domain.Config, error) {
				return existingConfig, nil
			},
			saveFn: func(_ context.Context, _ domain.Config, _ *domain.Config) error {
				t.Error("検証失敗時に Save が呼び出されました")
				return nil
			},
		}
		ui := &mockUIPort{
			confirmFn: func(_ context.Context, _, _ string) (bool, error) {
				t.Error("検証失敗時に ConfirmChange が呼び出されました")
				return false, nil
			},
		}

		err := runUseCase(t, useCaseDeps{
			repo: repo, ui: ui, fsys: newFailingFileSystem(), inputPath: "/same/repo/path",
		})
		assertValidationError(t, err)
	})
}

// --- TT-015: 異なるリポジトリへの設定変更テスト ---

func TestInitRepository_ChangeRepository(t *testing.T) {
	oldConfig := domain.Config{
		Version:        domain.CurrentConfigVersion,
		RepositoryPath: "/old/repo/path",
	}

	t.Run("承認された場合、設定が書き換えられる", func(t *testing.T) {
		saveCalled := false

		repo := &mockConfigRepository{
			loadFn: func(_ context.Context) (domain.Config, error) {
				return oldConfig, nil
			},
			saveFn: func(_ context.Context, config domain.Config, expectedOld *domain.Config) error {
				saveCalled = true
				if config.RepositoryPath != "/new/repo/path" {
					t.Errorf("保存された RepositoryPath = %q, 想定: %q", config.RepositoryPath, "/new/repo/path")
				}
				if expectedOld == nil {
					t.Error("expectedOld は nil であってはなりません")
				} else if expectedOld.RepositoryPath != "/old/repo/path" {
					t.Errorf("expectedOld.RepositoryPath = %q, 想定: %q", expectedOld.RepositoryPath, "/old/repo/path")
				}
				return nil
			},
		}
		ui := &mockUIPort{
			confirmFn: func(_ context.Context, currentPath, newPath string) (bool, error) {
				if currentPath != "/old/repo/path" {
					t.Errorf("ConfirmChange の currentPath = %q, 想定: %q", currentPath, "/old/repo/path")
				}
				if newPath != "/new/repo/path" {
					t.Errorf("ConfirmChange の newPath = %q, 想定: %q", newPath, "/new/repo/path")
				}
				return true, nil
			},
		}

		err := runUseCase(t, useCaseDeps{
			repo: repo, ui: ui, fsys: newPassingFileSystem(), inputPath: "/new/repo/path",
		})
		assertNoError(t, err)

		if !saveCalled {
			t.Error("Save が呼び出されませんでした")
		}
	})

	t.Run("拒否された場合、書き込みスキップで既存設定維持", func(t *testing.T) {
		repo := &mockConfigRepository{
			loadFn: func(_ context.Context) (domain.Config, error) {
				return oldConfig, nil
			},
			saveFn: func(_ context.Context, _ domain.Config, _ *domain.Config) error {
				t.Error("拒否時に Save が呼び出されました")
				return nil
			},
		}
		ui := &mockUIPort{
			confirmFn: func(_ context.Context, _, _ string) (bool, error) {
				return false, nil
			},
		}

		err := runUseCase(t, useCaseDeps{
			repo: repo, ui: ui, fsys: newPassingFileSystem(), inputPath: "/new/repo/path",
		})
		assertErrorIs(t, err, application.ErrChangeAborted)
	})

	t.Run("確認が中断された場合、書き込みスキップで既存設定維持", func(t *testing.T) {
		repo := &mockConfigRepository{
			loadFn: func(_ context.Context) (domain.Config, error) {
				return oldConfig, nil
			},
			saveFn: func(_ context.Context, _ domain.Config, _ *domain.Config) error {
				t.Error("中断時に Save が呼び出されました")
				return nil
			},
		}
		ui := &mockUIPort{
			confirmFn: func(_ context.Context, _, _ string) (bool, error) {
				return false, errInterrupted
			},
		}

		err := runUseCase(t, useCaseDeps{
			repo: repo, ui: ui, fsys: newPassingFileSystem(), inputPath: "/new/repo/path",
		})
		assertError(t, err)
	})
}

// --- TT-015-2: 検証失敗時に書き込みが行われないことの検証 ---

func TestInitRepository_ValidationFailurePreventsWrite(t *testing.T) {
	failSaveFn := func(t *testing.T) func(context.Context, domain.Config, *domain.Config) error {
		t.Helper()
		return func(_ context.Context, _ domain.Config, _ *domain.Config) error {
			t.Error("検証失敗時に Save が呼び出されました")
			return nil
		}
	}
	failConfirmFn := func(t *testing.T) func(context.Context, string, string) (bool, error) {
		t.Helper()
		return func(_ context.Context, _, _ string) (bool, error) {
			t.Error("検証失敗時に ConfirmChange が呼び出されました")
			return false, nil
		}
	}

	t.Run("リポジトリ検証失敗 - 初回設定時に書き込みが呼ばれない", func(t *testing.T) {
		err := runUseCase(t, useCaseDeps{
			repo: &mockConfigRepository{
				loadFn: func(_ context.Context) (domain.Config, error) {
					return domain.Config{}, fs.ErrNotExist
				},
				saveFn: failSaveFn(t),
			},
			ui:        &mockUIPort{confirmFn: failConfirmFn(t)},
			fsys:      newFailingFileSystem(),
			inputPath: "/invalid/repo",
		})
		assertValidationError(t, err)
	})

	t.Run("リポジトリ検証失敗 - 設定変更時に書き込みが呼ばれず既存設定が維持", func(t *testing.T) {
		err := runUseCase(t, useCaseDeps{
			repo: &mockConfigRepository{
				loadFn: func(_ context.Context) (domain.Config, error) {
					return domain.Config{
						Version:        domain.CurrentConfigVersion,
						RepositoryPath: "/existing/repo",
					}, nil
				},
				saveFn: failSaveFn(t),
			},
			ui:        &mockUIPort{confirmFn: failConfirmFn(t)},
			fsys:      newFailingFileSystem(),
			inputPath: "/invalid/repo",
		})
		assertValidationError(t, err)
	})

	t.Run("スキーマ検証失敗 - 未対応バージョンの既存設定でエラー終了", func(t *testing.T) {
		err := runUseCase(t, useCaseDeps{
			repo: &mockConfigRepository{
				loadFn: func(_ context.Context) (domain.Config, error) {
					return domain.Config{}, domain.ErrUnsupportedConfigVersion
				},
				saveFn: failSaveFn(t),
			},
			ui:        &mockUIPort{confirmFn: noopConfirmFn},
			fsys:      newPassingFileSystem(),
			inputPath: "/some/repo/path",
		})
		assertError(t, err)
	})

	t.Run("スキーマ検証失敗 - 不正な設定内容でエラー終了", func(t *testing.T) {
		err := runUseCase(t, useCaseDeps{
			repo: &mockConfigRepository{
				loadFn: func(_ context.Context) (domain.Config, error) {
					return domain.Config{}, domain.ErrInvalidConfig
				},
				saveFn: failSaveFn(t),
			},
			ui:        &mockUIPort{confirmFn: noopConfirmFn},
			fsys:      newPassingFileSystem(),
			inputPath: "/some/repo/path",
		})
		assertError(t, err)
	})
}
