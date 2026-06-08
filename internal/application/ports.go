package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/yukihito-jokyu/context-cli/internal/domain"
)

var (
	// ErrLockFailed は、排他的ファイルロックを取得できなかったことを示します。
	ErrLockFailed = errors.New("failed to acquire config lock")

	// ErrConfigConflict は、最後に読み込んだ後に設定が変更されたことを示します。
	ErrConfigConflict = errors.New("config conflict detected")

	// ErrPermissionTooBroad は、ディレクトリまたはファイルの権限が広すぎることを示します。
	ErrPermissionTooBroad = errors.New("config file or directory permissions are too broad")

	// ErrChangeAborted は、ユーザーが設定変更を拒否または中断したことを示します。
	ErrChangeAborted = errors.New("configuration change aborted")
)

// RepositoryValidationError は、リポジトリの検証に失敗したことを示します。
type RepositoryValidationError struct {
	Errors []domain.ValidationError
}

func (e *RepositoryValidationError) Error() string {
	return fmt.Sprintf("repository validation failed: %d errors", len(e.Errors))
}

// UIPort は、リポジトリ初期化時にユーザーと対話するためのインターフェースです。
type UIPort interface {
	// ConfirmChange は、リポジトリパスを currentPath から newPath へ変更するかユーザーに確認します。
	// ユーザーが承認した場合は true、拒否した場合は false、対話が中断された場合はエラーを返します。
	ConfirmChange(ctx context.Context, currentPath, newPath string) (bool, error)
}

// ConfigRepository は、グローバル設定を永続化および読み込むためのインターフェースです。
type ConfigRepository interface {
	// Load は、永続ストレージから設定を読み込みます。
	// 設定ファイルが存在しない場合は、fs.ErrNotExist をラップしたエラーを返す必要があります。
	// ファイルまたはディレクトリの権限が広すぎる場合は、ErrPermissionTooBroad を返す必要があります。
	Load(ctx context.Context) (domain.Config, error)

	// Save は、設定を永続ストレージへ保存します。
	// expectedOld が nil でない場合は、既存設定が expectedOld と一致することを確認します。
	// expectedOld が nil の場合は、設定ファイルが存在しないことを確認します。
	// ロックを即座に取得できない場合は、ErrLockFailed を返す必要があります。
	// 既存のディレクトリまたはファイルの権限が広すぎる場合は、ErrPermissionTooBroad を返す必要があります。
	// 競合を検出した場合は、ErrConfigConflict を返す必要があります。
	Save(ctx context.Context, config domain.Config, expectedOld *domain.Config) error
}
