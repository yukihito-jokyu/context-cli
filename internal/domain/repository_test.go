package domain_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/yukihito-jokyu/context-cli/internal/domain"
)

var errPermissionDenied = errors.New("permission denied")

type mockFileStatus struct {
	isDir     bool
	isRegular bool
	isSymlink bool
	mode      fs.FileMode
}

func (m mockFileStatus) IsDir() bool { return m.isDir }

func (m mockFileStatus) IsRegular() bool { return m.isRegular }

func (m mockFileStatus) IsSymlink() bool { return m.isSymlink }

func (m mockFileStatus) Mode() fs.FileMode { return m.mode }

type mockFileEntry struct {
	name  string
	isDir bool
}

func (m mockFileEntry) Name() string { return m.name }
func (m mockFileEntry) IsDir() bool  { return m.isDir }

type mockFileSystem struct {
	files        map[string]mockFileStatus
	dirs         map[string][]mockFileEntry
	fileContents map[string][]byte
	readErrors   map[string]error
}

func (m *mockFileSystem) LStat(_ context.Context, path string) (domain.FileStatus, error) {
	status, ok := m.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return status, nil
}

func (m *mockFileSystem) ReadDir(_ context.Context, path string) ([]domain.FileEntry, error) {
	if err, ok := m.readErrors[path]; ok {
		return nil, err
	}
	entries, ok := m.dirs[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	var res []domain.FileEntry
	for _, e := range entries {
		res = append(res, e)
	}
	return res, nil
}

func (m *mockFileSystem) ReadFile(_ context.Context, path string) ([]byte, error) {
	if err, ok := m.readErrors[path]; ok {
		return nil, err
	}
	content, ok := m.fileContents[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return content, nil
}

func setupValidFileSystem() *mockFileSystem {
	return &mockFileSystem{
		files: map[string]mockFileStatus{
			"/":                                      {isDir: true, mode: 0755},
			"/Users":                                 {isDir: true, mode: 0755},
			"/Users/hoge":                            {isDir: true, mode: 0700},
			"/Users/hoge/repo":                       {isDir: true, mode: 0755},
			"/Users/hoge/repo/projects":              {isDir: true, mode: 0755},
			"/Users/hoge/repo/projects/proj1":        {isDir: true, mode: 0755},
			"/Users/hoge/repo/projects/proj1/skills": {isDir: true, mode: 0755},
			"/Users/hoge/repo/projects/proj1/skills/skill1":          {isDir: true, mode: 0755},
			"/Users/hoge/repo/projects/proj1/skills/skill1/SKILL.md": {isRegular: true, mode: 0644},
			"/Users/hoge/repo/utils":                                 {isDir: true, mode: 0755},
			"/Users/hoge/repo/utils/skills":                          {isDir: true, mode: 0755},
			"/Users/hoge/repo/utils/skills/skill2":                   {isDir: true, mode: 0755},
			"/Users/hoge/repo/utils/skills/skill2/SKILL.md":          {isRegular: true, mode: 0644},
		},
		dirs: map[string][]mockFileEntry{
			"/Users/hoge/repo": {
				{name: "projects", isDir: true},
				{name: "utils", isDir: true},
			},
			"/Users/hoge/repo/projects": {
				{name: "proj1", isDir: true},
			},
			"/Users/hoge/repo/projects/proj1": {
				{name: "skills", isDir: true},
			},
			"/Users/hoge/repo/projects/proj1/skills": {
				{name: "skill1", isDir: true},
			},
			"/Users/hoge/repo/projects/proj1/skills/skill1": {
				{name: "SKILL.md", isDir: false},
			},
			"/Users/hoge/repo/utils": {
				{name: "skills", isDir: true},
			},
			"/Users/hoge/repo/utils/skills": {
				{name: "skill2", isDir: true},
			},
			"/Users/hoge/repo/utils/skills/skill2": {
				{name: "SKILL.md", isDir: false},
			},
		},
		fileContents: map[string][]byte{
			"/Users/hoge/repo/projects/proj1/skills/skill1/SKILL.md": []byte("# Skill 1"),
			"/Users/hoge/repo/utils/skills/skill2/SKILL.md":          []byte("# Skill 2"),
		},
		readErrors: map[string]error{},
	}
}

func TestRepositoryValidator_Valid(t *testing.T) {
	fsMock := setupValidFileSystem()
	validator := domain.NewRepositoryValidator(fsMock)

	errors, err := validator.Validate(context.Background(), "/Users/hoge/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(errors) > 0 {
		t.Errorf("expected no validation errors, got %d:", len(errors))
		for _, e := range errors {
			t.Logf("Path: %s, Reason: %s", e.Path, e.Reason)
		}
	}
}

func TestRepositoryValidator_MissingDirectories(t *testing.T) {
	tests := []struct {
		name         string
		modifyFS     func(*mockFileSystem)
		expectedPath string
	}{
		{
			name: "projects directory missing",
			modifyFS: func(m *mockFileSystem) {
				delete(m.files, "/Users/hoge/repo/projects")
				m.dirs["/Users/hoge/repo"] = []mockFileEntry{{name: "utils", isDir: true}}
			},
			expectedPath: "projects",
		},
		{
			name: "utils/skills directory missing",
			modifyFS: func(m *mockFileSystem) {
				delete(m.files, "/Users/hoge/repo/utils/skills")
				m.dirs["/Users/hoge/repo/utils"] = []mockFileEntry{}
			},
			expectedPath: "utils/skills",
		},
		{
			name: "no projects in projects directory",
			modifyFS: func(m *mockFileSystem) {
				m.dirs["/Users/hoge/repo/projects"] = []mockFileEntry{}
			},
			expectedPath: "projects",
		},
		{
			name: "project skills directory missing",
			modifyFS: func(m *mockFileSystem) {
				delete(m.files, "/Users/hoge/repo/projects/proj1/skills")
				m.dirs["/Users/hoge/repo/projects/proj1"] = []mockFileEntry{}
			},
			expectedPath: "projects/proj1/skills",
		},
		{
			name: "SKILL.md missing in project skill",
			modifyFS: func(m *mockFileSystem) {
				delete(m.files, "/Users/hoge/repo/projects/proj1/skills/skill1/SKILL.md")
			},
			expectedPath: "projects/proj1/skills/skill1/SKILL.md",
		},
		{
			name: "SKILL.md missing in common skill",
			modifyFS: func(m *mockFileSystem) {
				delete(m.files, "/Users/hoge/repo/utils/skills/skill2/SKILL.md")
			},
			expectedPath: "utils/skills/skill2/SKILL.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsMock := setupValidFileSystem()
			tt.modifyFS(fsMock)
			validator := domain.NewRepositoryValidator(fsMock)

			errors, err := validator.Validate(context.Background(), "/Users/hoge/repo")
			if err != nil {
				t.Fatalf("unexpected system error: %v", err)
			}

			found := false
			for _, e := range errors {
				if e.Path == tt.expectedPath {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected validation error on path %s, but got errors: %v", tt.expectedPath, errors)
			}
		})
	}
}

func TestRepositoryValidator_Symlinks(t *testing.T) {
	tests := []struct {
		name         string
		modifyFS     func(*mockFileSystem)
		expectedPath string
	}{
		{
			name: "ancestor component is symlink",
			modifyFS: func(m *mockFileSystem) {
				status := m.files["/Users/hoge"]
				status.isSymlink = true
				m.files["/Users/hoge"] = status
			},
			expectedPath: "..",
		},
		{
			name: "repo root itself is symlink",
			modifyFS: func(m *mockFileSystem) {
				status := m.files["/Users/hoge/repo"]
				status.isSymlink = true
				m.files["/Users/hoge/repo"] = status
			},
			expectedPath: ".",
		},
		{
			name: "projects is symlink",
			modifyFS: func(m *mockFileSystem) {
				status := m.files["/Users/hoge/repo/projects"]
				status.isSymlink = true
				m.files["/Users/hoge/repo/projects"] = status
			},
			expectedPath: "projects",
		},
		{
			name: "project directory is symlink",
			modifyFS: func(m *mockFileSystem) {
				status := m.files["/Users/hoge/repo/projects/proj1"]
				status.isSymlink = true
				m.files["/Users/hoge/repo/projects/proj1"] = status
			},
			expectedPath: "projects/proj1",
		},
		{
			name: "project skills is symlink",
			modifyFS: func(m *mockFileSystem) {
				status := m.files["/Users/hoge/repo/projects/proj1/skills"]
				status.isSymlink = true
				m.files["/Users/hoge/repo/projects/proj1/skills"] = status
			},
			expectedPath: "projects/proj1/skills",
		},
		{
			name: "skill directory is symlink",
			modifyFS: func(m *mockFileSystem) {
				status := m.files["/Users/hoge/repo/projects/proj1/skills/skill1"]
				status.isSymlink = true
				m.files["/Users/hoge/repo/projects/proj1/skills/skill1"] = status
			},
			expectedPath: "projects/proj1/skills/skill1",
		},
		{
			name: "SKILL.md is symlink",
			modifyFS: func(m *mockFileSystem) {
				status := m.files["/Users/hoge/repo/projects/proj1/skills/skill1/SKILL.md"]
				status.isSymlink = true
				m.files["/Users/hoge/repo/projects/proj1/skills/skill1/SKILL.md"] = status
			},
			expectedPath: "projects/proj1/skills/skill1/SKILL.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsMock := setupValidFileSystem()
			tt.modifyFS(fsMock)
			validator := domain.NewRepositoryValidator(fsMock)

			errors, err := validator.Validate(context.Background(), "/Users/hoge/repo")
			if err != nil {
				t.Fatalf("unexpected system error: %v", err)
			}

			found := false
			for _, e := range errors {
				if e.Path == tt.expectedPath {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected validation error on path %s, but got errors: %v", tt.expectedPath, errors)
			}
		})
	}
}

func TestRepositoryValidator_WritePermissions(t *testing.T) {
	tests := []struct {
		name         string
		modifyFS     func(*mockFileSystem)
		expectedPath string
	}{
		{
			name: "ancestor component has group write",
			modifyFS: func(m *mockFileSystem) {
				status := m.files["/Users/hoge"]
				status.mode |= 0020
				m.files["/Users/hoge"] = status
			},
			expectedPath: "..",
		},
		{
			name: "repo root has other write",
			modifyFS: func(m *mockFileSystem) {
				status := m.files["/Users/hoge/repo"]
				status.mode |= 0002
				m.files["/Users/hoge/repo"] = status
			},
			expectedPath: ".",
		},
		{
			name: "projects directory has group write",
			modifyFS: func(m *mockFileSystem) {
				status := m.files["/Users/hoge/repo/projects"]
				status.mode |= 0020
				m.files["/Users/hoge/repo/projects"] = status
			},
			expectedPath: "projects",
		},
		{
			name: "SKILL.md has other write",
			modifyFS: func(m *mockFileSystem) {
				status := m.files["/Users/hoge/repo/projects/proj1/skills/skill1/SKILL.md"]
				status.mode |= 0002
				m.files["/Users/hoge/repo/projects/proj1/skills/skill1/SKILL.md"] = status
			},
			expectedPath: "projects/proj1/skills/skill1/SKILL.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsMock := setupValidFileSystem()
			tt.modifyFS(fsMock)
			validator := domain.NewRepositoryValidator(fsMock)

			errors, err := validator.Validate(context.Background(), "/Users/hoge/repo")
			if err != nil {
				t.Fatalf("unexpected system error: %v", err)
			}

			found := false
			for _, e := range errors {
				if e.Path == tt.expectedPath {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected validation error on path %s, but got errors: %v", tt.expectedPath, errors)
			}
		})
	}
}

//nolint:gocognit // Related accessibility cases are kept together for validator coverage.
func TestRepositoryValidator_NotAccessible(t *testing.T) {
	t.Run("repository root does not exist", func(t *testing.T) {
		fsMock := setupValidFileSystem()
		delete(fsMock.files, "/Users/hoge/repo")
		validator := domain.NewRepositoryValidator(fsMock)

		errors, err := validator.Validate(context.Background(), "/Users/hoge/repo")
		if err != nil {
			t.Fatalf("unexpected system error: %v", err)
		}

		if len(errors) == 0 || errors[0].Path != "." {
			t.Errorf("expected validation error on repo root, got %v", errors)
		}
	})

	t.Run("repository root not readable", func(t *testing.T) {
		fsMock := setupValidFileSystem()
		fsMock.readErrors["/Users/hoge/repo"] = errPermissionDenied
		validator := domain.NewRepositoryValidator(fsMock)

		errors, err := validator.Validate(context.Background(), "/Users/hoge/repo")
		if err != nil {
			t.Fatalf("unexpected system error: %v", err)
		}

		if len(errors) == 0 || errors[0].Path != "." {
			t.Errorf("expected validation error on repo root, got %v", errors)
		}
	})

	t.Run("skill directory not readable", func(t *testing.T) {
		fsMock := setupValidFileSystem()
		skillPath := "/Users/hoge/repo/projects/proj1/skills/skill1"
		fsMock.readErrors[skillPath] = errPermissionDenied
		validator := domain.NewRepositoryValidator(fsMock)

		errors, err := validator.Validate(context.Background(), "/Users/hoge/repo")
		if err != nil {
			t.Fatalf("unexpected system error: %v", err)
		}

		found := false
		for _, validationErr := range errors {
			if validationErr.Path == "projects/proj1/skills/skill1" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected validation error on unreadable skill directory, got %v", errors)
		}
	})
}
