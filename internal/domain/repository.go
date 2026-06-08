// Package domain は、外部技術に依存しないドメイン規則を定義します。
package domain

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
)

// FileStatus は、信頼性チェックに必要なファイル情報を表します。
type FileStatus interface {
	IsDir() bool
	IsRegular() bool
	IsSymlink() bool
	Mode() fs.FileMode
}

// FileEntry は、ディレクトリのエントリを表します。
type FileEntry interface {
	Name() string
	IsDir() bool
}

// FileSystem は、OSのファイルシステムコールを抽象化します。
type FileSystem interface {
	LStat(ctx context.Context, path string) (FileStatus, error)
	ReadDir(ctx context.Context, path string) ([]FileEntry, error)
	ReadFile(ctx context.Context, path string) ([]byte, error)
}

// ValidationError は、リポジトリの構造または権限における単一の検証失敗を表します。
type ValidationError struct {
	Path   string
	Reason string
}

// RepositoryValidator は、リポジトリの構造と権限を検証します。
type RepositoryValidator struct {
	fs FileSystem
}

// NewRepositoryValidator は、新しい RepositoryValidator を作成します。
func NewRepositoryValidator(fs FileSystem) *RepositoryValidator {
	return &RepositoryValidator{fs: fs}
}

const writePermissionMask = 0o022

// Validate は、リポジトリの構造と権限をチェックします。
func (v *RepositoryValidator) Validate(ctx context.Context, repoRoot string) ([]ValidationError, error) {
	if repoRoot == "" || !filepath.IsAbs(repoRoot) {
		return nil, fmt.Errorf("%w: repository path must be absolute and normalized: %s", ErrInvalidRepositoryPath, repoRoot)
	}

	var valErrs []ValidationError

	addError := func(absPath string, reason string) {
		rel, err := filepath.Rel(repoRoot, absPath)
		if err != nil {
			rel = absPath
		}
		valErrs = append(valErrs, ValidationError{
			Path:   rel,
			Reason: reason,
		})
	}

	// 1. 祖先パスのチェック（ファイルシステムのルートから repoRoot の親ディレクトリまで）
	if err := v.validateAncestors(ctx, repoRoot, addError); err != nil {
		//nolint:nilerr // システムエラーの代わりに検証エラーを返すのは意図的です
		return valErrs, nil
	}

	// 2. repoRoot 自体のチェック
	if err := v.validateRepoRoot(ctx, repoRoot, addError); err != nil {
		//nolint:nilerr // システムエラーの代わりに検証エラーを返すのは意図的です
		return valErrs, nil
	}

	// リポジトリのルートディレクトリの読み取りを試行
	rootEntries, err := v.fs.ReadDir(ctx, repoRoot)
	if err != nil {
		addError(repoRoot, fmt.Sprintf("repository path is not readable/searchable: %v", err))
		return valErrs, nil
	}

	// 3. projects/ および utils/skills/ の検証
	projectsExist := v.validateProjectsDir(ctx, repoRoot, rootEntries, addError)
	utilsSkillsExist := v.validateUtilsSkillsDir(ctx, repoRoot, rootEntries, addError)

	// 4. プロジェクトおよびプロジェクト内のスキル（skills）の検証
	if projectsExist {
		v.validateProjects(ctx, repoRoot, addError)
	}

	// 5. utils 内のスキルの検証
	if utilsSkillsExist {
		utilsSkillsPath := filepath.Join(repoRoot, "utils", "skills")
		v.validateSkillsDir(ctx, utilsSkillsPath, addError)
	}

	return valErrs, nil
}

func (v *RepositoryValidator) validateAncestors(ctx context.Context, repoRoot string, addError func(string, string)) error {
	ancestors := getAncestors(repoRoot)
	for _, path := range ancestors {
		status, err := v.fs.LStat(ctx, path)
		if err != nil {
			addError(path, fmt.Sprintf("cannot access parent path: %v", err))
			return fmt.Errorf("failed to lstat ancestor path %s: %w", path, err)
		}

		if status.IsSymlink() {
			addError(path, "symbolic link is not allowed in repository path components")
			return fmt.Errorf("%w: symlink detected in ancestors", ErrInvalidRepositoryPath)
		}

		// 環境固有の問題を避けるため、ルート「/」に対する書き込み権限のチェックはスキップします。
		if path != "/" && (status.Mode()&writePermissionMask) != 0 {
			addError(path, "group or others write permission is not allowed in repository path components")
			return fmt.Errorf("%w: group/other write permission detected in ancestors", ErrInvalidRepositoryPath)
		}
	}
	return nil
}

func (v *RepositoryValidator) validateRepoRoot(ctx context.Context, repoRoot string, addError func(string, string)) error {
	status, err := v.fs.LStat(ctx, repoRoot)
	if err != nil {
		addError(repoRoot, "repository path does not exist or is not accessible")
		return fmt.Errorf("failed to lstat repository root %s: %w", repoRoot, err)
	}

	if status.IsSymlink() {
		addError(repoRoot, "repository path cannot be a symbolic link")
		return fmt.Errorf("%w: repo root is a symlink", ErrInvalidRepositoryPath)
	}

	if !status.IsDir() {
		addError(repoRoot, "repository path must be a directory")
		return fmt.Errorf("%w: repo root is not a directory", ErrInvalidRepositoryPath)
	}

	if (status.Mode() & writePermissionMask) != 0 {
		addError(repoRoot, "repository path has group or others write permission")
		return fmt.Errorf("%w: repo root has group/other write permission", ErrInvalidRepositoryPath)
	}

	return nil
}

func (v *RepositoryValidator) validateProjectsDir(ctx context.Context, repoRoot string, rootEntries []FileEntry, addError func(string, string)) bool {
	var projectsEntry FileEntry
	for _, entry := range rootEntries {
		if entry.Name() == "projects" {
			projectsEntry = entry
			break
		}
	}

	projectsPath := filepath.Join(repoRoot, "projects")
	if projectsEntry == nil {
		addError(projectsPath, "projects directory is missing")
		return false
	}

	status, err := v.fs.LStat(ctx, projectsPath)
	if err != nil {
		addError(projectsPath, "failed to inspect projects directory")
		return false
	}

	if !status.IsDir() {
		addError(projectsPath, "projects path must be a directory")
		return false
	}

	if status.IsSymlink() {
		addError(projectsPath, "projects directory cannot be a symbolic link")
		return false
	}

	if (status.Mode() & writePermissionMask) != 0 {
		addError(projectsPath, "projects directory has group or others write permission")
		return false
	}

	return true
}

func (v *RepositoryValidator) validateUtilsSkillsDir(ctx context.Context, repoRoot string, rootEntries []FileEntry, addError func(string, string)) bool {
	var utilsEntry FileEntry
	for _, entry := range rootEntries {
		if entry.Name() == "utils" {
			utilsEntry = entry
			break
		}
	}

	utilsSkillsPath := filepath.Join(repoRoot, "utils", "skills")
	if utilsEntry == nil {
		addError(utilsSkillsPath, "utils/skills directory is missing")
		return false
	}

	utilsPath := filepath.Join(repoRoot, "utils")
	utilsStatus, err := v.fs.LStat(ctx, utilsPath)
	if err != nil {
		addError(utilsPath, "failed to inspect utils directory")
		return false
	}

	if !utilsStatus.IsDir() {
		addError(utilsPath, "utils path must be a directory")
		return false
	}

	if utilsStatus.IsSymlink() {
		addError(utilsPath, "utils directory cannot be a symbolic link")
		return false
	}

	if (utilsStatus.Mode() & writePermissionMask) != 0 {
		addError(utilsPath, "utils directory has group or others write permission")
		return false
	}

	skillsStatus, err := v.fs.LStat(ctx, utilsSkillsPath)
	if err != nil {
		addError(utilsSkillsPath, "utils/skills directory is missing")
		return false
	}

	if !skillsStatus.IsDir() {
		addError(utilsSkillsPath, "utils/skills path must be a directory")
		return false
	}

	if skillsStatus.IsSymlink() {
		addError(utilsSkillsPath, "utils/skills directory cannot be a symbolic link")
		return false
	}

	if (skillsStatus.Mode() & writePermissionMask) != 0 {
		addError(utilsSkillsPath, "utils/skills directory has group or others write permission")
		return false
	}

	return true
}

func (v *RepositoryValidator) validateProjects(ctx context.Context, repoRoot string, addError func(string, string)) {
	projectsPath := filepath.Join(repoRoot, "projects")
	projectEntries, err := v.fs.ReadDir(ctx, projectsPath)
	if err != nil {
		addError(projectsPath, fmt.Sprintf("projects directory is not readable/searchable: %v", err))
		return
	}

	var projectDirs []string
	for _, entry := range projectEntries {
		if entry.IsDir() && entry.Name() != "" {
			projectDirs = append(projectDirs, entry.Name())
		}
	}

	if len(projectDirs) == 0 {
		addError(projectsPath, "projects directory must contain at least one project")
		return
	}

	for _, projName := range projectDirs {
		v.validateProject(ctx, projectsPath, projName, addError)
	}
}

func (v *RepositoryValidator) validateProject(ctx context.Context, projectsPath string, projName string, addError func(string, string)) {
	projPath := filepath.Join(projectsPath, projName)
	projStatus, err := v.fs.LStat(ctx, projPath)
	if err != nil {
		addError(projPath, "failed to inspect project directory")
		return
	}

	if projStatus.IsSymlink() {
		addError(projPath, "project directory cannot be a symbolic link")
		return
	}

	if (projStatus.Mode() & writePermissionMask) != 0 {
		addError(projPath, "project directory has group or others write permission")
		return
	}

	projSkillsPath := filepath.Join(projPath, "skills")
	projSkillsStatus, err := v.fs.LStat(ctx, projSkillsPath)
	if err != nil {
		addError(projSkillsPath, "skills directory is missing in project")
		return
	}

	if !projSkillsStatus.IsDir() {
		addError(projSkillsPath, "skills path must be a directory")
		return
	}

	if projSkillsStatus.IsSymlink() {
		addError(projSkillsPath, "skills directory cannot be a symbolic link")
		return
	}

	if (projSkillsStatus.Mode() & writePermissionMask) != 0 {
		addError(projSkillsPath, "skills directory has group or others write permission")
		return
	}

	v.validateSkillsDir(ctx, projSkillsPath, addError)
}

func (v *RepositoryValidator) validateSkillsDir(ctx context.Context, skillsPath string, addError func(string, string)) {
	entries, err := v.fs.ReadDir(ctx, skillsPath)
	if err != nil {
		addError(skillsPath, fmt.Sprintf("skills directory is not readable/searchable: %v", err))
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			v.validateSkillSubDir(ctx, skillsPath, entry.Name(), addError)
		}
	}
}

func (v *RepositoryValidator) validateSkillSubDir(ctx context.Context, skillsPath string, skillName string, addError func(string, string)) {
	skillPath := filepath.Join(skillsPath, skillName)
	skillStatus, err := v.fs.LStat(ctx, skillPath)
	if err != nil {
		addError(skillPath, "failed to inspect skill directory")
		return
	}

	if skillStatus.IsSymlink() {
		addError(skillPath, "skill directory cannot be a symbolic link")
		return
	}

	if (skillStatus.Mode() & writePermissionMask) != 0 {
		addError(skillPath, "skill directory has group or others write permission")
		return
	}

	if _, err := v.fs.ReadDir(ctx, skillPath); err != nil {
		addError(skillPath, fmt.Sprintf("skill directory is not readable/searchable: %v", err))
		return
	}

	skillMdPath := filepath.Join(skillPath, "SKILL.md")
	mdStatus, err := v.fs.LStat(ctx, skillMdPath)
	if err != nil {
		addError(skillMdPath, "SKILL.md is missing in skill directory")
		return
	}

	if mdStatus.IsDir() {
		addError(skillMdPath, "SKILL.md must be a regular file, but found a directory")
		return
	}

	if mdStatus.IsSymlink() {
		addError(skillMdPath, "SKILL.md cannot be a symbolic link")
		return
	}

	if !mdStatus.IsRegular() {
		addError(skillMdPath, "SKILL.md must be a regular file")
		return
	}

	if (mdStatus.Mode() & writePermissionMask) != 0 {
		addError(skillMdPath, "SKILL.md has group or others write permission")
		return
	}

	if _, err := v.fs.ReadFile(ctx, skillMdPath); err != nil {
		addError(skillMdPath, fmt.Sprintf("failed to read SKILL.md: %v", err))
	}
}

func getAncestors(path string) []string {
	var components []string
	curr := filepath.Clean(path)
	parent := filepath.Dir(curr)
	if parent == curr {
		return nil
	}
	curr = parent
	for {
		components = append(components, curr)
		p := filepath.Dir(curr)
		if p == curr {
			break
		}
		curr = p
	}
	// ルートから親ディレクトリへの順でチェックするため反転
	for i, j := 0, len(components)-1; i < j; i, j = i+1, j-1 {
		components[i], components[j] = components[j], components[i]
	}
	return components
}
