// Package application はユースケースを調整し、外部I/Oポートを定義します。
package application

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/yukihito-jokyu/context-cli/internal/domain"
)

// InitRepositoryUseCase は、指定パスの同一性評価、リポジトリ内容の検証、
// 既存設定の有無に応じたインタラクティブなUIプロンプト確認呼び出し、
// および書き込み保存のオーケストレーションを実装します。
type InitRepositoryUseCase struct {
	configRepo ConfigRepository
	ui         UIPort
	fs         domain.FileSystem
}

// NewInitRepositoryUseCase は、新しい InitRepositoryUseCase を作成します。
func NewInitRepositoryUseCase(
	configRepo ConfigRepository,
	ui UIPort,
	fs domain.FileSystem,
) *InitRepositoryUseCase {
	return &InitRepositoryUseCase{
		configRepo: configRepo,
		ui:         ui,
		fs:         fs,
	}
}

// Run は、リポジトリ初期化ユースケースを実行します。
//
// 処理フロー:
//  1. 入力パスを正規化する
//  2. 既存設定の読み込みを試みる
//  3. リポジトリの構造と権限を検証する
//  4. 同一リポジトリ判定による分岐
func (uc *InitRepositoryUseCase) Run(ctx context.Context, inputPath string) error {
	normalizedPath, err := domain.NormalizeRepositoryPath(inputPath)
	if err != nil {
		return fmt.Errorf("failed to normalize repository path: %w", err)
	}

	existingConfig, configExists, err := uc.loadExistingConfig(ctx)
	if err != nil {
		return err
	}

	valErrs, err := uc.validateRepository(ctx, normalizedPath)
	if err != nil {
		return err
	}

	if configExists {
		return uc.handleExistingConfig(ctx, existingConfig, normalizedPath, valErrs)
	}

	return uc.handleFirstTimeSetup(ctx, normalizedPath, valErrs)
}

// loadExistingConfig は既存設定の読み込みを試みます。
// 設定ファイルが存在しない場合は configExists=false を返します。
// スキーマ検証失敗（未対応バージョン、不正値等）の場合はエラーを返します。
func (uc *InitRepositoryUseCase) loadExistingConfig(ctx context.Context) (domain.Config, bool, error) {
	config, err := uc.configRepo.Load(ctx)
	if err == nil {
		return config, true, nil
	}

	if errors.Is(err, fs.ErrNotExist) {
		return domain.Config{}, false, nil
	}

	return domain.Config{}, false, fmt.Errorf("failed to load existing config: %w", err)
}

// validateRepository はリポジトリの構造と権限を検証します。
func (uc *InitRepositoryUseCase) validateRepository(
	ctx context.Context,
	path string,
) ([]domain.ValidationError, error) {
	validator := domain.NewRepositoryValidator(uc.fs)
	valErrs, err := validator.Validate(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("repository validation system error: %w", err)
	}

	return valErrs, nil
}

// handleExistingConfig は既存設定が存在する場合の分岐を処理します。
func (uc *InitRepositoryUseCase) handleExistingConfig(
	ctx context.Context,
	existingConfig domain.Config,
	normalizedPath string,
	valErrs []domain.ValidationError,
) error {
	isSameRepo := existingConfig.RepositoryPath == normalizedPath

	if isSameRepo {
		return uc.handleSameRepository(valErrs)
	}

	return uc.handleDifferentRepository(ctx, existingConfig, normalizedPath, valErrs)
}

// handleSameRepository は同一リポジトリ再実行を処理します。
// プロンプトと書き込みをスキップし、検証結果に基づき正常終了またはエラー終了します。
func (*InitRepositoryUseCase) handleSameRepository(valErrs []domain.ValidationError) error {
	if len(valErrs) > 0 {
		return &RepositoryValidationError{Errors: valErrs}
	}

	return nil
}

// handleDifferentRepository は異なるリポジトリへの設定変更を処理します。
func (uc *InitRepositoryUseCase) handleDifferentRepository(
	ctx context.Context,
	existingConfig domain.Config,
	normalizedPath string,
	valErrs []domain.ValidationError,
) error {
	if len(valErrs) > 0 {
		return &RepositoryValidationError{Errors: valErrs}
	}

	approved, err := uc.ui.ConfirmChange(ctx, existingConfig.RepositoryPath, normalizedPath)
	if err != nil {
		return fmt.Errorf("confirmation aborted: %w", err)
	}

	if !approved {
		return ErrChangeAborted
	}

	return uc.saveConfig(ctx, normalizedPath, &existingConfig)
}

// handleFirstTimeSetup は初回設定を処理します。
// プロンプトなしで永続化します。
func (uc *InitRepositoryUseCase) handleFirstTimeSetup(
	ctx context.Context,
	normalizedPath string,
	valErrs []domain.ValidationError,
) error {
	if len(valErrs) > 0 {
		return &RepositoryValidationError{Errors: valErrs}
	}

	return uc.saveConfig(ctx, normalizedPath, nil)
}

// saveConfig は設定を永続化します。
func (uc *InitRepositoryUseCase) saveConfig(
	ctx context.Context,
	repoPath string,
	expectedOld *domain.Config,
) error {
	newConfig := domain.Config{
		Version:        domain.CurrentConfigVersion,
		RepositoryPath: repoPath,
	}

	if err := uc.configRepo.Save(ctx, newConfig, expectedOld); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}
