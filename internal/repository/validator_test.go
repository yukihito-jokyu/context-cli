package repository

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

var errAbsolutePathTest = errors.New("absolute path conversion failed")

type validationCase struct {
	name       string
	path       string
	prepare    func(t *testing.T) string
	wantKind   error
	wantTarget string
}

func TestValidatorValidate(t *testing.T) {
	base := resolvedTempDir(t)
	validRepo := filepath.Join(base, "workspace", "context")
	createRepository(t, validRepo)

	tests := []validationCase{
		{
			name: "有効なリポジトリ",
			path: validRepo,
		},
		{
			name:       "指定パスが存在しない",
			path:       filepath.Join(base, "missing"),
			wantKind:   ErrPathNotExist,
			wantTarget: ".",
		},
		{
			name: "指定パスが通常ファイル",
			prepare: func(t *testing.T) string {
				t.Helper()
				path := filepath.Join(base, "file")
				writeFile(t, path)
				return path
			},
			wantKind:   ErrNotDirectory,
			wantTarget: ".",
		},
		{
			name: "projectsが存在しない",
			prepare: func(t *testing.T) string {
				t.Helper()
				path := filepath.Join(base, "missing-projects")
				mkdir(t, filepath.Join(path, "utils", "skills"))
				return path
			},
			wantKind:   ErrRequiredStructure,
			wantTarget: "projects",
		},
		{
			name: "utilsが通常ファイル",
			prepare: func(t *testing.T) string {
				t.Helper()
				path := filepath.Join(base, "file-utils")
				mkdir(t, filepath.Join(path, "projects"))
				writeFile(t, filepath.Join(path, "utils"))
				return path
			},
			wantKind:   ErrRequiredStructure,
			wantTarget: "utils",
		},
		{
			name: "utils/skillsが存在しない",
			prepare: func(t *testing.T) string {
				t.Helper()
				path := filepath.Join(base, "missing-skills")
				mkdir(t, filepath.Join(path, "projects"))
				mkdir(t, filepath.Join(path, "utils"))
				return path
			},
			wantKind:   ErrRequiredStructure,
			wantTarget: "utils/skills",
		},
		{
			name: "指定パスがシンボリックリンク",
			prepare: func(t *testing.T) string {
				t.Helper()
				target := filepath.Join(base, "repo-target")
				createRepository(t, target)
				link := filepath.Join(base, "repo-link")
				symlink(t, target, link)
				return link
			},
			wantKind:   ErrSymlink,
			wantTarget: ".",
		},
		{
			name: "親構成要素がシンボリックリンク",
			prepare: func(t *testing.T) string {
				t.Helper()
				target := filepath.Join(base, "parent-target")
				path := filepath.Join(target, "context")
				createRepository(t, path)
				link := filepath.Join(base, "parent-link")
				symlink(t, target, link)
				return filepath.Join(link, "context")
			},
			wantKind:   ErrSymlink,
			wantTarget: "repository parent",
		},
		{
			name: "projectsがシンボリックリンク",
			prepare: func(t *testing.T) string {
				t.Helper()
				path := filepath.Join(base, "linked-projects")
				mkdir(t, filepath.Join(path, "utils", "skills"))
				target := filepath.Join(base, "projects-target")
				mkdir(t, target)
				symlink(t, target, filepath.Join(path, "projects"))
				return path
			},
			wantKind:   ErrSymlink,
			wantTarget: "projects",
		},
		{
			name: "utilsがシンボリックリンク",
			prepare: func(t *testing.T) string {
				t.Helper()
				path := filepath.Join(base, "linked-utils")
				mkdir(t, filepath.Join(path, "projects"))
				target := filepath.Join(base, "utils-target")
				mkdir(t, filepath.Join(target, "skills"))
				symlink(t, target, filepath.Join(path, "utils"))
				return path
			},
			wantKind:   ErrSymlink,
			wantTarget: "utils",
		},
		{
			name: "utils/skillsがシンボリックリンク",
			prepare: func(t *testing.T) string {
				t.Helper()
				path := filepath.Join(base, "linked-skills")
				mkdir(t, filepath.Join(path, "projects"))
				mkdir(t, filepath.Join(path, "utils"))
				target := filepath.Join(base, "skills-target")
				mkdir(t, target)
				symlink(t, target, filepath.Join(path, "utils", "skills"))
				return path
			},
			wantKind:   ErrSymlink,
			wantTarget: "utils/skills",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runValidationCase(t, tt)
		})
	}
}

func runValidationCase(t *testing.T, tt validationCase) {
	t.Helper()
	path := tt.path
	if tt.prepare != nil {
		path = tt.prepare(t)
	}

	got, err := NewValidator(NewFileSystem()).Validate(path)
	if tt.wantKind == nil {
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		want, absErr := filepath.Abs(path)
		if absErr != nil {
			t.Fatalf("filepath.Abs() error = %v", absErr)
		}
		if got != filepath.Clean(want) {
			t.Fatalf("Validate() = %q, want %q", got, filepath.Clean(want))
		}
		return
	}

	if !errors.Is(err, tt.wantKind) {
		t.Fatalf("Validate() error = %v, want errors.Is(_, %v)", err, tt.wantKind)
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Validate() error type = %T, want *ValidationError", err)
	}
	if validationErr.Target != tt.wantTarget {
		t.Errorf("ValidationError.Target = %q, want %q", validationErr.Target, tt.wantTarget)
	}
	if strings.Contains(err.Error(), path) || strings.Contains(err.Error(), filepath.Clean(path)) {
		t.Errorf("error message leaks input path: %q", err)
	}
}

func TestValidatorValidateNormalizesRelativePath(t *testing.T) {
	base := resolvedTempDir(t)
	createRepository(t, filepath.Join(base, "workspace", "context"))
	withWorkingDirectory(t, base)

	got, err := NewValidator(NewFileSystem()).Validate(filepath.Join("workspace", "unused", "..", "context"))
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	want := filepath.Join(base, "workspace", "context")
	if got != want {
		t.Fatalf("Validate() = %q, want %q", got, want)
	}
}

func TestValidatorValidateInspectionBoundary(t *testing.T) {
	absolute := filepath.Join(string(filepath.Separator), "safe", "workspace", "context")
	tests := []struct {
		name      string
		input     string
		abs       string
		wantPaths []string
	}{
		{
			name:  "相対パスは作業ディレクトリを検査しない",
			input: filepath.Join("workspace", "context"),
			abs:   absolute,
			wantPaths: []string{
				filepath.Join(string(filepath.Separator), "safe", "workspace"),
				absolute,
				filepath.Join(absolute, "projects"),
				filepath.Join(absolute, "utils"),
				filepath.Join(absolute, "utils", "skills"),
			},
		},
		{
			name:  "一階層上のリポジトリでは作業ディレクトリの祖先を検査しない",
			input: filepath.Join("..", "context"),
			abs:   filepath.Join(string(filepath.Separator), "safe", "context"),
			wantPaths: []string{
				filepath.Join(string(filepath.Separator), "safe", "context"),
				filepath.Join(string(filepath.Separator), "safe", "context", "projects"),
				filepath.Join(string(filepath.Separator), "safe", "context", "utils"),
				filepath.Join(string(filepath.Separator), "safe", "context", "utils", "skills"),
			},
		},
		{
			name:  "複数階層上のリポジトリでは作業ディレクトリの祖先を検査しない",
			input: filepath.Join("..", "..", "context"),
			abs:   filepath.Join(string(filepath.Separator), "safe", "context"),
			wantPaths: []string{
				filepath.Join(string(filepath.Separator), "safe", "context"),
				filepath.Join(string(filepath.Separator), "safe", "context", "projects"),
				filepath.Join(string(filepath.Separator), "safe", "context", "utils"),
				filepath.Join(string(filepath.Separator), "safe", "context", "utils", "skills"),
			},
		},
		{
			name:  "上位階層配下の明示された通常名だけを検査する",
			input: filepath.Join("..", "parent", "context"),
			abs:   filepath.Join(string(filepath.Separator), "safe", "parent", "context"),
			wantPaths: []string{
				filepath.Join(string(filepath.Separator), "safe", "parent"),
				filepath.Join(string(filepath.Separator), "safe", "parent", "context"),
				filepath.Join(string(filepath.Separator), "safe", "parent", "context", "projects"),
				filepath.Join(string(filepath.Separator), "safe", "parent", "context", "utils"),
				filepath.Join(string(filepath.Separator), "safe", "parent", "context", "utils", "skills"),
			},
		},
		{
			name:  "絶対パスはルート以外の親を検査する",
			input: absolute,
			abs:   absolute,
			wantPaths: []string{
				filepath.Join(string(filepath.Separator), "safe"),
				filepath.Join(string(filepath.Separator), "safe", "workspace"),
				absolute,
				filepath.Join(absolute, "projects"),
				filepath.Join(absolute, "utils"),
				filepath.Join(absolute, "utils", "skills"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &recordingFileSystem{abs: tt.abs}
			_, err := NewValidator(fake).Validate(tt.input)
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if !reflect.DeepEqual(fake.paths, tt.wantPaths) {
				t.Fatalf("Lstat paths = %#v, want %#v", fake.paths, tt.wantPaths)
			}
		})
	}
}

func TestValidatorValidateWrapsIOErrorWithoutLeakingPath(t *testing.T) {
	internalPath := "/secret/internal/path"
	cause := &os.PathError{Op: "lstat", Path: internalPath, Err: fs.ErrPermission}
	fake := &recordingFileSystem{
		abs:     filepath.Join(string(filepath.Separator), "safe", "context"),
		failAt:  0,
		failErr: cause,
	}

	_, err := NewValidator(fake).Validate(filepath.Join(string(filepath.Separator), "safe", "context"))
	if !errors.Is(err, ErrIO) {
		t.Fatalf("Validate() error = %v, want ErrIO", err)
	}
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("Validate() error = %v, want wrapped permission error", err)
	}
	unwrapped := errors.Unwrap(err)
	if !errors.Is(unwrapped, cause) {
		t.Fatalf("errors.Unwrap() = %#v, want %#v", unwrapped, cause)
	}
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("errors.As() could not retrieve *os.PathError from %v", err)
	}
	if pathErr != cause {
		t.Fatalf("errors.As() path error = %#v, want %#v", pathErr, cause)
	}
	if strings.Contains(err.Error(), internalPath) {
		t.Fatalf("error message leaks internal path: %q", err)
	}
}

func TestValidatorValidateRelativeIOErrorDoesNotLeakPaths(t *testing.T) {
	inputPath := filepath.Join("..", "parent", "context")
	absolutePath := filepath.Join(string(filepath.Separator), "secret", "parent", "context")
	internalPath := filepath.Join(string(filepath.Separator), "internal", "failure", "path")
	cause := &os.PathError{Op: "lstat", Path: internalPath, Err: fs.ErrPermission}
	fake := &recordingFileSystem{
		abs:     absolutePath,
		failAt:  0,
		failErr: cause,
	}

	_, err := NewValidator(fake).Validate(inputPath)
	if !errors.Is(err, ErrIO) {
		t.Fatalf("Validate() error = %v, want ErrIO", err)
	}
	for _, secret := range []string{inputPath, absolutePath, internalPath} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("error message leaks path %q: %q", secret, err)
		}
	}
}

func TestValidatorValidateClassifiesIOErrorByInspectionTarget(t *testing.T) {
	absolute := filepath.Join(string(filepath.Separator), "safe", "context")
	tests := []struct {
		name       string
		failAt     int
		wantTarget string
	}{
		{name: "親構成要素", failAt: 0, wantTarget: "repository parent"},
		{name: "リポジトリ自体", failAt: 1, wantTarget: "."},
		{name: "必須構造", failAt: 2, wantTarget: "projects"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cause := &os.PathError{Op: "lstat", Path: "/secret/path", Err: fs.ErrPermission}
			fake := &recordingFileSystem{abs: absolute, failAt: tt.failAt, failErr: cause}

			_, err := NewValidator(fake).Validate(absolute)
			if !errors.Is(err, ErrIO) || !errors.Is(err, fs.ErrPermission) {
				t.Fatalf("Validate() error = %v, want ErrIO wrapping permission error", err)
			}
			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("Validate() error type = %T, want *ValidationError", err)
			}
			if validationErr.Target != tt.wantTarget {
				t.Errorf("ValidationError.Target = %q, want %q", validationErr.Target, tt.wantTarget)
			}
		})
	}
}

func TestValidatorValidateWrapsAbsolutePathError(t *testing.T) {
	fake := &recordingFileSystem{absErr: errAbsolutePathTest}

	_, err := NewValidator(fake).Validate("context")
	if !errors.Is(err, ErrIO) || !errors.Is(err, errAbsolutePathTest) {
		t.Fatalf("Validate() error = %v, want ErrIO wrapping absolute path error", err)
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) || validationErr.Target != "." {
		t.Fatalf("Validate() error = %#v, want target %q", err, ".")
	}
	if len(fake.paths) != 0 {
		t.Fatalf("Lstat paths = %#v, want no calls", fake.paths)
	}
}

type recordingFileSystem struct {
	abs     string
	absErr  error
	paths   []string
	failAt  int
	failErr error
}

func (f *recordingFileSystem) Abs(string) (string, error) {
	return f.abs, f.absErr
}

func (f *recordingFileSystem) Lstat(path string) (os.FileInfo, error) {
	index := len(f.paths)
	f.paths = append(f.paths, path)
	if f.failErr != nil && index == f.failAt {
		return nil, f.failErr
	}
	return fakeDirectoryInfo{}, nil
}

type fakeDirectoryInfo struct{}

func (fakeDirectoryInfo) Name() string       { return "directory" }
func (fakeDirectoryInfo) Size() int64        { return 0 }
func (fakeDirectoryInfo) Mode() os.FileMode  { return os.ModeDir }
func (fakeDirectoryInfo) ModTime() time.Time { return time.Time{} }
func (fakeDirectoryInfo) IsDir() bool        { return true }
func (fakeDirectoryInfo) Sys() any           { return nil }

func resolvedTempDir(t *testing.T) string {
	t.Helper()
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("一時ディレクトリの実体パスを取得できません: %v", err)
	}
	return path
}

func createRepository(t *testing.T, path string) {
	t.Helper()
	mkdir(t, filepath.Join(path, "projects"))
	mkdir(t, filepath.Join(path, "utils", "skills"))
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatalf("ディレクトリを作成できません: %v", err)
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	mkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
		t.Fatalf("ファイルを作成できません: %v", err)
	}
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("シンボリックリンクを作成できません: %v", err)
	}
}

func withWorkingDirectory(t *testing.T, path string) {
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
