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

// RecordedSkillRef は記録済みSkillの参照を表します。
type RecordedSkillRef struct {
	Name   string
	Source SkillSource
}

// SkillSource はSkillの供給元種別を表します。
type SkillSource string

const (
	// SkillSourceProject はプロジェクト固有Skillを表します。
	SkillSourceProject SkillSource = "project"
	// SkillSourceCommon は共通Skillを表します。
	SkillSourceCommon SkillSource = "common"
)

// SourceState は解決済み供給元の状態分類を表します。
type SourceState uint8

const (
	// SourceStateActive は供給元が実ディレクトリでSKILL.mdを持つ有効状態を表します。
	SourceStateActive SourceState = iota + 1
	// SourceStateMissing は個別Skillパスが欠落したことを表します。
	SourceStateMissing
	// SourceStateDisabled はSKILL.mdが欠落して無効化されたことを表します。
	SourceStateDisabled
)

// ResolvedSkillSource は記録済みSkillの解決結果を表します。
//
// State がActive以外の場合は Path を空にします。
// 基底ディレクトリやプロジェクトの構造破損、シンボリックリンク、未対応種別は
// エラーとして返し、個別Skillの消失・無効化とは区別します。
type ResolvedSkillSource struct {
	Name   string
	Source SkillSource
	State  SourceState
	Path   string
}

// ResolveRecordedSources は記録済みプロジェクトと必要な基底Skillディレクトリを検証し、
// 各記録済みSkillの供給元を状態分類付きで返します。
//
// 候補列挙は行わず、記録済みSkillだけを対象とします。
// 共通Skillが記録されていない場合は utils/skills の検証を省略します。
// 同一Skill名・供給元の重複は名前順で1件に正規化します。
func (c *Catalog) ResolveRecordedSources(project string, skills []RecordedSkillRef) ([]ResolvedSkillSource, error) {
	if !validName(project) {
		return nil, catalogError(ErrInvalidName, "project", nil)
	}
	refs, err := normalizeRecordedRefs(skills)
	if err != nil {
		return nil, err
	}

	projectBase := filepath.Join(c.root, "projects", project)
	projectSkillsBase := filepath.Join(projectBase, "skills")
	commonBase := filepath.Join(c.root, "utils", "skills")

	needsCommon := false
	for _, ref := range refs {
		if ref.Source == SkillSourceCommon {
			needsCommon = true
			break
		}
	}

	if err := c.validateRecordedProject(projectBase, projectSkillsBase); err != nil {
		return nil, err
	}
	if needsCommon {
		if err := c.validateRecordedCommonBase(commonBase); err != nil {
			return nil, err
		}
	}

	resolved := make([]ResolvedSkillSource, 0, len(refs))
	for _, ref := range refs {
		entry, resolveErr := c.resolveRecordedSkill(ref, projectSkillsBase, commonBase)
		if resolveErr != nil {
			return nil, resolveErr
		}
		resolved = append(resolved, entry)
	}
	return resolved, nil
}

func (c *Catalog) validateRecordedProject(projectPath, projectSkillsPath string) error {
	if err := c.validateRequiredRealDirectory(filepath.Join(c.root, "projects")); err != nil {
		return err
	}
	if err := c.validateRequiredRealDirectory(projectPath); err != nil {
		return err
	}
	if err := c.validateRequiredRealDirectory(projectSkillsPath); err != nil {
		return err
	}
	return nil
}

func (c *Catalog) validateRecordedCommonBase(commonPath string) error {
	if err := c.validateRequiredRealDirectory(filepath.Join(c.root, "utils")); err != nil {
		return err
	}
	if err := c.validateRequiredRealDirectory(commonPath); err != nil {
		return err
	}
	return nil
}

func (c *Catalog) resolveRecordedSkill(ref RecordedSkillRef, projectBase, commonBase string) (ResolvedSkillSource, error) {
	if !validName(ref.Name) {
		return ResolvedSkillSource{}, catalogError(ErrInvalidName, ref.Name, nil)
	}
	var base string
	switch ref.Source {
	case SkillSourceProject:
		base = projectBase
	case SkillSourceCommon:
		base = commonBase
	default:
		return ResolvedSkillSource{}, catalogError(ErrInvalidStructure, ref.Name, nil)
	}
	skillPath := filepath.Join(base, ref.Name)
	info, err := c.fs.Lstat(skillPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ResolvedSkillSource{Name: ref.Name, Source: ref.Source, State: SourceStateMissing}, nil
		}
		return ResolvedSkillSource{}, catalogError(ErrIO, ref.Name, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return ResolvedSkillSource{}, catalogError(ErrSymlink, ref.Name, nil)
	}
	if !info.IsDir() {
		return ResolvedSkillSource{}, catalogError(ErrInvalidStructure, ref.Name, nil)
	}
	manifestValid, err := c.inspectRecordedManifest(skillPath, ref.Name)
	if err != nil {
		return ResolvedSkillSource{}, err
	}
	if !manifestValid {
		return ResolvedSkillSource{Name: ref.Name, Source: ref.Source, State: SourceStateDisabled}, nil
	}
	if err := c.validateSkillContents(skillPath, ref.Name); err != nil {
		return ResolvedSkillSource{}, err
	}
	return ResolvedSkillSource{Name: ref.Name, Source: ref.Source, State: SourceStateActive, Path: skillPath}, nil
}

func (c *Catalog) inspectRecordedManifest(skillPath, name string) (bool, error) {
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
	if !info.Mode().IsRegular() {
		return false, catalogError(ErrInvalidStructure, name, nil)
	}
	return true, nil
}

func (c *Catalog) validateSkillContents(skillPath, name string) error {
	entries, err := c.fs.ReadDir(skillPath)
	if err != nil {
		return catalogError(ErrIO, name, err)
	}
	for _, entry := range entries {
		entryInfo, err := entry.Info()
		if err != nil {
			return catalogError(ErrIO, name, err)
		}
		mode := entryInfo.Mode()
		if mode&os.ModeSymlink != 0 {
			return catalogError(ErrSymlink, name, nil)
		}
		if !mode.IsDir() && !mode.IsRegular() {
			return catalogError(ErrInvalidStructure, name, nil)
		}
		if mode.IsDir() {
			if err := c.validateSkillContents(filepath.Join(skillPath, entry.Name()), name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Catalog) validateRequiredRealDirectory(path string) error {
	info, err := c.fs.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return catalogError(ErrInvalidStructure, filepath.Base(path), err)
		}
		return catalogError(ErrIO, filepath.Base(path), err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return catalogError(ErrSymlink, filepath.Base(path), nil)
	}
	if !info.IsDir() {
		return catalogError(ErrInvalidStructure, filepath.Base(path), nil)
	}
	return nil
}

func normalizeRecordedRefs(skills []RecordedSkillRef) ([]RecordedSkillRef, error) {
	deduped := make(map[string]RecordedSkillRef, len(skills))
	for _, skill := range skills {
		if skill.Source != SkillSourceProject && skill.Source != SkillSourceCommon {
			return nil, catalogError(ErrInvalidStructure, skill.Name, nil)
		}
		if !validName(skill.Name) {
			return nil, catalogError(ErrInvalidName, skill.Name, nil)
		}
		key := string(skill.Source) + "\x00" + skill.Name
		deduped[key] = skill
	}
	keys := make([]string, 0, len(deduped))
	for key := range deduped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	refs := make([]RecordedSkillRef, 0, len(keys))
	for _, key := range keys {
		refs = append(refs, deduped[key])
	}
	return refs, nil
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
