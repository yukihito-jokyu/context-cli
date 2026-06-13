package e2e_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yukihito-jokyu/context-cli/pkg/cmd"
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
			input:   "n\n",
			wantErr: cmd.ErrRepositoryChangeCanceled,
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
