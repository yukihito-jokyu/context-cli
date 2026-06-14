package skillcatalog

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

var errCatalogIOTest = errors.New("catalog io")

func TestCatalogListsProjectsAndSkillsInNameOrder(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "projects", "zeta", "skills", "z-skill"))
	mustWriteFile(t, filepath.Join(root, "projects", "zeta", "skills", "z-skill", "SKILL.md"))
	mustMkdirAll(t, filepath.Join(root, "projects", "alpha", "skills", "shared"))
	mustWriteFile(t, filepath.Join(root, "projects", "alpha", "skills", "shared", "SKILL.md"))
	mustMkdirAll(t, filepath.Join(root, "projects", "alpha", "skills", "a-skill"))
	mustWriteFile(t, filepath.Join(root, "projects", "alpha", "skills", "a-skill", "SKILL.md"))
	mustWriteFile(t, filepath.Join(root, "projects", "ignored"))
	mustMkdirAll(t, filepath.Join(root, "utils", "skills", "shared"))
	mustWriteFile(t, filepath.Join(root, "utils", "skills", "shared", "SKILL.md"))
	mustMkdirAll(t, filepath.Join(root, "utils", "skills", "common"))
	mustWriteFile(t, filepath.Join(root, "utils", "skills", "common", "SKILL.md"))
	mustMkdirAll(t, filepath.Join(root, "utils", "skills", "invalid"))

	catalog := New(root)
	projects, err := catalog.Projects()
	if err != nil {
		t.Fatalf("Projects() error = %v", err)
	}
	assertCandidateNames(t, projects, []string{"alpha", "zeta"})

	project, err := catalog.Project("alpha")
	if err != nil {
		t.Fatalf("Project() error = %v", err)
	}
	projectSkills, err := catalog.ProjectSkills(project)
	if err != nil {
		t.Fatalf("ProjectSkills() error = %v", err)
	}
	assertCandidateNames(t, projectSkills, []string{"a-skill", "shared"})

	commonSkills, err := catalog.CommonSkills(projectSkills)
	if err != nil {
		t.Fatalf("CommonSkills() error = %v", err)
	}
	assertCandidateNames(t, commonSkills, []string{"common"})
}

func TestCatalogRejectsInvalidProject(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "projects"))

	for _, name := range []string{"", ".", "..", "nested/project"} {
		t.Run(name, func(t *testing.T) {
			_, err := New(root).Project(name)
			if !errors.Is(err, ErrInvalidName) {
				t.Fatalf("Project(%q) error = %v, want ErrInvalidName", name, err)
			}
		})
	}

	_, err := New(root).Project("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Project() error = %v, want ErrNotFound", err)
	}
}

func TestCatalogProjectSkillsRejectsInvalidCandidate(t *testing.T) {
	root := realCatalogTempDir(t)
	expectedProject := filepath.Join(root, "projects", "project")
	mustMkdirAll(t, filepath.Join(expectedProject, "skills"))

	tests := []struct {
		name      string
		candidate Candidate
		want      error
	}{
		{
			name: "不正名",
			candidate: Candidate{
				Name: "../project",
				Path: expectedProject,
			},
			want: ErrInvalidName,
		},
		{
			name: "Repository外の実ディレクトリ",
			candidate: Candidate{
				Name: "project",
				Path: filepath.Join(root, "outside"),
			},
			want: ErrInvalidStructure,
		},
		{
			name: "同名の異なるパス",
			candidate: Candidate{
				Name: "project",
				Path: filepath.Join(root, "projects", "other"),
			},
			want: ErrInvalidStructure,
		},
	}
	mustMkdirAll(t, filepath.Join(root, "outside", "skills"))
	mustMkdirAll(t, filepath.Join(root, "projects", "other", "skills"))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(root).ProjectSkills(tt.candidate)
			if !errors.Is(err, tt.want) {
				t.Fatalf("ProjectSkills() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCatalogRejectsSymbolicLinks(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "projects", "real"))
	if err := os.Symlink(filepath.Join(root, "projects", "real"), filepath.Join(root, "projects", "linked")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	_, err := New(root).Projects()
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("Projects() error = %v, want ErrSymlink", err)
	}
}

func TestCatalogRejectsSkillDirectorySymbolicLink(t *testing.T) {
	root := realCatalogTempDir(t)
	target := filepath.Join(root, "skill-target")
	mustMkdirAll(t, filepath.Join(root, "projects", "project", "skills"))
	mustMkdirAll(t, target)
	mustWriteFile(t, filepath.Join(target, "SKILL.md"))
	mustSymlink(t, target, filepath.Join(root, "projects", "project", "skills", "linked"))

	project, err := New(root).Project("project")
	if err != nil {
		t.Fatalf("Project() error = %v", err)
	}
	_, err = New(root).ProjectSkills(project)
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("ProjectSkills() error = %v, want ErrSymlink", err)
	}
}

func TestCatalogRejectsRepositoryRootSymbolicLink(t *testing.T) {
	parent := realCatalogTempDir(t)
	target := filepath.Join(parent, "root-target")
	link := filepath.Join(parent, "root-link")
	mustMkdirAll(t, filepath.Join(target, "projects"))
	mustSymlink(t, target, link)

	_, err := New(link).Projects()
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("Projects() error = %v, want ErrSymlink", err)
	}
}

func TestCatalogRejectsContainerSymbolicLinksBeforeReadDir(t *testing.T) {
	tests := []struct {
		name string
		run  func(*Catalog) error
		make func(*testing.T, string)
	}{
		{
			name: "projects",
			make: func(t *testing.T, root string) {
				t.Helper()
				target := filepath.Join(root, "projects-target")
				mustMkdirAll(t, target)
				mustSymlink(t, target, filepath.Join(root, "projects"))
			},
			run: func(catalog *Catalog) error {
				_, err := catalog.Projects()
				return err
			},
		},
		{
			name: "project skills",
			make: func(t *testing.T, root string) {
				t.Helper()
				target := filepath.Join(root, "skills-target")
				mustMkdirAll(t, filepath.Join(root, "projects", "project"))
				mustMkdirAll(t, target)
				mustSymlink(t, target, filepath.Join(root, "projects", "project", "skills"))
			},
			run: func(catalog *Catalog) error {
				_, err := catalog.ProjectSkills(Candidate{
					Name: "project",
					Path: filepath.Join(catalog.root, "projects", "project"),
				})
				return err
			},
		},
		{
			name: "utils",
			make: func(t *testing.T, root string) {
				t.Helper()
				target := filepath.Join(root, "utils-target")
				mustMkdirAll(t, target)
				mustSymlink(t, target, filepath.Join(root, "utils"))
			},
			run: func(catalog *Catalog) error {
				_, err := catalog.CommonSkills(nil)
				return err
			},
		},
		{
			name: "common skills",
			make: func(t *testing.T, root string) {
				t.Helper()
				target := filepath.Join(root, "common-target")
				mustMkdirAll(t, filepath.Join(root, "utils"))
				mustMkdirAll(t, target)
				mustSymlink(t, target, filepath.Join(root, "utils", "skills"))
			},
			run: func(catalog *Catalog) error {
				_, err := catalog.CommonSkills(nil)
				return err
			},
		},
		{
			name: "project",
			make: func(t *testing.T, root string) {
				t.Helper()
				target := filepath.Join(root, "project-target")
				mustMkdirAll(t, filepath.Join(root, "projects"))
				mustMkdirAll(t, target)
				mustSymlink(t, target, filepath.Join(root, "projects", "project"))
			},
			run: func(catalog *Catalog) error {
				_, err := catalog.ProjectSkills(Candidate{
					Name: "project",
					Path: filepath.Join(catalog.root, "projects", "project"),
				})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := realCatalogTempDir(t)
			tt.make(t, root)
			err := tt.run(New(root))
			if !errors.Is(err, ErrSymlink) {
				t.Fatalf("error = %v, want ErrSymlink", err)
			}
		})
	}
}

func TestCatalogRejectsLinkedSkillManifest(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "projects", "project", "skills", "skill")
	mustMkdirAll(t, skill)
	target := filepath.Join(root, "manifest")
	mustWriteFile(t, target)
	if err := os.Symlink(target, filepath.Join(skill, "SKILL.md")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	project, err := New(root).Project("project")
	if err != nil {
		t.Fatalf("Project() error = %v", err)
	}
	_, err = New(root).ProjectSkills(project)
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("ProjectSkills() error = %v, want ErrSymlink", err)
	}
}

func TestCatalogExcludesNonRegularSkillManifest(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "projects", "project", "skills", "skill")
	mustMkdirAll(t, filepath.Join(skill, "SKILL.md"))

	project, err := New(root).Project("project")
	if err != nil {
		t.Fatalf("Project() error = %v", err)
	}
	candidates, err := New(root).ProjectSkills(project)
	if err != nil {
		t.Fatalf("ProjectSkills() error = %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("ProjectSkills() = %v, want empty", candidates)
	}
}

func TestCatalogRejectsSpecifiedNonDirectory(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "projects"))
	mustWriteFile(t, filepath.Join(root, "projects", "project"))

	_, err := New(root).Project("project")
	if !errors.Is(err, ErrInvalidStructure) {
		t.Fatalf("Project() error = %v, want ErrInvalidStructure", err)
	}
}

func TestCatalogReportsInjectedIOFailures(t *testing.T) {
	root := "/context"
	tests := []struct {
		name string
		fs   FileSystem
		run  func(*Catalog) error
	}{
		{
			name: "container lstat",
			fs: &stubCatalogFileSystem{
				infos: map[string]os.FileInfo{
					root: catalogFileInfo{mode: os.ModeDir},
				},
				lstatErr: map[string]error{filepath.Join(root, "projects"): errCatalogIOTest},
			},
			run: func(catalog *Catalog) error {
				_, err := catalog.Projects()
				return err
			},
		},
		{
			name: "read directory",
			fs: &stubCatalogFileSystem{
				infos: map[string]os.FileInfo{
					root:                            catalogFileInfo{mode: os.ModeDir},
					filepath.Join(root, "projects"): catalogFileInfo{mode: os.ModeDir},
				},
				readDirErr: map[string]error{filepath.Join(root, "projects"): errCatalogIOTest},
			},
			run: func(catalog *Catalog) error {
				_, err := catalog.Projects()
				return err
			},
		},
		{
			name: "candidate lstat",
			fs: &stubCatalogFileSystem{
				infos: map[string]os.FileInfo{
					root:                            catalogFileInfo{mode: os.ModeDir},
					filepath.Join(root, "projects"): catalogFileInfo{mode: os.ModeDir},
				},
				entries: map[string][]os.DirEntry{
					filepath.Join(root, "projects"): {catalogDirEntry{name: "project"}},
				},
				lstatErr: map[string]error{
					filepath.Join(root, "projects", "project"): errCatalogIOTest,
				},
			},
			run: func(catalog *Catalog) error {
				_, err := catalog.Projects()
				return err
			},
		},
		{
			name: "manifest lstat",
			fs: &stubCatalogFileSystem{
				infos: map[string]os.FileInfo{
					root:                            catalogFileInfo{mode: os.ModeDir},
					filepath.Join(root, "projects"): catalogFileInfo{mode: os.ModeDir},
					filepath.Join(root, "projects", "project"):                    catalogFileInfo{mode: os.ModeDir},
					filepath.Join(root, "projects", "project", "skills"):          catalogFileInfo{mode: os.ModeDir},
					filepath.Join(root, "projects", "project", "skills", "skill"): catalogFileInfo{mode: os.ModeDir},
				},
				entries: map[string][]os.DirEntry{
					filepath.Join(root, "projects", "project", "skills"): {catalogDirEntry{name: "skill"}},
				},
				lstatErr: map[string]error{
					filepath.Join(root, "projects", "project", "skills", "skill", "SKILL.md"): errCatalogIOTest,
				},
			},
			run: func(catalog *Catalog) error {
				_, err := catalog.ProjectSkills(Candidate{
					Name: "project",
					Path: filepath.Join(root, "projects", "project"),
				})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(NewWithFileSystem(root, tt.fs))
			if !errors.Is(err, ErrIO) || !errors.Is(err, errCatalogIOTest) {
				t.Fatalf("error = %v, want ErrIO wrapping injected error", err)
			}
		})
	}
}

func assertCandidateNames(t *testing.T, candidates []Candidate, want []string) {
	t.Helper()
	got := make([]string, len(candidates))
	for i, candidate := range candidates {
		got[i] = candidate.Name
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidate names = %v, want %v", got, want)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func mustSymlink(t *testing.T, target, path string) {
	t.Helper()
	if err := os.Symlink(target, path); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
}

func unixMkfifo(path string, mode os.FileMode) error {
	if err := unix.Mkfifo(path, uint32(mode)); err != nil {
		return fmt.Errorf("failed to create fifo: %w", err)
	}
	return nil
}

func realCatalogTempDir(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks() error = %v", err)
	}
	return root
}

type stubCatalogFileSystem struct {
	infos      map[string]os.FileInfo
	lstatErr   map[string]error
	entries    map[string][]os.DirEntry
	readDirErr map[string]error
}

func (f *stubCatalogFileSystem) Lstat(path string) (os.FileInfo, error) {
	if err := f.lstatErr[path]; err != nil {
		return nil, err
	}
	if info := f.infos[path]; info != nil {
		return info, nil
	}
	return nil, fs.ErrNotExist
}

func (f *stubCatalogFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	if err := f.readDirErr[path]; err != nil {
		return nil, err
	}
	return f.entries[path], nil
}

type catalogFileInfo struct {
	mode os.FileMode
}

func (i catalogFileInfo) Name() string       { return "entry" }
func (i catalogFileInfo) Size() int64        { return 0 }
func (i catalogFileInfo) Mode() os.FileMode  { return i.mode }
func (i catalogFileInfo) ModTime() time.Time { return time.Time{} }
func (i catalogFileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i catalogFileInfo) Sys() any           { return nil }

type catalogDirEntry struct {
	name string
}

func (e catalogDirEntry) Name() string               { return e.name }
func (e catalogDirEntry) IsDir() bool                { return true }
func (e catalogDirEntry) Type() os.FileMode          { return os.ModeDir }
func (e catalogDirEntry) Info() (os.FileInfo, error) { return catalogFileInfo{mode: os.ModeDir}, nil }

func TestCatalogResolvesRecordedSources(t *testing.T) {
	root := realCatalogTempDir(t)
	// プロジェクト固有Skillはalpha有効、betaはSKILL.md欠落、gammaは個別Skill欠落
	mustMkdirAll(t, filepath.Join(root, "projects", "project", "skills", "alpha"))
	mustWriteFile(t, filepath.Join(root, "projects", "project", "skills", "alpha", "SKILL.md"))
	mustMkdirAll(t, filepath.Join(root, "projects", "project", "skills", "beta"))
	// 共通Skillはcommon有効、vanishは個別Skill欠落
	mustMkdirAll(t, filepath.Join(root, "utils", "skills", "common"))
	mustWriteFile(t, filepath.Join(root, "utils", "skills", "common", "SKILL.md"))

	refs := []RecordedSkillRef{
		{Name: "common", Source: SkillSourceCommon},
		{Name: "alpha", Source: SkillSourceProject},
		{Name: "beta", Source: SkillSourceProject},
		{Name: "gamma", Source: SkillSourceProject},
		{Name: "vanish", Source: SkillSourceCommon},
	}

	resolved, err := New(root).ResolveRecordedSources("project", refs)
	if err != nil {
		t.Fatalf("ResolveRecordedSources() error = %v", err)
	}
	if len(resolved) != len(refs) {
		t.Fatalf("resolved length = %d, want %d", len(resolved), len(refs))
	}
	// 供給元種別→名前順で正規化されていることを検証（common系: common, vanish / project系: alpha, beta, gamma）
	wantOrder := []string{"common", "vanish", "alpha", "beta", "gamma"}
	for index, want := range wantOrder {
		if resolved[index].Name != want {
			t.Fatalf("resolved[%d].Name = %q, want %q", index, resolved[index].Name, want)
		}
	}
	assertResolvedState(t, resolved, "common", SourceStateActive)
	assertResolvedState(t, resolved, "alpha", SourceStateActive)
	assertResolvedState(t, resolved, "beta", SourceStateDisabled)
	assertResolvedState(t, resolved, "gamma", SourceStateMissing)
	assertResolvedState(t, resolved, "vanish", SourceStateMissing)
}

func TestCatalogResolveRecordedSourcesRejectsProjectBaseMissing(t *testing.T) {
	root := realCatalogTempDir(t)
	// projects/project/skills を作らない（基底ディレクトリ欠落）
	mustMkdirAll(t, filepath.Join(root, "projects"))

	_, err := New(root).ResolveRecordedSources("project", []RecordedSkillRef{
		{Name: "alpha", Source: SkillSourceProject},
	})
	if !errors.Is(err, ErrInvalidStructure) {
		t.Fatalf("error = %v, want ErrInvalidStructure", err)
	}
}

func TestCatalogResolveRecordedSourcesRejectsProjectBaseSymlink(t *testing.T) {
	root := realCatalogTempDir(t)
	mustMkdirAll(t, filepath.Join(root, "projects", "project"))
	target := filepath.Join(root, "skills-target")
	mustMkdirAll(t, target)
	mustSymlink(t, target, filepath.Join(root, "projects", "project", "skills"))

	_, err := New(root).ResolveRecordedSources("project", []RecordedSkillRef{
		{Name: "alpha", Source: SkillSourceProject},
	})
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("error = %v, want ErrSymlink", err)
	}
}

func TestCatalogResolveRecordedSourcesRejectsProjectBaseNonDirectory(t *testing.T) {
	root := realCatalogTempDir(t)
	mustMkdirAll(t, filepath.Join(root, "projects", "project"))
	mustWriteFile(t, filepath.Join(root, "projects", "project", "skills"))

	_, err := New(root).ResolveRecordedSources("project", []RecordedSkillRef{
		{Name: "alpha", Source: SkillSourceProject},
	})
	if !errors.Is(err, ErrInvalidStructure) {
		t.Fatalf("error = %v, want ErrInvalidStructure", err)
	}
}

func TestCatalogResolveRecordedSourcesRejectsCommonBaseMissingWhenCommonRecorded(t *testing.T) {
	root := realCatalogTempDir(t)
	// 共通Skillが記録されているのに utils/skills が欠落
	mustMkdirAll(t, filepath.Join(root, "projects", "project", "skills"))

	_, err := New(root).ResolveRecordedSources("project", []RecordedSkillRef{
		{Name: "common", Source: SkillSourceCommon},
	})
	if !errors.Is(err, ErrInvalidStructure) {
		t.Fatalf("error = %v, want ErrInvalidStructure", err)
	}
}

func TestCatalogResolveRecordedSourcesSkipsCommonBaseWhenNotRecorded(t *testing.T) {
	root := realCatalogTempDir(t)
	// 共通Skillが未記録なら utils/skills が欠落していてもプロジェクト固有Skillは解決可能
	mustMkdirAll(t, filepath.Join(root, "projects", "project", "skills", "alpha"))
	mustWriteFile(t, filepath.Join(root, "projects", "project", "skills", "alpha", "SKILL.md"))

	resolved, err := New(root).ResolveRecordedSources("project", []RecordedSkillRef{
		{Name: "alpha", Source: SkillSourceProject},
	})
	if err != nil {
		t.Fatalf("ResolveRecordedSources() error = %v", err)
	}
	if len(resolved) != 1 || resolved[0].State != SourceStateActive {
		t.Fatalf("resolved = %#v", resolved)
	}
}

func TestCatalogResolveRecordedSourcesRejectsSkillSymlink(t *testing.T) {
	root := realCatalogTempDir(t)
	mustMkdirAll(t, filepath.Join(root, "projects", "project", "skills"))
	target := filepath.Join(root, "skill-target")
	mustMkdirAll(t, target)
	mustWriteFile(t, filepath.Join(target, "SKILL.md"))
	mustSymlink(t, target, filepath.Join(root, "projects", "project", "skills", "linked"))

	_, err := New(root).ResolveRecordedSources("project", []RecordedSkillRef{
		{Name: "linked", Source: SkillSourceProject},
	})
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("error = %v, want ErrSymlink", err)
	}
}

func TestCatalogResolveRecordedSourcesRejectsSkillNonDirectory(t *testing.T) {
	root := realCatalogTempDir(t)
	mustMkdirAll(t, filepath.Join(root, "projects", "project", "skills"))
	mustWriteFile(t, filepath.Join(root, "projects", "project", "skills", "file"))

	_, err := New(root).ResolveRecordedSources("project", []RecordedSkillRef{
		{Name: "file", Source: SkillSourceProject},
	})
	if !errors.Is(err, ErrInvalidStructure) {
		t.Fatalf("error = %v, want ErrInvalidStructure", err)
	}
}

func TestCatalogResolveRecordedSourcesRejectsManifestSymlink(t *testing.T) {
	root := realCatalogTempDir(t)
	skill := filepath.Join(root, "projects", "project", "skills", "alpha")
	mustMkdirAll(t, skill)
	target := filepath.Join(root, "manifest")
	mustWriteFile(t, target)
	mustSymlink(t, target, filepath.Join(skill, "SKILL.md"))

	_, err := New(root).ResolveRecordedSources("project", []RecordedSkillRef{
		{Name: "alpha", Source: SkillSourceProject},
	})
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("error = %v, want ErrSymlink", err)
	}
}

func TestCatalogResolveRecordedSourcesRejectsManifestNonRegular(t *testing.T) {
	root := realCatalogTempDir(t)
	skill := filepath.Join(root, "projects", "project", "skills", "alpha")
	mustMkdirAll(t, filepath.Join(skill, "SKILL.md"))

	_, err := New(root).ResolveRecordedSources("project", []RecordedSkillRef{
		{Name: "alpha", Source: SkillSourceProject},
	})
	// SKILL.mdがディレクトリの場合は無効化（Disabled）ではなく構造エラー
	if !errors.Is(err, ErrInvalidStructure) {
		t.Fatalf("error = %v, want ErrInvalidStructure", err)
	}
}

func TestCatalogResolveRecordedSourcesRejectsUnsupportedChildType(t *testing.T) {
	root := realCatalogTempDir(t)
	skill := filepath.Join(root, "projects", "project", "skills", "alpha")
	mustMkdirAll(t, skill)
	mustWriteFile(t, filepath.Join(skill, "SKILL.md"))
	// FIFOは配下の未対応種別
	if err := unixMkfifo(filepath.Join(skill, "pipe"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := New(root).ResolveRecordedSources("project", []RecordedSkillRef{
		{Name: "alpha", Source: SkillSourceProject},
	})
	if !errors.Is(err, ErrInvalidStructure) {
		t.Fatalf("error = %v, want ErrInvalidStructure", err)
	}
}

func TestCatalogResolveRecordedSourcesRejectsInvalidProjectName(t *testing.T) {
	root := realCatalogTempDir(t)
	_, err := New(root).ResolveRecordedSources("../escape", []RecordedSkillRef{
		{Name: "alpha", Source: SkillSourceProject},
	})
	if !errors.Is(err, ErrInvalidName) {
		t.Fatalf("error = %v, want ErrInvalidName", err)
	}
}

func TestCatalogResolveRecordedSourcesRejectsInvalidSourceKind(t *testing.T) {
	root := realCatalogTempDir(t)
	_, err := New(root).ResolveRecordedSources("project", []RecordedSkillRef{
		{Name: "alpha", Source: SkillSource("unknown")},
	})
	if !errors.Is(err, ErrInvalidStructure) {
		t.Fatalf("error = %v, want ErrInvalidStructure", err)
	}
}

func assertResolvedState(t *testing.T, resolved []ResolvedSkillSource, name string, want SourceState) {
	t.Helper()
	for _, entry := range resolved {
		if entry.Name != name {
			continue
		}
		if entry.State != want {
			t.Fatalf("resolved %q state = %v, want %v", name, entry.State, want)
		}
		if want == SourceStateActive && entry.Path == "" {
			t.Fatalf("resolved %q active source must carry path", name)
		}
		if want != SourceStateActive && entry.Path != "" {
			t.Fatalf("resolved %q non-active source must not carry path: %q", name, entry.Path)
		}
		return
	}
	t.Fatalf("resolved entry %q not found", name)
}
