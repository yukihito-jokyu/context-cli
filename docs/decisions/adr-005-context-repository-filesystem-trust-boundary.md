# ADR-005: Context Repositoryのファイルシステム信頼境界

Status: Accepted
Status Reason: 最重レビューにおいて、上流要件（prd-001/requirements.md）および開発規則との完全な整合性が確認され、承認されたため。
Decision ID: ADR-005
Date: 2026-06-07。

## Context

Context Repositoryを唯一の正本として安全に使用するため、どの権限状態とパス構成を信頼可能として受け入れるかを実装前に固定する必要がある。

この判断は、`docs/prds/prd-001-safe-context-repository-setup/adr-candidates.md`の`ADC-003`を正式化する。

## Decision

現在の利用者が読み取りとディレクトリ検索を行え、グループと他ユーザーに書き込みが許可されず、パス構成要素と必須対象にシンボリックリンクがないRepositoryだけを受け入れる。検証失敗時は設定を書き込まず、既存設定を維持する。

## Options

### Option A: 所有者限定とグループ・他者書き込み拒否（厳格案）

Pros:

- 完全に現在の利用者が所有する領域に限定されるため、他者による置換や改ざんのリスクが最小化される。

Cons:

- 共有開発環境や、特定の親ディレクトリ権限を持つ正当なローカルRepositoryであっても過度に拒否される可能性があり、実装が複雑になる。

Rejection Reason:

- 正当なローカルRepositoryを過度に拒否する可能性があり、初期版の実装複雑度を抑えるため。所有者と親ディレクトリによる置換可能性は残存リスクとして受容する。

### Option B: 利用者の読み取りと検索、グループ・他者書き込み拒否、シンボリックリンク排除（推奨案）

Pros:

- 初期版の個人利用、単一ユーザー、ローカル実行という前提に適合する。
- グループや他ユーザーによる意図しない書き込みを拒否し、シンボリックリンク経由の参照先変更リスクを排除する。
- macOSとLinuxで一貫して検証可能。

Cons:

- 所有者自身や親ディレクトリの所有関係による置換可能性は完全に排除できない（残存リスクとして受容）。

Rejection Reason:

- Not applicable

### Option C: グループ共有書き込みの許可、他者書き込み・リンク拒否

Pros:

- 同一グループ内の他のユーザーと共同でリポジトリを編集するユースケースに対応できる。

Cons:

- 他のグループメンバーによって不正な内容が書き込まれるリスクを受容することになり、信頼境界の安全性が低下する。

Rejection Reason:

- 初期版は個人開発者による単一ユーザー実行を前提としており、グループ書き込みを許可することは安全性とのトレードオフにおいてリスクが高いため。

### Option D: 読み取り可能性と必須構造のみ検証、書き込み権限・リンク受容

Pros:

- 最も制約が緩く、どのような権限状態やシンボリックリンク構成のディレクトリであっても設定できる。

Cons:

- シンボリックリンクの張り替えによる意図しない配布元変更や、他ユーザーによる書き込みによって、安全性が著しく低下する。

Rejection Reason:

- 配布内容の正本としての信頼境界を保証できず、不正な内容の配布やデータ整合性の喪失につながる重大なリスクがあるため。

## Rationale

初期版の個人利用、単一ユーザー、ローカル実行という前提に適合し、Context Repositoryへの意図しない書き込みとシンボリックリンクによる参照先変更を拒否できるため。所有者と親ディレクトリを検証する厳格案は、正当なローカルRepositoryを拒否する可能性と実装複雑度が増すため初期版では採用しない。所有者と親ディレクトリによる置換可能性は、下流の設計および実装レビューで継続確認する残存リスクとして受容する。

## Consequences

- 設定時および再実行時に、リポジトリパスおよび必須対象がシンボリックリンクではないこと、グループ・他者の書き込み権限がないことが厳格に検証される。
- グループや他人に書き込み権限があるディレクトリ、またはシンボリックリンクを含むリポジトリは設定できない。

## Risks

- 所有者と親ディレクトリによる置換可能性というリスクが残存する。

## Mitigations

- 下流の設計および実装レビューにおいて、この残存リスクに対する評価を継続する。

## Related PRDs

- docs/prds/prd-001-safe-context-repository-setup/prd.md

## Related Stories

- prd-001-safe-context-repository-setup/ST-001
- prd-001-safe-context-repository-setup/ST-002
- prd-001-safe-context-repository-setup/ST-003
- prd-001-safe-context-repository-setup/ST-004

## Related Requirements

- prd-001-safe-context-repository-setup/AC-010
- prd-001-safe-context-repository-setup/AC-012
- prd-001-safe-context-repository-setup/AC-013
- prd-001-safe-context-repository-setup/AC-014
- prd-001-safe-context-repository-setup/AC-015
- prd-001-safe-context-repository-setup/TR-001
- prd-001-safe-context-repository-setup/TR-004
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
- 利用者はContext Repositoryを所有し、安全な権限で管理している。

## Open Questions

- Not applicable
