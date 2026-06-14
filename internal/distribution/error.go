package distribution

import "errors"

var (
	// ErrStructure は供給元または配布先の構造不正を表します。
	ErrStructure = errors.New("distribution structure is invalid")
	// ErrUnsafePath は安全に扱えない経路を表します。
	ErrUnsafePath = errors.New("distribution path is unsafe")
	// ErrSymlink は経路またはSkill内のシンボリックリンクを表します。
	ErrSymlink = errors.New("distribution path contains a symbolic link")
	// ErrFileType は未対応のファイル種別を表します。
	ErrFileType = errors.New("distribution file type is invalid")
	// ErrPermission は期待外の権限を表します。
	ErrPermission = errors.New("distribution permission is invalid")
	// ErrConflict は計画時から状態が変化した競合を表します。
	ErrConflict = errors.New("distribution state changed concurrently")
	// ErrManaged は同じWorkspaceの管理記録が既に存在することを表します。
	ErrManaged = errors.New("workspace is already managed")
	// ErrIO は配布中の入出力失敗を表します。
	ErrIO = errors.New("distribution I/O failed")
	// ErrRollback はロールバック失敗を表します。
	ErrRollback = errors.New("distribution rollback failed")
	// ErrCommitted は管理情報コミット後の失敗を表します。
	ErrCommitted = errors.New("distribution committed with an error")
)

// Error は配布処理の分類、原因、後処理失敗を保持します。
type Error struct {
	Operation  string
	Kind       error
	Err        error
	Cleanup    error
	Unrestored []string
}

func (e *Error) Error() string {
	return "distribution " + e.Operation + " failed"
}

func (e *Error) Unwrap() []error {
	causes := []error{e.Kind}
	if e.Err != nil {
		causes = append(causes, e.Err)
	}
	if e.Cleanup != nil {
		causes = append(causes, e.Cleanup)
	}
	return causes
}

func newError(operation string, kind, cause error) error {
	return &Error{Operation: operation, Kind: kind, Err: cause}
}
