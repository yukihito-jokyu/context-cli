package skillcatalog

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Candidate は選択候補の名前と検証済みパスを表します。
type Candidate struct {
	Name string
	Path string
}

// FileSystem はカタログ検査に必要なファイルシステム操作を表します。
type FileSystem interface {
	Lstat(string) (os.FileInfo, error)
	ReadDir(string) ([]os.DirEntry, error)
}

type standardFileSystem struct{}

func (standardFileSystem) Lstat(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect catalog path: %w", err)
	}
	return info, nil
}

func (standardFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog directory: %w", err)
	}
	return entries, nil
}

// Catalog はContext Repository内のプロジェクトとSkillを列挙します。
type Catalog struct {
	root string
	fs   FileSystem
}

// New は検証済みContext Repositoryを使うCatalogを返します。
func New(root string) *Catalog {
	return NewWithFileSystem(root, standardFileSystem{})
}

// NewWithFileSystem は指定したファイルシステムを使うCatalogを返します。
func NewWithFileSystem(root string, fileSystem FileSystem) *Catalog {
	return &Catalog{root: root, fs: fileSystem}
}

// Projects は有効なプロジェクトを名前順で返します。
func (c *Catalog) Projects() ([]Candidate, error) {
	projectsPath := filepath.Join(c.root, "projects")
	if err := c.validateRequiredDirectories(c.root, projectsPath); err != nil {
		return nil, err
	}
	return c.listDirectories(projectsPath, false)
}

// Project は指定名のプロジェクトを検証して返します。
func (c *Catalog) Project(name string) (Candidate, error) {
	if !validName(name) {
		return Candidate{}, catalogError(ErrInvalidName, "project", nil)
	}
	path := filepath.Join(c.root, "projects", name)
	if err := c.validateRequiredDirectories(c.root, filepath.Join(c.root, "projects")); err != nil {
		return Candidate{}, err
	}
	exists, err := c.validateDirectory(path, name, true)
	if err != nil {
		return Candidate{}, err
	}
	if !exists {
		return Candidate{}, catalogError(ErrNotFound, name, fs.ErrNotExist)
	}
	return Candidate{Name: name, Path: path}, nil
}

// ProjectSkills はプロジェクト固有Skillを名前順で返します。
func (c *Catalog) ProjectSkills(project Candidate) ([]Candidate, error) {
	if !validName(project.Name) {
		return nil, catalogError(ErrInvalidName, "project", nil)
	}
	projectPath := filepath.Join(c.root, "projects", project.Name)
	if project.Path != projectPath {
		return nil, catalogError(ErrInvalidStructure, project.Name, nil)
	}
	if err := c.validateRequiredDirectories(c.root, filepath.Join(c.root, "projects"), projectPath); err != nil {
		return nil, err
	}
	skillsPath := filepath.Join(projectPath, "skills")
	exists, err := c.validateDirectory(skillsPath, "skills", true)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []Candidate{}, nil
	}
	return c.listSkills(skillsPath)
}

// CommonSkills はプロジェクト固有Skillと同名の候補を除いた共通Skillを返します。
func (c *Catalog) CommonSkills(projectSkills []Candidate) ([]Candidate, error) {
	commonPath := filepath.Join(c.root, "utils", "skills")
	if err := c.validateRequiredDirectories(c.root, filepath.Join(c.root, "utils"), commonPath); err != nil {
		return nil, err
	}
	candidates, err := c.listSkills(commonPath)
	if err != nil {
		return nil, err
	}
	excluded := make(map[string]struct{}, len(projectSkills))
	for _, candidate := range projectSkills {
		excluded[candidate.Name] = struct{}{}
	}
	filtered := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if _, found := excluded[candidate.Name]; !found {
			filtered = append(filtered, candidate)
		}
	}
	return filtered, nil
}

func (c *Catalog) listSkills(path string) ([]Candidate, error) {
	return c.listDirectories(path, true)
}

func (c *Catalog) listDirectories(path string, requireManifest bool) ([]Candidate, error) {
	entries, err := c.fs.ReadDir(path)
	if err != nil {
		return nil, catalogError(ErrIO, filepath.Base(path), err)
	}

	candidates := make([]Candidate, 0, len(entries))
	for _, entry := range entries {
		candidate, valid, inspectErr := c.inspectCandidate(path, entry.Name(), requireManifest)
		if inspectErr != nil {
			return nil, inspectErr
		}
		if !valid {
			continue
		}
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Name < candidates[j].Name
	})
	return candidates, nil
}

func (c *Catalog) inspectCandidate(parent, name string, requireManifest bool) (Candidate, bool, error) {
	entryPath := filepath.Join(parent, name)
	info, err := c.fs.Lstat(entryPath)
	if err != nil {
		return Candidate{}, false, catalogError(ErrIO, name, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return Candidate{}, false, catalogError(ErrSymlink, name, nil)
	}
	if !info.IsDir() {
		return Candidate{}, false, nil
	}
	if requireManifest {
		valid, manifestErr := c.validateManifest(entryPath, name)
		if manifestErr != nil || !valid {
			return Candidate{}, false, manifestErr
		}
	}
	return Candidate{Name: name, Path: entryPath}, true, nil
}

func (c *Catalog) validateManifest(skillPath, name string) (bool, error) {
	info, err := c.fs.Lstat(filepath.Join(skillPath, "SKILL.md"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, catalogError(ErrIO, name, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, catalogError(ErrSymlink, name, nil)
	}
	return info.Mode().IsRegular(), nil
}

func (c *Catalog) validateRequiredDirectories(paths ...string) error {
	for _, path := range paths {
		if _, err := c.validateDirectory(path, filepath.Base(path), false); err != nil {
			return err
		}
	}
	return nil
}

func (c *Catalog) validateDirectory(path, target string, allowMissing bool) (bool, error) {
	info, err := c.fs.Lstat(path)
	if err != nil {
		if allowMissing && errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		if errors.Is(err, fs.ErrNotExist) {
			return false, catalogError(ErrInvalidStructure, target, err)
		}
		return false, catalogError(ErrIO, target, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, catalogError(ErrSymlink, target, nil)
	}
	if !info.IsDir() {
		return false, catalogError(ErrInvalidStructure, target, nil)
	}
	return true, nil
}

func validName(name string) bool {
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name {
		return false
	}
	return !strings.ContainsAny(name, `/\`)
}

func catalogError(kind error, target string, err error) *Error {
	return &Error{Kind: kind, Target: target, Err: err}
}
