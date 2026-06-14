package distribution

import "io/fs"

// SkillSource はSkillの供給元を表します。
type SkillSource string

const (
	// SkillSourceProject はプロジェクト固有Skillを表します。
	SkillSourceProject SkillSource = "project"
	// SkillSourceCommon は共通Skillを表します。
	SkillSourceCommon SkillSource = "common"
)

// Destination はSkillの配布先を表します。
type Destination string

const (
	// DestinationCodex はCodex向け配布先を表します。
	DestinationCodex Destination = "codex"
	// DestinationClaude はClaude向け配布先を表します。
	DestinationClaude Destination = "claude"
)

// SelectedSkill は選択されたSkillと供給元を表します。
type SelectedSkill struct {
	Name       string
	Source     SkillSource
	SourcePath string
}

// Selection は対話で確定した配布対象を表します。
type Selection struct {
	WorkspaceRoot string
	Project       string
	Skills        []SelectedSkill
	Destinations  []Destination
}

// Revision は管理情報全体の比較更新用リビジョンを表します。
type Revision string

const (
	// EmptyRevision はmap.yaml未作成状態を表します。
	EmptyRevision Revision = "absent"
)

// SkillRecord は1つのSkill配布記録を表します。
type SkillRecord struct {
	Name         string
	Source       SkillSource
	Destination  Destination
	RelativePath string
	Hash         string
}

// WorkspaceRecord は1つのWorkspaceに紐づく初回配布記録を表します。
type WorkspaceRecord struct {
	WorkspaceRoot string
	Project       string
	Destinations  []Destination
	Skills        []SkillRecord
}

// MapSnapshot は管理情報と比較更新用リビジョンを表します。
type MapSnapshot struct {
	Revision   Revision
	Workspaces map[string]WorkspaceRecord
}

// CommitResult は管理情報のコミット点を通過したかを表します。
type CommitResult struct {
	Committed bool
}

// MapStore は管理情報の読込とロック付き比較更新を表します。
type MapStore interface {
	Load() (MapSnapshot, error)
	Begin(expected Revision) (MapTransaction, MapSnapshot, error)
}

// MapTransaction はロック下の初回記録保存を表します。
type MapTransaction interface {
	Commit(workspace WorkspaceRecord) (CommitResult, error)
	Close() error
}

// PathKind は期待するファイル種別を表します。
type PathKind uint8

const (
	// PathKindDirectory は実ディレクトリを表します。
	PathKindDirectory PathKind = iota + 1
	// PathKindRegularFile は通常ファイルを表します。
	PathKindRegularFile
)

// PathExpectation は計画時に固定した経路要素の状態を表します。
type PathExpectation struct {
	Path   string
	Exists bool
	Kind   PathKind
	Perm   fs.FileMode
	Device uint64
	Inode  uint64
}

// CreateOperation は初回配布で作成する1つのSkillを表します。
type CreateOperation struct {
	Name             string
	Source           SkillSource
	SourcePath       string
	Destination      Destination
	RelativePath     string
	FinalPath        string
	Hash             string
	SourcePathStates []PathExpectation
	TargetPathStates []PathExpectation
	IsConflict       bool
	IsLocalEdit      bool
}

// DeleteOperation は前回配布済みで今回削除する1つのSkillを表します。
type DeleteOperation struct {
	Name             string
	Destination      Destination
	RelativePath     string
	FinalPath        string
	TargetPathStates []PathExpectation
	IsLocalEdit      bool
}

// Plan はロック下で再検証する初回配布計画を表します。
type Plan struct {
	ExpectedRevision Revision
	Workspace        WorkspaceRecord
	Creates          []CreateOperation
	Deletes          []DeleteOperation
}

// Result は初回配布の実行結果を表します。
type Result struct {
	Created      int
	Destinations []Destination
}
