# 競合とローカル編集を保護する

- **ステータス**: 完了 (Completed)
- **対象ストーリー**: ST-005, ST-006

## 1. 処理フローチャート (Flowchart)

```mermaid
flowchart TD
    A["AddOptions.Run を開始"] --> B["選択結果を基に Planner.Plan を呼び出し"]
    B --> C["Planner.Plan 内で配布対象ごとに状態検証"]

    C --> D{"配布予定先は既存管理下か"}

    D -- "いいえ（未管理）" --> E{"予定先に同名ファイル・フォルダがあるか"}
    E -- "はい" --> F["IsConflict = true とマーク\n（エラーとせず計画へ記録）"]
    E -- "いいえ" --> G["正常な新規追加として計画へ記録"]

    D -- "はい（管理対象）" --> H{"実ファイルのハッシュ値が前回と一致するか\n（または欠落していないか）"}
    H -- "いいえ" --> I["IsLocalEdit = true とマーク"]
    H -- "はい" --> J["正常な更新として計画へ記録"]

    F & G & I & J --> K["作成・置換計画（Creates）と削除計画（Deletes）を構築"]
    K --> L["AddOptions.Run へPlanを返却"]

    L --> M{"Plan内に IsConflict または IsLocalEdit を持つ要素があるか"}

    M -- "はい" --> N["Prompt.ConfirmOverwrite で競合・ローカル編集一覧を表示し確認"]
    N --> O{"利用者は承認（はい）を選択したか"}
    O -- "いいえ（拒否またはキャンセル）" --> Z["何も変更せずに正常終了"]
    O -- "はい" --> P["Executor.Execute で計画をロック下で実行"]

    M -- "いいえ" --> P

    P --> Q["排他ロックを取得して期待状態（期待ハッシュ・リビジョン・存在）を再検証"]
    Q --> R{"状態は一致するか"}
    R -- "いいえ" --> X["競合エラーで終了（ロールバック不要）"]
    R -- "はい" --> S["原子的に配布と管理情報保存を実行し、正常終了"]
```

## 2. シーケンス図 (Sequence Diagram)

```mermaid
sequenceDiagram
    actor User as 利用者
    participant Add as AddOptions
    participant Prompt as pkg/cmd.Prompt
    participant Planner as distribution.Planner
    participant Executor as distribution.Executor
    participant Map as distributionmap.Store
    participant FS as distribution.FileSystem

    User->>Add: context add
    Add->>Map: map.yamlから管理情報を取得
    Map-->>Add: MapSnapshot (既存レコード)

    Add->>Prompt: プロジェクト、Skill、配布先を選択
    Prompt-->>Add: 選択結果 (Selection)

    Add->>Planner: Plan(snapshot, Selection)
    Planner->>FS: 配布先の現在のハッシュ値を計算
    FS-->>Planner: ハッシュ / 欠落状態 / リンク検証

    Note over Planner: 未管理同名の存在 -> IsConflict = true<br/>管理対象のハッシュ不一致・欠落 -> IsLocalEdit = true
    Planner-->>Add: Plan (Creates / Deletes にフラグを設定)

    alt 競合またはローカル編集が存在する
        Add->>Prompt: ConfirmOverwrite(conflicts, localEdits)
        Prompt-->>User: 変更点一覧と一括承認の確認表示
        User-->>Prompt: 承認（はい）
        Prompt-->>Add: true
    end

    Add->>Executor: Execute(Plan)
    Executor->>Map: Begin(Revision)
    Map-->>Executor: Transaction

    Executor->>FS: ロック下での期待状態の再検証
    FS-->>Executor: 一致

    Executor->>FS: 新旧Skillの退避・配置を実行
    Executor->>Map: Commit(WorkspaceRecord)
    Map-->>Executor: Commit成功

    Executor->>FS: 一時退避フォルダの削除
    Executor->>Map: Close() (ロック解放)
    Executor-->>Add: 成功
    Add-->>User: 完了表示
```

## 3. ファイル配置・責務定義

### internal/distribution

- **[MODIFY] [model.go](file:///Users/yukihito/Documents/github_projects/context-cli/internal/distribution/model.go)**
  - `CreateOperation` 構造体に `IsConflict bool` および `IsLocalEdit bool` フィールドを追加。
  - `DeleteOperation` 構造体に `IsLocalEdit bool` フィールドを追加。

- **[MODIFY] [planner.go](file:///Users/yukihito/Documents/github_projects/context-cli/internal/distribution/planner.go)**
  - `Plan` メソッドにおける `ErrConflict` による即時エラー返却ロジックを削除。
  - `fileSystem.HashSkill` を用いて、配布予定先に存在する実ファイルのハッシュ値と、前回の配布記録上のハッシュ値を比較し、不一致または欠落がある場合に `IsLocalEdit` を `true` に設定する。
  - 未管理の同名フォルダが存在する場合に `IsConflict` を `true` に設定する。
  - `Deletes` 計画構築ループ内においても、ディスク上の実ファイルの状態を検証し、前回のハッシュと異なる、あるいは欠落している場合に `DeleteOperation.IsLocalEdit` を `true` に設定する。

- **[MODIFY] [planner_test.go](file:///Users/yukihito/Documents/github_projects/context-cli/internal/distribution/planner_test.go)**
  - `TestPlannerDetectsConflictsAndLocalEdits` を追加。未管理の同名Skillが存在する場合に `IsConflict` が、ローカルでの変更・欠落がある場合に `IsLocalEdit` が正しく検出され、エラーにならず計画が返ることを検証する。

### pkg/cmd

- **[MODIFY] [prompt.go](file:///Users/yukihito/Documents/github_projects/context-cli/pkg/cmd/prompt.go)**
  - `Prompt` インターフェースに `ConfirmOverwrite(conflicts []string, localEdits []string) (bool, error)` を追加。
  - `huhPrompt` で `ConfirmOverwrite` メソッドを実装。`huh.NewConfirm` を用いて、衝突または変更のあるパスを一覧表示し、初期選択を「いいえ」（拒否）とした一括承認を求める。

- **[MODIFY] [add.go](file:///Users/yukihito/Documents/github_projects/context-cli/pkg/cmd/add.go)**
  - `Planner` から計画を受け取った後、`Creates` および `Deletes` 内に `IsConflict` または `IsLocalEdit` が設定された要素があるかスキャンする。
  - 該当要素がある場合、`Prompt.ConfirmOverwrite` を呼び出す。
  - ユーザーが承認しなかった場合（「いいえ」の選択、または対話キャンセル時）は、変更を適用せず正常終了（`return nil`）とする。

- **[MODIFY] [add_test.go](file:///Users/yukihito/Documents/github_projects/context-cli/pkg/cmd/add_test.go)**
  - `stubPrompt` に `ConfirmOverwrite` メソッドの実装を追加。
  - `TestAddOptionsRunPromptsForConflictsAndLocalEdits` を追加。モックプロンプトとMapStoreを用いて、競合やローカル編集が検出された際の承認・拒否に伴う実行可否、およびキャンセル時の無変更終了を検証する。

### test/e2e

- **[MODIFY] [add_test.go](file:///Users/yukihito/Documents/github_projects/context-cli/test/e2e/add_test.go)**
  - `TestAddProtectsConflictsAndLocalEdits` を追加。未管理の同名Skillがある場合の一括承認/拒否、およびローカル編集がある場合の一括承認/拒否のシナリオを、実端末対話プロセスを起動して検証する。

## 4. 実装チェックリスト

- [x] `model.go` へのフラグフィールド追加
- [x] `Planner` での競合・ローカル編集・欠落の検出ロジックの実装と単体テストのパス
- [x] `Prompt` への `ConfirmOverwrite` の追加と `huhPrompt` 実装の修正
- [x] `add.go` での承認フロー接続の実装とCLI単体テストのパス
- [x] E2Eテストへの競合・ローカル編集保護テストケースの追加とパス
- [x] 品質ゲートの実行（`golangci-lint run`, `go test ./...`）

## 5. テスト・検証計画

### E2E/結合テスト方法

- `go test -v ./test/e2e -run TestAddProtectsConflictsAndLocalEdits` を実行し、以下のシナリオを検証する：
  1. **未管理競合（拒否）**: 配布予定先に未管理の同名Skillが存在する状態で `context add` を実行。競合警告画面が表示されることを確認し、「いいえ」を選択。exitCode == 0で、配布先および `map.yaml` が変更されないことを検証。
  2. **未管理競合（承認）**: 同様の状態で実行し、競合警告に「はい」を選択。正常に上書き配布され、`map.yaml` が更新されることを検証。
  3. **ローカル編集（拒否）**: 配布済みSkillのファイルをローカル編集した状態で `context add` を実行。変更警告が表示されることを確認し、「いいえ」を選択。配布先が元の編集状態を維持していること、および `map.yaml` が変更されないことを検証。
  4. **ローカル編集（承認）**: 同様の状態で実行し、「はい」を選択。正常に新Skillで上書き再配布され、`map.yaml` が更新されることを検証。

### 単体テスト対象

- **`Planner`**: 衝突、ローカル編集、および欠落が検出されたときに、即時エラーではなく対応するフラグが `true` に設定された計画が正しく生成されること。
- **`AddOptions`**: 競合が検出された場合に `ConfirmOverwrite` が呼び出され、承認されたときのみ `Executor` が実行されること。
