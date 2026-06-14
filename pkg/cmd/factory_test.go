package cmd

import (
	"path/filepath"
	"testing"
)

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
