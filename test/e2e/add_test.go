package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

//nolint:gocognit,cyclop // 実対話、配布内容、権限、管理情報を一つのE2Eシナリオで検証します。
func TestAddDistributesSkillsAndPersistsMap(t *testing.T) {
	repository := createRepositoryFixture(t)
	projectSkill := filepath.Join(repository, "projects", "project", "skills", "project-skill")
	projectSkill2 := filepath.Join(repository, "projects", "project", "skills", "project-skill-2")
	commonSkill := filepath.Join(repository, "utils", "skills", "common-skill")
	for _, skill := range []string{projectSkill, projectSkill2, commonSkill} {
		if err := os.MkdirAll(filepath.Join(skill, "scripts"), 0o755); err != nil { // #nosec G301 -- Skillディレクトリの配布要件を再現します。
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte(filepath.Base(skill)), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skill, "scripts", "run.sh"), []byte("#!/bin/sh\n"), 0o750); err != nil { // #nosec G306 -- 実行権限の維持をE2Eで検証します。
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

	result := runContextTerminal(t, processRequest{
		xdgConfigHome:    configHome,
		workingDirectory: workspace,
		args:             []string{"add", "project"},
	}, []terminalStep{
		{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"},
		{waitFor: "共通Skillを追加しますか?", input: "y\r"},
		{waitFor: "共通Skillを選択してください", input: " \r"},
		{waitFor: "配布先を選択してください", input: " \x1b[B \r"},
	})
	if result.exitCode != 0 {
		t.Fatalf("exit code = %d\noutput:\n%s", result.exitCode, result.stdout)
	}
	if !strings.Contains(result.stdout, "4件 of Skillをclaude, codexへ配布しました") && !strings.Contains(result.stdout, "4件のSkillをclaude, codexへ配布しました") {
		t.Fatalf("success output is missing:\n%s", result.stdout)
	}

	for _, destination := range []string{".codex", ".claude"} {
		for _, skill := range []string{"project-skill", "common-skill"} {
			target := filepath.Join(workspace, destination, "skills", skill)
			if _, err := os.Stat(filepath.Join(target, "SKILL.md")); err != nil {
				t.Fatalf("distributed file is missing: %v", err)
			}
			info, err := os.Stat(filepath.Join(target, "scripts", "run.sh"))
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0o750 {
				t.Fatalf("run.sh mode = %o, want 750", info.Mode().Perm())
			}
		}
	}
	mapPath := filepath.Join(configHome, "context", "map.yaml")
	info, err := os.Stat(mapPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("map.yaml mode = %o, want 600", info.Mode().Perm())
	}
	data, err := os.ReadFile(mapPath)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		SchemaVersion int            `yaml:"schema_version"`
		Workspaces    map[string]any `yaml:"workspaces"`
	}
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if document.SchemaVersion != 1 || len(document.Workspaces) != 1 {
		t.Fatalf("map.yaml = %s", data)
	}

	// --- 2回目の実行：前回の選択を復元し、一部Skillの解除と追加を行う ---
	result2 := runContextTerminal(t, processRequest{
		xdgConfigHome:    configHome,
		workingDirectory: workspace,
		args:             []string{"add", "project"},
	}, []terminalStep{
		{waitFor: "プロジェクト固有Skillを選択してください", input: " \x1b[B \r"}, // project-skill解除、project-skill-2選択
		{waitFor: "共通Skillを追加しますか?", input: "y\r"},               // 共通Skill追加を承認
		{waitFor: "共通Skillを選択してください", input: "\r"},               // common-skillは選択されたままなのでそのまま決定
		{waitFor: "配布先を選択してください", input: "\r"},                   // claude, codexは選択されたままなのでそのまま決定
	})
	if result2.exitCode != 0 {
		t.Fatalf("exit code = %d\noutput:\n%s", result2.exitCode, result2.stdout)
	}

	// project-skillが削除されていること、project-skill-2が配置されていることを検証
	for _, destination := range []string{".codex", ".claude"} {
		// 削除されているべきファイル
		oldSkill := filepath.Join(workspace, destination, "skills", "project-skill")
		if _, err := os.Stat(oldSkill); !os.IsNotExist(err) {
			t.Fatalf("removed skill path still exists: %s", oldSkill)
		}

		// 新たに配置されているべきファイル
		newSkill := filepath.Join(workspace, destination, "skills", "project-skill-2")
		if _, err := os.Stat(filepath.Join(newSkill, "SKILL.md")); err != nil {
			t.Fatalf("distributed file is missing: %v", err)
		}

		// common-skillは維持されていること
		keepSkill := filepath.Join(workspace, destination, "skills", "common-skill")
		if _, err := os.Stat(filepath.Join(keepSkill, "SKILL.md")); err != nil {
			t.Fatalf("maintained file is missing: %v", err)
		}
	}

	// map.yamlが正しく更新されていることを検証
	data, err = os.ReadFile(mapPath)
	if err != nil {
		t.Fatal(err)
	}
	var document2 struct {
		SchemaVersion int            `yaml:"schema_version"`
		Workspaces    map[string]any `yaml:"workspaces"`
	}
	if err := yaml.Unmarshal(data, &document2); err != nil {
		t.Fatal(err)
	}
	if document2.SchemaVersion != 1 || len(document2.Workspaces) != 1 {
		t.Fatalf("map.yaml mismatch after 2nd run = %s", data)
	}

	// --- 3回目の実行：すべてのSkillを解除し、全削除を実行する ---
	result3 := runContextTerminal(t, processRequest{
		xdgConfigHome:    configHome,
		workingDirectory: workspace,
		args:             []string{"add", "project"},
	}, []terminalStep{
		{waitFor: "プロジェクト固有Skillを選択してください", input: "\x1b[B \r"}, // project-skill-2を解除
		{waitFor: "共通Skillを追加しますか?", input: "n\r"},              // 共通Skillを追加しない（common-skillが解除される）
	})
	if result3.exitCode != 0 {
		t.Fatalf("exit code = %d\noutput:\n%s", result3.exitCode, result3.stdout)
	}

	// すべての配布先からSkillディレクトリが削除されていることを検証
	for _, destination := range []string{".codex", ".claude"} {
		skillsDir := filepath.Join(workspace, destination, "skills")
		if _, err := os.Stat(skillsDir); err == nil {
			// ディレクトリ自体は残っているかもしれないが、中は空のはず
			entries, err := os.ReadDir(skillsDir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) > 0 {
				t.Fatalf("skills directory is not empty: %s", skillsDir)
			}
		}
	}

	// map.yamlからWorkspaceの記述が完全に削除されていることを検証
	data, err = os.ReadFile(mapPath)
	if err != nil {
		t.Fatal(err)
	}
	var document3 struct {
		SchemaVersion int            `yaml:"schema_version"`
		Workspaces    map[string]any `yaml:"workspaces"`
	}
	if err := yaml.Unmarshal(data, &document3); err != nil {
		t.Fatal(err)
	}
	if len(document3.Workspaces) != 0 {
		t.Fatalf("expected 0 workspaces in map.yaml, but got: %s", data)
	}
}

func TestAddRejectsExistingUnmanagedSkillWithoutChanges(t *testing.T) {
	repository := createRepositoryFixture(t)
	source := filepath.Join(repository, "projects", "project", "skills", "skill")
	if err := os.MkdirAll(source, 0o755); err != nil { // #nosec G301 -- 配布元Skillの権限要件を再現します。
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	workspace, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(workspace, ".codex", "skills", "skill")
	if err := os.MkdirAll(existing, 0o755); err != nil { // #nosec G301 -- 未管理同名Skillの権限要件を再現します。
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existing, "SKILL.md"), []byte("local"), 0o600); err != nil {
		t.Fatal(err)
	}
	configHome, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	initializeRepository(t, configHome, workspace, repository)

	result := runContextTerminal(t, processRequest{
		xdgConfigHome:    configHome,
		workingDirectory: workspace,
		args:             []string{"add", "project"},
	}, []terminalStep{
		{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"},
		{waitFor: "配布先を選択してください", input: " \r"},
		{waitFor: "以下の未管理ファイルが衝突しています", input: "n\r"}, // 拒否
	})
	if result.exitCode != 0 {
		t.Fatalf("existing unmanaged skill rejection failed\noutput:\n%s", result.stdout)
	}
	data, err := os.ReadFile(filepath.Join(existing, "SKILL.md")) // #nosec G304 -- テストが作成した隔離Workspace内の固定相対パスです。
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "local" {
		t.Fatalf("existing skill was changed: %q", data)
	}
	if _, err := os.Stat(filepath.Join(configHome, "context", "map.yaml")); !os.IsNotExist(err) {
		t.Fatalf("map.yaml was created: %v", err)
	}
}

func initializeRepository(t *testing.T, configHome, workspace, repository string) {
	t.Helper()
	result := runContextProcess(t, processRequest{
		xdgConfigHome:    configHome,
		workingDirectory: workspace,
		args:             []string{"init", "--repo", repository},
	})
	if result.exitCode != 0 {
		t.Fatalf("context init failed: stderr=%s", result.stderr)
	}
}

//nolint:gocognit,cyclop // 一連のE2E対話と結果検証の流れを一つのテストケースとして表現します。
func TestAddSwitchProject(t *testing.T) {
	// 既存管理中のプロジェクトから異なるプロジェクトに切り替える際、
	// 旧プロジェクト由来の管理対象がすべて削除され、新プロジェクトの選択Skillのみが正しく配置されることを検証します。
	repository := createRepositoryFixture(t)
	projectASkill := filepath.Join(repository, "projects", "project-a", "skills", "skill-a")
	projectBSkill := filepath.Join(repository, "projects", "project-b", "skills", "skill-b")
	commonSkill := filepath.Join(repository, "utils", "skills", "common-skill")

	for _, skill := range []string{projectASkill, projectBSkill, commonSkill} {
		// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
		if err := os.MkdirAll(skill, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte(filepath.Base(skill)), 0o600); err != nil {
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

	// 1. 初回実行：project-a の skill-a と共通Skill を claude と codex へ配布
	result1 := runContextTerminal(t, processRequest{
		xdgConfigHome:    configHome,
		workingDirectory: workspace,
		args:             []string{"add", "project-a"},
	}, []terminalStep{
		{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"},
		{waitFor: "共通Skillを追加しますか?", input: "y\r"},
		{waitFor: "共通Skillを選択してください", input: " \r"},
		{waitFor: "配布先を選択してください", input: " \x1b[B \r"},
	})
	if result1.exitCode != 0 {
		t.Fatalf("exit code = %d\noutput:\n%s", result1.exitCode, result1.stdout)
	}

	// 2. 2回目：引数なしで add を実行し、プロジェクトを project-b へ切り替えて、skill-b のみを配布
	result2 := runContextTerminal(t, processRequest{
		xdgConfigHome:    configHome,
		workingDirectory: workspace,
		args:             []string{"add"},
	}, []terminalStep{
		{waitFor: "プロジェクトを選択してください", input: "\x1b[B\r"},   // 下キーで project-b に切り替える
		{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"}, // skill-bを選択（初期選択がクリアされているためスペースキーでチェック）
		{waitFor: "共通Skillを追加しますか?", input: "n\r"},        // 共通Skillを追加しない（前回値も引き継がれずクリアされる）
		{waitFor: "配布先を選択してください", input: "\r"},            // 前回配布先（claude, codex）が初期選択された状態でEnter
	})
	if result2.exitCode != 0 {
		t.Fatalf("exit code = %d\noutput:\n%s", result2.exitCode, result2.stdout)
	}

	// 3. 検証：プロジェクトAの旧Skillおよび共通Skillが削除され、プロジェクトBの新Skillのみが配置されていること
	for _, destination := range []string{".codex", ".claude"} {
		// 削除されているべき旧Skill
		oldSkill := filepath.Join(workspace, destination, "skills", "skill-a")
		if _, err := os.Stat(oldSkill); !os.IsNotExist(err) {
			t.Fatalf("removed project-a skill still exists: %s", oldSkill)
		}

		// 削除されているべき共通Skill
		oldCommon := filepath.Join(workspace, destination, "skills", "common-skill")
		if _, err := os.Stat(oldCommon); !os.IsNotExist(err) {
			t.Fatalf("removed common skill still exists: %s", oldCommon)
		}

		// 新規配置されているべきSkill
		newSkill := filepath.Join(workspace, destination, "skills", "skill-b")
		if _, err := os.Stat(filepath.Join(newSkill, "SKILL.md")); err != nil {
			t.Fatalf("distributed project-b skill is missing: %v", err)
		}
	}

	// 4. map.yamlがプロジェクトBの情報へ正しく更新されていることを検証
	mapPath := filepath.Join(configHome, "context", "map.yaml")
	data, err := os.ReadFile(mapPath)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		SchemaVersion int `yaml:"schema_version"`
		Workspaces    map[string]struct {
			Project      string   `yaml:"project"`
			Destinations []string `yaml:"destinations"`
			Skills       []struct {
				Name string `yaml:"name"`
			} `yaml:"skills"`
		} `yaml:"workspaces"`
	}
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}

	ws, exists := document.Workspaces[workspace]
	if !exists {
		t.Fatalf("workspace record is missing from map.yaml: %s", data)
	}
	if ws.Project != "project-b" {
		t.Fatalf("expected project to be project-b, got %q", ws.Project)
	}
	if len(ws.Skills) != 2 { // Codex向けとClaude向けで計2レコード
		t.Fatalf("expected 2 skill records (Codex/Claude), got %d: %#v", len(ws.Skills), ws.Skills)
	}
	for _, skill := range ws.Skills {
		if skill.Name != "skill-b" {
			t.Fatalf("expected skill name to be skill-b, got %q", skill.Name)
		}
	}
}

//nolint:gocognit,cyclop // 一連のE2E対話と結果検証の流れを一つのテストケースとして表現します。
func TestAddProtectsConflictsAndLocalEdits(t *testing.T) {
	// 未管理競合やローカル編集が存在する場合に、承認プロンプトを介して保護および上書きができることを検証します。
	repository := createRepositoryFixture(t)
	projectSkill := filepath.Join(repository, "projects", "project", "skills", "project-skill")
	commonSkill := filepath.Join(repository, "utils", "skills", "common-skill")

	for _, skill := range []string{projectSkill, commonSkill} {
		// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
		if err := os.MkdirAll(skill, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("repository-skill"), 0o600); err != nil {
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

	// 1. 未管理の同名ファイルをあらかじめ配置しておく
	conflictDest := filepath.Join(workspace, ".codex", "skills", "project-skill")
	// #nosec G301 -- テスト環境のディレクトリ作成のため、検証要件に合わせて0755を使用します。
	if err := os.MkdirAll(conflictDest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(conflictDest, "SKILL.md"), []byte("local-unmanaged"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 2. 未管理競合の拒否ケース：警告画面で「いいえ」を選択
	result1 := runContextTerminal(t, processRequest{
		xdgConfigHome:    configHome,
		workingDirectory: workspace,
		args:             []string{"add", "project"},
	}, []terminalStep{
		{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"},
		{waitFor: "共通Skillを追加しますか?", input: "n\r"},
		{waitFor: "配布先を選択してください", input: " \r"},
		{waitFor: "以下の未管理ファイルが衝突しています", input: "n\r"}, // 拒否
	})
	if result1.exitCode != 0 {
		t.Fatalf("exit code = %d\noutput:\n%s", result1.exitCode, result1.stdout)
	}

	// 変更されていないことを検証
	data, err := os.ReadFile(filepath.Join(conflictDest, "SKILL.md")) // #nosec G304 -- テストが作成した隔離Workspace内の固定相対パスです。
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "local-unmanaged" {
		t.Fatalf("expected file to remain unchanged, but got %q", data)
	}

	// 3. 未管理競合の承認ケース：警告画面で「はい」を選択
	result2 := runContextTerminal(t, processRequest{
		xdgConfigHome:    configHome,
		workingDirectory: workspace,
		args:             []string{"add", "project"},
	}, []terminalStep{
		{waitFor: "プロジェクト固有Skillを選択してください", input: " \r"},
		{waitFor: "共通Skillを追加しますか?", input: "n\r"},
		{waitFor: "配布先を選択してください", input: " \r"},
		{waitFor: "以下の未管理ファイルが衝突しています", input: "y\r"}, // 承認
	})
	if result2.exitCode != 0 {
		t.Fatalf("exit code = %d\noutput:\n%s", result2.exitCode, result2.stdout)
	}

	// 上書きされたことを検証
	data, err = os.ReadFile(filepath.Join(conflictDest, "SKILL.md")) // #nosec G304 -- テストが作成した隔離Workspace内の固定相対パスです。
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "repository-skill" {
		t.Fatalf("expected file to be overwritten, but got %q", data)
	}

	// 4. ローカル編集の拒否ケース：管理対象となったファイルをローカルで編集する
	if err := os.WriteFile(filepath.Join(conflictDest, "SKILL.md"), []byte("local-edited"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 警告画面で「いいえ」を選択
	result3 := runContextTerminal(t, processRequest{
		xdgConfigHome:    configHome,
		workingDirectory: workspace,
		args:             []string{"add", "project"},
	}, []terminalStep{
		{waitFor: "プロジェクト固有Skillを選択してください", input: "\r"}, // 前回選択維持のまま決定
		{waitFor: "共通Skillを追加しますか?", input: "n\r"},
		{waitFor: "配布先を選択してください", input: "\r"},
		{waitFor: "以下のファイルがローカルで編集されています", input: "n\r"}, // 拒否
	})
	if result3.exitCode != 0 {
		t.Fatalf("exit code = %d\noutput:\n%s", result3.exitCode, result3.stdout)
	}

	// 変更されていないことを検証
	data, err = os.ReadFile(filepath.Join(conflictDest, "SKILL.md")) // #nosec G304 -- テストが作成した隔離Workspace内の固定相対パスです。
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "local-edited" {
		t.Fatalf("expected file to remain local-edited, but got %q", data)
	}

	// 5. ローカル編集の承認ケース：警告画面で「はい」を選択
	result4 := runContextTerminal(t, processRequest{
		xdgConfigHome:    configHome,
		workingDirectory: workspace,
		args:             []string{"add", "project"},
	}, []terminalStep{
		{waitFor: "プロジェクト固有Skillを選択してください", input: "\r"}, // 前回選択維持のまま決定
		{waitFor: "共通Skillを追加しますか?", input: "n\r"},
		{waitFor: "配布先を選択してください", input: "\r"},
		{waitFor: "以下のファイルがローカルで編集されています", input: "y\r"}, // 承認
	})
	if result4.exitCode != 0 {
		t.Fatalf("exit code = %d\noutput:\n%s", result4.exitCode, result4.stdout)
	}

	// 上書きされたことを検証
	data, err = os.ReadFile(filepath.Join(conflictDest, "SKILL.md")) // #nosec G304 -- テストが作成した隔離Workspace内の固定相対パスです。
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "repository-skill" {
		t.Fatalf("expected file to be overwritten, but got %q", data)
	}
}
