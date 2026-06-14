package e2e_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

type initTestCase struct {
	name            string
	prepare         func(t *testing.T) (argument string, repository string, sensitivePaths []string)
	initialRepo     func(t *testing.T, repository string) string
	input           string
	wantErr         error
	wantErrContains string
	wantRepository  func(repository, initialRepository string) string
	wantSetCalls    int
	wantOutput      func(repository string) string
}

type initTestResult struct {
	repositoryPath    string
	initialRepository string
}

func TestE2E_Init(t *testing.T) {
	tests := []initTestCase{
		{
			name: "INIT-001",
			prepare: func(t *testing.T) (string, string, []string) {
				t.Helper()
				repositoryPath := createRepositoryFixture(t)
				changeWorkingDirectory(t, filepath.Dir(repositoryPath))
				return filepath.Base(repositoryPath), repositoryPath, nil
			},
			wantRepository: func(repository, _ string) string {
				return repository
			},
			wantSetCalls: 1,
			wantOutput: func(repository string) string {
				return fmt.Sprintf("Successfully initialized context repository at: %s\n", repository)
			},
		},
		{
			name: "INIT-002",
			prepare: func(t *testing.T) (string, string, []string) {
				t.Helper()
				repositoryPath := createRepositoryFixture(t)
				if err := os.RemoveAll(filepath.Join(repositoryPath, "utils", "skills")); err != nil {
					t.Fatalf("fixtureを変更できません: %v", err)
				}
				return repositoryPath, repositoryPath, []string{repositoryPath}
			},
			wantErrContains: "required structure",
		},
		{
			name: "INIT-003",
			prepare: func(t *testing.T) (string, string, []string) {
				t.Helper()
				repositoryPath := createRepositoryFixture(t)
				linkPath := filepath.Join(filepath.Dir(repositoryPath), "context-link")
				if err := os.Symlink(repositoryPath, linkPath); err != nil {
					t.Fatalf("シンボリックリンクを作成できません: %v", err)
				}
				return linkPath, repositoryPath, []string{linkPath, repositoryPath}
			},
			wantErrContains: "symbolic link",
		},
		{
			name: "INIT-004",
			prepare: func(t *testing.T) (string, string, []string) {
				t.Helper()
				repositoryPath := createRepositoryFixture(t)
				projectsPath := filepath.Join(repositoryPath, "projects")
				if err := os.Remove(projectsPath); err != nil {
					t.Fatalf("projectsを削除できません: %v", err)
				}
				projectsTarget := filepath.Join(filepath.Dir(repositoryPath), "projects-target")
				if err := os.Mkdir(projectsTarget, 0o700); err != nil {
					t.Fatalf("projects targetを作成できません: %v", err)
				}
				if err := os.Symlink(projectsTarget, projectsPath); err != nil {
					t.Fatalf("projectsリンクを作成できません: %v", err)
				}
				return repositoryPath, repositoryPath, []string{repositoryPath, projectsPath, projectsTarget}
			},
			wantErrContains: "symbolic link",
		},
		{
			name: "INIT-005",
			prepare: func(t *testing.T) (string, string, []string) {
				t.Helper()
				repositoryPath := createRepositoryFixture(t)
				return repositoryPath, repositoryPath, nil
			},
			initialRepo: func(t *testing.T, _ string) string {
				t.Helper()
				return createRepositoryFixture(t)
			},
			input: "y\n",
			wantRepository: func(repository, _ string) string {
				return repository
			},
			wantSetCalls: 1,
			wantOutput: func(repository string) string {
				return "Current context repository: {{initial}}\n" +
					fmt.Sprintf("New context repository: %s\n", repository) +
					"変更しますか? [y/N] " +
					fmt.Sprintf("Successfully initialized context repository at: %s\n", repository)
			},
		},
		{
			name: "INIT-006",
			prepare: func(t *testing.T) (string, string, []string) {
				t.Helper()
				repositoryPath := createRepositoryFixture(t)
				return repositoryPath, repositoryPath, nil
			},
			initialRepo: func(t *testing.T, _ string) string {
				t.Helper()
				return createRepositoryFixture(t)
			},
			input: "n\n",
			wantRepository: func(_, initialRepository string) string {
				return initialRepository
			},
			wantOutput: func(repository string) string {
				return "Current context repository: {{initial}}\n" +
					fmt.Sprintf("New context repository: %s\n", repository) +
					"変更しますか? [y/N] "
			},
		},
		{
			name: "INIT-007",
			prepare: func(t *testing.T) (string, string, []string) {
				t.Helper()
				repositoryPath := createRepositoryFixture(t)
				return repositoryPath, repositoryPath, nil
			},
			initialRepo: func(t *testing.T, repository string) string {
				t.Helper()
				return repository
			},
			wantRepository: func(repository, _ string) string {
				return repository
			},
			wantOutput: func(repository string) string {
				return fmt.Sprintf("Successfully initialized context repository at: %s\n", repository)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runInitTestCase(t, tt)
		})
	}
}

func runInitTestCase(t *testing.T, tt initTestCase) {
	t.Helper()
	argument, repositoryPath, sensitivePaths := tt.prepare(t)
	initialRepository := ""
	if tt.initialRepo != nil {
		initialRepository = tt.initialRepo(t, repositoryPath)
	}
	h := newHarness(initialRepository, tt.input)

	err := h.execute("init", "--repo", argument)
	assertInitError(t, err, tt.wantErr, tt.wantErrContains)
	assertInitResult(t, h, initTestResult{
		repositoryPath:    repositoryPath,
		initialRepository: initialRepository,
	}, tt)
	assertSensitivePathsHidden(t, h, err, sensitivePaths)
}

func assertInitError(t *testing.T, err, wantErr error, wantErrContains string) {
	t.Helper()
	if wantErr == nil && wantErrContains == "" {
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		return
	}
	if wantErr != nil && !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want errors.Is(_, %v)", err, wantErr)
	}
	if wantErrContains != "" && (err == nil || !strings.Contains(err.Error(), wantErrContains)) {
		t.Fatalf("Execute() error = %v, want containing %q", err, wantErrContains)
	}
}

func assertInitResult(
	t *testing.T,
	h *harness,
	result initTestResult,
	tt initTestCase,
) {
	t.Helper()
	wantRepository := result.initialRepository
	if tt.wantRepository != nil {
		wantRepository = tt.wantRepository(result.repositoryPath, result.initialRepository)
	}
	if h.config.repository != wantRepository {
		t.Fatalf("saved repository = %q, want %q", h.config.repository, wantRepository)
	}
	if h.config.setCalls != tt.wantSetCalls {
		t.Fatalf("SetContextRepository calls = %d, want %d", h.config.setCalls, tt.wantSetCalls)
	}
	wantOutput := ""
	if tt.wantOutput != nil {
		wantOutput = strings.ReplaceAll(
			tt.wantOutput(result.repositoryPath),
			"{{initial}}",
			result.initialRepository,
		)
	}
	if h.output.String() != wantOutput {
		t.Fatalf("output = %q, want %q", h.output.String(), wantOutput)
	}
	if h.errOut.String() != "" {
		t.Fatalf("error output = %q, want empty", h.errOut.String())
	}
}

func assertSensitivePathsHidden(t *testing.T, h *harness, err error, sensitivePaths []string) {
	t.Helper()
	if err == nil {
		return
	}
	for _, path := range sensitivePaths {
		for outputName, output := range map[string]string{
			"error":  err.Error(),
			"stdout": h.output.String(),
			"stderr": h.errOut.String(),
		} {
			if strings.Contains(output, path) {
				t.Fatalf("%s leaks path %q: %q", outputName, path, output)
			}
		}
	}
}

type persistedConfig struct {
	SchemaVersion     int    `yaml:"schema_version"`
	ContextRepository string `yaml:"context_repository"`
}

type persistenceScenario struct {
	name       string
	input      string
	wantOutput func(currentRepository, newRepository string) string
	wantPath   func(currentRepository, newRepository string) string
}

func TestE2E_InitPersistence(t *testing.T) {
	scenarios := []persistenceScenario{
		{
			name:  "PERSIST-001-same-path",
			input: "",
			wantOutput: func(currentRepository, _ string) string {
				return fmt.Sprintf(
					"Successfully initialized context repository at: %s\n",
					currentRepository,
				)
			},
			wantPath: func(currentRepository, _ string) string {
				return currentRepository
			},
		},
		{
			name:  "PERSIST-002-approve-change",
			input: "y\n",
			wantOutput: func(currentRepository, newRepository string) string {
				return repositoryChangeOutput(currentRepository, newRepository) +
					fmt.Sprintf(
						"Successfully initialized context repository at: %s\n",
						newRepository,
					)
			},
			wantPath: func(_, newRepository string) string {
				return newRepository
			},
		},
		{
			name:       "PERSIST-003-reject-change",
			input:      "n\n",
			wantOutput: repositoryChangeOutput,
			wantPath: func(currentRepository, _ string) string {
				return currentRepository
			},
		},
		{
			name:       "PERSIST-004-eof",
			input:      "",
			wantOutput: repositoryChangeOutput,
			wantPath: func(currentRepository, _ string) string {
				return currentRepository
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			runPersistenceScenario(t, scenario)
		})
	}
}

func runPersistenceScenario(t *testing.T, scenario persistenceScenario) {
	t.Helper()
	base := realTemporaryDirectory(t)
	xdgConfigHome := filepath.Join(base, "xdg")
	currentRepository := createRepositoryFixtureAt(t, base, "current")
	newRepository := currentRepository
	if scenario.name != "PERSIST-001-same-path" {
		newRepository = createRepositoryFixtureAt(t, base, "new")
	}

	first := runContextProcess(t, processRequest{
		xdgConfigHome:    xdgConfigHome,
		workingDirectory: base,
		args:             []string{"init", "--repo", currentRepository},
	})
	assertProcessResult(t, first, processResult{
		exitCode: 0,
		stdout: fmt.Sprintf(
			"Successfully initialized context repository at: %s\n",
			currentRepository,
		),
	})

	configPath := filepath.Join(xdgConfigHome, "context", "config.yaml")
	assertPersistedConfig(t, configPath, currentRepository)
	beforeData, beforeInfo := readConfigSnapshot(t, configPath)
	time.Sleep(10 * time.Millisecond)

	second := runContextProcess(t, processRequest{
		xdgConfigHome:    xdgConfigHome,
		workingDirectory: base,
		stdin:            scenario.input,
		args:             []string{"init", "--repo", newRepository},
	})
	assertProcessResult(t, second, processResult{
		exitCode: 0,
		stdout:   scenario.wantOutput(currentRepository, newRepository),
	})

	wantPath := scenario.wantPath(currentRepository, newRepository)
	assertPersistedConfig(t, configPath, wantPath)
	if wantPath == currentRepository {
		assertConfigUnchanged(t, configPath, beforeData, beforeInfo)
	}
}

func TestE2E_InitPersistence_InvalidRepository(t *testing.T) {
	base := realTemporaryDirectory(t)
	xdgConfigHome := filepath.Join(base, "xdg")
	invalidRepository := filepath.Join(base, "invalid")
	if err := os.Mkdir(invalidRepository, 0o700); err != nil {
		t.Fatalf("無効Repository fixtureを作成できません: %v", err)
	}

	result := runContextProcess(t, processRequest{
		xdgConfigHome:    xdgConfigHome,
		workingDirectory: base,
		args:             []string{"init", "--repo", invalidRepository},
	})
	wantError := "Error: failed to validate context repository: " +
		"context repository required structure is missing (projects)\n"
	assertProcessResult(t, result, processResult{
		exitCode: 1,
		stderr:   wantError,
	})
	if strings.Count(result.stderr, "Error:") != 1 {
		t.Fatalf("stderrのError出現回数 = %d, want 1: %q", strings.Count(result.stderr, "Error:"), result.stderr)
	}
	if strings.Contains(result.stderr, invalidRepository) {
		t.Fatalf("stderrにRepositoryパスが含まれています: %q", result.stderr)
	}
	if _, err := os.Stat(filepath.Join(xdgConfigHome, "context")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("無効Repositoryで設定ディレクトリが作成されました: %v", err)
	}
}

func TestE2E_CommandErrorBoundary(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{
			name:       "unknown-command",
			args:       []string{"unknown-command"},
			wantStderr: "Error: unknown command \"unknown-command\" for \"context\"\n",
		},
		{
			name:       "unknown-flag",
			args:       []string{"init", "--unknown"},
			wantStderr: "Error: unknown flag: --unknown\n",
		},
		{
			name:       "required-repository",
			args:       []string{"init"},
			wantStderr: "Error: --repo flag is required\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := realTemporaryDirectory(t)
			xdgConfigHome := filepath.Join(base, "xdg")
			result := runContextProcess(t, processRequest{
				xdgConfigHome:    xdgConfigHome,
				workingDirectory: base,
				args:             tt.args,
			})
			assertProcessResult(t, result, processResult{
				exitCode: 1,
				stderr:   tt.wantStderr,
			})
			if strings.Count(result.stderr, "Error:") != 1 {
				t.Fatalf(
					"stderrのError出現回数 = %d, want 1: %q",
					strings.Count(result.stderr, "Error:"),
					result.stderr,
				)
			}
			if _, err := os.Stat(filepath.Join(xdgConfigHome, "context")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("CLI入力エラーで設定ディレクトリが作成されました: %v", err)
			}
		})
	}
}

func repositoryChangeOutput(currentRepository, newRepository string) string {
	return fmt.Sprintf(
		"Current context repository: %s\nNew context repository: %s\n変更しますか? [y/N] ",
		currentRepository,
		newRepository,
	)
}

func realTemporaryDirectory(t *testing.T) string {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("一時ディレクトリの実体パスを取得できません: %v", err)
	}
	return base
}

func createRepositoryFixtureAt(t *testing.T, base, name string) string {
	t.Helper()
	repository := filepath.Join(base, name)
	for _, path := range []string{
		filepath.Join(repository, "projects"),
		filepath.Join(repository, "utils", "skills"),
	} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatalf("Repository fixtureを作成できません: %v", err)
		}
	}
	return repository
}

func assertProcessResult(t *testing.T, got, want processResult) {
	t.Helper()
	if got.exitCode != want.exitCode {
		t.Errorf("exit code = %d, want %d", got.exitCode, want.exitCode)
	}
	if got.stdout != want.stdout {
		t.Errorf("stdout = %q, want %q", got.stdout, want.stdout)
	}
	if got.stderr != want.stderr {
		t.Errorf("stderr = %q, want %q", got.stderr, want.stderr)
	}
}

func assertPersistedConfig(t *testing.T, configPath, wantRepository string) {
	t.Helper()
	data, err := os.ReadFile(configPath) //nolint:gosec // テスト専用一時ディレクトリ配下の固定ファイルを読む。
	if err != nil {
		t.Fatalf("設定ファイルを読み込めません: %v", err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var config persistedConfig
	if err := decoder.Decode(&config); err != nil {
		t.Fatalf("設定YAMLをデコードできません: %v", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		t.Fatalf("設定YAMLに複数文書または末尾データがあります: %v", err)
	}
	if config.SchemaVersion != 1 {
		t.Errorf("schema_version = %d, want 1", config.SchemaVersion)
	}
	if config.ContextRepository != wantRepository {
		t.Errorf("context_repository = %q, want %q", config.ContextRepository, wantRepository)
	}
	if !filepath.IsAbs(config.ContextRepository) {
		t.Errorf("context_repositoryが絶対パスではありません: %q", config.ContextRepository)
	}
}

func readConfigSnapshot(t *testing.T, configPath string) ([]byte, os.FileInfo) {
	t.Helper()
	data, err := os.ReadFile(configPath) //nolint:gosec // テスト専用一時ディレクトリ配下の固定ファイルを読む。
	if err != nil {
		t.Fatalf("設定ファイルを読み込めません: %v", err)
	}
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("設定ファイルの情報を取得できません: %v", err)
	}
	return data, info
}

func assertConfigUnchanged(
	t *testing.T,
	configPath string,
	beforeData []byte,
	beforeInfo os.FileInfo,
) {
	t.Helper()
	afterData, afterInfo := readConfigSnapshot(t, configPath)
	if !bytes.Equal(afterData, beforeData) {
		t.Errorf("設定ファイルの内容が変更されました")
	}
	if !afterInfo.ModTime().Equal(beforeInfo.ModTime()) {
		t.Errorf(
			"設定ファイルの更新日時が変更されました: before=%s after=%s",
			beforeInfo.ModTime(),
			afterInfo.ModTime(),
		)
	}
	if !os.SameFile(beforeInfo, afterInfo) {
		t.Errorf("設定ファイルが置換されました")
	}
}
