package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestNewFactoryProvidesAddDependencies(t *testing.T) {
	factory := NewFactory()

	if factory.IsTerminal == nil {
		t.Error("IsTerminal = nil")
	}
	if factory.WorkspaceValidator == nil {
		t.Error("WorkspaceValidator = nil")
	}
	if factory.SkillCatalog == nil {
		t.Error("SkillCatalog = nil")
	}
	if factory.Prompt == nil {
		t.Error("Prompt = nil")
	}
	prompt, ok := factory.Prompt(factory.IOIn, factory.IOOut).(*huhPrompt)
	if !ok {
		t.Fatalf("Prompt type = %T, want *huhPrompt", factory.Prompt)
	}
	if prompt.input != os.Stdin || prompt.output != os.Stdout {
		t.Error("Prompt does not use Factory default input/output")
	}
}

func TestFactoryAddDependenciesFollowReplacedIO(t *testing.T) {
	factory := NewFactory()
	input := bytes.NewBufferString("input")
	output := &bytes.Buffer{}
	factory.IOIn = input
	factory.IOOut = output

	if factory.IsTerminal(factory.IOIn, factory.IOOut) {
		t.Fatal("IsTerminal() = true for buffers")
	}
	prompt, ok := factory.Prompt(factory.IOIn, factory.IOOut).(*huhPrompt)
	if !ok {
		t.Fatalf("Prompt type = %T, want *huhPrompt", factory.Prompt(factory.IOIn, factory.IOOut))
	}
	if prompt.input != input || prompt.output != output {
		t.Fatal("Prompt does not follow replaced Factory IO")
	}
}

func TestFactoryConfigPersistsAcrossInstances(t *testing.T) {
	configHome, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("一時設定ディレクトリの実体パスを取得できません: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", configHome)

	first, err := NewFactory().Config()
	if err != nil {
		t.Fatalf("1つ目のFactory.Config() error = %v", err)
	}
	if err := first.SetContextRepository("", "/tmp/context-repository"); err != nil {
		t.Fatalf("SetContextRepository() error = %v", err)
	}

	second, err := NewFactory().Config()
	if err != nil {
		t.Fatalf("2つ目のFactory.Config() error = %v", err)
	}
	if got := second.GetContextRepository(); got != "/tmp/context-repository" {
		t.Errorf("GetContextRepository() = %q, want /tmp/context-repository", got)
	}
}
