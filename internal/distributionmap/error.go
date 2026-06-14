package distributionmap

import "errors"

var (
	// ErrDiscovery は設定保存先の探索失敗を表します。
	ErrDiscovery = errors.New("distribution map discovery failed")
	// ErrSchema はmap.yamlの形式または値の不正を表します。
	ErrSchema = errors.New("distribution map schema is invalid")
	// ErrPermission は設定経路の権限不正を表します。
	ErrPermission = errors.New("distribution map permission is unsafe")
	// ErrSymlink は設定経路のシンボリックリンクを表します。
	ErrSymlink = errors.New("distribution map path contains a symbolic link")
	// ErrFileType は設定経路のファイル種別不正を表します。
	ErrFileType = errors.New("distribution map file type is invalid")
	// ErrLock は別プロセスがmap.yaml更新中であることを表します。
	ErrLock = errors.New("distribution map lock is held")
	// ErrConflict は比較更新の競合を表します。
	ErrConflict = errors.New("distribution map changed concurrently")
	// ErrIO は管理情報の入出力失敗を表します。
	ErrIO = errors.New("distribution map I/O failed")
	// ErrCommitted はmap.yaml置換後の後処理失敗を表します。
	ErrCommitted = errors.New("distribution map committed with an error")
)

// Error は管理情報操作の失敗分類と原因を保持します。
type Error struct {
	Operation string
	Kind      error
	Err       error
	Cleanup   error
}

func (e *Error) Error() string {
	return "distribution map " + e.Operation + " failed"
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
