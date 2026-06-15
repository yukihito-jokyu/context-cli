# 01-plan-delete-logic

- **対象ストーリー**: ST-001、ST-003、ST-004、ST-005、ST-006

## 1. 処理フローチャート (Flowchart)

`spec.md` を参照。本タスクは `Planner.PlanDelete` のビジネスロジックと、そのユニットテストの実装を担当する。

## 2. シーケンス図 (Sequence Diagram)

`spec.md` を参照。

## 3. ファイル配置・責務定義

- `[NEW]` [delete_planner.go](file:///Users/yukihito/Documents/github_projects/context-cli/internal/distribution/delete_planner.go) : `Planner` 構造体に `PlanDelete` メソッドを実装する。
  - **シグネチャ**: `PlanDelete(snapshot MapSnapshot, workspaceRoot string, skillNames []string) (Plan, error)`
  - **仕様**:
    1. `snapshot.Workspaces` から `workspaceRoot` に対応する `oldRecord` が存在するか検証。存在しない場合は `ErrUnmanagedWorkspace` を返す。
    2. 引数 `skillNames` を重複排除（Deduplicate）する。
    3. `skillNames` 内のすべてのSkillが `oldRecord.Skills` に実在するか検証。1つでも存在しないSkill名があれば `ErrPrecondition`（または適切なエラー）を返す。
    4. `oldRecord.Skills` から、削除対象のSkillと、残す（維持する）Skillを分類する。
    5. 削除対象のSkillについて、`Inspect` を用いて配布先のファイル状態（`TargetPathStates`）を収集し、実在かつハッシュ不一致の場合や、欠落している場合は `DeleteOperation.IsLocalEdit = true` とする。
    6. 維持するSkillについて、残す配布先（Destination）を算出し、新しく構成される `WorkspaceRecord` に設定する。
    7. もし維持するSkillが0件になった場合、`WorkspaceRecord.Skills` を空にし、`WorkspaceRecord.Destinations` も空にする（`map.yaml` からの完全削除に繋がる）。
    8. 削除されるSkillにより、いずれのSkillも配布されなくなったDestinationは `WorkspaceRecord.Destinations` から除外する。
    9. `Plan` を構築して返す（`Creates` は空、`Deletes` に削除操作一覧を格納）。
- `[NEW]` [delete_planner_test.go](file:///Users/yukihito/Documents/github_projects/context-cli/internal/distribution/delete_planner_test.go) : `PlanDelete` の動作を検証する単体テスト。
  - テストケース:
    - 正常系: 複数Skillの中から一部のSkillを削除し、適切な `Deletes` が生成されること。維持されたSkillの宛先（Destinations）が正しく再構成されること。
    - 正常系: すべてのSkillを削除した場合に、`WorkspaceRecord` のSkill and宛先が空になり、全削除の計画が作られること。
    - 異常系: 未管理のWorkspaceに対して実行した場合に `ErrUnmanagedWorkspace` を返すこと。
    - 異常系: 管理外のSkill名を指定した場合にエラー（一切の計画作成を拒否）となること。
    - 異常系: 削除対象のSkillファイルにローカル編集がある場合、`IsLocalEdit` が `true` に設定されること。
- `[MODIFY]` [factory.go](file:///Users/yukihito/Documents/github_projects/context-cli/pkg/cmd/factory.go) : `DistributionPlanner` インターフェースに `PlanDelete` メソッドを追加する。

## 4. 実装チェックリスト

- [x] `DistributionPlanner` インターフェースの拡張
- [x] `PlanDelete` メソッドの骨子作成
- [x] `delete_planner_test.go` のテストケース作成（TDD）
- [x] `PlanDelete` のビジネスロジック詳細実装とエラーハンドリング
- [x] 単体テストがすべてパスすることを確認

## 5. テスト・検証計画

- **単体テスト対象**:
  - `go test ./internal/distribution -run TestPlanDelete`
