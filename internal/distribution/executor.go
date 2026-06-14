package distribution

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"sort"
)

const distributionDirectoryPermission fs.FileMode = 0o755

// Executor は初回配布計画をロック下で実行します。
type Executor struct {
	fileSystem FileSystem
	mapStore   MapStore
}

// NewExecutor は初回配布Executorを返します。
func NewExecutor(fileSystem FileSystem, mapStore MapStore) *Executor {
	return &Executor{fileSystem: fileSystem, mapStore: mapStore}
}

type stagedOperation struct {
	operation CreateOperation
	path      string
}

// Execute は計画を再検証し、配布と管理情報保存を一つの処理境界で実行します。
//
//nolint:gocognit,cyclop // コミット点を含む処理順とロールバック分岐を一つの関数で明示します。
func (e *Executor) Execute(plan Plan) (Result, error) {
	transaction, _, err := e.mapStore.Begin(plan.ExpectedRevision)
	if err != nil {
		return Result{}, newError("begin", ErrIO, err)
	}

	createdDirectories := []string{}
	staged := []stagedOperation{}
	backedUp := []backedUpOperation{}
	placed := []CreateOperation{}
	committed := false
	var primaryErr error

	if err := e.revalidatePlan(plan); err != nil {
		primaryErr = err
	} else if createdDirectories, err = e.createDirectories(plan); err != nil {
		primaryErr = err
	} else if backedUp, err = e.backupAll(plan); err != nil {
		primaryErr = err
	} else if staged, err = e.stageAll(plan); err != nil {
		primaryErr = err
	} else if err = e.revalidateBeforePlacement(plan, staged); err != nil {
		primaryErr = err
	} else if placed, err = e.placeAll(staged); err != nil {
		primaryErr = err
	} else {
		result, commitErr := transaction.Commit(plan.Workspace)
		committed = result.Committed || errors.Is(commitErr, ErrCommitted)
		if commitErr != nil {
			if committed {
				primaryErr = &Error{Operation: "commit", Kind: ErrCommitted, Err: commitErr}
			} else {
				primaryErr = newError("commit", ErrIO, commitErr)
			}
		} else if !committed {
			primaryErr = newError("commit", ErrIO, nil)
		}
	}

	if primaryErr != nil && !committed {
		unrestored, rollbackErr := e.rollback(staged, placed, backedUp, createdDirectories)
		if rollbackErr != nil {
			primaryErr = &Error{
				Operation:  "rollback",
				Kind:       ErrRollback,
				Err:        primaryErr,
				Cleanup:    rollbackErr,
				Unrestored: unrestored,
			}
		}
	}
	if committed {
		var cleanupErr error
		if err := e.cleanupStaging(staged); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
		for _, backup := range backedUp {
			if err := e.fileSystem.RemoveAll(backup.backupPath); err != nil {
				cleanupErr = errors.Join(cleanupErr, err)
			}
		}
		if cleanupErr != nil {
			primaryErr = errors.Join(primaryErr, &Error{
				Operation: "cleanup",
				Kind:      ErrCommitted,
				Cleanup:   cleanupErr,
			})
		}
	}

	closeErr := transaction.Close()
	if primaryErr != nil {
		return Result{}, joinPrimaryAndClose(primaryErr, closeErr, committed)
	}
	if closeErr != nil {
		return Result{}, &Error{Operation: "close", Kind: ErrCommitted, Cleanup: closeErr}
	}
	return Result{
		Created:      len(plan.Creates),
		Destinations: append([]Destination(nil), plan.Workspace.Destinations...),
	}, nil
}

func (e *Executor) revalidatePlan(plan Plan) error {
	for _, operation := range plan.Creates {
		if err := e.fileSystem.Revalidate(operation.SourcePathStates); err != nil {
			return newError("revalidate source", ErrConflict, err)
		}
		if err := e.fileSystem.Revalidate(operation.TargetPathStates); err != nil {
			return newError("revalidate target", ErrConflict, err)
		}
		hash, err := e.fileSystem.HashSkill(operation.SourcePath)
		if err != nil {
			return newError("hash source", ErrIO, err)
		}
		if hash != operation.Hash {
			return newError("revalidate source", ErrConflict, nil)
		}
	}
	for _, operation := range plan.Deletes {
		if err := e.fileSystem.Revalidate(operation.TargetPathStates); err != nil {
			return newError("revalidate delete target", ErrConflict, err)
		}
	}
	return nil
}

type backedUpOperation struct {
	originalPath string
	backupPath   string
}

func (e *Executor) backupAll(plan Plan) ([]backedUpOperation, error) {
	backedUp := []backedUpOperation{}
	for _, operation := range plan.Deletes {
		backupPath, ok, err := e.backupPath(operation.FinalPath)
		if err != nil {
			e.cleanupBackups(backedUp)
			return nil, err
		}
		if ok {
			backedUp = append(backedUp, backedUpOperation{originalPath: operation.FinalPath, backupPath: backupPath})
		}
	}
	for _, operation := range plan.Creates {
		backupPath, ok, err := e.backupPath(operation.FinalPath)
		if err != nil {
			e.cleanupBackups(backedUp)
			return nil, err
		}
		if ok {
			backedUp = append(backedUp, backedUpOperation{originalPath: operation.FinalPath, backupPath: backupPath})
		}
	}
	return backedUp, nil
}

func (e *Executor) backupPath(path string) (string, bool, error) {
	state, err := e.fileSystem.Inspect(path, PathKindDirectory, true)
	if err != nil {
		return "", false, newError("backup inspect", ErrIO, err)
	}
	if !state.Exists {
		return "", false, nil
	}
	backupPath, err := e.fileSystem.Backup(path)
	if err != nil {
		return "", false, fmt.Errorf("避難の実行に失敗しました: %w", err)
	}
	return backupPath, true, nil
}

func (e *Executor) cleanupBackups(backups []backedUpOperation) {
	for _, backup := range backups {
		_ = e.fileSystem.RemoveAll(backup.backupPath)
	}
}

//nolint:gocognit // 重複排除、親子順序、作成直後検証を同じ処理単位で扱います。
func (e *Executor) createDirectories(plan Plan) ([]string, error) {
	required := make(map[string]struct{})
	for _, operation := range plan.Creates {
		for index, state := range operation.TargetPathStates {
			if index == len(operation.TargetPathStates)-1 {
				continue
			}
			if !state.Exists {
				required[state.Path] = struct{}{}
			}
		}
	}
	paths := make([]string, 0, len(required))
	for path := range required {
		paths = append(paths, path)
	}
	sort.Slice(paths, func(i, j int) bool {
		leftDepth := pathDepth(paths[i])
		rightDepth := pathDepth(paths[j])
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
		return paths[i] < paths[j]
	})
	created := make([]string, 0, len(paths))
	for _, path := range paths {
		if err := e.fileSystem.Mkdir(path, distributionDirectoryPermission); err != nil {
			return created, newError("create directory", ErrIO, err)
		}
		created = append(created, path)
		state, err := e.fileSystem.Inspect(path, PathKindDirectory, false)
		if err != nil || !state.Exists || state.Perm != distributionDirectoryPermission {
			return created, newError("verify directory", ErrPermission, err)
		}
	}
	return created, nil
}

func (e *Executor) stageAll(plan Plan) ([]stagedOperation, error) {
	staged := make([]stagedOperation, 0, len(plan.Creates))
	for _, operation := range plan.Creates {
		path, err := e.fileSystem.Stage(operation.SourcePath, filepath.Dir(operation.FinalPath))
		if err != nil {
			return staged, newError("stage", ErrIO, err)
		}
		staged = append(staged, stagedOperation{operation: operation, path: path})
		hash, err := e.fileSystem.HashSkill(path)
		if err != nil {
			return staged, newError("hash staging", ErrIO, err)
		}
		if hash != operation.Hash {
			return staged, newError("verify staging", ErrConflict, nil)
		}
	}
	return staged, nil
}

func (e *Executor) revalidateBeforePlacement(plan Plan, staged []stagedOperation) error {
	for _, item := range staged {
		state, err := e.fileSystem.Inspect(item.operation.FinalPath, PathKindDirectory, true)
		if err != nil {
			return newError("revalidate final path", ErrConflict, err)
		}
		if state.Exists {
			return newError("revalidate final path", ErrConflict, nil)
		}
		hash, err := e.fileSystem.HashSkill(item.path)
		if err != nil {
			return newError("hash staging", ErrIO, err)
		}
		if hash != item.operation.Hash {
			return newError("verify staging", ErrConflict, nil)
		}
	}
	for _, operation := range plan.Creates {
		if err := e.fileSystem.Revalidate(operation.SourcePathStates); err != nil {
			return newError("revalidate source", ErrConflict, err)
		}
	}
	return nil
}

func (e *Executor) placeAll(staged []stagedOperation) ([]CreateOperation, error) {
	placed := make([]CreateOperation, 0, len(staged))
	for _, item := range staged {
		if err := e.fileSystem.Rename(item.path, item.operation.FinalPath); err != nil {
			return placed, newError("place", ErrIO, err)
		}
		placed = append(placed, item.operation)
	}
	return placed, nil
}

func (e *Executor) rollback(
	staged []stagedOperation,
	placed []CreateOperation,
	backedUp []backedUpOperation,
	createdDirectories []string,
) ([]string, error) {
	var rollbackErr error
	var unrestored []string
	for _, operation := range slices.Backward(placed) {
		if err := e.fileSystem.RemoveAll(operation.FinalPath); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
			unrestored = append(unrestored, operation.RelativePath)
		}
	}
	for _, backup := range slices.Backward(backedUp) {
		if err := e.fileSystem.Rename(backup.backupPath, backup.originalPath); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
			unrestored = append(unrestored, filepath.Base(backup.originalPath))
		}
	}
	for _, item := range slices.Backward(staged) {
		if err := e.fileSystem.RemoveAll(item.path); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
		}
	}
	for _, path := range slices.Backward(createdDirectories) {
		if err := e.fileSystem.Remove(path); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
			unrestored = append(unrestored, filepath.Base(path))
		}
	}
	return unrestored, rollbackErr
}

func (e *Executor) cleanupStaging(staged []stagedOperation) error {
	var cleanupErr error
	for _, item := range staged {
		if err := e.fileSystem.RemoveAll(item.path); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	return cleanupErr
}

func joinPrimaryAndClose(primaryErr, closeErr error, committed bool) error {
	if closeErr == nil {
		return primaryErr
	}
	if committed {
		return errors.Join(primaryErr, &Error{Operation: "close", Kind: ErrCommitted, Cleanup: closeErr})
	}
	var distributionErr *Error
	if errors.As(primaryErr, &distributionErr) {
		return &Error{
			Operation:  distributionErr.Operation,
			Kind:       distributionErr.Kind,
			Err:        distributionErr.Err,
			Cleanup:    errors.Join(distributionErr.Cleanup, closeErr),
			Unrestored: distributionErr.Unrestored,
		}
	}
	return errors.Join(primaryErr, closeErr)
}

func pathDepth(path string) int {
	depth := 0
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		depth++
		parent := filepath.Dir(current)
		if parent == current {
			return depth
		}
	}
}
