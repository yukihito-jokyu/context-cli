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
const minimumTargetPathStates = 2

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
	parent    PathExpectation
	state     PathExpectation
}

type createdDirectory struct {
	state  PathExpectation
	parent PathExpectation
}

// Execute は計画を再検証し、配布と管理情報保存を一つの処理境界で実行します。
//
//nolint:gocognit,cyclop // コミット点を含む処理順とロールバック分岐を一つの関数で明示します。
func (e *Executor) Execute(plan Plan) (Result, error) {
	transaction, _, err := e.mapStore.Begin(plan.ExpectedRevision)
	if err != nil {
		return Result{}, newError("begin", ErrIO, err)
	}

	createdDirectories := []createdDirectory{}
	staged := []stagedOperation{}
	backedUp := []backedUpOperation{}
	placed := []CreateOperation{}
	committed := false
	var primaryErr error

	if err := e.revalidatePlan(plan); err != nil {
		primaryErr = err
	} else if createdDirectories, err = e.createDirectories(&plan); err != nil {
		primaryErr = err
	} else if staged, err = e.stageAll(plan); err != nil {
		primaryErr = err
	} else if err = e.revalidateBeforeBackup(plan, staged); err != nil {
		primaryErr = err
	} else if backedUp, err = e.backupAll(plan); err != nil {
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
				Unrestored: mergeUnrestored(primaryErr, unrestored),
			}
		}
	}
	if committed {
		var cleanupErr error
		for _, backup := range backedUp {
			if err := e.fileSystem.RemoveAll(backup.backupPath, backup.parent, backup.state); err != nil {
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
	return executorResult(plan), nil
}

// executorResult はPlanの実行モードに応じて結果集計を切り替えます。
//
// 初回配布（context add）は作成件数と配布先一覧を返します。
// 同期（context sync）は一意Skill名単位の更新数・削除数を返します。
func executorResult(plan Plan) Result {
	if !plan.IsSync {
		return Result{
			Created:      len(plan.Creates),
			Destinations: append([]Destination(nil), plan.Workspace.Destinations...),
		}
	}
	updated := make(map[string]struct{}, len(plan.Creates))
	for _, operation := range plan.Creates {
		updated[operation.Name] = struct{}{}
	}
	deleted := make(map[string]struct{}, len(plan.Deletes))
	for _, operation := range plan.Deletes {
		deleted[operation.Name] = struct{}{}
	}
	return Result{
		Created:       len(plan.Creates),
		Destinations:  append([]Destination(nil), plan.Workspace.Destinations...),
		UniqueUpdated: len(updated),
		UniqueDeleted: len(deleted),
	}
}

func (e *Executor) revalidatePlan(plan Plan) error {
	for _, operation := range plan.Creates {
		if err := e.fileSystem.Revalidate(operation.SourcePathStates); err != nil {
			return newError("revalidate source", ErrConflict, err)
		}
		if err := e.revalidateTarget(operation.TargetPathStates); err != nil {
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
		if err := e.revalidateTarget(operation.TargetPathStates); err != nil {
			return newError("revalidate delete target", ErrConflict, err)
		}
	}
	return nil
}

func (e *Executor) revalidateTarget(states []PathExpectation) error {
	for index, state := range states {
		if state.Exists {
			if err := e.fileSystem.Revalidate([]PathExpectation{state}); err != nil {
				return newError("revalidate target state", ErrConflict, err)
			}
			continue
		}
		if index != len(states)-1 {
			continue
		}
		current, err := e.fileSystem.Inspect(state.Path, state.Kind, true)
		if err != nil {
			return newError("inspect target state", ErrConflict, err)
		}
		if current.Exists {
			return newError("revalidate target", ErrConflict, nil)
		}
	}
	return nil
}

type backedUpOperation struct {
	originalPath string
	backupPath   string
	relativePath string
	parent       PathExpectation
	state        PathExpectation
}

func (e *Executor) backupAll(plan Plan) ([]backedUpOperation, error) {
	backedUp := []backedUpOperation{}
	for _, operation := range plan.Deletes {
		backupPath, ok, err := e.backupPath(operation.FinalPath, operation.TargetPathStates)
		if err != nil {
			return backedUp, err
		}
		if ok {
			backedUp = append(backedUp, backedUpOperation{
				originalPath: operation.FinalPath,
				backupPath:   backupPath,
				relativePath: operation.RelativePath,
				parent:       targetParent(operation.TargetPathStates),
				state:        operation.TargetPathStates[len(operation.TargetPathStates)-1],
			})
		}
	}
	for _, operation := range plan.Creates {
		backupPath, ok, err := e.backupPath(operation.FinalPath, operation.TargetPathStates)
		if err != nil {
			return backedUp, err
		}
		if ok {
			backedUp = append(backedUp, backedUpOperation{
				originalPath: operation.FinalPath,
				backupPath:   backupPath,
				relativePath: operation.RelativePath,
				parent:       targetParent(operation.TargetPathStates),
				state:        operation.TargetPathStates[len(operation.TargetPathStates)-1],
			})
		}
	}
	return backedUp, nil
}

func (e *Executor) backupPath(path string, states []PathExpectation) (string, bool, error) {
	if len(states) == 0 {
		return "", false, newError("backup inspect", ErrStructure, nil)
	}
	finalState := states[len(states)-1]
	if !finalState.Exists {
		return "", false, nil
	}
	current, err := e.fileSystem.Inspect(path, finalState.Kind, false)
	if err != nil {
		return "", false, newError("backup inspect", ErrConflict, err)
	}
	if current.Device != finalState.Device || current.Inode != finalState.Inode ||
		current.Kind != finalState.Kind || current.Perm != finalState.Perm {
		return "", false, newError("backup inspect", ErrConflict, nil)
	}
	parent := targetParent(states)
	if !parent.Exists {
		return "", false, newError("backup parent", ErrConflict, nil)
	}
	backupPath, err := e.fileSystem.Backup(path, parent, finalState)
	if err != nil {
		return "", false, fmt.Errorf("避難の実行に失敗しました: %w", err)
	}
	return backupPath, true, nil
}

//nolint:gocognit // 重複排除、親子順序、作成直後検証を同じ処理単位で扱います。
func (e *Executor) createDirectories(plan *Plan) ([]createdDirectory, error) {
	required := make(map[string]PathExpectation)
	for _, operation := range plan.Creates {
		for index, state := range operation.TargetPathStates {
			if index == len(operation.TargetPathStates)-1 {
				continue
			}
			if !state.Exists {
				required[state.Path] = operation.TargetPathStates[index-1]
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
	created := make([]createdDirectory, 0, len(paths))
	for _, path := range paths {
		parent := required[path]
		if updated, ok := createdExpectation(created, parent.Path); ok {
			parent = updated
		}
		if err := e.fileSystem.Mkdir(path, parent, distributionDirectoryPermission); err != nil {
			return created, newError("create directory", ErrIO, err)
		}
		state, err := e.fileSystem.Inspect(path, PathKindDirectory, false)
		if err != nil || !state.Exists || state.Perm != distributionDirectoryPermission {
			return created, newError("verify directory", ErrPermission, err)
		}
		created = append(created, createdDirectory{state: state, parent: parent})
		updateTargetExpectation(plan, state)
	}
	return created, nil
}

func (e *Executor) stageAll(plan Plan) ([]stagedOperation, error) {
	staged := make([]stagedOperation, 0, len(plan.Creates))
	for _, operation := range plan.Creates {
		parent := targetParent(operation.TargetPathStates)
		path, err := e.fileSystem.Stage(operation.SourcePath, parent)
		if err != nil {
			if errors.Is(err, ErrRollback) {
				return staged, fmt.Errorf("ステージング清掃に失敗しました: %w", err)
			}
			return staged, newError("stage", ErrIO, err)
		}
		state, err := e.fileSystem.Inspect(path, PathKindDirectory, false)
		if err != nil {
			return staged, newError("inspect staging", ErrIO, err)
		}
		staged = append(staged, stagedOperation{
			operation: operation,
			path:      path,
			parent:    parent,
			state:     state,
		})
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

//nolint:gocognit // ステージング、供給元、配布先の全件再検証を退避前に完結させます。
func (e *Executor) revalidateBeforeBackup(plan Plan, staged []stagedOperation) error {
	for _, item := range staged {
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
		if err := e.revalidateTarget(operation.TargetPathStates); err != nil {
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
		if err := e.revalidateTarget(operation.TargetPathStates); err != nil {
			return newError("revalidate delete target", ErrConflict, err)
		}
	}
	return nil
}

func (e *Executor) placeAll(staged []stagedOperation) ([]CreateOperation, error) {
	placed := make([]CreateOperation, 0, len(staged))
	for _, item := range staged {
		if err := e.fileSystem.Rename(RenameOperation{
			OldPath:     item.path,
			OldParent:   item.parent,
			OldExpected: item.state,
			NewPath:     item.operation.FinalPath,
			NewParent:   targetParent(item.operation.TargetPathStates),
			NewExpected: PathExpectation{Path: item.operation.FinalPath, Kind: PathKindDirectory},
		}); err != nil {
			return placed, newError("place", ErrIO, err)
		}
		placed = append(placed, item.operation)
	}
	return placed, nil
}

//nolint:gocognit // 配置、退避、ステージング、作成親を逆順に復元します。
func (e *Executor) rollback(
	staged []stagedOperation,
	placed []CreateOperation,
	backedUp []backedUpOperation,
	createdDirectories []createdDirectory,
) ([]string, error) {
	var rollbackErr error
	var unrestored []string
	for _, operation := range slices.Backward(placed) {
		item, ok := stagedByRelativePath(staged, operation.RelativePath)
		if !ok {
			rollbackErr = errors.Join(rollbackErr, newError("rollback placed state", ErrStructure, nil))
			unrestored = append(unrestored, operation.RelativePath)
			continue
		}
		if err := e.fileSystem.RemoveAll(
			operation.FinalPath, targetParent(operation.TargetPathStates), item.state,
		); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
			unrestored = append(unrestored, operation.RelativePath)
		}
	}
	for _, backup := range slices.Backward(backedUp) {
		if err := e.fileSystem.Rename(RenameOperation{
			OldPath:     backup.backupPath,
			OldParent:   backup.parent,
			OldExpected: backup.state,
			NewPath:     backup.originalPath,
			NewParent:   backup.parent,
			NewExpected: PathExpectation{Path: backup.originalPath, Kind: backup.state.Kind},
		}); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
			unrestored = append(unrestored, backup.relativePath)
		}
	}
	for _, item := range slices.Backward(staged) {
		if slices.ContainsFunc(placed, func(operation CreateOperation) bool {
			return operation.RelativePath == item.operation.RelativePath
		}) {
			continue
		}
		if err := e.fileSystem.RemoveAll(item.path, item.parent, item.state); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
		}
	}
	for _, directory := range slices.Backward(createdDirectories) {
		if err := e.fileSystem.Remove(directory.state.Path, directory.parent, directory.state); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
			unrestored = append(unrestored, filepath.Base(directory.state.Path))
		}
	}
	return unrestored, rollbackErr
}

func stagedByRelativePath(staged []stagedOperation, relativePath string) (stagedOperation, bool) {
	for _, item := range staged {
		if item.operation.RelativePath == relativePath {
			return item, true
		}
	}
	return stagedOperation{}, false
}

func targetParent(states []PathExpectation) PathExpectation {
	if len(states) < minimumTargetPathStates {
		return PathExpectation{}
	}
	return states[len(states)-2]
}

func createdExpectation(directories []createdDirectory, path string) (PathExpectation, bool) {
	for _, directory := range directories {
		if directory.state.Path == path {
			return directory.state, true
		}
	}
	return PathExpectation{}, false
}

func updateTargetExpectation(plan *Plan, current PathExpectation) {
	for operationIndex := range plan.Creates {
		for stateIndex := range plan.Creates[operationIndex].TargetPathStates {
			if plan.Creates[operationIndex].TargetPathStates[stateIndex].Path == current.Path {
				plan.Creates[operationIndex].TargetPathStates[stateIndex] = current
			}
		}
	}
	for operationIndex := range plan.Deletes {
		for stateIndex := range plan.Deletes[operationIndex].TargetPathStates {
			if plan.Deletes[operationIndex].TargetPathStates[stateIndex].Path == current.Path {
				plan.Deletes[operationIndex].TargetPathStates[stateIndex] = current
			}
		}
	}
}

func mergeUnrestored(primaryErr error, rollbackUnrestored []string) []string {
	merged := append([]string(nil), rollbackUnrestored...)
	var distributionErr *Error
	if !errors.As(primaryErr, &distributionErr) {
		return merged
	}
	for _, path := range distributionErr.Unrestored {
		if !slices.Contains(merged, path) {
			merged = append(merged, path)
		}
	}
	return merged
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
