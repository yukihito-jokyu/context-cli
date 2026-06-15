# 02-delete-command-interface

- **対象ストーリー**: ST-001、ST-002、ST-003、ST-004、ST-005、ST-006

## 1. 処理フローチャート (Flowchart)

`spec.md` を参照。本タスクは `context delete` コマンド、引数/フラグ検証、対話選択UI、および全体の実行フローを担当する。

## 2. シーケンス図 (Sequence Diagram)

`spec.md` を参照。

## 3. ファイル配置・責務定義

- `[MODIFY]` [prompt.go](file:///Users/yukihito/Documents/github_projects/context-cli/pkg/cmd/prompt.go) : `Prompt` インターフェースに `SelectSkillsToDelete(candidates []string) ([]string, error)` を追加し、`huhPrompt` に実装する。
- `[NEW]` [delete.go](file:///Users/yukihito/Documents/github_projects/context-cli/pkg/cmd/delete.go) : `context delete` コマンド本体を実装する。
  - **構造体**: `DeleteOptions` を定義し、`Factory`、削除対象Skill名、`All` フラグを保持する。
  - **Runメソッドのフロー**:
    1. Workspaceの妥当性を検証（`WorkspaceValidator`）。
    2. `MapStore` から管理情報を読み込む（`snapshot`）。
    3. カレントディレクトリが管理対象か確認。未管理なら `ErrUnmanagedWorkspace`。
    4. 削除対象の決定:
       - **`--all` / `-a` が指定された場合**: 管理情報内のすべてのSkillを削除対象に設定。
       - **引数（Skill名）が指定された場合**: 引数を削除対象に設定。管理情報にないSkill名があれば、即座にエラー終了。
       - **引数と `--all` のいずれも指定されていない場合**: TTYでなければ `ErrNonTTY` エラー。TTYなら `SelectSkillsToDelete` を呼び出して削除対象を選択。
    5. 削除対象が空の場合、`削除対象のSkillはありません` と出力して正常終了（Exit 0）。
    6. `Planner.PlanDelete` を呼び出し、削除計画（`Plan`）を作成。
    7. 計画にローカル編集（`plan.Deletes` 内の `IsLocalEdit`）が含まれる場合、TTYなら `ConfirmOverwrite`（または `ConfirmSync`）で確認。拒否・キャンセル時は無変更で正常終了。非TTYなら `ErrLocalChange` を返して終了。
    8. `Executor.Execute(plan)` を実行。
    9. 成功時、`X件のSkillを削除しました` と標準出力へ表示。
- `[MODIFY]` [root.go](file:///Users/yukihito/Documents/github_projects/context-cli/pkg/cmd/root.go) : `NewCmdRoot` に `NewCmdDelete(f)` を追加する。
- `[NEW]` [delete_test.go](file:///Users/yukihito/Documents/github_projects/context-cli/pkg/cmd/delete_test.go) : `DeleteOptions.Run` を直接実行する単体テスト。
  - テストケース:
    - 引数指定による特定のSkill削除（正常終了、出力の検証）。
    - `--all` 指定による全Skillの削除（正常終了、出力の検証）。
    - 対話UI経由でのSkill削除（正常終了、キャンセル時の無変更終了）。
    - 未管理Workspaceでのエラー。
    - 存在しないSkill名を引数に含めた際のエラー。
    - ローカル編集検知時の確認プロンプト（承認による削除、拒否による無変更終了、非TTYでのエラー）。
    - `stubDistributionPlanner` への `PlanDelete` の追加（モック実装）。
- `[MODIFY]` [add_test.go](file:///Users/yukihito/Documents/github_projects/context-cli/pkg/cmd/add_test.go) : `stubDistributionPlanner` および `stubPrompt` に新しいメソッドを追加（インターフェース適合のため）。

## 4. 実装チェックリスト

- [x] `Prompt` インターフェースおよび `huhPrompt` の拡張
- [x] `root.go` へのコマンド登録
- [x] `DeleteOptions` および `NewCmdDelete` コマンドの作成
- [x] `add_test.go` 等のテストモックへのメソッド追加
- [x] `delete_test.go` にテーブル駆動テストを作成
- [x] `delete.go` の全体実行フロー実装とエラーハンドリング
- [x] 単体テストがすべてパスすることを確認

## 5. テスト・検証計画

- **単体テスト対象**:
  - `go test ./pkg/cmd -run TestCmdDelete`
