# ADR-004: Context Repositoryの同一性の判定方式

Status: Accepted
Status Reason: 最重レビューにおいて、上流要件（prd-001/requirements.md）および開発規則との完全な整合性が確認され、承認されたため。
Decision ID: ADR-004
Date: 2026-06-07。

## Context

Accepted済みRequirementsで固定した、Context Repositoryの同一性を判定する方式について、採用理由、代替案、Repository移動や別名パスに関する制約を実装開始前にADRへ正式記録する必要がある。

この判断は、`docs/prds/prd-001-safe-context-repository-setup/adr-candidates.md`の`ADC-002`を正式化する。

## Decision

シンボリックリンクを解決せず、字句的に正規化した絶対パスの文字列で同一性を判定する。判定後もRepositoryを再検証し、同一パス上の内容または安全性が変化した場合は設定を書き換えずエラー終了する。

## Options

### Option A: 字句的に正規化した絶対パス文字列による判定

Pros:

- macOSとLinuxで挙動が一貫し、説明が容易。
- 外部の複雑なAPIやOS固有の実装を必要としない。
- 利用者が設定ファイル（`config.yaml`）に保存されたパスを見て、直感的に診断しやすい。
- シンボリックリンクを排除する安全性要件と整合する。

Cons:

- 同一のRepositoryディレクトリであっても、シンボリックリンク経由の別名パスや、移動・置換された場合は別Repositoryとして判定される。

Rejection Reason:

- Not applicable

### Option B: シンボリックリンクを解決した実体パスによる判定

Pros:

- シンボリックリンク経由であっても、同じ実体ディレクトリであれば同一と判定できる。

Cons:

- シンボリックリンクを許可しないという安全要件と矛盾する、または同一性判定のロジックがシンボリックリンク検証と交絡する。
- OSやファイルシステムによる解決方法の違い（macOSの特定のボリュームマウントなど）に依存する可能性がある。

Rejection Reason:

- シンボリックリンクを排除する安全性要件と整合しにくく、同一性を判定する処理の独立性を維持するため。

### Option C: デバイス番号とinodeによる判定

Pros:

- パス文字列に依存せず、ファイルシステムレベルで確実に同一ディレクトリであることを特定できる。

Cons:

- 保存された識別子（デバイス番号等）は人間が読み取って診断することが極めて難しい。
- ファイルシステムの再フォーマットや、クローン後の再配置、ネットワークマウントの再割り当て等で情報が変わり、誤判定される可能性がある。

Rejection Reason:

- 設定内容を利用者が確認・診断しやすくするという運用の容易さを満たさず、ファイルシステム側の変更やマウント差異に対して脆弱であるため。

## Rationale

Accepted済みRequirementsのTR-007とTR-008に整合し、macOSとLinuxで説明可能かつ利用者が診断しやすい識別子を、外部ランタイムなしで実装できるため。Repositoryの移動またはシンボリックリンク経由の別名パスは別Repositoryとして扱われ、設定変更の確認が必要になる制約を受容する。

## Consequences

- ユーザー設定には字句的に正規化された絶対パスが保存される。
- Repositoryが別の場所に移動された場合は、同一のものとは判定されず、設定変更の確認フローに入る。

## Risks

- 同一内容の別パスへの切り替え時、および移動時に、設定の再実行時に確認ダイアログまたはエラー（変更拒否時）が発生する。

## Mitigations

- ユーザーに対し、正規化した絶対パスでの指定を促し、設定変更の確認フローでパスの差分を明示する。

## Related PRDs

- docs/prds/prd-001-safe-context-repository-setup/prd.md

## Related Stories

- prd-001-safe-context-repository-setup/ST-002
- prd-001-safe-context-repository-setup/ST-003
- prd-001-safe-context-repository-setup/ST-004

## Related Requirements

- prd-001-safe-context-repository-setup/AC-004
- prd-001-safe-context-repository-setup/AC-007
- prd-001-safe-context-repository-setup/AC-008
- prd-001-safe-context-repository-setup/AC-009
- prd-001-safe-context-repository-setup/AC-010
- prd-001-safe-context-repository-setup/AC-013
- prd-001-safe-context-repository-setup/AC-014
- prd-001-safe-context-repository-setup/AC-015
- prd-001-safe-context-repository-setup/TR-007
- prd-001-safe-context-repository-setup/TR-008
- prd-001-safe-context-repository-setup/TR-009
- prd-001-safe-context-repository-setup/TR-010
- prd-001-safe-context-repository-setup/TR-011
- prd-001-safe-context-repository-setup/TR-012

## Related Architecture Changes

- None

## Supersedes

- None

## Superseded By

- None

## Assumptions

- 初期版は個人開発者による単一ユーザー、単一マシンのローカル実行を対象とし、ネットワークアクセスを行わない。
- 同一性判定はローカルパス文字列の比較のみで機能する。

## Open Questions

- Not applicable
