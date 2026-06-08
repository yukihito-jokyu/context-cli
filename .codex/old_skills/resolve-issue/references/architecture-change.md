# Architecture Change Update

## Responsibility

- PRDによる実装前のシステム変更案と、API、データ、境界、運用、移行への影響を保持する。

## Sources

- `docs/ai-driven-development/templates/architecture-change.md`
- Accepted済みPRD、Backlog、Requirements、ADR
- Engineering Foundationと現在Architecture

## Update Rules

- Issueの結論を要件ではなく設計へ具体化する。
- 既存Architectureとの差分、互換性、移行、失敗時動作、テスト境界を明記する。
- 設計の意味を変更した場合は`Draft`へ戻し、Tasksと実装への影響を記録する。

## Verification

- 全AC、TR、NFRへの設計上の対応を追跡できる。
- ADRと矛盾せず、未確定の重要判断を暗黙に確定していない。
- Tasks作成に必要な境界と変更対象が揃っている。
