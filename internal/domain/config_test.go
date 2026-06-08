package domain

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    string
		want    Config
		wantErr error
	}{
		{
			name: "valid",
			data: "version: 1\nrepository_path: /work/context\n",
			want: Config{
				Version:        CurrentConfigVersion,
				RepositoryPath: "/work/context",
			},
		},
		{
			name:    "unknown field",
			data:    "version: 1\nrepository_path: /work/context\nunexpected: true\n",
			wantErr: ErrInvalidConfig,
		},
		{
			name:    "malformed yaml",
			data:    "version: [\n",
			wantErr: ErrInvalidConfig,
		},
		{
			name:    "multiple documents",
			data:    "version: 1\nrepository_path: /work/context\n---\nversion: 1\nrepository_path: /work/other\n",
			wantErr: ErrInvalidConfig,
		},
		{
			name:    "missing repository path",
			data:    "version: 1\n",
			wantErr: ErrInvalidConfig,
		},
		{
			name:    "relative repository path",
			data:    "version: 1\nrepository_path: relative/context\n",
			wantErr: ErrInvalidConfig,
		},
		{
			name:    "unclean repository path",
			data:    "version: 1\nrepository_path: /work/../context\n",
			wantErr: ErrInvalidConfig,
		},
		{
			name:    "unsupported version",
			data:    "version: 2\nrepository_path: /work/context\n",
			wantErr: ErrUnsupportedConfigVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseConfig([]byte(tt.data))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseConfig() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got != tt.want {
				t.Fatalf("ParseConfig() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name: "valid",
			config: Config{
				Version:        CurrentConfigVersion,
				RepositoryPath: "/work/context",
			},
		},
		{
			name: "zero version",
			config: Config{
				RepositoryPath: "/work/context",
			},
			wantErr: ErrUnsupportedConfigVersion,
		},
		{
			name: "empty repository path",
			config: Config{
				Version: CurrentConfigVersion,
			},
			wantErr: ErrInvalidConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeRepositoryPath(t *testing.T) {
	tempDir := t.TempDir()
	workingDir := filepath.Join(tempDir, "working")
	targetDir := filepath.Join(tempDir, "target")
	if err := os.Mkdir(workingDir, 0o700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := os.Mkdir(targetDir, 0o700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	oldWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWorkingDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	currentWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() after Chdir error = %v", err)
	}
	want := filepath.Clean(filepath.Join(currentWorkingDir, "..", "target"))

	got, err := NormalizeRepositoryPath(filepath.Join("..", "target", "."))
	if err != nil {
		t.Fatalf("NormalizeRepositoryPath() error = %v", err)
	}
	if got != want {
		t.Fatalf("NormalizeRepositoryPath() = %q, want %q", got, want)
	}
}

func TestNormalizeRepositoryPathDoesNotResolveSymlink(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "target")
	linkPath := filepath.Join(tempDir, "link")
	if err := os.Mkdir(targetDir, 0o700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := os.Symlink(targetDir, linkPath); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	got, err := NormalizeRepositoryPath(linkPath)
	if err != nil {
		t.Fatalf("NormalizeRepositoryPath() error = %v", err)
	}
	if got != linkPath {
		t.Fatalf("NormalizeRepositoryPath() = %q, want lexical path %q", got, linkPath)
	}
}

func TestNormalizeRepositoryPathRejectsEmptyPath(t *testing.T) {
	t.Parallel()

	_, err := NormalizeRepositoryPath("")
	if !errors.Is(err, ErrInvalidRepositoryPath) {
		t.Fatalf("NormalizeRepositoryPath() error = %v, want %v", err, ErrInvalidRepositoryPath)
	}
}
