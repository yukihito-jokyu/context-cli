package repository

import "errors"

var (
	// ErrPathNotExist は指定されたリポジトリが存在しないことを表します。
	ErrPathNotExist = errors.New("context repository path does not exist")
	// ErrNotDirectory は指定されたリポジトリがディレクトリでないことを表します。
	ErrNotDirectory = errors.New("context repository path is not a directory")
	// ErrRequiredStructure は必須ディレクトリが不足していることを表します。
	ErrRequiredStructure = errors.New("context repository required structure is missing")
	// ErrSymlink は検査対象でシンボリックリンクを検出したことを表します。
	ErrSymlink = errors.New("context repository contains a symbolic link")
	// ErrIO は検証中のファイルシステム操作に失敗したことを表します。
	ErrIO = errors.New("context repository filesystem operation failed")
)

// ValidationError は安全な表示用対象と内部原因を分離して保持します。
type ValidationError struct {
	Kind   error
	Target string
	Err    error
}

// Error は内部パスを含めず、分類と表示用対象だけを返します。
func (e *ValidationError) Error() string {
	return e.Kind.Error() + " (" + e.Target + ")"
}

// Is は検証エラーの分類と内部原因の両方を判定可能にします。
func (e *ValidationError) Is(target error) bool {
	return target == e.Kind || errors.Is(e.Err, target)
}

// Unwrap は呼び出し側が内部原因を判定できるようにします。
func (e *ValidationError) Unwrap() error {
	return e.Err
}
