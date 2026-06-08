# ADR Update

## Responsibility

- 1つの重要な技術判断についてContext、Decision、Options、理由、結果、リスクを保持する。

## Sources

- `docs/ai-driven-development/templates/adr.md`
- 対応するAccepted済みADR候補
- 関連Requirements、Engineering Foundation、既存ADR

## Update Rules

- Accepted済みADRの判断を上書きしない。判断変更は新しいADRでSupersedeする。
- 誤字、リンク、事実同期だけは既存Statusを維持できる。
- 判断内容を変更するIssueでは新ADR作成を提案し、既存ADRは必要な参照更新だけに限定する。

## Verification

- Decisionが一意で実装可能である。
- 採用・却下した選択肢と理由、受容リスクが明確である。
- Supersedes、Superseded By、関連成果物が整合する。
