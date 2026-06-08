# ADR Candidates Update

## Responsibility

- 重要な技術判断候補について境界、選択肢、評価軸、推奨案、ADR化要否を保持する。

## Sources

- `docs/ai-driven-development/templates/adr-candidates.md`
- Sourceに対応するRequirementsまたはEngineering Foundation
- 関連ADR

## Update Rules

- 1候補を1つの判断境界に限定する。
- Issueで判断が確定した候補は`Accepted`または`Rejected`、条件待ちは`Deferred`にする。
- `Human Decision Reason`へ人間が確定した理由と受容リスクを記録する。
- 採用候補のADRが未作成なら`Resulting ADR`は`Not applicable`のままとし、`create-adr`へ委譲する。
- 全候補が確定した場合だけ文書全体の`Accepted`を候補にする。

## Verification

- 選択肢が公平で、有力案を省略していない。
- Related PRDs、Stories、Requirements、ADRsを追跡できる。
- 判断内容が未確定の候補を`Accepted`にしていない。
