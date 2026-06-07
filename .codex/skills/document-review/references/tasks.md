# tasks.md Review

## Recognition

- Path pattern: `docs/prds/*/tasks.md`
- Template: `docs/ai-driven-development/templates/tasks.md`
- Default importance: `軽`
- Status required: yes

## Inputs

<!-- 固定の定義または対応関係を1行で維持するため。 -->
<!-- textlint-disable preset-ja-technical-writing/max-comma -->

- Required: template, Accepted PRD, backlog, requirements, architecture-change, engineering documents

<!-- textlint-enable preset-ja-technical-writing/max-comma -->

- Conditional: Accepted ADRs and current architecture
- Code/config: verify planned files, dependencies, commands, test locations as needed

## Review Points

- Task粒度が実装可能な大きさか
- 全Story、AC、TR、NFRにTask、Test Task、Quality Gate、Release Checkのいずれかとの紐づきがあるか
- 共通基盤Taskが関連する全Story IDと必要理由を持つか
- 実装順序と依存関係が妥当か
- Test TasksがTask内の検証可能な振る舞い単位か
- ユーザー価値に近いテストを優先しているか
- Documentation Updates、Files to Change、Definition of Doneに漏れがないか
- Files to Changeが確定リストではなく変更予定・調査対象として適切か
- 不明な変更箇所が単なるTBDではなく調査作業として明示されているか
- 適用フローとリスク判定の根拠が妥当か
- 並行作業の競合、品質ゲート、release-check、リリース後検証の記録先があるか
  <!-- 固定の定義または対応関係を1行で維持するため。 -->

- Release CheckにScope、Checked At、標準チェック項目ごとの判定と根拠、Overall Result、Remaining Risks、Follow-up、Human Decisionがあるか

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->

- 各TaskのCode ReviewとPRD全体のCode Review Summaryに、対象差分、指摘、修正、再検証、残存リスク、人間判断の記録先があるか
  <!-- 固定の定義または対応関係を1行で維持するため。 -->

- Architecture UpdateにStatus、Scope、Evidence、Files Updated、Inconsistencies、Unreflected Items、Human Decisionがあるか

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->

- Open QuestionsがBlockingとDeferredに分類され、Deferredに判断条件、判断工程、影響Taskがあるか
- 既存Task IDとTest Task IDが維持され、CancelledまたはSupersededの理由と置換先が記録されているか
  <!-- 固定の定義または対応関係を1行で維持するため。 -->

- Task Statusが `Pending / In Progress / Blocked / Completion Candidate / Completed / Cancelled / Superseded` のいずれかで、Status Reasonと整合しているか

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->

- `Completion Candidate` が依存Taskの完了条件として扱われていないか
- 実装後の再レビューでは、Completedが人間の確定後にだけ設定され、Blockedに原因と解除条件があるか
- 限定コードレビューだけでPRD全体のCode Review SummaryがCompletedになっていないか
- Architecture UpdateがCompletedの場合、architecture-changeがImplementedで、更新対象をdocument-reviewへ引き渡せるか

## Completion

- 実装AIがTask順に作業を開始できる
- 標準フローではテスト計画レビューが完了している
- Blocking Open Questionがない
- Allowed transition: `Review -> Accepted`
