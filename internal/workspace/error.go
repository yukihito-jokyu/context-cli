package workspace

import "errors"

var (
	// ErrCurrentDirectory はカレントディレクトリの取得失敗を表します。
	ErrCurrentDirectory = errors.New("workspace current directory is unavailable")
	// ErrAbsolutePath は絶対パス化の失敗を表します。
	ErrAbsolutePath = errors.New("workspace absolute path conversion failed")
	// ErrNotExist はWorkspaceが存在しないことを表します。
	ErrNotExist = errors.New("workspace does not exist")
	// ErrNotDirectory はWorkspaceがディレクトリでないことを表します。
	ErrNotDirectory = errors.New("workspace is not a directory")
	// ErrSymlink はWorkspaceの経路にシンボリックリンクがあることを表します。
	ErrSymlink = errors.New("workspace path contains a symbolic link")
	// ErrIO はWorkspaceの検査失敗を表します。
	ErrIO = errors.New("workspace filesystem operation failed")
)

// Error はWorkspace検証エラーの分類と内部原因を保持します。
type Error struct {
	Kind error
	Err  error
}

// Error は安全な分類メッセージを返します。
func (e *Error) Error() string {
	return e.Kind.Error()
}

// Is は分類と内部原因の両方を判定可能にします。
func (e *Error) Is(target error) bool {
	return target == e.Kind || errors.Is(e.Err, target)
}

// Unwrap は内部原因を返します。
func (e *Error) Unwrap() error {
	return e.Err
}
