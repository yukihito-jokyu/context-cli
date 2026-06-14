# init / add E2E テスト

`test/e2e` は公開されたCobraコマンド境界から `context init --repo` の処理フローを確認する。
各テストは一時ディレクトリとメモリ内Configを使用し、利用者の実設定を変更しない。
相対パスのテストでプロセスの作業ディレクトリを変更するため、テストは直列実行する。

## シナリオ

| ID       | 事前条件                                                              | CLI 操作                                | 期待結果                                                         | Go サブテスト           |
| -------- | --------------------------------------------------------------------- | --------------------------------------- | ---------------------------------------------------------------- | ----------------------- |
| INIT-001 | `projects/` と `utils/skills/` を持つリポジトリを相対パスで参照できる | `context init --repo context`           | 正規化済み絶対パスを保存し、成功メッセージへ表示する             | `TestE2E_Init/INIT-001` |
| INIT-002 | `utils/skills/` が存在しない                                          | `context init --repo <repository>`      | 必須構造不足として拒否し、設定を変更しない                       | `TestE2E_Init/INIT-002` |
| INIT-003 | 指定したリポジトリ自体がシンボリックリンク                            | `context init --repo <repository-link>` | リンク検出として拒否し、設定を変更せず、エラーへパスを漏洩しない | `TestE2E_Init/INIT-003` |
| INIT-004 | `projects/` がシンボリックリンク                                      | `context init --repo <repository>`      | リンク検出として拒否し、設定を変更せず、エラーへパスを漏洩しない | `TestE2E_Init/INIT-004` |
| INIT-005 | 現在値と異なる有効なリポジトリが設定済み                              | `context init --repo <new>`、`y` を入力 | 現在値と変更先を表示し、新しいパスを1回保存して成功する          | `TestE2E_Init/INIT-005` |
| INIT-006 | 現在値と異なる有効なリポジトリが設定済み                              | `context init --repo <new>`、`n` を入力 | 現在値と変更先を表示し、設定を変更せず正常終了する               | `TestE2E_Init/INIT-006` |
| INIT-007 | 指定した有効なリポジトリと同じパスが設定済み                          | `context init --repo <same>`            | 確認と保存を省略し、成功メッセージだけを表示する                 | `TestE2E_Init/INIT-007` |

## 永続化E2E

`TestE2E_InitPersistence` は実際の `context` バイナリを遅延ビルドし、ケース専用の
`XDG_CONFIG_HOME` を渡した別プロセスとして実行する。利用者の設定は参照せず、
同じシナリオ内のプロセスだけが設定ファイルを共有する。

| ID                | 操作                                      | 期待結果                                                           |
| ----------------- | ----------------------------------------- | ------------------------------------------------------------------ |
| PERSIST-001       | 初回設定後、別プロセスで同一パスを指定    | 終了コード0。確認なし。設定ファイルの内容、更新日時、実体を維持    |
| PERSIST-002       | 初回設定後、別パスを指定して `y` を入力   | 終了コード0。確認後に新しい絶対パスを保存                          |
| PERSIST-003       | 初回設定後、別パスを指定して `n` を入力   | 終了コード0。成功表示なし。既存設定を維持                          |
| PERSIST-004       | 初回設定後、別パスを指定してstdinを閉じる | 終了コード0。成功表示なし。既存設定を維持                          |
| InvalidRepository | 必須構造を欠くRepositoryを指定            | 終了コード1。パスを含まないエラーを1回だけ表示し、設定を作成しない |

各ケースはstdout、stderr、終了コードを完全一致で検証する。保存されたYAMLは
未知フィールドと複数文書を拒否してデコードし、スキーマバージョンと保存パスを確認する。

## 実行方法

全シナリオを一括で実行する。

```bash
task test:e2e
```

個別のシナリオを実行する。

```bash
go test ./test/e2e/... -run 'TestE2E_Init/INIT-001' -v
go test ./test/e2e/... -run 'TestE2E_InitPersistence/PERSIST-001-same-path$' -v
```

E2Eテストはシナリオを一覧しやすいテーブル駆動テストとして記述する。
シナリオを追加または変更する場合は、対応するテーブル要素とこの表を同じ変更で更新する。

## add初回配布

`TestAddDistributesSkillsAndPersistsMap` は実バイナリを固定サイズの擬似TTY上で起動し、
各プロンプト文字列を待ってからキー入力する。プロジェクト固有Skillと共通Skillを
Codex・Claude双方へ配布し、複数ファイル、実行権限、`map.yaml`の内容と`0600`を確認する。

`TestAddRejectsExistingUnmanagedSkillWithoutChanges` は配布予定先に同名Skillがある状態で
初回配布し、既存内容と管理情報を変更せず競合終了することを確認する。

```bash
go test ./test/e2e -run TestAdd -v
```
