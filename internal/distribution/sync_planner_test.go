package distribution

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// syncFixture は同期計画テスト用の環境を構築します。
type syncFixture struct {
	t           *testing.T
	root        string
	workspace   string
	projectBase string
	commonBase  string
}

func newSyncFixture(t *testing.T) *syncFixture {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	fixture := &syncFixture{
		t:           t,
		root:        base,
		workspace:   filepath.Join(base, "workspace"),
		projectBase: filepath.Join(base, "context", "projects", "project", "skills"),
		commonBase:  filepath.Join(base, "context", "utils", "skills"),
	}
	if err := os.MkdirAll(fixture.workspace, 0o755); err != nil { // #nosec G301 -- 配布先要件を再現します。
		t.Fatal(err)
	}
	if err := os.MkdirAll(fixture.projectBase, 0o755); err != nil { // #nosec G301 -- Skillディレクトリ権限を再現します。
		t.Fatal(err)
	}
	if err := os.MkdirAll(fixture.commonBase, 0o755); err != nil { // #nosec G301 -- Skillディレクトリ権限を再現します。
		t.Fatal(err)
	}
	return fixture
}

// writeProjectSource はプロジェクト固有供給元Skillを作成し現在ハッシュを返します。
func (f *syncFixture) writeProjectSource(name, body string) (string, string) {
	f.t.Helper()
	path := filepath.Join(f.projectBase, name)
	if err := os.MkdirAll(path, 0o755); err != nil { // #nosec G301 -- Skillディレクトリ権限を再現します。
		f.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte(body), 0o600); err != nil {
		f.t.Fatal(err)
	}
	hash, err := HashSkill(path)
	if err != nil {
		f.t.Fatal(err)
	}
	return path, hash
}

// distributeAlpha はalpha配布先へalpha-v1内容のSkillを配置し、記録ハッシュを返します。
func (f *syncFixture) distributeAlpha(destination Destination) string {
	f.t.Helper()
	relative, err := destinationRelativePath(destination, "alpha")
	if err != nil {
		f.t.Fatal(err)
	}
	finalPath := filepath.Join(f.workspace, relative)
	if err := os.MkdirAll(finalPath, 0o755); err != nil { // #nosec G301 -- 配布先要件を再現します。
		f.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(finalPath, "SKILL.md"), []byte("alpha-v1\n"), 0o600); err != nil {
		f.t.Fatal(err)
	}
	hash, err := HashSkill(finalPath)
	if err != nil {
		f.t.Fatal(err)
	}
	return hash
}

func (f *syncFixture) runPlanner(input SyncInput, sources []ResolvedSource) SyncPlan {
	f.t.Helper()
	plan, err := NewSyncPlanner(NewOSFileSystem()).Plan(MapSnapshot{Revision: "rev"}, input, sources)
	if err != nil {
		f.t.Fatalf("Plan() error = %v", err)
	}
	return plan
}

//nolint:gocognit,cyclop // テーブル駆動テストのため、認知・循環複雑度の上限を無視します。
func TestSyncPlanner(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "UpdatesWhenSourceChanged",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncFixture(t)
				sourcePath, sourceHash := fixture.writeProjectSource("alpha", "alpha-v1\n")
				targetHash := fixture.distributeAlpha(DestinationCodex)
				// 記録ハッシュは旧ハッシュ（targetHash == sourceHash）。供給元だけ更新する。
				if err := os.WriteFile(filepath.Join(sourcePath, "SKILL.md"), []byte("alpha-v2\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				newHash, err := HashSkill(sourcePath)
				if err != nil {
					t.Fatal(err)
				}

				input := SyncInput{
					WorkspaceRoot: fixture.workspace,
					Project:       "project",
					Destinations:  []Destination{DestinationCodex},
					Skills: []RecordedSkill{{
						Name: "alpha", Source: SkillSourceProject, Destination: DestinationCodex,
						RelativePath: ".codex/skills/alpha", RecordedHash: targetHash,
					}},
				}
				sources := []ResolvedSource{{
					Name: "alpha", Source: SkillSourceProject, State: SourceStateActive, Path: sourcePath,
					Hash: newHash,
				}}
				plan := fixture.runPlanner(input, sources)

				if len(plan.Updates) != 1 {
					t.Fatalf("Updates = %d, want 1", len(plan.Updates))
				}
				if plan.Updates[0].IsLocalEdit {
					t.Fatalf("供給元更新だけの場合はローカル変更ではない想定ですがLocalEdit=true")
				}
				_ = sourceHash
			},
		},
		{
			name: "UpdatesWhenDestinationEditedAndSourceUnchanged",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncFixture(t)
				sourcePath, sourceHash := fixture.writeProjectSource("alpha", "alpha-v1\n")
				targetHash := fixture.distributeAlpha(DestinationCodex)

				input := SyncInput{
					WorkspaceRoot: fixture.workspace,
					Project:       "project",
					Destinations:  []Destination{DestinationCodex},
					Skills: []RecordedSkill{{
						Name: "alpha", Source: SkillSourceProject, Destination: DestinationCodex,
						RelativePath: ".codex/skills/alpha", RecordedHash: sourceHash,
					}},
				}
				sources := []ResolvedSource{{
					Name: "alpha", Source: SkillSourceProject, State: SourceStateActive, Path: sourcePath,
					Hash: sourceHash,
				}}
				// RecordedHash=sourceHashと一致し、配布先=targetHash=sourceHash（同一内容）。
				// 配布先を少し編集してsourceHashと一致しない状態にする。
				if err := os.WriteFile(
					filepath.Join(fixture.workspace, ".codex", "skills", "alpha", "SKILL.md"),
					[]byte("local-edit\n"), 0o600,
				); err != nil {
					t.Fatal(err)
				}
				plan := fixture.runPlanner(input, sources)

				if len(plan.Updates) != 1 {
					t.Fatalf("Updates = %d, want 1", len(plan.Updates))
				}
				if !plan.Updates[0].IsLocalEdit {
					t.Fatal("配布先ローカル編集を検出する必要があります")
				}
				if len(plan.LocalChanges) != 1 {
					t.Fatalf("LocalChanges = %d, want 1", len(plan.LocalChanges))
				}
				_ = targetHash
			},
		},
		{
			name: "KeepsWhenNoChange",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncFixture(t)
				sourcePath, sourceHash := fixture.writeProjectSource("alpha", "alpha-v1\n")
				_ = fixture.distributeAlpha(DestinationCodex)

				input := SyncInput{
					WorkspaceRoot: fixture.workspace,
					Project:       "project",
					Destinations:  []Destination{DestinationCodex},
					Skills: []RecordedSkill{{
						Name: "alpha", Source: SkillSourceProject, Destination: DestinationCodex,
						RelativePath: ".codex/skills/alpha", RecordedHash: sourceHash,
					}},
				}
				sources := []ResolvedSource{{
					Name: "alpha", Source: SkillSourceProject, State: SourceStateActive, Path: sourcePath,
					Hash: sourceHash,
				}}
				plan := fixture.runPlanner(input, sources)

				if len(plan.Updates) != 0 || len(plan.Deletes) != 0 {
					t.Fatalf("Updates=%d Deletes=%d, want both 0", len(plan.Updates), len(plan.Deletes))
				}
				if len(plan.Keeps) != 1 {
					t.Fatalf("Keeps = %d, want 1", len(plan.Keeps))
				}
				if len(plan.UpdatedWorkspace.Skills) != 1 {
					t.Fatalf("UpdatedWorkspace.Skills = %d, want 1", len(plan.UpdatedWorkspace.Skills))
				}
			},
		},
		{
			name: "UpdatesWhenDestinationMissing",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncFixture(t)
				sourcePath, sourceHash := fixture.writeProjectSource("alpha", "alpha-v1\n")

				input := SyncInput{
					WorkspaceRoot: fixture.workspace,
					Project:       "project",
					Destinations:  []Destination{DestinationCodex},
					Skills: []RecordedSkill{{
						Name: "alpha", Source: SkillSourceProject, Destination: DestinationCodex,
						RelativePath: ".codex/skills/alpha", RecordedHash: sourceHash,
					}},
				}
				sources := []ResolvedSource{{
					Name: "alpha", Source: SkillSourceProject, State: SourceStateActive, Path: sourcePath,
					Hash: sourceHash,
				}}
				plan := fixture.runPlanner(input, sources)

				if len(plan.Updates) != 1 {
					t.Fatalf("Updates = %d, want 1", len(plan.Updates))
				}
				if !plan.Updates[0].IsLocalEdit {
					t.Fatal("配布先欠落はローカル変更扱いにする必要があります")
				}
			},
		},
		{
			name: "UpdatesRegularFileLeaf",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncFixture(t)
				sourcePath, sourceHash := fixture.writeProjectSource("alpha", "alpha-v1\n")
				// 配布先を通常ファイルへ置き換える
				leafPath := filepath.Join(fixture.workspace, ".codex", "skills", "alpha")
				if err := os.MkdirAll(filepath.Dir(leafPath), 0o755); err != nil { // #nosec G301 -- 配布先要件を再現します。
					t.Fatal(err)
				}
				if err := os.WriteFile(leafPath, []byte("file"), 0o600); err != nil {
					t.Fatal(err)
				}

				input := SyncInput{
					WorkspaceRoot: fixture.workspace,
					Project:       "project",
					Destinations:  []Destination{DestinationCodex},
					Skills: []RecordedSkill{{
						Name: "alpha", Source: SkillSourceProject, Destination: DestinationCodex,
						RelativePath: ".codex/skills/alpha", RecordedHash: sourceHash,
					}},
				}
				sources := []ResolvedSource{{
					Name: "alpha", Source: SkillSourceProject, State: SourceStateActive, Path: sourcePath,
					Hash: sourceHash,
				}}
				plan := fixture.runPlanner(input, sources)

				if len(plan.Updates) != 1 {
					t.Fatalf("Updates = %d, want 1", len(plan.Updates))
				}
				if !plan.Updates[0].IsLocalEdit {
					t.Fatal("通常ファイルへの種別変化はローカル変更にする必要があります")
				}
			},
		},
		{
			name: "UpdatesFIFOLeaf",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncFixture(t)
				sourcePath, sourceHash := fixture.writeProjectSource("alpha", "alpha-v1\n")
				leafPath := filepath.Join(fixture.workspace, ".codex", "skills", "alpha")
				if err := os.MkdirAll(filepath.Dir(leafPath), 0o755); err != nil { // #nosec G301 -- 配布先要件を再現します。
					t.Fatal(err)
				}
				if err := unix.Mkfifo(leafPath, 0o600); err != nil {
					t.Fatal(err)
				}

				input := SyncInput{
					WorkspaceRoot: fixture.workspace,
					Project:       "project",
					Destinations:  []Destination{DestinationCodex},
					Skills: []RecordedSkill{{
						Name: "alpha", Source: SkillSourceProject, Destination: DestinationCodex,
						RelativePath: ".codex/skills/alpha", RecordedHash: sourceHash,
					}},
				}
				sources := []ResolvedSource{{
					Name: "alpha", Source: SkillSourceProject, State: SourceStateActive, Path: sourcePath,
					Hash: sourceHash,
				}}
				plan := fixture.runPlanner(input, sources)

				if len(plan.Updates) != 1 || !plan.Updates[0].IsLocalEdit {
					t.Fatalf("FIFO末端は更新＋ローカル変更にする必要があります: %+v", plan.Updates)
				}
			},
		},
		{
			name: "RejectsLeafSymlink",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncFixture(t)
				sourcePath, sourceHash := fixture.writeProjectSource("alpha", "alpha-v1\n")
				leafPath := filepath.Join(fixture.workspace, ".codex", "skills", "alpha")
				if err := os.MkdirAll(filepath.Dir(leafPath), 0o755); err != nil { // #nosec G301 -- 配布先要件を再現します。
					t.Fatal(err)
				}
				outside := filepath.Join(fixture.root, "outside")
				if err := os.MkdirAll(outside, 0o755); err != nil { // #nosec G301 -- リンク先ディレクトリを作成します。
					t.Fatal(err)
				}
				if err := os.Symlink(outside, leafPath); err != nil {
					t.Fatal(err)
				}

				input := SyncInput{
					WorkspaceRoot: fixture.workspace,
					Project:       "project",
					Destinations:  []Destination{DestinationCodex},
					Skills: []RecordedSkill{{
						Name: "alpha", Source: SkillSourceProject, Destination: DestinationCodex,
						RelativePath: ".codex/skills/alpha", RecordedHash: sourceHash,
					}},
				}
				sources := []ResolvedSource{{
					Name: "alpha", Source: SkillSourceProject, State: SourceStateActive, Path: sourcePath,
					Hash: sourceHash,
				}}
				_, err := NewSyncPlanner(NewOSFileSystem()).Plan(MapSnapshot{Revision: "rev"}, input, sources)
				if !errors.Is(err, ErrSymlink) && !errors.Is(err, ErrFileType) {
					t.Fatalf("error = %v, want ErrSymlink or ErrFileType", err)
				}
			},
		},
		{
			name: "RejectsParentSymlink",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncFixture(t)
				sourcePath, sourceHash := fixture.writeProjectSource("alpha", "alpha-v1\n")
				parent := filepath.Join(fixture.workspace, ".codex", "skills")
				if err := os.MkdirAll(parent, 0o755); err != nil { // #nosec G301 -- 配布先親を作成します。
					t.Fatal(err)
				}
				outside := filepath.Join(fixture.root, "outside")
				if err := os.MkdirAll(outside, 0o755); err != nil { // #nosec G301 -- リンク先を作成します。
					t.Fatal(err)
				}
				swapped := parent + "-real"
				if err := os.Rename(parent, swapped); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, parent); err != nil {
					t.Fatal(err)
				}

				input := SyncInput{
					WorkspaceRoot: fixture.workspace,
					Project:       "project",
					Destinations:  []Destination{DestinationCodex},
					Skills: []RecordedSkill{{
						Name: "alpha", Source: SkillSourceProject, Destination: DestinationCodex,
						RelativePath: ".codex/skills/alpha", RecordedHash: sourceHash,
					}},
				}
				sources := []ResolvedSource{{
					Name: "alpha", Source: SkillSourceProject, State: SourceStateActive, Path: sourcePath,
					Hash: sourceHash,
				}}
				_, err := NewSyncPlanner(NewOSFileSystem()).Plan(MapSnapshot{Revision: "rev"}, input, sources)
				if !errors.Is(err, ErrSymlink) && !errors.Is(err, ErrFileType) {
					t.Fatalf("error = %v, want ErrSymlink or ErrFileType", err)
				}
			},
		},
		{
			name: "DeletesWhenSourceMissing",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncFixture(t)
				targetHash := fixture.distributeAlpha(DestinationCodex)

				input := SyncInput{
					WorkspaceRoot: fixture.workspace,
					Project:       "project",
					Destinations:  []Destination{DestinationCodex},
					Skills: []RecordedSkill{{
						Name: "alpha", Source: SkillSourceProject, Destination: DestinationCodex,
						RelativePath: ".codex/skills/alpha", RecordedHash: targetHash,
					}},
				}
				sources := []ResolvedSource{{
					Name: "alpha", Source: SkillSourceProject, State: SourceStateMissing,
				}}
				plan := fixture.runPlanner(input, sources)

				if len(plan.Deletes) != 1 {
					t.Fatalf("Deletes = %d, want 1", len(plan.Deletes))
				}
				if len(plan.UpdatedWorkspace.Skills) != 0 {
					t.Fatalf("消失Skillは更新後記録から除外する必要があります: %v", plan.UpdatedWorkspace.Skills)
				}
			},
		},
		{
			name: "DeletesWhenSourceDisabled",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncFixture(t)
				// SKILL.mdを欠落させた無効化状態の供給元
				disabledPath := filepath.Join(fixture.projectBase, "alpha")
				if err := os.MkdirAll(disabledPath, 0o755); err != nil { // #nosec G301 -- Skillディレクトリを作成します。
					t.Fatal(err)
				}
				targetHash := fixture.distributeAlpha(DestinationCodex)

				input := SyncInput{
					WorkspaceRoot: fixture.workspace,
					Project:       "project",
					Destinations:  []Destination{DestinationCodex},
					Skills: []RecordedSkill{{
						Name: "alpha", Source: SkillSourceProject, Destination: DestinationCodex,
						RelativePath: ".codex/skills/alpha", RecordedHash: targetHash,
					}},
				}
				sources := []ResolvedSource{{
					Name: "alpha", Source: SkillSourceProject, State: SourceStateDisabled,
				}}
				plan := fixture.runPlanner(input, sources)

				if len(plan.Deletes) != 1 {
					t.Fatalf("Deletes = %d, want 1", len(plan.Deletes))
				}
			},
		},
		{
			name: "HandlesMultipleDestinations",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncFixture(t)
				sourcePath, sourceHash := fixture.writeProjectSource("alpha", "alpha-v1\n")
				targetHashCodex := fixture.distributeAlpha(DestinationCodex)
				targetHashClaude := fixture.distributeAlpha(DestinationClaude)

				input := SyncInput{
					WorkspaceRoot: fixture.workspace,
					Project:       "project",
					Destinations:  []Destination{DestinationCodex, DestinationClaude},
					Skills: []RecordedSkill{
						{
							Name: "alpha", Source: SkillSourceProject, Destination: DestinationCodex,
							RelativePath: ".codex/skills/alpha", RecordedHash: sourceHash,
						},
						{
							Name: "alpha", Source: SkillSourceProject, Destination: DestinationClaude,
							RelativePath: ".claude/skills/alpha", RecordedHash: sourceHash,
						},
					},
				}
				sources := []ResolvedSource{{
					Name: "alpha", Source: SkillSourceProject, State: SourceStateActive, Path: sourcePath,
					Hash: sourceHash,
				}}
				plan := fixture.runPlanner(input, sources)

				if len(plan.Keeps) != 2 {
					t.Fatalf("Keeps = %d, want 2", len(plan.Keeps))
				}
				_ = targetHashCodex
				_ = targetHashClaude
			},
		},
		{
			name: "EmptiesWorkspaceWhenAllSkillsRemoved",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncFixture(t)
				codexHash := fixture.distributeAlpha(DestinationCodex)

				input := SyncInput{
					WorkspaceRoot: fixture.workspace,
					Project:       "project",
					Destinations:  []Destination{DestinationCodex},
					Skills: []RecordedSkill{{
						Name: "alpha", Source: SkillSourceProject, Destination: DestinationCodex,
						RelativePath: ".codex/skills/alpha", RecordedHash: codexHash,
					}},
				}
				sources := []ResolvedSource{{
					Name: "alpha", Source: SkillSourceProject, State: SourceStateMissing,
				}}
				plan := fixture.runPlanner(input, sources)

				if len(plan.Deletes) != 1 {
					t.Fatalf("Deletes = %d, want 1", len(plan.Deletes))
				}
				if len(plan.UpdatedWorkspace.Skills) != 0 {
					t.Fatalf("全消失時は空Workspace記録にする必要があります: %v", plan.UpdatedWorkspace.Skills)
				}
			},
		},
		{
			name: "OrdersOperationsDeterministically",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncFixture(t)
				sourcePath, sourceHash := fixture.writeProjectSource("beta", "beta-v1\n")
				sourcePathAlpha, sourceHashAlpha := fixture.writeProjectSource("alpha", "alpha-v1\n")

				input := SyncInput{
					WorkspaceRoot: fixture.workspace,
					Project:       "project",
					Destinations:  []Destination{DestinationCodex, DestinationClaude},
					Skills: []RecordedSkill{
						{
							Name: "beta", Source: SkillSourceProject, Destination: DestinationCodex,
							RelativePath: ".codex/skills/beta", RecordedHash: sourceHash,
						},
						{
							Name: "alpha", Source: SkillSourceProject, Destination: DestinationClaude,
							RelativePath: ".claude/skills/alpha", RecordedHash: sourceHashAlpha,
						},
					},
				}
				sources := []ResolvedSource{
					{Name: "beta", Source: SkillSourceProject, State: SourceStateActive, Path: sourcePath, Hash: sourceHash},
					{Name: "alpha", Source: SkillSourceProject, State: SourceStateActive, Path: sourcePathAlpha, Hash: sourceHashAlpha},
				}
				plan := fixture.runPlanner(input, sources)

				if len(plan.Updates) != 2 {
					t.Fatalf("Updates = %d, want 2", len(plan.Updates))
				}
				if plan.Updates[0].Name != "alpha" {
					t.Fatalf("Updates[0].Name = %q, want alpha", plan.Updates[0].Name)
				}
				if plan.Updates[1].Name != "beta" {
					t.Fatalf("Updates[1].Name = %q, want beta", plan.Updates[1].Name)
				}
			},
		},
		{
			name: "RejectsInvalidInput",
			run: func(t *testing.T) {
				t.Helper()
				tests := []struct {
					name  string
					input SyncInput
				}{
					{
						name: "相対WorkspaceRoot",
						input: SyncInput{
							WorkspaceRoot: "relative",
							Project:       "project",
						},
					},
					{
						name: "不正プロジェクト名",
						input: SyncInput{
							WorkspaceRoot: "/workspace",
							Project:       "../escape",
						},
					},
					{
						name: "重複配布先",
						input: SyncInput{
							WorkspaceRoot: "/workspace",
							Project:       "project",
							Destinations:  []Destination{DestinationCodex, DestinationCodex},
						},
					},
				}
				for _, test := range tests {
					t.Run(test.name, func(t *testing.T) {
						_, err := NewSyncPlanner(NewOSFileSystem()).Plan(MapSnapshot{}, test.input, nil)
						if err == nil {
							t.Fatalf("Plan() error = nil, want error")
						}
					})
				}
			},
		},
		{
			name: "ToPlanConvertsOperations",
			run: func(t *testing.T) {
				t.Helper()
				fixture := newSyncFixture(t)
				sourcePath, sourceHash := fixture.writeProjectSource("alpha", "alpha-v1\n")
				targetHash := fixture.distributeAlpha(DestinationCodex)

				input := SyncInput{
					WorkspaceRoot: fixture.workspace,
					Project:       "project",
					Destinations:  []Destination{DestinationCodex},
					Skills: []RecordedSkill{{
						Name: "alpha", Source: SkillSourceProject, Destination: DestinationCodex,
						RelativePath: ".codex/skills/alpha", RecordedHash: targetHash,
					}},
				}
				sources := []ResolvedSource{{
					Name: "alpha", Source: SkillSourceProject, State: SourceStateMissing,
				}}
				plan := fixture.runPlanner(input, sources)

				execPlan, err := plan.ToPlan()
				if err != nil {
					t.Fatalf("ToPlan() error = %v", err)
				}
				if !execPlan.IsSync {
					t.Fatal("IsSync should be true")
				}
				if len(execPlan.Deletes) != 1 {
					t.Fatalf("Deletes = %d, want 1", len(execPlan.Deletes))
				}
				if len(execPlan.Creates) != 0 {
					t.Fatalf("Creates = %d, want 0", len(execPlan.Creates))
				}
				_ = sourcePath
				_ = sourceHash
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}
