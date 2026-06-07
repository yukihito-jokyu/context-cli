# architecture-change.md Review

## Recognition

- Path pattern: `docs/prds/*/architecture-change.md`
- Template: `docs/ai-driven-development/templates/architecture-change.md`
- Default importance: `重`
- Status required: yes

## Inputs

<!-- 固定の定義または対応関係を1行で維持するため。 -->
<!-- textlint-disable preset-ja-technical-writing/max-comma -->

- Required: template, Accepted PRD, backlog, requirements, engineering documents

<!-- textlint-enable preset-ja-technical-writing/max-comma -->

- Conditional: Accepted ADRs and current `docs/architecture/*.md`
  <!-- 固定の定義または対応関係を1行で維持するため。 -->

- Code/config before implementation: inspect current structure and feasibility; lack of implementation is not a defect

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->
<!-- 固定の定義または対応関係を1行で維持するため。 -->

- Code/config after implementation: inspect the implementation diff, tests, tasks.md, and actual resulting behavior

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->

## Review Points

- Requirementsを満たす変更案か
- 現在のarchitectureが存在する場合、それとの差分が明確か
- 初回でarchitectureがない場合、RequirementsとADRから初期設計が明確か
- 設計判断とADR候補が区別されているか
- 変更予定・調査対象と確定事項が区別されているか
- Open QuestionsがBlockingとDeferredに分類され、Deferredに判断条件と判断する工程があるか
- 影響範囲、依存関係、互換性、移行、廃止、切り戻しが明確か
- データ、API、セキュリティ、プライバシー、運用への影響が評価されているか
- 実装後に `docs/architecture/`へ反映する対象が明確か
- 実装後レビューでは、変更案ではなく実装結果と一致し、未実装の予定が確定事項として残っていないか
- 実装後レビューでは、設計判断の変更が文書更新だけで正当化されず、必要なRequirementsまたはADRの再確認を経ているか

## Completion

- 実装前: tasks.mdを作成できる設計情報が揃い、必要なADRがAcceptedで、Blocking Open Questionがない
- 実装後: コード、テスト、tasks.md、関連ADRと一致し、docs/architecture更新へ引き渡せる
- Allowed transition before implementation: `Review -> Accepted`
- Allowed transition after implementation: `Review -> Implemented`
