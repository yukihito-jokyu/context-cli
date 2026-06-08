# Backlog Update

## Responsibility

- PRD Scopeを成立させるStory、Priority、Dependencies、Supportsを保持する。
- Acceptance Criteriaや実装Taskを含めない。

## Sources

- `docs/ai-driven-development/templates/backlog.md`
- Accepted済みPRD

## Update Rules

- Storyの追加、分割、統合、取消では既存IDを再利用しない。
- 不要なStoryは削除せず`Cancelled`または`Superseded`として理由と置換先を残す。
- Storyの意味、優先度、依存関係を変更した場合は文書と該当Storyを`Draft`へ戻す。

## Verification

- PRD Scopeが全Storyで過不足なく成立する。
- 各Storyがユーザー価値単位で独立している。
- Requirements、Architecture Change、Tasksへの影響を列挙する。
