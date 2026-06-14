# 管理対象のプロジェクトを切り替える

- **ステータス**: 完了 (Completed)
- **対象ストーリー**: ST-004, ST-006

## 1. 処理フローチャート (Flowchart)

```mermaid
flowchart TD
    A["context add [project-name] を開始"] --> B["map.yamlから管理情報を取得"]
    B --> C{"既存の管理情報が存在するか"}

    C -- "いいえ" --> D["初期設定なしでプロジェクト選択へ"]
    C -- "はい" --> E["前回の管理プロジェクトを特定"]

    E --> F{"今回指定または選択したプロジェクトと一致するか"}

    F -- "はい" --> G["前回選択されたSkill（プロジェクト固有・共通）を復元して初期値にする"]
    F -- "いいえ" --> H["初期選択（デフォルト値）をクリアし、すべて未選択とする"]

    G & H --> I["対話プロンプトで新プロジェクトのSkillを選択"]

    I --> J["配布計画（Planner.Plan）を作成"]
    J --> K{"プロジェクトが変更されたか"}

    K -- "はい" --> L["旧プロジェクト由来の全SkillをDeletes（削除計画）に追加\n新Skillは同名であっても新規/置換（Creates）として扱う"]
    K -- "いいえ" --> M["通常の差分更新計画を作成"]

    L & M --> N["計画の検証・排他ロックの取得"]
    N --> O["Executorにより旧Skillを退避・削除し、新Skillを配置"]
    O --> P["map.yamlを新プロジェクト情報で上書き保存して完了"]
```

## 2. シーケンス図 (Sequence Diagram)

```mermaid
sequenceDiagram
    actor User as 利用者
    participant Add as AddOptions
    participant Prompt as pkg/cmd.Prompt
    participant Catalog as skillcatalog.Catalog
    participant Planner as distribution.Planner
    participant Executor as distribution.Executor
    participant Map as distributionmap.Store
    participant FS as distribution.FileSystem

    User->>Add: context add
    Add->>Map: map.yamlから管理情報を取得
    Map-->>Add: MapSnapshot (既存レコード)

    Add->>Catalog: プロジェクト一覧を取得
    Catalog-->>Add: [ProjectA, ProjectB]

    Add->>Prompt: SelectProject(candidates, default: ProjectA)
    Prompt-->>User: プロジェクト選択（初期選択: ProjectA）
    User-->>Prompt: ProjectB を選択確定
    Prompt-->>Add: ProjectB

    Note over Add: プロジェクトがAからBに変更されたため、<br/>Skillのデフォルト初期値をクリアする

    Add->>Catalog: ProjectB のSkill一覧を取得
    Catalog-->>Add: [SkillB1, SkillB2]

    Add->>Prompt: SelectSkills(ProjectB, candidates, default: 空)
    Prompt-->>User: Skill選択（初期選択なし）
    User-->>Prompt: SkillB1 を選択確定
    Prompt-->>Add: SkillB1

    Add->>Prompt: ConfirmCommonSkills(default: false)
    Prompt-->>User: 共通追加確認
    User-->>Prompt: いいえ
    Prompt-->>Add: false

    Add->>Prompt: SelectDestinations(default: Codex, Claude)
    Prompt-->>User: 配布先選択（初期選択は前回値維持）
    User-->>Prompt: 決定
    Prompt-->>Add: [Codex, Claude]

    Add->>Planner: Plan(snapshot, Selection)
    Note over Planner: ProjectA != ProjectB のため、<br/>ProjectA由来のSkillをDeletesに追加、<br/>ProjectBのSkillB1をCreatesに追加
    Planner-->>Add: Plan (Creates: SkillB1, Deletes: SkillA1, SkillA2)

    Add->>Executor: Execute(Plan)
    Executor->>Map: Begin(Revision)
    Map-->>Executor: Transaction

    Executor->>FS: ProjectAの旧Skillを退避
    Executor->>FS: ProjectBのSkillB1を配置

    Executor->>Map: Commit(WorkspaceRecord for ProjectB)
    Map-->>Executor: Commit成功

    Executor->>FS: 退避した旧Skillを完全削除
    Executor->>Map: Close() (ロック解放)
    Executor-->>Add: 成功
    Add-->>User: 完了表示
```

## 3. ファイル配置・責務定義

本タスクのビジネスロジックはすでに `internal/distribution/planner.go` および `pkg/cmd/add.go` に実装されている。したがって、本タスクでの主な変更は検証用テストコードの追加である。

### テストコード

- **[MODIFY] [planner_test.go](file:///Users/yukihito/Documents/github_projects/context-cli/internal/distribution/planner_test.go)**
  - `TestPlannerHandlesProjectSwitch` を追加。プロジェクト変更時に、旧プロジェクトのSkillがすべて `Deletes` に分類され、新しいプロジェクトのSkillが `Creates` に入ることを検証する。

- **[MODIFY] [add_test.go](file:///Users/yukihito/Documents/github_projects/context-cli/pkg/cmd/add_test.go)**
  - `TestAddOptionsRunClearsDefaultSkillsOnProjectSwitch` を追加。前回のプロジェクトと異なるプロジェクトが対話UIで選択された際に、Skill選択プロンプトに渡される初期選択値（`defaultNames`）が空であることをモックプロンプト経由で検証する。

- **[MODIFY] [add_test.go](file:///Users/yukihito/Documents/github_projects/context-cli/test/e2e/add_test.go)**
  - `TestAddSwitchProject` E2Eテストを新規追加。
  - プロジェクトAのSkillを配布した状態から、再度 `context add` を実行してプロジェクトBに切り替えた際、配布先の旧Skillがすべて消去され新Skillのみが配置されること、および `map.yaml` が更新されることを実端末対話プロセスを起動して検証する。

## 4. 実装チェックリスト

- [x] プロジェクト切り替え時のビジネスロジックの実装状況（`planner.go`, `add.go`）のコード確認
- [x] `Planner` 単体テストへのプロジェクト切り替え検証ケースの追加とパス
- [x] CLI単体テスト (`add_test.go`) への初期選択クリア検証ケースの追加とパス
- [x] E2Eテストへのプロジェクト切り替え検証ケースの追加とパス
- [x] 品質ゲートの実行（`golangci-lint run`, `go test ./...`）

## 5. テスト・検証計画

### E2E/結合テスト方法

- `go test -v ./test/e2e -run TestAddSwitchProject` を実行し、以下のシナリオを検証する：
  1. プロジェクトA의 Skill（例: `project-skill`）と共通Skill（例: `common-skill`）をCodex/Claudeへ配布。
  2. 再度 `context add` を実行し、対話プロンプトで別プロジェクトB（例: `project-b`）を選択。
  3. Skill選択プロンプトで初期チェックが外れていることを確認し、プロジェクトBのSkillを選択して決定。
  4. 完了後、配置先でプロジェクトAのSkillおよび前回の共通Skillが削除され、プロジェクトBのSkillのみが配置されていること、`map.yaml` の記録がプロジェクトBへ更新されていることを検証。

### 単体テスト対象

- **`Planner`**: プロジェクト切り替え時に、前回のSkillレコードが同一名の別Skillであっても `Deletes` と `Creates` の両方に正しく算出されること。
- **`AddOptions`**: プロジェクトが変更された場合に、`selectAllSkills` がプロンプト呼び出し時に渡すデフォルトSkill名のスライスを空にすること。
