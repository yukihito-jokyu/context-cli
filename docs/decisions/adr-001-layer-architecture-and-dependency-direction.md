# ADR-001: レイヤー構造と依存方向

Status: Accepted
Status Reason: 最重レビューで判断境界、選択肢、却下理由、受容リスク、ADR候補との追跡性を確認し、採用を確定した。
Decision ID: ADR-001
Date: 2026-06-07。

## Context

`context` は、CLI、ユースケース、ドメイン規則、ファイルシステムやYAMLなどの外部技術を扱う。実装開始前に責務配置と依存方向を固定し、domainをCLI、対話UI、永続化、OS固有APIから分離した状態を一貫して維持する必要がある。

この判断は、`docs/engineering/adr-candidates.md`の`ADC-001`を正式化する。

## Decision

`cli → application → domain` を主要な依存方向とする4境界を採用する。infrastructureはapplicationが定義するI/Oポートとdomainの型にのみ依存し、具象ユースケースやcliには依存しない。`cmd/context` は依存の組み立てのため、cli、application、infrastructureを参照できる。

## Options

### Option A: 4境界と依存逆転

Pros:

- domainをCLI、対話UI、永続化、OS固有APIから独立させられる。
- 依存方向と責務境界を明示し、循環依存を防止できる。
- domainとapplicationを外部I/Oなしでテストできる。
- 外部技術の変更に伴う修正範囲を境界内へ限定できる。

Cons:

- applicationの利用規則に応じたI/Oポートの定義が必要になる。
- 小規模な初期実装でも4つの責務境界を維持する必要がある。

Rejection Reason:

- Not applicable

### Option B: domain中心の直接依存

Pros:

- 各外側レイヤーからdomainを直接利用できる。
- applicationを経由しない単純な処理では依存関係を短くできる。

Cons:

- applicationが所有すべきユースケース調整とI/Oポートの境界が不明確になる。
- 外側レイヤー間の責務分担とテスト境界を一貫して説明しにくい。

Rejection Reason:

- applicationの責務とI/Oポートの所有者を明確にできず、システム全体で依存方向とテスト境界を一貫させにくいため。

### Option C: 明示的な依存境界を設けない

Pros:

- 初期のファイル配置と実装を最小限にできる。
- 各パッケージから必要な実装を直接参照できる。

Cons:

- domainへ外部技術の依存が入り込むことを防止できない。
- 循環依存や責務の混在が発生しやすい。
- 機能追加や外部技術の変更時に修正範囲が広がる。

Rejection Reason:

- domainの独立性、依存方向、責務境界、テスト境界をシステム全体で維持できないため。

## Rationale

domainを外部技術から分離し、依存方向、責務境界、テスト境界をシステム全体で一貫させる必要がある。採用案は、infrastructureをapplicationのI/Oポートへ依存させることで依存逆転を適用しつつ、初期版に不要な下位階層を先行作成しない構成を取れる。

## Consequences

- `cmd/context`、`internal/cli`、`internal/application`、`internal/domain`、`internal/infrastructure`を基本境界とする。
- 機能ごとの具体的なパッケージ構成は、関連PRDの`architecture-change.md`で定義する。
- 外部機能のインターフェースは、その利用規則を所有するapplicationまたはdomainへ配置する。
- domainとapplicationのテストは外部I/Oへ依存させない。

## Risks

- 単純な処理にも不要なインターフェースやパッケージを追加し、初期版の複雑度を高める可能性がある。
- 責務の所有者を誤ると、infrastructureから具象ユースケースへの逆向き依存が生じる可能性がある。

## Mitigations

- 空の階層や将来予測による抽象化を作らず、ビルドまたはテスト可能な最小単位だけを追加する。
- インターフェースは利用側で定義し、infrastructureからapplicationの具象ユースケースおよびcliへの依存を禁止する。
- 機能追加時に依存方向と循環依存をレビューする。

## Related PRDs

- Not applicable

## Related Stories

- Not applicable

## Related Requirements

- Not applicable

## Related Architecture Changes

- Not applicable

## Supersedes

- None

## Superseded By

- None

## Assumptions

- 初期版の機能数とチーム規模では、単一Goモジュールで十分である。
- 初回bootstrapでは、空ディレクトリではなくビルドまたはテスト可能な最小単位だけを作成する。

## Open Questions

- Not applicable
