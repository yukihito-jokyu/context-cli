# Tasks Update

## Responsibility

- Accepted済み成果物を実装可能なTask、Test Task、順序、変更予定ファイル、品質ゲートへ分解する。

## Sources

- `docs/ai-driven-development/templates/tasks.md`
- Accepted済みPRD、Backlog、Requirements、Architecture Change、ADR

## Update Rules

- Issue解決による設計・要件変更に影響されるTaskだけを追加、置換、再計画する。
- 完了済みTaskを削除または未完了へ戻さず、置換理由と後継Taskを記録する。
- 実装内容が未確定ならTaskへ推測を書かず、上流成果物の確定を優先する。
- 計画の意味を変更した場合は`Draft`へ戻す。

## Verification

- 各Taskが要件と設計へ追跡でき、独立して検証可能である。
- Test Task、変更予定ファイル、依存順序、品質ゲートが揃っている。
