# ADR Review

## Recognition

- Path pattern: `docs/decisions/adr-*.md`
- Template: `docs/ai-driven-development/templates/adr.md`
- Default importance: `最重`
- Status required: yes

## Inputs

- Required: 対象の `docs/decisions/adr-*.md`
- Resolve automatically: 対応する `adr-candidates.md`、Candidate ID、判断を確定した上流成果物
- Conditional: template, existing ADRs, engineering documents, architecture documents
- Code/config: inspect when current feasibility or constraints are material

## Review Points

- ADR化する重要性と判断境界が明確か
- Contextが解決策を先取りせず判断理由を説明しているか
- 選択肢が公平に比較されているか
- Decisionと採用理由が明確か
- 不採用理由が将来の再検討に耐えるか
- Consequencesと受容するリスクが明示されているか
- Related Decisionsと関連成果物が追跡可能か
- 既存判断を変更する場合、新しいADRでSupersedesしているか
- DateがDraft生成日として固定されているか

## Completion

- 人間が採否と受容リスクを明示的に確定する
- Allowed transitions: `Review -> Accepted` or `Review -> Rejected`
- Accepted後の判断変更は既存ADRを改変しない
- SupersedesするADRをAcceptedにした場合だけ、旧ADRの`Superseded By`とStatus更新を提案する
