# ADR Candidates Review

## Recognition

- Target: `docs/prds/prd-XXX-*/adr-candidates.md` または `docs/engineering/adr-candidates.md`
- Template: `docs/ai-driven-development/templates/adr-candidates.md`
- Default importance: `最重`
- Status required: no independent status

## Inputs

- Required: 対象の `adr-candidates.md`
- Resolve automatically: Sourceに対応する上流成果物
- Conditional: existing ADRs, engineering documents, architecture documents
- Code/config: inspect when a candidate depends on current implementation facts

## Review Points

- 後からの変更コスト、横断影響、重大リスク、有力な代替案があるか
- ADR化せず設定、コード、Git履歴で十分ではないか
- 候補の境界が1つの判断として明確か
- 選択肢、評価軸、推奨案が公平か
- 採否を今決める必要があるか、Deferred可能か
- 関連PRD、Story、Requirement、Architecture Changeが特定されているか

## Completion

- 候補一覧をまとめて提示し、採否は一件ずつ人間が確定する
- 各候補を `Accepted / Rejected / Deferred` のいずれかにし、人間の確定理由を記録する
- 全候補の確認が完了した場合だけ成果物全体のAccepted化を提案する
- 限定レビューでは成果物全体のAccepted化を行わない
- 採用候補はADR作成スキルへ委譲する
- ADR作成スキルへはSourceとCandidate IDを渡す
