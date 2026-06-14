package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

//nolint:gocognit,cyclop // テーブル駆動テストのため、認知・循環複雑度の上限を無視します。
func TestFactory(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "NewFactoryProvidesAddDependencies",
			run: func(t *testing.T) {
				t.Helper()
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
				if factory.SyncPlanner == nil {
					t.Error("SyncPlanner = nil")
				}
				prompt, ok := factory.Prompt(factory.IOIn, factory.IOOut).(*huhPrompt)
				if !ok {
					t.Fatalf("Prompt type = %T, want *huhPrompt", factory.Prompt)
				}
				if prompt.input != os.Stdin || prompt.output != os.Stdout {
					t.Error("Prompt does not use Factory default input/output")
				}
			},
		},
		{
			name: "FactoryAddDependenciesFollowReplacedIO",
			run: func(t *testing.T) {
				t.Helper()
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
			},
		},
		{
			name: "FactoryConfigPersistsAcrossInstances",
			run: func(t *testing.T) {
				t.Helper()
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}
