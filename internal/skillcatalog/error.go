package skillcatalog

import "errors"

var (
	// ErrInvalidName は候補名が単一の安全なパス要素でないことを表します。
	ErrInvalidName = errors.New("skill catalog name is invalid")
	// ErrNotFound は指定されたプロジェクトが存在しないことを表します。
	ErrNotFound = errors.New("skill catalog project does not exist")
	// ErrInvalidStructure は指定された候補の構造が不正であることを表します。
	ErrInvalidStructure = errors.New("skill catalog structure is invalid")
	// ErrNoCandidates は選択可能なプロジェクトが存在しないことを表します。
	ErrNoCandidates = errors.New("skill catalog has no candidates")
	// ErrSymlink は候補内にシンボリックリンクが存在することを表します。
	ErrSymlink = errors.New("skill catalog contains a symbolic link")
	// ErrIO は候補の検査に失敗したことを表します。
	ErrIO = errors.New("skill catalog filesystem operation failed")
)

// Error はカタログエラーの分類と安全な対象名を保持します。
type Error struct {
	Kind   error
	Target string
	Err    error
}

// Error は内部パスを含めず、分類と対象だけを返します。
func (e *Error) Error() string {
	if e.Target == "" {
		return e.Kind.Error()
	}
	return e.Kind.Error() + " (" + e.Target + ")"
}

// Is は分類と内部原因の両方を判定可能にします。
func (e *Error) Is(target error) bool {
	return target == e.Kind || errors.Is(e.Err, target)
}

// Unwrap は内部原因を返します。
func (e *Error) Unwrap() error {
	return e.Err
}
