# backlog.md Review

## Recognition

- Path pattern: `docs/prds/*/backlog.md`
- Template: `docs/ai-driven-development/templates/backlog.md`
- Default importance: `軽`
- Status required: yes

## Inputs

- Required: template and Accepted PRD
- Conditional: Accepted requirements when re-reviewing an established backlog
- Code/config: not normally required

## Review Points

- StoryがPRD Scopeから逸脱していないか
- Storyがユーザー価値に紐づいているか
- 内部改善がTechnical Storyとして明示されているか
- EpicがStoryの多い場合のgroupingに限定されているか
- Story IDがPRD内で一意な連番か
- 優先度と依存関係が自然か
- Storyが実装Taskや技術設計へ細分化されすぎていないか

## Completion

- 各Storyをrequirements.mdへ一意に引き継げる
- Allowed transition: `Review -> Accepted`
