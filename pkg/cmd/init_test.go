package cmd

import (
	"bytes"
	"testing"
)

// mockConfig はテスト用の Config モック実装です。
type mockConfig struct {
	repoPath string
}

func (m *mockConfig) GetContextRepository() string {
	return m.repoPath
}

func (m *mockConfig) SetContextRepository(path string) error {
	m.repoPath = path
	return nil
}

func TestInitOptions_Run(t *testing.T) {
	tests := []struct {
		name           string
		repoPath       string
		expectedOutput string
		wantErr        bool
	}{
		{
			name:           "success initialization",
			repoPath:       "/path/to/my-context-repo",
			expectedOutput: "Successfully initialized context repository at: /path/to/my-context-repo\n",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			cfg := &mockConfig{}
			f := &Factory{
				IOOut: buf,
				Config: func() (Config, error) {
					return cfg, nil
				},
			}

			opts := &InitOptions{
				Factory:  f,
				RepoPath: tt.repoPath,
			}

			err := opts.Run()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if buf.String() != tt.expectedOutput {
					t.Errorf("Run() output = %q, expected %q", buf.String(), tt.expectedOutput)
				}
				if cfg.GetContextRepository() != tt.repoPath {
					t.Errorf("config repo path = %q, expected %q", cfg.GetContextRepository(), tt.repoPath)
				}
			}
		})
	}
}
