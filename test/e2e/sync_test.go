package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

//nolint:gocognit,cyclop // テーブル駆動テストのため、認知・循環複雑度の上限を無視します。
func TestSyncE2E(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "Success_NoTTY",
			run: func(t *testing.T) {
				t.Helper()
				repository := createRepositoryFixture(t)
				projectSkill := filepath.Join(repository, "projects", "project", "skills", "project-skill")
				commonSkill := filepath.Join(repository, "utils", "skills", "common-skill")

				// 供給元のSkillを作成
				if err := os.MkdirAll(projectSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(projectSkill, "SKILL.md"), []byte("project-skill v1"), 0o600); err != nil {
					t.Fatal(err)
				}

				if err := os.MkdirAll(commonSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(commonSkill, "SKILL.md"), []byte("common-skill v1"), 0o600); err != nil {
					t.Fatal(err)
				}

				workspace, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				configHome, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				initializeRepository(t, configHome, workspace, repository)

				// 初期配布を実行
				resultAdd := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"add", "project"},
				}, []terminalStep{
					{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"},
					{waitFor: "共通Skillを追加しますか?", input: "y\r"},
					{waitFor: "共通Skillを選択してください", input: " \r"},
					{waitFor: "配布先を選択してください", input: " \x1b[B \r"},
				})
				if resultAdd.exitCode != 0 {
					t.Fatalf("context add failed: %s", resultAdd.stdout)
				}

				// 供給元のSkill内容を変更
				if err := os.WriteFile(filepath.Join(projectSkill, "SKILL.md"), []byte("project-skill v2"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(commonSkill, "SKILL.md"), []byte("common-skill v2"), 0o600); err != nil {
					t.Fatal(err)
				}

				// 非TTY同期を実行
				resultSync := runContextProcess(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"sync"},
				})
				if resultSync.exitCode != 0 {
					t.Fatalf("context sync failed: %s\nstderr: %s", resultSync.stdout, resultSync.stderr)
				}

				// 出力の件数が一意のSkill名単位で数えられていることを検証
				// project-skill と common-skill はそれぞれ codex/claude 両方に配布されたが、更新件数は「2件」であるべき
				if !strings.Contains(resultSync.stdout, "2件のSkillを更新し、0件を削除しました") {
					t.Fatalf("unexpected success output: %s", resultSync.stdout)
				}

				// 配布先が更新されていることを検証
				for _, dest := range []string{".codex", ".claude"} {
					pContent, err := os.ReadFile(filepath.Join(workspace, dest, "skills", "project-skill", "SKILL.md"))
					if err != nil {
						t.Fatal(err)
					}
					if string(pContent) != "project-skill v2" {
						t.Fatalf("expected project-skill to be v2, got %q", pContent)
					}

					cContent, err := os.ReadFile(filepath.Join(workspace, dest, "skills", "common-skill", "SKILL.md"))
					if err != nil {
						t.Fatal(err)
					}
					if string(cContent) != "common-skill v2" {
						t.Fatalf("expected common-skill to be v2, got %q", cContent)
					}
				}
			},
		},
		{
			name: "NoChange",
			run: func(t *testing.T) {
				t.Helper()
				repository := createRepositoryFixture(t)
				projectSkill := filepath.Join(repository, "projects", "project", "skills", "project-skill")
				commonSkill := filepath.Join(repository, "utils", "skills", "common-skill")

				if err := os.MkdirAll(projectSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(projectSkill, "SKILL.md"), []byte("project-skill v1"), 0o600); err != nil {
					t.Fatal(err)
				}

				if err := os.MkdirAll(commonSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(commonSkill, "SKILL.md"), []byte("common-skill v1"), 0o600); err != nil {
					t.Fatal(err)
				}

				workspace, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				configHome, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				initializeRepository(t, configHome, workspace, repository)

				resultAdd := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"add", "project"},
				}, []terminalStep{
					{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"},
					{waitFor: "共通Skillを追加しますか?", input: "n\r"},
					{waitFor: "配布先を選択してください", input: " \r"},
				})
				if resultAdd.exitCode != 0 {
					t.Fatalf("context add failed: %s", resultAdd.stdout)
				}

				mapPath := filepath.Join(configHome, "context", "map.yaml")
				mapStatBefore, err := os.Stat(mapPath)
				if err != nil {
					t.Fatal(err)
				}
				destPath := filepath.Join(workspace, ".codex", "skills", "project-skill", "SKILL.md")
				destStatBefore, err := os.Stat(destPath)
				if err != nil {
					t.Fatal(err)
				}

				// 僅かに時間経過を待つ
				time.Sleep(100 * time.Millisecond)

				// 変更なし状態でsyncを実行
				resultSync := runContextProcess(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"sync"},
				})
				if resultSync.exitCode != 0 {
					t.Fatalf("context sync failed: %s", resultSync.stdout)
				}

				if !strings.Contains(resultSync.stdout, "同期対象に変更はありません") {
					t.Fatalf("unexpected output: %s", resultSync.stdout)
				}

				// mtimeなどのファイルメタデータが変わっていないことを検証
				mapStatAfter, err := os.Stat(mapPath)
				if err != nil {
					t.Fatal(err)
				}
				if mapStatAfter.ModTime() != mapStatBefore.ModTime() {
					t.Fatal("map.yaml modified time changed, but should be unchanged")
				}

				destStatAfter, err := os.Stat(destPath)
				if err != nil {
					t.Fatal(err)
				}
				if destStatAfter.ModTime() != destStatBefore.ModTime() {
					t.Fatal("distributed file modified time changed, but should be unchanged")
				}
			},
		},
		{
			name: "DeleteSkill",
			run: func(t *testing.T) {
				t.Helper()
				repository := createRepositoryFixture(t)
				projectSkill := filepath.Join(repository, "projects", "project", "skills", "project-skill")
				commonSkill := filepath.Join(repository, "utils", "skills", "common-skill")

				if err := os.MkdirAll(projectSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(projectSkill, "SKILL.md"), []byte("project-skill v1"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(commonSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(commonSkill, "SKILL.md"), []byte("common-skill v1"), 0o600); err != nil {
					t.Fatal(err)
				}

				workspace, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				configHome, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				initializeRepository(t, configHome, workspace, repository)

				resultAdd := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"add", "project"},
				}, []terminalStep{
					{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"},
					{waitFor: "共通Skillを追加しますか?", input: "y\r"},
					{waitFor: "共通Skillを選択してください", input: " \r"},
					{waitFor: "配布先を選択してください", input: " \r"},
				})
				if resultAdd.exitCode != 0 {
					t.Fatalf("context add failed: %s", resultAdd.stdout)
				}

				// 供給元から project-skill を完全に削除（消失）させる
				if err := os.RemoveAll(projectSkill); err != nil {
					t.Fatal(err)
				}

				resultSync := runContextProcess(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"sync"},
				})
				if resultSync.exitCode != 0 {
					t.Fatalf("context sync failed: %s", resultSync.stdout)
				}

				if !strings.Contains(resultSync.stdout, "0件 of Skillを更新し、1件を削除しました") && !strings.Contains(resultSync.stdout, "0件のSkillを更新し、1件を削除しました") {
					t.Fatalf("unexpected success output: %s", resultSync.stdout)
				}

				// 配布先から削除されたことを検証
				removedPath := filepath.Join(workspace, ".codex", "skills", "project-skill")
				if _, err := os.Stat(removedPath); !os.IsNotExist(err) {
					t.Fatalf("project-skill should be deleted, but still exists: %s", removedPath)
				}

				// 共通Skillは残っていることを検証
				keepPath := filepath.Join(workspace, ".codex", "skills", "common-skill", "SKILL.md")
				if _, err := os.Stat(keepPath); err != nil {
					t.Fatalf("common-skill should remain, but error: %v", err)
				}

				// map.yaml の管理記録から project-skill が削除され、common-skill のみになっていることを検証
				mapPath := filepath.Join(configHome, "context", "map.yaml")
				data, err := os.ReadFile(mapPath)
				if err != nil {
					t.Fatal(err)
				}
				var document struct {
					Workspaces map[string]struct {
						Skills []struct {
							Name string `yaml:"name"`
						} `yaml:"skills"`
					} `yaml:"workspaces"`
				}
				if err := yaml.Unmarshal(data, &document); err != nil {
					t.Fatal(err)
				}
				skills := document.Workspaces[workspace].Skills
				if len(skills) != 1 || skills[0].Name != "common-skill" {
					t.Fatalf("expected only common-skill in map.yaml records, got: %#v", skills)
				}
			},
		},
		{
			name: "DeleteAllSkills",
			run: func(t *testing.T) {
				t.Helper()
				repository := createRepositoryFixture(t)
				projectSkill := filepath.Join(repository, "projects", "project", "skills", "project-skill")
				commonSkill := filepath.Join(repository, "utils", "skills", "common-skill")

				if err := os.MkdirAll(projectSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(projectSkill, "SKILL.md"), []byte("project-skill v1"), 0o600); err != nil {
					t.Fatal(err)
				}

				if err := os.MkdirAll(commonSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(commonSkill, "SKILL.md"), []byte("common-skill v1"), 0o600); err != nil {
					t.Fatal(err)
				}

				workspace, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				configHome, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				initializeRepository(t, configHome, workspace, repository)

				resultAdd := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"add", "project"},
				}, []terminalStep{
					{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"},
					{waitFor: "共通Skillを追加しますか?", input: "n\r"},
					{waitFor: "配布先を選択してください", input: " \r"},
				})
				if resultAdd.exitCode != 0 {
					t.Fatalf("context add failed: %s", resultAdd.stdout)
				}

				// 唯一のSkillを消失させる
				if err := os.RemoveAll(projectSkill); err != nil {
					t.Fatal(err)
				}

				resultSync := runContextProcess(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"sync"},
				})
				if resultSync.exitCode != 0 {
					t.Fatalf("context sync failed: %s", resultSync.stdout)
				}

				if !strings.Contains(resultSync.stdout, "0件のSkillを更新し、1件を削除しました") {
					t.Fatalf("unexpected success output: %s", resultSync.stdout)
				}

				// 配布先からSkillが削除されていること
				removedPath := filepath.Join(workspace, ".codex", "skills", "project-skill")
				if _, err := os.Stat(removedPath); !os.IsNotExist(err) {
					t.Fatalf("project-skill should be deleted, but still exists: %s", removedPath)
				}

				// map.yaml からWorkspace記録が削除されていることを検証
				mapPath := filepath.Join(configHome, "context", "map.yaml")
				data, err := os.ReadFile(mapPath)
				if err != nil {
					t.Fatal(err)
				}
				var document struct {
					Workspaces map[string]any `yaml:"workspaces"`
				}
				if err := yaml.Unmarshal(data, &document); err != nil {
					t.Fatal(err)
				}
				if len(document.Workspaces) != 0 {
					t.Fatalf("expected workspaces to be empty, but got: %s", data)
				}
			},
		},
		{
			name: "UnselectedSkill",
			run: func(t *testing.T) {
				t.Helper()
				repository := createRepositoryFixture(t)
				projectSkill := filepath.Join(repository, "projects", "project", "skills", "project-skill")
				unselectedSkill := filepath.Join(repository, "projects", "project", "skills", "unselected-skill")
				commonSkill := filepath.Join(repository, "utils", "skills", "common-skill")

				// 登録するSkillと、未選択のSkillを両方供給元に作成
				if err := os.MkdirAll(projectSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(projectSkill, "SKILL.md"), []byte("project-skill v1"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(unselectedSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(unselectedSkill, "SKILL.md"), []byte("unselected-skill v1"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(commonSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(commonSkill, "SKILL.md"), []byte("common-skill v1"), 0o600); err != nil {
					t.Fatal(err)
				}

				workspace, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				configHome, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				initializeRepository(t, configHome, workspace, repository)

				// project-skill のみを配布（unselected-skillは選択しない）
				resultAdd := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"add", "project"},
				}, []terminalStep{
					{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"},
					{waitFor: "共通Skillを追加しますか?", input: "n\r"},
					{waitFor: "配布先を選択してください", input: " \r"},
				})
				if resultAdd.exitCode != 0 {
					t.Fatalf("context add failed: %s", resultAdd.stdout)
				}

				// 既存の未管理配布物（同名の未管理フォルダー・ファイル）を手動で配布先に作成しておく
				unmanagedDir := filepath.Join(workspace, ".codex", "skills", "unselected-skill")
				if err := os.MkdirAll(unmanagedDir, 0o700); err != nil {
					t.Fatal(err)
				}
				unmanagedFile := filepath.Join(unmanagedDir, "SKILL.md")
				if err := os.WriteFile(unmanagedFile, []byte("local-unmanaged-content"), 0o600); err != nil {
					t.Fatal(err)
				}
				mtimeBefore := time.Now().Add(-1 * time.Hour)
				if err := os.Chtimes(unmanagedFile, mtimeBefore, mtimeBefore); err != nil {
					t.Fatal(err)
				}

				// 供給元のSkill内容を変更
				if err := os.WriteFile(filepath.Join(projectSkill, "SKILL.md"), []byte("project-skill v2"), 0o600); err != nil {
					t.Fatal(err)
				}
				// 未選択側の供給元も変更
				if err := os.WriteFile(filepath.Join(unselectedSkill, "SKILL.md"), []byte("unselected-skill v2"), 0o600); err != nil {
					t.Fatal(err)
				}

				// 同期を実行
				resultSync := runContextProcess(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"sync"},
				})
				if resultSync.exitCode != 0 {
					t.Fatalf("context sync failed: %s", resultSync.stdout)
				}

				// 配布先の unselected-skill が一切変更されていないこと（内容と mtime の不変）を検証
				// #nosec G304 -- テスト作成用の隔離パスです。
				unmanagedContent, err := os.ReadFile(unmanagedFile)
				if err != nil {
					t.Fatal(err)
				}
				if string(unmanagedContent) != "local-unmanaged-content" {
					t.Fatalf("unselected skill content changed: %q", unmanagedContent)
				}
				stat, err := os.Stat(unmanagedFile)
				if err != nil {
					t.Fatal(err)
				}
				if !stat.ModTime().Equal(mtimeBefore) {
					t.Fatalf("unselected skill mtime changed: %s", stat.ModTime())
				}

				// map.yaml の管理記録にも unselected-skill が追加されていないことを検証
				mapPath := filepath.Join(configHome, "context", "map.yaml")
				data, err := os.ReadFile(mapPath)
				if err != nil {
					t.Fatal(err)
				}
				var document struct {
					Workspaces map[string]struct {
						Skills []struct {
							Name string `yaml:"name"`
						} `yaml:"skills"`
					} `yaml:"workspaces"`
				}
				if err := yaml.Unmarshal(data, &document); err != nil {
					t.Fatal(err)
				}
				skills := document.Workspaces[workspace].Skills
				if len(skills) != 1 || skills[0].Name != "project-skill" {
					t.Fatalf("expected only project-skill in map.yaml records, got: %#v", skills)
				}
			},
		},
		{
			name: "Interactive_Confirm",
			run: func(t *testing.T) {
				t.Helper()
				repository := createRepositoryFixture(t)
				projectSkill := filepath.Join(repository, "projects", "project", "skills", "project-skill")
				commonSkill := filepath.Join(repository, "utils", "skills", "common-skill")

				if err := os.MkdirAll(projectSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(projectSkill, "SKILL.md"), []byte("project-skill v1"), 0o600); err != nil {
					t.Fatal(err)
				}

				if err := os.MkdirAll(commonSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(commonSkill, "SKILL.md"), []byte("common-skill v1"), 0o600); err != nil {
					t.Fatal(err)
				}

				workspace, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				configHome, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				initializeRepository(t, configHome, workspace, repository)

				resultAdd := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"add", "project"},
				}, []terminalStep{
					{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"},
					{waitFor: "共通Skillを追加しますか?", input: "n\r"},
					{waitFor: "配布先を選択してください", input: " \r"},
				})
				if resultAdd.exitCode != 0 {
					t.Fatalf("context add failed: %s", resultAdd.stdout)
				}

				targetFile := filepath.Join(workspace, ".codex", "skills", "project-skill", "SKILL.md")

				// 1. TTY 拒否 (Negative)
				// 配布先のファイルをローカル変更する
				if err := os.WriteFile(targetFile, []byte("project-skill local-edit"), 0o600); err != nil {
					t.Fatal(err)
				}
				resultReject := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"sync"},
				}, []terminalStep{
					{waitFor: "これらの変更を承認し、同期を実行しますか？", input: "n\r"},
				})
				if resultReject.exitCode != 0 {
					t.Fatalf("sync reject exit code = %d", resultReject.exitCode)
				}
				// ファイル内容が変わっていないことを検証
				rejectContent, err := os.ReadFile(targetFile)
				if err != nil {
					t.Fatal(err)
				}
				if string(rejectContent) != "project-skill local-edit" {
					t.Fatalf("expected file to remain local-edit, but got %q", rejectContent)
				}

				// 2. TTY キャンセル (Ctrl-C / User Abort)
				resultCancel := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"sync"},
				}, []terminalStep{
					{waitFor: "これらの変更を承認し、同期を実行しますか？", input: "\x03"}, // Ctrl-C
				})
				// キャンセル時も正常終了 (exit code 0) または abort処理となる
				if resultCancel.exitCode != 0 {
					t.Fatalf("sync cancel exit code = %d", resultCancel.exitCode)
				}
				cancelContent, err := os.ReadFile(targetFile)
				if err != nil {
					t.Fatal(err)
				}
				if string(cancelContent) != "project-skill local-edit" {
					t.Fatalf("expected file to remain local-edit, but got %q", cancelContent)
				}

				// 3. TTY 承認 (Affirmative)
				resultApprove := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"sync"},
				}, []terminalStep{
					{waitFor: "これらの変更を承認し、同期を実行しますか？", input: "y\r"},
				})
				if resultApprove.exitCode != 0 {
					t.Fatalf("sync approve exit code = %d", resultApprove.exitCode)
				}
				// ファイル内容が供給元の内容に更新されたことを検証
				approveContent, err := os.ReadFile(targetFile)
				if err != nil {
					t.Fatal(err)
				}
				if string(approveContent) != "project-skill v1" {
					t.Fatalf("expected file to be updated to v1, but got %q", approveContent)
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
