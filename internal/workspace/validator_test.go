package workspace

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var (
	errWorkspaceAbsTest   = errors.New("absolute path")
	errWorkspaceLstatTest = errors.New("lstat")
)

func TestValidatorReturnsCleanAbsoluteDirectory(t *testing.T) {
	root := realTempDir(t)
	validator := NewValidator(func() (string, error) {
		return filepath.Join(root, "child", ".."), nil
	})

	got, err := validator.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got != filepath.Clean(root) {
		t.Fatalf("Validate() = %q, want %q", got, filepath.Clean(root))
	}
}

func TestValidatorRejectsUnsafeWorkspace(t *testing.T) {
	root := realTempDir(t)
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("content"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	tests := []struct {
		name string
		get  func() (string, error)
		want error
	}{
		{name: "取得失敗", get: func() (string, error) { return "", os.ErrPermission }, want: ErrCurrentDirectory},
		{name: "不在", get: func() (string, error) { return filepath.Join(root, "missing"), nil }, want: ErrNotExist},
		{name: "非ディレクトリ", get: func() (string, error) { return file, nil }, want: ErrNotDirectory},
		{name: "リンク", get: func() (string, error) { return link, nil }, want: ErrSymlink},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewValidator(tt.get).Validate()
			if !errors.Is(err, tt.want) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestValidatorRejectsInjectedPathFailures(t *testing.T) {
	tests := []struct {
		name string
		fs   FileSystem
		want error
	}{
		{
			name: "絶対化失敗",
			fs: &stubWorkspaceFileSystem{
				absErr: errWorkspaceAbsTest,
			},
			want: ErrAbsolutePath,
		},
		{
			name: "中間リンク",
			fs: &stubWorkspaceFileSystem{
				absolute: "/safe/workspace",
				infos: map[string]os.FileInfo{
					"/safe": catalogWorkspaceInfo{mode: os.ModeDir | os.ModeSymlink},
				},
			},
			want: ErrSymlink,
		},
		{
			name: "中間非ディレクトリ",
			fs: &stubWorkspaceFileSystem{
				absolute: "/safe/workspace",
				infos: map[string]os.FileInfo{
					"/safe": catalogWorkspaceInfo{mode: 0},
				},
			},
			want: ErrIO,
		},
		{
			name: "中間I/O失敗",
			fs: &stubWorkspaceFileSystem{
				absolute: "/safe/workspace",
				lstatErr: map[string]error{"/safe": errWorkspaceLstatTest},
			},
			want: ErrIO,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewValidatorWithFileSystem(func() (string, error) {
				return "workspace", nil
			}, tt.fs).Validate()
			if !errors.Is(err, tt.want) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.want)
			}
			if errors.Is(tt.want, ErrIO) && tt.name == "中間I/O失敗" && !errors.Is(err, errWorkspaceLstatTest) {
				t.Fatalf("Validate() error = %v, want injected error", err)
			}
		})
	}
}

func realTempDir(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks() error = %v", err)
	}
	return root
}

type stubWorkspaceFileSystem struct {
	absolute string
	absErr   error
	infos    map[string]os.FileInfo
	lstatErr map[string]error
}

func (f *stubWorkspaceFileSystem) Abs(string) (string, error) {
	return f.absolute, f.absErr
}

func (f *stubWorkspaceFileSystem) Lstat(path string) (os.FileInfo, error) {
	if err := f.lstatErr[path]; err != nil {
		return nil, err
	}
	if info := f.infos[path]; info != nil {
		return info, nil
	}
	return nil, fs.ErrNotExist
}

type catalogWorkspaceInfo struct {
	mode os.FileMode
}

func (i catalogWorkspaceInfo) Name() string       { return "entry" }
func (i catalogWorkspaceInfo) Size() int64        { return 0 }
func (i catalogWorkspaceInfo) Mode() os.FileMode  { return i.mode }
func (i catalogWorkspaceInfo) ModTime() time.Time { return time.Time{} }
func (i catalogWorkspaceInfo) IsDir() bool        { return i.mode.IsDir() }
func (i catalogWorkspaceInfo) Sys() any           { return nil }
