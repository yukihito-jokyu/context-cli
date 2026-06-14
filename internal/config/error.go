package config

import "errors"

var (
	// ErrDiscovery は設定保存先の探索失敗を表します。
	ErrDiscovery = errors.New("configuration discovery failed")
	// ErrFormat は設定ファイルの形式不正を表します。
	ErrFormat = errors.New("configuration format is invalid")
	// ErrSchema は設定値またはスキーマの不正を表します。
	ErrSchema = errors.New("configuration schema is invalid")
	// ErrPermission は安全でない権限を表します。
	ErrPermission = errors.New("configuration permission is unsafe")
	// ErrSymlink は設定経路にシンボリックリンクが含まれることを表します。
	ErrSymlink = errors.New("configuration path contains a symbolic link")
	// ErrFileType は設定経路のファイル種別が不正であることを表します。
	ErrFileType = errors.New("configuration file type is invalid")
	// ErrLockConflict は別プロセスが設定更新中であることを表します。
	ErrLockConflict = errors.New("configuration lock is held")
	// ErrUpdateConflict は読み込み後に設定が変更されたことを表します。
	ErrUpdateConflict = errors.New("configuration was changed concurrently")
	// ErrIO は設定の入出力失敗を表します。
	ErrIO = errors.New("configuration I/O failed")
	// ErrCleanup はコミット前のクリーンアップ失敗を表します。
	ErrCleanup = errors.New("configuration cleanup failed")
	// ErrCommitted は設定更新後の永続化またはクリーンアップ失敗を表します。
	ErrCommitted = errors.New("configuration was committed with an error")
)

// Error は内部情報を公開せずに設定操作の失敗分類と原因を保持します。
type Error struct {
	Operation string
	Kind      error
	Err       error
	Cleanup   error
}

func (e *Error) Error() string {
	return "configuration " + e.Operation + " failed"
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

func newError(operation string, kind, err error) error {
	return &Error{
		Operation: operation,
		Kind:      kind,
		Err:       err,
	}
}

func classifyUpdateError(committed bool, primaryErr, cleanupErr error) error {
	if committed {
		if primaryErr == nil && cleanupErr == nil {
			return nil
		}
		return &Error{
			Operation: "commit",
			Kind:      ErrCommitted,
			Err:       primaryErr,
			Cleanup:   cleanupErr,
		}
	}
	if primaryErr != nil {
		if cleanupErr == nil {
			return primaryErr
		}
		var configErr *Error
		if errors.As(primaryErr, &configErr) {
			return &Error{
				Operation: configErr.Operation,
				Kind:      configErr.Kind,
				Err:       configErr.Err,
				Cleanup:   cleanupErr,
			}
		}
		return &Error{
			Operation: "update",
			Kind:      ErrIO,
			Err:       primaryErr,
			Cleanup:   cleanupErr,
		}
	}
	if cleanupErr != nil {
		return &Error{
			Operation: "cleanup",
			Kind:      ErrCleanup,
			Cleanup:   cleanupErr,
		}
	}
	return nil
}
