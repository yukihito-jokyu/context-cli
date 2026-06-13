---
name: run-task-loop
description: plan-tasks で作成されたタスクファイル群を読み込み、各タスクに対して「個別設計 (design-task)」「設計レビュー」「実装」「コードレビュー」のサイクルをループ実行して開発を進行管理するフロー。
---

# タスク開発ループ実行 (Run Task Loop)

タスク分割によって作成されたタスクファイル（例: `01-xxx.md`, `02-xxx.md`）について、各タスクを順に完了させていく開発プロセスを一気通貫で管理・実行する。

## 入力

- 開発対象のタスクファイルの一覧または再開するタスクファイル（例: `docs/specs/spec-XXX-<slug>/tasks/01-init.md`）

## ループ

依存順に各タスクへ次を実行する。

1. 親エージェントが [design-task](../design-task/SKILL.md) を実行する。
2. `task_design_reviewer` に [review-task-design](../review-task-design/SKILL.md) を実行させる。
3. 指摘があれば親エージェントがユーザーと修正内容を合意し、再レビューする。
4. ユーザーの設計承認後、親エージェントがステータスを `仕掛中 (In Progress)` にする。
5. `task_implementer` に [implement-task](../implement-task/SKILL.md) を実行させる。
6. 品質ゲート通過後、親エージェントがステータスを `レビュー中 (Under Review)` にする。
7. `code_reviewer` に [review-implementation](../review-implementation/SKILL.md) を実行させる。
8. 指摘があれば実装担当が修正し、再度品質ゲートを実行してレビューする。
9. 変更概要、検証結果、レビュー対象ファイルへの絶対パスリンクを提示し、ユーザーの実装承認を得る。
10. 親エージェントがステータスを `完了 (Completed)` にして次のタスクへ進む。

状態遷移と承認の規則は [AGENTS.md](../../../AGENTS.md) に従う。レビューサブエージェントは変更しない。全タスク完了後に [update-architecture](../update-architecture/SKILL.md) を実行する。
