package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

//nolint:gocognit,cyclop // テーブル駆動のE2Eテストのため、複雑度の上限を無視します。
func TestDeleteE2E(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "Success_Specific",
			run: func(t *testing.T) {
				t.Helper()
				repository := createRepositoryFixture(t)
				projectSkillA := filepath.Join(repository, "projects", "project", "skills", "skill-a")
				projectSkillB := filepath.Join(repository, "projects", "project", "skills", "skill-b")

				for _, skill := range []string{projectSkillA, projectSkillB} {
					if err := os.MkdirAll(skill, 0o700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("skill content"), 0o600); err != nil {
						t.Fatal(err)
					}
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

				// 初期配布（両方のSkillをCodexへ配布）
				resultAdd := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"add", "project"},
				}, []terminalStep{
					{waitFor: "プロジェクト固有Skillを選択してください", input: " \x1b[B \r"}, // 両方選択
					{waitFor: "配布先を選択してください", input: " \r"},                  // Codexのみ
				})
				if resultAdd.exitCode != 0 {
					t.Fatalf("context add failed: %s", resultAdd.stdout)
				}

				// skill-a のみを削除
				resultDelete := runContextProcess(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"delete", "skill-a"},
				})
				if resultDelete.exitCode != 0 {
					t.Fatalf("context delete failed: %s\nstderr: %s", resultDelete.stdout, resultDelete.stderr)
				}

				if !strings.Contains(resultDelete.stdout, "1件のSkillを削除しました") {
					t.Fatalf("unexpected success output: %s", resultDelete.stdout)
				}

				// skill-a が削除され、skill-b は維持されていることを検証
				removedPath := filepath.Join(workspace, ".codex", "skills", "skill-a")
				if _, err := os.Stat(removedPath); !os.IsNotExist(err) {
					t.Fatalf("skill-a should be deleted, but still exists: %s", removedPath)
				}

				keepPath := filepath.Join(workspace, ".codex", "skills", "skill-b", "SKILL.md")
				if _, err := os.Stat(keepPath); err != nil {
					t.Fatalf("skill-b should remain, but error: %v", err)
				}

				// map.yaml の管理記録を検証
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
				if len(skills) != 1 || skills[0].Name != "skill-b" {
					t.Fatalf("expected only skill-b in map.yaml records, got: %#v", skills)
				}
			},
		},
		{
			name: "Success_All",
			run: func(t *testing.T) {
				t.Helper()
				repository := createRepositoryFixture(t)
				projectSkill := filepath.Join(repository, "projects", "project", "skills", "skill-a")

				if err := os.MkdirAll(projectSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(projectSkill, "SKILL.md"), []byte("skill content"), 0o600); err != nil {
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

				// 初期配布
				resultAdd := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"add", "project"},
				}, []terminalStep{
					{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"},
					{waitFor: "配布先を選択してください", input: " \r"},
				})
				if resultAdd.exitCode != 0 {
					t.Fatalf("context add failed: %s", resultAdd.stdout)
				}

				// 全削除を実行
				resultDelete := runContextProcess(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"delete", "--all"},
				})
				if resultDelete.exitCode != 0 {
					t.Fatalf("context delete --all failed: %s\nstderr: %s", resultDelete.stdout, resultDelete.stderr)
				}

				if !strings.Contains(resultDelete.stdout, "1件のSkillを削除しました") {
					t.Fatalf("unexpected success output: %s", resultDelete.stdout)
				}

				// 配布ファイルが消えていること
				removedPath := filepath.Join(workspace, ".codex", "skills", "skill-a")
				if _, err := os.Stat(removedPath); !os.IsNotExist(err) {
					t.Fatalf("skill-a should be deleted, but still exists: %s", removedPath)
				}

				// map.yaml から Workspace レコードが削除されていること
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
			name: "Interactive_Select",
			run: func(t *testing.T) {
				t.Helper()
				repository := createRepositoryFixture(t)
				projectSkillA := filepath.Join(repository, "projects", "project", "skills", "skill-a")
				projectSkillB := filepath.Join(repository, "projects", "project", "skills", "skill-b")

				for _, skill := range []string{projectSkillA, projectSkillB} {
					if err := os.MkdirAll(skill, 0o700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("content"), 0o600); err != nil {
						t.Fatal(err)
					}
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

				// 初期配布
				resultAdd := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"add", "project"},
				}, []terminalStep{
					{waitFor: "プロジェクト固有Skillを選択してください", input: " \x1b[B \r"},
					{waitFor: "配布先を選択してください", input: " \r"},
				})
				if resultAdd.exitCode != 0 {
					t.Fatalf("context add failed: %s", resultAdd.stdout)
				}

				// 対話UIで skill-a を選択して削除
				// 初期状態では未選択なので、スペースキーで最初の skill-a をチェックし、Enterで決定
				resultDelete := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"delete"},
				}, []terminalStep{
					{waitFor: "削除するSkillを選択してください", input: " \r"},
				})
				if resultDelete.exitCode != 0 {
					t.Fatalf("context delete failed: %s", resultDelete.stdout)
				}

				if !strings.Contains(resultDelete.stdout, "1件のSkillを削除しました") {
					t.Fatalf("unexpected success output: %s", resultDelete.stdout)
				}

				// skill-a が削除され、skill-b は維持されていることを検証
				removedPath := filepath.Join(workspace, ".codex", "skills", "skill-a")
				if _, err := os.Stat(removedPath); !os.IsNotExist(err) {
					t.Fatalf("skill-a should be deleted: %s", removedPath)
				}

				keepPath := filepath.Join(workspace, ".codex", "skills", "skill-b", "SKILL.md")
				if _, err := os.Stat(keepPath); err != nil {
					t.Fatalf("skill-b should remain: %v", err)
				}
			},
		},
		{
			name: "Interactive_Cancel",
			run: func(t *testing.T) {
				t.Helper()
				repository := createRepositoryFixture(t)
				projectSkill := filepath.Join(repository, "projects", "project", "skills", "skill-a")

				if err := os.MkdirAll(projectSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(projectSkill, "SKILL.md"), []byte("content"), 0o600); err != nil {
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

				// 初期配布
				resultAdd := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"add", "project"},
				}, []terminalStep{
					{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"},
					{waitFor: "配布先を選択してください", input: " \r"},
				})
				if resultAdd.exitCode != 0 {
					t.Fatalf("context add failed: %s", resultAdd.stdout)
				}

				// 対話UIをキャンセル (Ctrl-C)
				resultDelete := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"delete"},
				}, []terminalStep{
					{waitFor: "削除するSkillを選択してください", input: "\x03"}, // Ctrl-C
				})
				if resultDelete.exitCode != 0 {
					t.Fatalf("context delete cancel failed: %s", resultDelete.stdout)
				}

				// skill-a が削除されていないことを検証
				keepPath := filepath.Join(workspace, ".codex", "skills", "skill-a", "SKILL.md")
				if _, err := os.Stat(keepPath); err != nil {
					t.Fatalf("skill-a should remain: %v", err)
				}
			},
		},
		{
			name: "Unmanaged",
			run: func(t *testing.T) {
				t.Helper()
				workspace, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				configHome, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}

				// 未管理Workspaceで実行
				resultDelete := runContextProcess(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"delete", "skill-a"},
				})
				if resultDelete.exitCode != 1 {
					t.Fatalf("expected exit code 1, got %d. output:\n%s\nstderr:\n%s", resultDelete.exitCode, resultDelete.stdout, resultDelete.stderr)
				}
				if !strings.Contains(resultDelete.stderr, "workspace is not managed") {
					t.Fatalf("unexpected error message: %s", resultDelete.stderr)
				}
			},
		},
		{
			name: "NonexistentSkill",
			run: func(t *testing.T) {
				t.Helper()
				repository := createRepositoryFixture(t)
				projectSkill := filepath.Join(repository, "projects", "project", "skills", "skill-a")

				if err := os.MkdirAll(projectSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(projectSkill, "SKILL.md"), []byte("content"), 0o600); err != nil {
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

				// 初期配布
				resultAdd := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"add", "project"},
				}, []terminalStep{
					{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"},
					{waitFor: "配布先を選択してください", input: " \r"},
				})
				if resultAdd.exitCode != 0 {
					t.Fatalf("context add failed: %s", resultAdd.stdout)
				}

				// 存在しないSkill名を指定して削除を実行
				resultDelete := runContextProcess(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"delete", "nonexistent-skill"},
				})
				if resultDelete.exitCode != 1 {
					t.Fatalf("expected exit code 1, got %d. output:\n%s\nstderr:\n%s", resultDelete.exitCode, resultDelete.stdout, resultDelete.stderr)
				}
				if !strings.Contains(resultDelete.stderr, `skill "nonexistent-skill" is not distributed in this workspace`) {
					t.Fatalf("unexpected error message: %s", resultDelete.stderr)
				}
			},
		},
		{
			name: "Interactive_LocalEdit",
			run: func(t *testing.T) {
				t.Helper()
				repository := createRepositoryFixture(t)
				projectSkill := filepath.Join(repository, "projects", "project", "skills", "skill-a")

				if err := os.MkdirAll(projectSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(projectSkill, "SKILL.md"), []byte("content"), 0o600); err != nil {
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

				// 初期配布
				resultAdd := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"add", "project"},
				}, []terminalStep{
					{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"},
					{waitFor: "配布先を選択してください", input: " \r"},
				})
				if resultAdd.exitCode != 0 {
					t.Fatalf("context add failed: %s", resultAdd.stdout)
				}

				targetFile := filepath.Join(workspace, ".codex", "skills", "skill-a", "SKILL.md")

				// 1. ローカル編集ありの拒否ケース
				if err := os.WriteFile(targetFile, []byte("local modified content"), 0o600); err != nil {
					t.Fatal(err)
				}

				resultReject := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"delete", "skill-a"},
				}, []terminalStep{
					{waitFor: "以下のファイルがローカルで編集されています", input: "n\r"}, // いいえ
				})
				if resultReject.exitCode != 0 {
					t.Fatalf("expected exit code 0, got %d", resultReject.exitCode)
				}

				// 削除されていないことを検証
				if _, err := os.Stat(targetFile); err != nil {
					t.Fatalf("target file should remain: %v", err)
				}

				// 2. ローカル編集ありの承認ケース
				resultApprove := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"delete", "skill-a"},
				}, []terminalStep{
					{waitFor: "以下のファイルがローカルで編集されています", input: "y\r"}, // はい
				})
				if resultApprove.exitCode != 0 {
					t.Fatalf("expected exit code 0, got %d", resultApprove.exitCode)
				}

				// 削除されたことを検証
				removedPath := filepath.Join(workspace, ".codex", "skills", "skill-a")
				if _, err := os.Stat(removedPath); !os.IsNotExist(err) {
					t.Fatalf("skill-a should be deleted: %s", removedPath)
				}
			},
		},
		{
			name: "LocalEdit_NoTTY",
			run: func(t *testing.T) {
				t.Helper()
				repository := createRepositoryFixture(t)
				projectSkill := filepath.Join(repository, "projects", "project", "skills", "skill-a")

				if err := os.MkdirAll(projectSkill, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(projectSkill, "SKILL.md"), []byte("content"), 0o600); err != nil {
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

				// 初期配布
				resultAdd := runContextTerminal(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"add", "project"},
				}, []terminalStep{
					{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"},
					{waitFor: "配布先を選択してください", input: " \r"},
				})
				if resultAdd.exitCode != 0 {
					t.Fatalf("context add failed: %s", resultAdd.stdout)
				}

				// ローカルで編集
				targetFile := filepath.Join(workspace, ".codex", "skills", "skill-a", "SKILL.md")
				if err := os.WriteFile(targetFile, []byte("local modified content"), 0o600); err != nil {
					t.Fatal(err)
				}

				// 非TTYで削除を実行
				resultDelete := runContextProcess(t, processRequest{
					xdgConfigHome:    configHome,
					workingDirectory: workspace,
					args:             []string{"delete", "skill-a"},
				})
				if resultDelete.exitCode != 1 {
					t.Fatalf("expected exit code 1, got %d. output:\n%s\nstderr:\n%s", resultDelete.exitCode, resultDelete.stdout, resultDelete.stderr)
				}
				if !strings.Contains(resultDelete.stderr, "distribution local change requires approval") {
					t.Fatalf("unexpected error message: %s", resultDelete.stderr)
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
