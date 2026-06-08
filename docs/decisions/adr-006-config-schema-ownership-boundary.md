# ADR-006: config.yamlのスキーマ解釈の所有境界

Status: Accepted
Status Reason: 人間が、Accepted済みADC-003の推奨案を正式な技術判断として承認した。
Decision ID: ADR-006
Date: 2026-06-07。

## Context

`config.yaml`の未知フィールド拒否、スキーマバージョン検証、値検証をドメイン契約として一箇所で強制しつつ、その他のYAML処理とファイルI/Oをinfrastructureへ隔離する所有境界を実装前に確定する必要がある。

この判断は、`docs/engineering/adr-candidates.md`の`ADC-003`を正式化する。

## Decision

`config.yaml`の厳格デコードと値検証だけをdomainへ配置する。domainはファイルを直接読み書きせず、受け取ったバイト列をドメイン設定へ変換する。YAMLエンコード、ファイルI/O、`map.yaml`を含むその他のYAML処理はinfrastructureへ配置する。

## Options

### Option A: config.yamlのデコードと検証をdomainへ配置

Pros:

- スキーマバージョン、未知フィールド、値の妥当性を1つのドメイン契約として強制できる。
- ファイルI/O、XDGパス、ロック、原子的置換をdomainから分離できる。
- 初期版に追加のcodecパッケージや変換処理を必要としない。

Cons:

- domainが`config.yaml`の厳格デコードに使用するYAMLライブラリへ依存する限定例外が生じる。

Rejection Reason:

- Not applicable

### Option B: すべてのYAMLデコードをinfrastructureへ配置

Pros:

- domainをYAMLライブラリから完全に独立させられる。

Cons:

- `config.yaml`のスキーマ解釈と値検証が複数境界へ分かれる。
- デコード済みの値がdomainへ渡る前に契約違反を見落とす可能性がある。

Rejection Reason:

- `config.yaml`のスキーマ契約をdomainで一貫して強制できず、所有境界が分散するため。

### Option C: 専用codecパッケージを追加

Pros:

- YAML固有処理をdomainとinfrastructureの両方から分離できる。

Cons:

- 初期版に新しい責務境界と変換が増える。
- `config.yaml`の契約所有者が不明確になりやすい。

Rejection Reason:

- 初期版には不要なパッケージと変換処理を増やし、ドメイン契約の所有者を不明確にするため。

## Rationale

`config.yaml`のスキーマバージョン、未知フィールド、値の妥当性を一貫して強制しながら、ファイルI/Oとその他のYAML処理をdomainから分離できる。例外を`config.yaml`の厳格デコードだけに限定することで、`map.yaml`や将来追加されるYAML処理への拡大を防ぐ。

## Consequences

- `internal/domain`は`config.yaml`のデコードに必要なYAMLライブラリへ依存できる。
- domainはファイルパスを解決せず、読み書き、ロック、原子的置換もしない。
- `map.yaml`を含むその他のYAMLデコードと、すべてのYAMLエンコードはinfrastructureへ配置する。

## Risks

- 限定例外が将来のYAML処理へ拡大し、domainへ外部技術依存が増える可能性がある。

## Mitigations

- 例外対象を`config.yaml`の厳格デコードと値検証だけに限定し、その他のYAML処理をinfrastructureへ配置する規則をレビューで確認する。

## Related PRDs

- docs/prds/prd-001-safe-context-repository-setup/prd.md

## Related Stories

- prd-001-safe-context-repository-setup/ST-001
- prd-001-safe-context-repository-setup/ST-003
- prd-001-safe-context-repository-setup/ST-005

## Related Requirements

- prd-001-safe-context-repository-setup/AC-016
- prd-001-safe-context-repository-setup/AC-018
- prd-001-safe-context-repository-setup/TR-013
- prd-001-safe-context-repository-setup/TR-014
- prd-001-safe-context-repository-setup/TR-015

## Related Architecture Changes

- docs/prds/prd-001-safe-context-repository-setup/architecture-change.md

## Supersedes

- None

## Superseded By

- None

## Assumptions

- 初期版は単一ユーザー、単一マシンのローカル実行を対象とする。
- `config.yaml`以外のYAMLスキーマ解釈はinfrastructureが所有する。

## Open Questions

- Not applicable
