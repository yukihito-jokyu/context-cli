# Requirements Update

## Responsibility

- StoryごとのAcceptance Criteria、Technical Requirements、共通NFRを保持する。
- 技術選定や具体的な実装方式を本文で確定しない。

## Sources

- `docs/ai-driven-development/templates/requirements.md`
- Accepted済みPRDとBacklog
- Accepted済みADRとEngineering Foundation

## Update Rules

- ACはユーザーから観測できる独立した判定条件にする。
- TRはStory固有の保証、NFRは複数Storyに共通する制約にする。
- AC、TR、NFRの既存IDを削除、再利用、振り直ししない。置換時は旧IDへ状態と置換先を残す。
- 意味を変更した場合は`Draft`へ戻し、再レビュー理由と下流影響を記録する。
- 解決策の選択が必要ならADR候補として分離する。

## Verification

- PRDと全Storyを追跡できる。
- 正常系、主要な失敗、境界条件、非機能カテゴリを確認する。
- Architecture Change、ADR候補、Tasks、テストへの影響を列挙する。
