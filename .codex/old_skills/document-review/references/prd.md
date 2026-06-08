# prd.md Review

## Recognition

- Path pattern: `docs/prds/*/prd.md`
- Template: `docs/ai-driven-development/templates/prd.md`
- Default importance: `重`
- Status required: yes

## Inputs

- Required: template, Accepted product, Accepted engineering documents
- Conditional: related Accepted ADRs
- Code/config: inspect only for necessary existing-product facts

## Review Points

- 1つのユーザー課題に絞られているか
- Problem、Target User、Goalが具体的か
- 原則として解決策や設計判断が入り込んでいないか
- Scopeが対象領域を定め、Out of Scopeとの境界が明確か
- Expected OutcomeとSuccess Metricsが観測可能か
- Success Metricsの評価方法と観測時期が明確か
- ConstraintsとAssumptionsが上位文脈に整合しているか
- PRD候補や別課題が混在していないか

## Completion

- Scopeを下流でStoryへ分解できる
- Blocking Open Questionがない
- Allowed transition: `Review -> Accepted`
