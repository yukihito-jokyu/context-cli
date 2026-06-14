package distribution

import (
	"errors"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

var errCommitTest = errors.New("commit failed")

//nolint:gocognit,cyclop // テーブル駆動テストのため、認知・循環複雑度の上限を無視します。
func TestExecutor(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "CreatesSkillsAndCommitsMap",
			run: func(t *testing.T) {
				t.Helper()
				plan, workspace := createExecutorPlan(t)
				store := &fakeMapStore{
					snapshot: MapSnapshot{Revision: plan.ExpectedRevision, Workspaces: map[string]WorkspaceRecord{}},
					tx:       &fakeMapTransaction{},
				}

				result, err := NewExecutor(NewOSFileSystem(), store).Execute(plan)
				if err != nil {
					t.Fatalf("Execute() error = %v", err)
				}
				if result.Created != len(plan.Creates) {
					t.Fatalf("Created = %d, want %d", result.Created, len(plan.Creates))
				}
				for _, operation := range plan.Creates {
					data, err := os.ReadFile(filepath.Join(operation.FinalPath, "SKILL.md"))
					if err != nil {
						t.Fatalf("read distributed skill: %v", err)
					}
					if len(data) == 0 {
						t.Fatal("distributed skill is empty")
					}
				}
				if store.expected != plan.ExpectedRevision {
					t.Fatalf("Begin() expected = %q", store.expected)
				}
				if store.tx.committed.WorkspaceRoot != workspace {
					t.Fatalf("committed workspace = %q, want %q", store.tx.committed.WorkspaceRoot, workspace)
				}
				if store.tx.closeCalls != 1 {
					t.Fatalf("Close() calls = %d, want 1", store.tx.closeCalls)
				}
			},
		},
		{
			name: "RollsBackWhenCommitFailsBeforeCommitPoint",
			run: func(t *testing.T) {
				t.Helper()
				plan, _ := createExecutorPlan(t)
				store := &fakeMapStore{
					snapshot: MapSnapshot{Revision: plan.ExpectedRevision, Workspaces: map[string]WorkspaceRecord{}},
					tx: &fakeMapTransaction{
						commitErr: errCommitTest,
					},
				}

				_, err := NewExecutor(NewOSFileSystem(), store).Execute(plan)
				if !errors.Is(err, ErrIO) {
					t.Fatalf("Execute() error = %v, want ErrIO", err)
				}
				for _, operation := range plan.Creates {
					if _, statErr := os.Lstat(operation.FinalPath); !errors.Is(statErr, os.ErrNotExist) {
						t.Fatalf("final path remains after rollback: %s (%v)", operation.FinalPath, statErr)
					}
				}
			},
		},
		{
			name: "DoesNotRollbackAfterCommitPoint",
			run: func(t *testing.T) {
				t.Helper()
				plan, _ := createExecutorPlan(t)
				store := &fakeMapStore{
					snapshot: MapSnapshot{Revision: plan.ExpectedRevision, Workspaces: map[string]WorkspaceRecord{}},
					tx: &fakeMapTransaction{
						commitResult: CommitResult{Committed: true},
						commitErr:    ErrCommitted,
					},
				}

				_, err := NewExecutor(NewOSFileSystem(), store).Execute(plan)
				if !errors.Is(err, ErrCommitted) {
					t.Fatalf("Execute() error = %v, want ErrCommitted", err)
				}
				for _, operation := range plan.Creates {
					if _, statErr := os.Lstat(operation.FinalPath); statErr != nil {
						t.Fatalf("committed final path was rolled back: %v", statErr)
					}
				}
			},
		},
		{
			name: "StagesAllCreatesBeforeBackingUpTargets",
			run: func(t *testing.T) {
				t.Helper()
				plan, _ := createExecutorPlan(t)
				for index := range plan.Creates {
					finalPath := plan.Creates[index].FinalPath
					if err := os.MkdirAll(finalPath, 0o755); err != nil { // #nosec G301 -- 配布先要件の0755を再現します。
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(finalPath, "old"), []byte("old"), 0o600); err != nil {
						t.Fatal(err)
					}
					refreshTargetStates(t, &plan.Creates[index])
				}
				recording := &recordingFileSystem{FileSystem: NewOSFileSystem()}
				store := &fakeMapStore{tx: &fakeMapTransaction{}}

				if _, err := NewExecutor(recording, store).Execute(plan); err != nil {
					t.Fatalf("Execute() error = %v", err)
				}

				lastStage := -1
				firstBackup := len(recording.operations)
				for index, operation := range recording.operations {
					if strings.HasPrefix(operation, "stage:") {
						lastStage = index
					}
					if strings.HasPrefix(operation, "backup:") && firstBackup == len(recording.operations) {
						firstBackup = index
					}
				}
				if lastStage < 0 || firstBackup == len(recording.operations) || lastStage > firstBackup {
					t.Fatalf("operations = %v, want all stages before backups", recording.operations)
				}
			},
		},
		{
			name: "PreservesStageCleanupFailureForManualRecovery",
			run: func(t *testing.T) {
				t.Helper()
				plan, _ := createExecutorPlan(t)
				remaining := filepath.Join(filepath.Dir(plan.Creates[0].FinalPath), ".context-stage-remains")
				stageErr := &Error{
					Operation:  "stage cleanup",
					Kind:       ErrRollback,
					Err:        ErrFileType,
					Cleanup:    fs.ErrPermission,
					Unrestored: []string{remaining},
				}
				fileSystem := &stageFailureFileSystem{
					FileSystem: NewOSFileSystem(),
					err:        stageErr,
				}
				store := &fakeMapStore{tx: &fakeMapTransaction{}}

				_, err := NewExecutor(fileSystem, store).Execute(plan)
				if !errors.Is(err, ErrRollback) || !errors.Is(err, ErrFileType) {
					t.Fatalf("Execute() error = %v, want ErrRollback and ErrFileType", err)
				}
				var distributionErr *Error
				if !errors.As(err, &distributionErr) {
					t.Fatalf("Execute() error type = %T, want *Error", err)
				}
				if !slices.Contains(distributionErr.Unrestored, remaining) {
					t.Fatalf("Unrestored = %v, want %q", distributionErr.Unrestored, remaining)
				}
			},
		},
		{
			name: "RevalidatesAllTargetsBeforeFirstBackup",
			run: func(t *testing.T) {
				t.Helper()
				plan, _ := createExecutorPlan(t)
				for index := range plan.Creates {
					finalPath := plan.Creates[index].FinalPath
					if err := os.MkdirAll(finalPath, 0o755); err != nil { // #nosec G301 -- 配布先要件の0755を再現します。
						t.Fatal(err)
					}
					refreshTargetStates(t, &plan.Creates[index])
				}
				recording := &recordingFileSystem{FileSystem: NewOSFileSystem()}
				store := &fakeMapStore{tx: &fakeMapTransaction{}}

				if _, err := NewExecutor(recording, store).Execute(plan); err != nil {
					t.Fatalf("Execute() error = %v", err)
				}
				lastRevalidate := -1
				firstBackup := len(recording.operations)
				for index, operation := range recording.operations {
					if strings.HasPrefix(operation, "revalidate:") {
						lastRevalidate = index
					}
					if strings.HasPrefix(operation, "backup:") && firstBackup == len(recording.operations) {
						firstBackup = index
					}
				}
				if lastRevalidate < 0 || firstBackup == len(recording.operations) || lastRevalidate > firstBackup {
					t.Fatalf("operations = %v, want all revalidation before first backup", recording.operations)
				}
			},
		},
		{
			name: "RestoresBackupsInReverseOrderWhenLaterBackupFails",
			run: func(t *testing.T) {
				t.Helper()
				plan, _ := createExecutorPlan(t)
				for index := range plan.Creates {
					finalPath := plan.Creates[index].FinalPath
					if err := os.MkdirAll(finalPath, 0o755); err != nil { // #nosec G301 -- 配布先要件の0755を再現します。
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(finalPath, "old"), []byte("old"), 0o600); err != nil {
						t.Fatal(err)
					}
					refreshTargetStates(t, &plan.Creates[index])
				}
				failing := &recordingFileSystem{FileSystem: NewOSFileSystem(), failBackupAt: 2}
				store := &fakeMapStore{tx: &fakeMapTransaction{}}

				_, err := NewExecutor(failing, store).Execute(plan)
				if !errors.Is(err, ErrIO) {
					t.Fatalf("Execute() error = %v, want ErrIO", err)
				}
				for _, operation := range plan.Creates {
					data, readErr := os.ReadFile(filepath.Join(operation.FinalPath, "old"))
					if readErr != nil || string(data) != "old" {
						t.Fatalf("original target was not restored: %s (%q, %v)", operation.RelativePath, data, readErr)
					}
				}
			},
		},
		{
			name: "ReplacesFIFOLeafAndCommits",
			run: func(t *testing.T) {
				t.Helper()
				plan, _ := createExecutorPlan(t)
				plan.Creates = plan.Creates[:1]
				operation := &plan.Creates[0]
				if err := os.MkdirAll(filepath.Dir(operation.FinalPath), 0o755); err != nil { // #nosec G301 -- 配布先要件の0755を再現します。
					t.Fatal(err)
				}
				if err := unix.Mkfifo(operation.FinalPath, 0o600); err != nil {
					t.Fatal(err)
				}
				for index, state := range operation.TargetPathStates {
					kind := PathKindDirectory
					if index == len(operation.TargetPathStates)-1 {
						kind = PathKindAny
					}
					current, err := NewOSFileSystem().Inspect(state.Path, kind, false)
					if err != nil {
						t.Fatal(err)
					}
					operation.TargetPathStates[index] = current
				}
				store := &fakeMapStore{tx: &fakeMapTransaction{}}

				if _, err := NewExecutor(NewOSFileSystem(), store).Execute(plan); err != nil {
					t.Fatalf("Execute() error = %v", err)
				}
				info, err := os.Stat(operation.FinalPath)
				if err != nil {
					t.Fatal(err)
				}
				if !info.IsDir() {
					t.Fatalf("final mode = %v, want directory", info.Mode())
				}
			},
		},
		{
			name: "RestoresNonDirectoryLeafOnCommitFailure",
			run: func(t *testing.T) {
				t.Helper()
				tests := []struct {
					name   string
					short  bool
					create func(t *testing.T, path string)
					verify func(t *testing.T, path string)
				}{
					{
						name: "regular file",
						create: func(t *testing.T, path string) {
							t.Helper()
							if err := os.WriteFile(path, []byte("local"), 0o600); err != nil {
								t.Fatal(err)
							}
						},
						verify: func(t *testing.T, path string) {
							t.Helper()
							data, err := os.ReadFile(path) // #nosec G304 -- テスト専用一時ディレクトリ内の対象です。
							if err != nil || string(data) != "local" {
								t.Fatalf("restored file = %q, %v", data, err)
							}
						},
					},
					{
						name: "fifo",
						create: func(t *testing.T, path string) {
							t.Helper()
							if err := unix.Mkfifo(path, 0o600); err != nil {
								t.Fatal(err)
							}
						},
						verify: func(t *testing.T, path string) {
							t.Helper()
							info, err := os.Lstat(path)
							if err != nil || info.Mode()&fs.ModeNamedPipe == 0 {
								t.Fatalf("restored mode = %v, %v", info, err)
							}
						},
					},
					{
						name:  "unix socket",
						short: true,
						create: func(t *testing.T, path string) {
							t.Helper()
							listener, err := net.Listen("unix", path)
							if err != nil {
								t.Skipf("Unixソケットを作成できない実行環境のためスキップします: %v", err)
							}
							t.Cleanup(func() { _ = listener.Close() })
						},
						verify: func(t *testing.T, path string) {
							t.Helper()
							info, err := os.Lstat(path)
							if err != nil || info.Mode()&fs.ModeSocket == 0 {
								t.Fatalf("restored mode = %v, %v", info, err)
							}
						},
					},
				}
				for _, test := range tests {
					t.Run(test.name, func(t *testing.T) {
						plan, _ := createExecutorPlan(t)
						plan.Creates = plan.Creates[:1]
						operation := &plan.Creates[0]
						if test.short {
							shortRoot, err := os.MkdirTemp("/tmp", "context-socket-")
							if err != nil {
								t.Fatal(err)
							}
							t.Cleanup(func() { _ = os.RemoveAll(shortRoot) })
							shortRoot, err = filepath.EvalSymlinks(shortRoot)
							if err != nil {
								t.Fatal(err)
							}
							operation.FinalPath = filepath.Join(shortRoot, "alpha")
							operation.RelativePath = "alpha"
							states, err := NewPlanner(NewOSFileSystem()).inspectChain(
								operation.FinalPath, PathKindAny, true,
							)
							if err != nil {
								t.Fatal(err)
							}
							operation.TargetPathStates = states
						}
						if err := os.MkdirAll(filepath.Dir(operation.FinalPath), 0o755); err != nil { // #nosec G301 -- 配布先要件を再現します。
							t.Fatal(err)
						}
						test.create(t, operation.FinalPath)
						refreshTargetStatesAnyLeaf(t, operation)
						store := &fakeMapStore{tx: &fakeMapTransaction{commitErr: errCommitTest}}

						_, err := NewExecutor(NewOSFileSystem(), store).Execute(plan)
						if !errors.Is(err, ErrIO) {
							t.Fatalf("Execute() error = %v, want ErrIO", err)
						}
						test.verify(t, operation.FinalPath)
					})
				}
			},
		},
		{
			name: "RejectsParentReplacementBeforePlacementWithoutWritingOutside",
			run: func(t *testing.T) {
				t.Helper()
				plan, _ := createExecutorPlan(t)
				plan.Creates = plan.Creates[:1]
				parent := filepath.Dir(plan.Creates[0].FinalPath)
				outside := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(parent))), "outside")
				if err := os.Mkdir(outside, 0o755); err != nil { // #nosec G301 -- 差し替え先ディレクトリを作成します。
					t.Fatal(err)
				}
				swapped := false
				fileSystem := osFileSystem{hooks: fileSystemHooks{
					beforeRename: func(oldPath, _ string) {
						if swapped || !strings.Contains(filepath.Base(oldPath), ".context-stage-") {
							return
						}
						swapped = true
						if err := os.Rename(parent, parent+"-moved"); err != nil {
							t.Fatal(err)
						}
						if err := os.Symlink(outside, parent); err != nil {
							t.Fatal(err)
						}
					},
				}}
				store := &fakeMapStore{tx: &fakeMapTransaction{}}

				_, err := NewExecutor(fileSystem, store).Execute(plan)
				if !errors.Is(err, ErrConflict) && !errors.Is(err, ErrRollback) {
					t.Fatalf("Execute() error = %v, want ErrConflict or ErrRollback", err)
				}
				entries, readErr := os.ReadDir(outside)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if len(entries) != 0 {
					t.Fatalf("outside entries = %v, want empty", entries)
				}
			},
		},
		{
			name: "RejectsCreatedParentReplacementBeforeStaging",
			run: func(t *testing.T) {
				t.Helper()
				plan, _ := createExecutorPlan(t)
				plan.Creates = plan.Creates[:1]
				parent := filepath.Dir(plan.Creates[0].FinalPath)
				outside := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(parent))), "outside-stage")
				if err := os.Mkdir(outside, 0o755); err != nil { // #nosec G301 -- 差し替え先ディレクトリを作成します。
					t.Fatal(err)
				}
				swapped := false
				fileSystem := osFileSystem{hooks: fileSystemHooks{
					beforeStage: func(stageParent string) {
						if swapped || stageParent != parent {
							return
						}
						swapped = true
						if err := os.Rename(parent, parent+"-moved"); err != nil {
							t.Fatal(err)
						}
						if err := os.Symlink(outside, parent); err != nil {
							t.Fatal(err)
						}
					},
				}}
				store := &fakeMapStore{tx: &fakeMapTransaction{}}

				_, err := NewExecutor(fileSystem, store).Execute(plan)
				if !errors.Is(err, ErrConflict) && !errors.Is(err, ErrRollback) {
					t.Fatalf("Execute() error = %v, want ErrConflict or ErrRollback", err)
				}
				entries, readErr := os.ReadDir(outside)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if len(entries) != 0 {
					t.Fatalf("outside entries = %v, want empty", entries)
				}
			},
		},
		{
			name: "ReportsPlacedTargetWhenPlacementAndRollbackFail",
			run: func(t *testing.T) {
				t.Helper()
				plan, _ := createExecutorPlan(t)
				failing := &recordingFileSystem{
					FileSystem:        NewOSFileSystem(),
					failRenameAt:      2,
					failRemoveAllPath: plan.Creates[0].FinalPath,
				}
				store := &fakeMapStore{tx: &fakeMapTransaction{}}

				_, err := NewExecutor(failing, store).Execute(plan)
				if !errors.Is(err, ErrRollback) {
					t.Fatalf("Execute() error = %v, want ErrRollback", err)
				}
				var distributionErr *Error
				if !errors.As(err, &distributionErr) {
					t.Fatalf("Execute() error type = %T, want *Error", err)
				}
				if !slices.Contains(distributionErr.Unrestored, plan.Creates[0].RelativePath) {
					t.Fatalf("Unrestored = %v, want %q", distributionErr.Unrestored, plan.Creates[0].RelativePath)
				}
			},
		},
		{
			name: "DeletesAndRollsBack",
			run: func(t *testing.T) {
				t.Helper()
				setup := createExecutorPlanWithDeletes(t)
				store := &fakeMapStore{
					snapshot: MapSnapshot{Revision: setup.plan.ExpectedRevision, Workspaces: map[string]WorkspaceRecord{}},
					tx: &fakeMapTransaction{
						commitErr: errCommitTest,
					},
				}

				_, err := NewExecutor(NewOSFileSystem(), store).Execute(setup.plan)
				if !errors.Is(err, ErrIO) {
					t.Fatalf("Execute() error = %v, want ErrIO", err)
				}

				if _, err := os.Lstat(filepath.Join(setup.betaPath, "SKILL.md")); err != nil {
					t.Fatalf("deleted skill should be restored but got error: %v", err)
				}

				for _, operation := range setup.plan.Creates {
					if _, err := os.Lstat(operation.FinalPath); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("newly created path should be removed but it still exists: %s", operation.FinalPath)
					}
				}
			},
		},
		{
			name: "DeletesAndCommits",
			run: func(t *testing.T) {
				t.Helper()
				setup := createExecutorPlanWithDeletes(t)
				store := &fakeMapStore{
					snapshot: MapSnapshot{Revision: setup.plan.ExpectedRevision, Workspaces: map[string]WorkspaceRecord{}},
					tx:       &fakeMapTransaction{},
				}

				result, err := NewExecutor(NewOSFileSystem(), store).Execute(setup.plan)
				if err != nil {
					t.Fatalf("Execute() error = %v, want ErrIO", err)
				}
				if result.Created != len(setup.plan.Creates) {
					t.Fatalf("Created = %d, want %d", result.Created, len(setup.plan.Creates))
				}

				if _, err := os.Lstat(setup.betaPath); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("deleted skill path should not exist: %v", err)
				}

				for _, operation := range setup.plan.Creates {
					if _, err := os.Lstat(filepath.Join(operation.FinalPath, "SKILL.md")); err != nil {
						t.Fatalf("newly created skill does not exist: %v", err)
					}
				}

				files, err := os.ReadDir(filepath.Dir(setup.betaPath))
				if err != nil {
					t.Fatal(err)
				}
				for _, file := range files {
					if file.IsDir() && len(file.Name()) > 15 && file.Name()[:16] == ".context-backup-" {
						t.Fatalf("backup directory remains: %s", file.Name())
					}
				}
			},
		},
		{
			name: "SyncUpdatesAndDeletesAndCountsUniqueSkills",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncExecutorFixture(t)
				_, execPlan := fixture.buildSyncPlan(fixture.alphaHash, SourceStateMissing)

				result, err := NewExecutor(NewOSFileSystem(), fixture.store).Execute(execPlan)
				if err != nil {
					t.Fatalf("Execute() error = %v", err)
				}
				if result.UniqueUpdated != 1 {
					t.Fatalf("UniqueUpdated = %d, want 1", result.UniqueUpdated)
				}
				if result.UniqueDeleted != 1 {
					t.Fatalf("UniqueDeleted = %d, want 1", result.UniqueDeleted)
				}

				data, err := os.ReadFile(filepath.Join(fixture.alphaDest, "SKILL.md"))
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != "alpha-v1\n" {
					t.Fatalf("updated alpha content = %q, want %q", data, "alpha-v1\n")
				}
				if _, err := os.Lstat(fixture.betaDest); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("beta destination should be removed: %v", err)
				}
				committed := fixture.store.tx.committed
				if len(committed.Skills) != 1 || committed.Skills[0].Name != "alpha" {
					t.Fatalf("committed skills = %#v", committed.Skills)
				}
			},
		},
		{
			name: "SyncCommitsEmptyWorkspaceWhenAllSkillsRemoved",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncExecutorFixture(t)
				sources := []ResolvedSource{
					{Name: "alpha", Source: SkillSourceProject, State: SourceStateMissing},
					{Name: "beta", Source: SkillSourceProject, State: SourceStateMissing},
				}
				input := SyncInput{
					WorkspaceRoot: fixture.workspace,
					Project:       "project",
					Destinations:  []Destination{DestinationCodex},
					Skills:        append([]RecordedSkill(nil), fixture.skillInput...),
				}
				plan, err := NewSyncPlanner(NewOSFileSystem()).Plan(fixture.snapshot, input, sources)
				if err != nil {
					t.Fatalf("Plan() error = %v", err)
				}
				execPlan, err := plan.ToPlan()
				if err != nil {
					t.Fatalf("ToPlan() error = %v", err)
				}

				result, err := NewExecutor(NewOSFileSystem(), fixture.store).Execute(execPlan)
				if err != nil {
					t.Fatalf("Execute() error = %v", err)
				}
				if result.UniqueDeleted != 2 {
					t.Fatalf("UniqueDeleted = %d, want 2", result.UniqueDeleted)
				}
				if len(fixture.store.tx.committed.Skills) != 0 {
					t.Fatalf("committed skills = %d, want 0", len(fixture.store.tx.committed.Skills))
				}
			},
		},
		{
			name: "SyncRollsBackWhenCommitFails",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncExecutorFixture(t)
				fixture.store.tx = &fakeMapTransaction{commitErr: errCommitTest}
				_, execPlan := fixture.buildSyncPlan(fixture.alphaHash, SourceStateMissing)

				_, err := NewExecutor(NewOSFileSystem(), fixture.store).Execute(execPlan)
				if !errors.Is(err, ErrIO) {
					t.Fatalf("Execute() error = %v, want ErrIO", err)
				}
				data, readErr := os.ReadFile(filepath.Join(fixture.alphaDest, "SKILL.md"))
				if readErr != nil || string(data) != "alpha-old\n" {
					t.Fatalf("alpha should be restored: %q (%v)", data, readErr)
				}
				if _, err := os.Lstat(filepath.Join(fixture.betaDest, "SKILL.md")); err != nil {
					t.Fatalf("beta should be restored: %v", err)
				}
			},
		},
		{
			name: "SyncDoesNotRollbackAfterCommit",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncExecutorFixture(t)
				fixture.store.tx = &fakeMapTransaction{
					commitResult: CommitResult{Committed: true},
					commitErr:    ErrCommitted,
				}
				_, execPlan := fixture.buildSyncPlan(fixture.alphaHash, SourceStateMissing)

				_, err := NewExecutor(NewOSFileSystem(), fixture.store).Execute(execPlan)
				if !errors.Is(err, ErrCommitted) {
					t.Fatalf("Execute() error = %v, want ErrCommitted", err)
				}
				data, readErr := os.ReadFile(filepath.Join(fixture.alphaDest, "SKILL.md"))
				if readErr != nil || string(data) != "alpha-v1\n" {
					t.Fatalf("alpha should remain updated after commit: %q (%v)", data, readErr)
				}
				if _, err := os.Lstat(fixture.betaDest); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("beta should remain removed after commit: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func refreshTargetStates(t *testing.T, operation *CreateOperation) {
	t.Helper()
	for index, state := range operation.TargetPathStates {
		current, err := NewOSFileSystem().Inspect(state.Path, PathKindDirectory, false)
		if err != nil {
			t.Fatal(err)
		}
		operation.TargetPathStates[index] = current
	}
}

func refreshTargetStatesAnyLeaf(t *testing.T, operation *CreateOperation) {
	t.Helper()
	for index, state := range operation.TargetPathStates {
		kind := PathKindDirectory
		if index == len(operation.TargetPathStates)-1 {
			kind = PathKindAny
		}
		current, err := NewOSFileSystem().Inspect(state.Path, kind, false)
		if err != nil {
			t.Fatal(err)
		}
		operation.TargetPathStates[index] = current
	}
}

func createExecutorPlan(t *testing.T) (Plan, string) {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(base, "workspace")
	source := filepath.Join(base, "context", "projects", "project", "skills", "alpha")
	if err := os.MkdirAll(workspace, 0o755); err != nil { // #nosec G301 -- 配布先ディレクトリ要件を再現します。
		t.Fatal(err)
	}
	if err := os.MkdirAll(source, 0o755); err != nil { // #nosec G301 -- Skillディレクトリの権限維持を検証します。
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("skill\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := NewPlanner(NewOSFileSystem()).Plan(
		MapSnapshot{Revision: EmptyRevision, Workspaces: map[string]WorkspaceRecord{}},
		Selection{
			WorkspaceRoot: workspace,
			Project:       "project",
			Skills:        []SelectedSkill{{Name: "alpha", Source: SkillSourceProject, SourcePath: source}},
			Destinations:  []Destination{DestinationCodex, DestinationClaude},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return plan, workspace
}

type fakeMapStore struct {
	expected Revision
	snapshot MapSnapshot
	tx       *fakeMapTransaction
	beginErr error
}

func (s *fakeMapStore) Load() (MapSnapshot, error) {
	return s.snapshot, nil
}

func (s *fakeMapStore) Begin(expected Revision) (MapTransaction, MapSnapshot, error) {
	s.expected = expected
	if s.beginErr != nil {
		return nil, MapSnapshot{}, s.beginErr
	}
	return s.tx, s.snapshot, nil
}

type fakeMapTransaction struct {
	committed    WorkspaceRecord
	commitResult CommitResult
	commitErr    error
	closeErr     error
	closeCalls   int
}

type recordingFileSystem struct {
	FileSystem
	operations        []string
	backupCalls       int
	renameCalls       int
	failBackupAt      int
	failRenameAt      int
	failRemoveAllPath string
}

type stageFailureFileSystem struct {
	FileSystem
	err error
}

func (f *stageFailureFileSystem) Stage(string, PathExpectation) (string, error) {
	return "", f.err
}

func (f *recordingFileSystem) Revalidate(expectations []PathExpectation) error {
	for _, expectation := range expectations {
		f.operations = append(f.operations, "revalidate:"+expectation.Path)
	}
	if err := f.FileSystem.Revalidate(expectations); err != nil {
		return newError("record revalidate", ErrConflict, err)
	}
	return nil
}

func (f *recordingFileSystem) Stage(source string, parent PathExpectation) (string, error) {
	f.operations = append(f.operations, "stage:"+source)
	path, err := f.FileSystem.Stage(source, parent)
	if err != nil {
		return "", newError("record stage", ErrIO, err)
	}
	return path, nil
}

func (f *recordingFileSystem) Backup(
	path string,
	parent PathExpectation,
	expected PathExpectation,
) (string, error) {
	f.backupCalls++
	f.operations = append(f.operations, "backup:"+path)
	if f.backupCalls == f.failBackupAt {
		return "", &Error{Operation: "backup", Kind: ErrIO, Err: fs.ErrPermission}
	}
	backup, err := f.FileSystem.Backup(path, parent, expected)
	if err != nil {
		return "", newError("record backup", ErrIO, err)
	}
	return backup, nil
}

func (f *recordingFileSystem) Rename(operation RenameOperation) error {
	f.renameCalls++
	f.operations = append(f.operations, "rename:"+operation.OldPath+"->"+operation.NewPath)
	if f.renameCalls == f.failRenameAt {
		return &Error{Operation: "rename", Kind: ErrIO, Err: fs.ErrPermission}
	}
	if err := f.FileSystem.Rename(operation); err != nil {
		return newError("record rename", ErrIO, err)
	}
	return nil
}

func (f *recordingFileSystem) RemoveAll(
	path string,
	parent PathExpectation,
	expected PathExpectation,
) error {
	f.operations = append(f.operations, "remove-all:"+path)
	if path == f.failRemoveAllPath {
		return &Error{Operation: "remove tree", Kind: ErrIO, Err: fs.ErrPermission}
	}
	if err := f.FileSystem.RemoveAll(path, parent, expected); err != nil {
		return newError("record remove tree", ErrIO, err)
	}
	return nil
}

func (t *fakeMapTransaction) Commit(workspace WorkspaceRecord) (CommitResult, error) {
	t.committed = workspace
	if t.commitResult == (CommitResult{}) && t.commitErr == nil {
		return CommitResult{Committed: true}, nil
	}
	return t.commitResult, t.commitErr
}

func (t *fakeMapTransaction) Close() error {
	t.closeCalls++
	return t.closeErr
}

type deleteTestSetup struct {
	plan      Plan
	workspace string
	betaPath  string
	alphaPath string
}

func createExecutorPlanWithDeletes(t *testing.T) deleteTestSetup {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(base, "workspace")

	// 供給元ディレクトリの作成
	alphaSource := filepath.Join(base, "context", "projects", "project", "skills", "alpha")
	betaSource := filepath.Join(base, "context", "projects", "project", "skills", "beta")

	// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
	if err := os.MkdirAll(alphaSource, 0o755); err != nil {
		t.Fatal(err)
	}
	// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
	if err := os.MkdirAll(betaSource, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(alphaSource, "SKILL.md"), []byte("alpha\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(betaSource, "SKILL.md"), []byte("beta\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 配布先ディレクトリと既存のSkillを配置
	alphaDest := filepath.Join(workspace, ".codex", "skills", "alpha")
	betaDest := filepath.Join(workspace, ".codex", "skills", "beta")
	// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
	if err := os.MkdirAll(filepath.Dir(betaDest), 0o755); err != nil {
		t.Fatal(err)
	}
	// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
	if err := os.MkdirAll(betaDest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(betaDest, "SKILL.md"), []byte("beta\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 前回の管理情報（Snapshot）の作成（betaが配置済み）
	snapshot := MapSnapshot{
		Revision: "rev-1234",
		Workspaces: map[string]WorkspaceRecord{
			workspace: {
				WorkspaceRoot: workspace,
				Project:       "project",
				Destinations:  []Destination{DestinationCodex},
				Skills: []SkillRecord{
					{
						Name:         "beta",
						Source:       SkillSourceProject,
						Destination:  DestinationCodex,
						RelativePath: ".codex/skills/beta",
						Hash:         "some-hash",
					},
				},
			},
		},
	}

	// 今回の選択（alphaのみを選択、betaは解除）
	selection := Selection{
		WorkspaceRoot: workspace,
		Project:       "project",
		Skills: []SelectedSkill{
			{
				Name:       "alpha",
				Source:     SkillSourceProject,
				SourcePath: alphaSource,
			},
		},
		Destinations: []Destination{DestinationCodex},
	}

	plan, err := NewPlanner(NewOSFileSystem()).Plan(snapshot, selection)
	if err != nil {
		t.Fatal(err)
	}

	return deleteTestSetup{
		plan:      plan,
		workspace: workspace,
		betaPath:  betaDest,
		alphaPath: alphaDest,
	}
}

// syncExecutorFixture は同期Executorテスト用の共通セットアップを保持します。
type syncExecutorFixture struct {
	t          *testing.T
	base       string
	workspace  string
	alphaSrc   string
	betaSrc    string
	alphaDest  string
	betaDest   string
	alphaHash  string
	betaHash   string
	snapshot   MapSnapshot
	store      *fakeMapStore
	skillInput []RecordedSkill
}

func newSyncExecutorFixture(t *testing.T) *syncExecutorFixture {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(base, "workspace")
	alphaSrc := filepath.Join(base, "context", "projects", "project", "skills", "alpha")
	betaSrc := filepath.Join(base, "context", "projects", "project", "skills", "beta")
	alphaDest := filepath.Join(workspace, ".codex", "skills", "alpha")
	betaDest := filepath.Join(workspace, ".codex", "skills", "beta")

	for _, path := range []string{workspace, alphaSrc, betaSrc, filepath.Dir(betaDest)} {
		// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(alphaSrc, "SKILL.md"), []byte("alpha-v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(betaSrc, "SKILL.md"), []byte("beta-v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	alphaHash, err := HashSkill(alphaSrc)
	if err != nil {
		t.Fatal(err)
	}
	betaHash, err := HashSkill(betaSrc)
	if err != nil {
		t.Fatal(err)
	}
	// 配布先alphaは旧内容で配置して「更新対象」にする。
	if err := os.MkdirAll(alphaDest, 0o755); err != nil { // #nosec G301 -- 配布先ディレクトリ要件を再現します。
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(alphaDest, "SKILL.md"), []byte("alpha-old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// 配布先betaは記録ハッシュと同一内容で配置して「削除対象（供給元消失）」にする。
	if err := os.MkdirAll(betaDest, 0o755); err != nil { // #nosec G301 -- 配布先ディレクトリ要件を再現します。
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(betaDest, "SKILL.md"), []byte("beta-v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	fixture := &syncExecutorFixture{
		t:         t,
		base:      base,
		workspace: workspace,
		alphaSrc:  alphaSrc,
		betaSrc:   betaSrc,
		alphaDest: alphaDest,
		betaDest:  betaDest,
		alphaHash: alphaHash,
		betaHash:  betaHash,
	}
	fixture.skillInput = []RecordedSkill{
		{
			Name: "alpha", Source: SkillSourceProject, Destination: DestinationCodex,
			RelativePath: ".codex/skills/alpha", RecordedHash: alphaHash,
		},
		{
			Name: "beta", Source: SkillSourceProject, Destination: DestinationCodex,
			RelativePath: ".codex/skills/beta", RecordedHash: betaHash,
		},
	}
	fixture.snapshot = MapSnapshot{
		Revision: "rev-sync",
		Workspaces: map[string]WorkspaceRecord{
			workspace: {
				WorkspaceRoot: workspace,
				Project:       "project",
				Destinations:  []Destination{DestinationCodex},
				Skills: []SkillRecord{
					{
						Name: "alpha", Source: SkillSourceProject, Destination: DestinationCodex,
						RelativePath: ".codex/skills/alpha", Hash: alphaHash,
					},
					{
						Name: "beta", Source: SkillSourceProject, Destination: DestinationCodex,
						RelativePath: ".codex/skills/beta", Hash: betaHash,
					},
				},
			},
		},
	}
	fixture.store = &fakeMapStore{
		snapshot: fixture.snapshot,
		tx:       &fakeMapTransaction{},
	}
	return fixture
}

// buildSyncPlan は同期計画を組み立ててPlanへ変換します。
// sourcesで供給元状態を与え、alphaは有効、betaは消失とします。
func (f *syncExecutorFixture) buildSyncPlan(alphaActiveHash string, betaState SourceState) (SyncPlan, Plan) {
	f.t.Helper()
	sources := []ResolvedSource{
		{
			Name: "alpha", Source: SkillSourceProject, State: SourceStateActive,
			Path: f.alphaSrc, Hash: alphaActiveHash,
		},
		{
			Name: "beta", Source: SkillSourceProject, State: betaState,
		},
	}
	input := SyncInput{
		WorkspaceRoot: f.workspace,
		Project:       "project",
		Destinations:  []Destination{DestinationCodex},
		Skills:        append([]RecordedSkill(nil), f.skillInput...),
	}
	plan, err := NewSyncPlanner(NewOSFileSystem()).Plan(f.snapshot, input, sources)
	if err != nil {
		f.t.Fatalf("Plan() error = %v", err)
	}
	execPlan, err := plan.ToPlan()
	if err != nil {
		f.t.Fatalf("ToPlan() error = %v", err)
	}
	return plan, execPlan
}
