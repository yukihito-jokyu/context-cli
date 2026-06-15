# 03-delete-e2e-tests

- **対象ストーリー**: ST-001、ST-002、ST-003、ST-004、ST-005、ST-006

## 1. 処理フローチャート (Flowchart)

`spec.md` を参照。本タスクは実バイナリを別プロセスで駆動するE2Eテストの実装を担当する。

## 2. シーケンス図 (Sequence Diagram)

`spec.md` を参照。

## 3. ファイル配置・責務定義

- `[NEW]` [delete_test.go](file:///Users/yukihito/Documents/github_projects/context-cli/test/e2e/delete_test.go) : `context delete` コマンドの実動作を検証するE2Eテスト。
  - テストケース:
    - 正常系 (引数指定): `map.yaml` と配布先ファイルが存在するテスト環境で `context delete skill-a` を実行し、ファイル消去と `map.yaml` からの該当Skill削除を確認する。
    - 正常系 (一括削除): `context delete --all` で全Skillを削除し、ファイルがすべて消え、`map.yaml` からWorkspaceレコード自体が削除されること。
    - 正常系 (対話UI): 擬似TTYを起動し、複数選択画面で特定のSkillを選んで削除が完了すること、また選択をキャンセルした際に無変更で終了すること。
    - 異常系 (未管理): 未管理ディレクトリで実行した際に適切なエラーが表示され終了コード1となること。
    - 異常系 (存在しないSkill名): `context delete nonexistent-skill` を実行した際、エラーになりファイルが一切変更されないこと。
    - 異常系 (ローカル変更時の動作): ローカル変更（退避対象）がある場合、擬似TTYの確認で「いいえ」を選んだ場合に処理が中止され無変更で終了すること。「はい」を選んだ場合は削除されること。非TTYでローカル変更がある場合はエラー終了（Exit 1）となること。
- `[MODIFY]` [README.md](file:///Users/yukihito/Documents/github_projects/context-cli/test/e2e/README.md) : `context delete` 用のE2EテストシナリオID、操作、事前条件、期待結果、実行方法を追加する。

## 4. 実装チェックリスト

- [x] `README.md` にE2Eシナリオを追加
- [x] `delete_test.go` に隔離されたテストケースを作成
- [x] 実バイナリをビルドしてテストを実行（`task test` もしくは `go test ./test/e2e -run TestE2EDelete`）
- [x] すべてのE2Eテストがパスすることを確認
- [x] 品質ゲート（`gofmt`、`go vet`、`golangci-lint`、`govulncheck`、`go test ./...`）を実行して問題ないことを確認

## 5. テスト・検証計画

- **E2Eテスト方法**:
  - `go test ./test/e2e -run TestE2EDelete`
  - リポジトリ全体テスト: `go test ./...` (または `task test`)
