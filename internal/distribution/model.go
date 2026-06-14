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
	// PathKindFIFO はFIFOを表します。
	PathKindFIFO
	// PathKindSocket はソケットを表します。
	PathKindSocket
	// PathKindDevice はデバイスファイルを表します。
	PathKindDevice
	// PathKindOther は上記以外のシンボリックリンクでない種別を表します。
	PathKindOther
	// PathKindAny は末端のシンボリックリンクでない任意種別を要求します。
	PathKindAny
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

// RenameOperation は同一性確認付きの名前変更を表します。
type RenameOperation struct {
	OldPath     string
	OldParent   PathExpectation
	OldExpected PathExpectation
	NewPath     string
	NewParent   PathExpectation
	NewExpected PathExpectation
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
//
// 同期（SyncPlanner）でも更新対象をCreates、削除対象をDeletesへ格納して再利用します。
// IsSync をtrueにした場合は Result の同期用集計フィールドへ一意Skill件数を格納します。
type Plan struct {
	ExpectedRevision Revision
	Workspace        WorkspaceRecord
	Creates          []CreateOperation
	Deletes          []DeleteOperation
	// IsSync は同期計画として実行するかを表します。
	IsSync bool
}

// Result は初回配布および同期の実行結果を表します。
//
// 初回配布（context add）は Created と Destinations を使用します。
// 同期（context sync）は UniqueUpdated と UniqueDeleted を一意Skill名単位の件数として使用します。
type Result struct {
	Created       int
	Destinations  []Destination
	UniqueUpdated int
	UniqueDeleted int
}

// RecordedSkill は同期入力となる記録済みSkillを表します。
type RecordedSkill struct {
	Name        string
	Source      SkillSource
	Destination Destination
	// RelativePath は配布先のWorkspace相対パスです。
	RelativePath string
	// RecordedHash は前回配布時に記録した供給元ハッシュです。
	RecordedHash string
}

// SyncInput は同期計画の入力を表します。
type SyncInput struct {
	// WorkspaceRoot は同期対象Workspaceの正規化済み絶対パスです。
	WorkspaceRoot string
	// Project は記録済みプロジェクト名です。
	Project string
	// Destinations は記録済み配布先です。
	Destinations []Destination
	// Skills は記録済みSkill単位の入力です。
	Skills []RecordedSkill
}

// SourceState は解決済み供給元の状態分類を表します。
type SourceState uint8

const (
	// SourceStateActive は供給元が有効で更新・維持対象になることを表します。
	SourceStateActive SourceState = iota + 1
	// SourceStateMissing は個別Skillパスが欠落したことを表します。
	SourceStateMissing
	// SourceStateDisabled はSKILL.mdが欠落して無効化されたことを表します。
	SourceStateDisabled
)

// ResolvedSource は同期計画が利用する解決済み供給元を表します。
//
// Name と Source は同じSkill名・供給元種別を持つ全RecordedSkillで共有します。
// State がActive以外の場合は Path・Hash・SourcePathStates・ManifestState を空にします。
type ResolvedSource struct {
	Name             string
	Source           SkillSource
	State            SourceState
	Path             string
	Hash             string
	SourcePathStates []PathExpectation
	ManifestState    PathExpectation
}

// SyncOperationKind は同期計画の操作種別を表します。
type SyncOperationKind uint8

const (
	// SyncOperationKeep は更新も削除もしない維持を表します。
	SyncOperationKeep SyncOperationKind = iota + 1
	// SyncOperationUpdate は配布先を供給元の現在内容へ更新することを表します。
	SyncOperationUpdate
	// SyncOperationDelete は供給元消失または無効化による削除を表します。
	SyncOperationDelete
)

// SyncOperation は同期計画の1操作を表します。
//
// Plan.Creates または Plan.Deletes へ変換してExecutorへ渡します。
type SyncOperation struct {
	Kind         SyncOperationKind
	Name         string
	Source       SkillSource
	Destination  Destination
	RelativePath string
	// FinalPath は配布先の絶対パスです。
	FinalPath string
	// RecordedHash は前回配布時ハッシュです。
	RecordedHash string
	// CurrentHash は今回計画時の配布先ハッシュです（欠落時は空）。
	CurrentHash string
	// IsLocalEdit はローカル変更（欠落・ハッシュ差異・種別変化）を表します。
	IsLocalEdit bool
	// SourcePath は更新操作の供給元絶対パスです（Keep/Deleteでは空）。
	SourcePath string
	// SourceHash は更新操作の供給元現在ハッシュです（Keep/Deleteでは空）。
	SourceHash string
	// SourcePathStates は供給元経路の期待状態です（Keep/Deleteでは空）。
	SourcePathStates []PathExpectation
	// TargetPathStates は配布先経路の期待状態です。
	TargetPathStates []PathExpectation
}

// SyncPlan は同期計画を表します。
//
// Plan へ変換可能な操作一覧と、確認対象・結果件数の元情報を保持します。
type SyncPlan struct {
	ExpectedRevision Revision
	WorkspaceRoot    string
	Project          string
	Destinations     []Destination
	Updates          []SyncOperation
	Deletes          []SyncOperation
	Keeps            []SyncOperation
	LocalChanges     []SyncOperation
	ResolvedSources  []ResolvedSource
	// UpdatedWorkspace はコミットへ渡す更新後Workspace記録です。
	UpdatedWorkspace WorkspaceRecord
}
