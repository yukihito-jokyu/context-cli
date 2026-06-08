# product.md Review

## Recognition

- Path: `docs/product.md`
- Template: `docs/ai-driven-development/templates/product.md`
- Default importance: `重`
- Status required: yes

## Inputs

- Required: template and current product document
- Conditional: Accepted PRDs and Accepted ADRs affected by the change
- Code/config: inspect only when a stated current-product fact requires verification

## Review Points

- 現在のプロダクト意図を表す正本として明確か
- Product Vision、Target Users、Core User Problems、Value Propositionが具体的か
- Product GoalsとCurrent FocusがPRD候補の判断に使えるか
- Product PrinciplesとConstraintsが下流判断を十分に制約するか
- Scope外の実装詳細や個別PRDの設計を含んでいないか
- Open Questions、Assumptions、Out of Scopeが曖昧でないか
- 変更が既存PRD、Engineering Foundationへ与える影響が特定されているか

## Completion

- 中核項目にBlocking Open Questionがない
- 下流成果物との矛盾がない、または修正対象が確定している
- Allowed transition: `Review -> Accepted`
