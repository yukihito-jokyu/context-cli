---
name: run-task-loop
description: 事前に設計承認されたタスクファイル群を読み込み、各タスクに対して「実装 (implement-task)」「コードレビュー」のサイクルをループ実行して開発を進行管理するフロー。
---

# タスク開発ループ実行 (Run Task Loop)

具体設計が完了し承認されたタスクファイル（例: `01-xxx.md`, `02-xxx.md`）について、各タスクを順に実装・完了させていく開発プロセスを一気通貫で管理・実行する。

## 入力

- 開発対象のタスクファイルの一覧または再開するタスクファイル（例: `docs/specs/spec-XXX-<slug>/tasks/01-init.md`）

## ループ

依存順に各タスクへ次を実行する。

1. 親エージェントが対象タスクのステータスを `仕掛中 (In Progress)` に更新する。
2. `task_implementer` に [implement-task](../implement-task/SKILL.md) を実行させる。
3. 品質ゲート通過後、親エージェントがステータスを `レビュー中 (Under Review)` に更新する。
4. `code_reviewer` に [review-implementation](../review-implementation/SKILL.md) を実行させる。
5. 指摘があれば実装担当が修正し、再度品質ゲートを実行してレビューする。
6. 変更概要、検証結果、レビュー対象ファイルへの絶対パスリンクを提示し、ユーザーの実装承認を得る。
7. 親エージェントがステータスを `完了 (Completed)` にして次のタスクへ進む。

状態遷移と承認の規則は [AGENTS.md](../../../AGENTS.md) に従う。レビューサブエージェントは変更しない。全タスク完了後に [update-architecture](../update-architecture/SKILL.md) を実行する。
