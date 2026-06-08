# requirements.md Review

## Recognition

- Path pattern: `docs/prds/*/requirements.md`
- Template: `docs/ai-driven-development/templates/requirements.md`
- Default importance: `重`
- Status required: yes

## Inputs

- Required: template, Accepted PRD, Accepted backlog, Accepted engineering documents
- Conditional: related Accepted ADRs
- Code/config: inspect when existing interfaces or constraints must be verified

## Review Points

- ACがユーザー視点の完成条件に限定されているか
- Technical Requirementsが設計の制約・保証条件に限定されているか
- 実装方式や設計判断が混ざっていないか
- 共通要件とStory固有要件が区別され、必要なStory IDが付いているか
- NFR、データ、セキュリティ、アクセシビリティ、運用制約に漏れがないか
- Story ID、AC、TRの対応が追跡可能か
- Test Tasksへ渡す検証可能な振る舞いが明確か
- Open Questionsが下流設計へ暗黙に流れていないか

## Completion

- 各Storyの完成条件と技術制約を設計へ渡せる
- Blocking Open Questionがない
- Allowed transition: `Review -> Accepted`
