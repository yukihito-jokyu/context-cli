package e2e_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/yukihito-jokyu/context-cli/pkg/cmd"
)

type harness struct {
	factory *cmd.Factory
	config  *memoryConfig
	output  *bytes.Buffer
	errOut  *bytes.Buffer
}

func newHarness(initialRepository, input string) *harness {
	output := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	config := &memoryConfig{repository: initialRepository}
	factory := cmd.NewFactory()
	factory.IOOut = output
	factory.IOErr = errOut
	factory.IOIn = bytes.NewBufferString(input)
	factory.Config = func() (cmd.Config, error) {
		return config, nil
	}
	return &harness{factory: factory, config: config, output: output, errOut: errOut}
}

func (h *harness) execute(args ...string) error {
	root := cmd.NewCmdRoot(h.factory)
	root.SetArgs(args)
	root.SetOut(h.output)
	root.SetErr(h.factory.IOErr)
	root.SilenceUsage = true
	root.SilenceErrors = true
	if err := root.Execute(); err != nil {
		return fmt.Errorf("CLI execution failed: %w", err)
	}
	return nil
}

type memoryConfig struct {
	repository string
	setCalls   int
}

func (c *memoryConfig) GetContextRepository() string {
	return c.repository
}

func (c *memoryConfig) SetContextRepository(path string) error {
	c.setCalls++
	c.repository = path
	return nil
}

func createRepositoryFixture(t *testing.T) string {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("一時ディレクトリの実体パスを取得できません: %v", err)
	}
	repository := filepath.Join(base, "context")
	for _, path := range []string{
		filepath.Join(repository, "projects"),
		filepath.Join(repository, "utils", "skills"),
	} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatalf("fixtureを作成できません: %v", err)
		}
	}
	return repository
}

func changeWorkingDirectory(t *testing.T, path string) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("作業ディレクトリを取得できません: %v", err)
	}
	if err := os.Chdir(path); err != nil {
		t.Fatalf("作業ディレクトリを変更できません: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("作業ディレクトリを復元できません: %v", err)
		}
	})
}
