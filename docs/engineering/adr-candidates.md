# ADR Candidates

Status: Accepted
Status Reason: 人間がADC-003の推奨案と、既存のレイヤー判断に対する限定例外を承認した。
Source: engineering-foundation。

## Candidates

### ADC-001: レイヤー構造と依存方向

<!-- 判断時期の固定値を1行で定義するため。 -->

Status: Accepted
Decision Needed: システム全体の責務配置と依存方向を実装開始前に固定し、外部技術からdomainを分離した状態を一貫して維持する必要がある。
Decision Timing: Now。

Related PRDs:

- Not applicable

Related Stories:

- Not applicable

Related Requirements:

- Not applicable

Related Architecture Changes:

- Not applicable

Related ADRs:

- Not applicable

Options:

- `cli → application → domain` を主要な依存方向とし、infrastructureがapplicationのI/Oポートとdomainの型へ依存する4境界を採用する。
- domainを中心に置き、cli、application、infrastructureがdomainへ直接依存する構造を採用する。
- 明示的な依存境界を設けず、各パッケージが必要なパッケージへ直接依存する構造を採用する。

Evaluation Criteria:

- domainをCLI、対話UI、永続化、OS固有APIから独立させられること。
- 正しい依存方向を説明し、循環依存を防止できること。
- domainとapplicationを外部I/Oなしでテストできること。
- 初期版の規模に対して不要な依存境界や抽象化を増やさないこと。
- 機能追加や外部技術の変更に伴う修正範囲を限定できること。

Recommendation:

- `cli → application → domain` を主要な依存方向とし、infrastructureがapplicationのI/Oポートとdomainの型へ依存する4境界を採用する。パッケージ編成の全体原則と配置規則は `docs/engineering/structure.md`、機能ごとの具体的な構成は関連PRDの `architecture-change.md` で定義する。

ADR Recommendation: Create ADR
ADR Recommendation Reason:

- 全PRDと複数モジュールの責務配置、依存方向、テスト境界へ影響する。
- 後から構造を変更する場合の修正範囲が広く、Engineering Foundationの原則を規定する判断である。

Human Decision Reason:

- domainを外部技術から分離し、依存方向、責務境界、テスト境界をシステム全体で一貫させるため、推奨案を採用してADR化する。

Resulting ADR:

- `docs/decisions/adr-001-layer-architecture-and-dependency-direction.md`

### ADC-002: ユーザー設定と配布管理情報の永続化方式および分割境界

<!-- 判断時期の固定値を1行で定義するため。 -->

Status: Accepted
Decision Needed: ユーザー単位の全体設定と配布先ごとの管理情報について、永続化方式と分割境界を実装開始前に固定する必要がある。
Decision Timing: Now。

Related PRDs:

- Not applicable

Related Stories:

- Not applicable

Related Requirements:

- Not applicable

Related Architecture Changes:

- Not applicable

Related ADRs:

- Not applicable

Options:

- 全体設定を `config.yaml`、配布管理情報を `map.yaml` とする独立した2つのYAMLファイルへ保存する。
- 全体設定と配布管理情報を単一のYAMLファイルへ保存する。
- 全体設定と配布管理情報を単一のSQLiteデータベースへ保存する。

Evaluation Criteria:

- 初期版の単一ユーザー、単一マシン、ローカル実行に適合すること。
- 利用者が保存内容を確認しやすく、運用上の問題を診断しやすいこと。
- 実装および運用の複雑度を必要最小限にできること。
- 設定と配布管理情報の責務およびスキーマ変更を分離できること。
- 採用方式に適した排他制御と原子的更新によりデータ整合性を維持できること。
- スキーマの互換性を検証し、未対応バージョンを安全に拒否できること。
- データ量または同時実行要件が変化した場合に代替ストレージへ移行できること。

Recommendation:

- `XDG_CONFIG_HOME/context/` 配下へ、全体設定を保持する `config.yaml` と配布先ごとの管理情報を保持する `map.yaml` を独立して保存する。
- 採用時の必須要件として、各ファイルに独立したスキーマバージョンを設け、未知フィールドの拒否、書き込み前検証、排他ロック、同一ディレクトリでの一時ファイル作成と原子的な置換を実施する。複数ファイルの完全なトランザクションは初期版では保証せず、途中失敗時に変更済みと未変更の対象を報告する。

ADR Recommendation: Create ADR
ADR Recommendation Reason:

- データ形式、互換性、権限、同時更新、障害時の整合性へシステム横断で影響する。
- 後から保存形式または分割境界を変更するコストが高く、安全性に関する意図的な制約を記録する必要がある。

Human Decision Reason:

- 初期版の単一ユーザー、単一マシンでの運用に適合し、設定と配布管理情報の責務およびスキーマ変更を分離できるため、推奨案を採用してADR化する。

Resulting ADR:

- `docs/decisions/adr-002-yaml-persistence-and-storage-boundaries.md`

### ADC-003: config.yamlのスキーマ解釈の所有境界

<!-- 判断時期の固定値を1行で定義するため。 -->

Status: Accepted
Decision Needed: `config.yaml`の未知フィールド拒否、スキーマバージョン検証、値検証をドメイン契約として一箇所で強制しつつ、その他のYAML処理とファイルI/Oをinfrastructureへ隔離する所有境界を実装前に確定する必要がある。
Decision Timing: Before Implementation。

Related PRDs:

- `docs/prds/prd-001-safe-context-repository-setup/prd.md`

Related Stories:

- `prd-001-safe-context-repository-setup/ST-001`
- `prd-001-safe-context-repository-setup/ST-003`
- `prd-001-safe-context-repository-setup/ST-005`

Related Requirements:

- `prd-001-safe-context-repository-setup/AC-016`
- `prd-001-safe-context-repository-setup/AC-018`
- `prd-001-safe-context-repository-setup/TR-013`
- `prd-001-safe-context-repository-setup/TR-014`
- `prd-001-safe-context-repository-setup/TR-015`

Related Architecture Changes:

- `docs/prds/prd-001-safe-context-repository-setup/architecture-change.md`

Related ADRs:

- `docs/decisions/adr-001-layer-architecture-and-dependency-direction.md`: domainを外部技術から分離する既存判断に対する限定例外。
- `docs/decisions/adr-002-yaml-persistence-and-storage-boundaries.md`: `config.yaml`の厳格デコード要件を適用する。

Options:

- `config.yaml`の厳格デコードと値検証をdomainへ配置し、YAMLエンコード、ファイルI/O、`map.yaml`を含むその他のYAML処理をinfrastructureへ配置する。
- `config.yaml`を含むすべてのYAMLデコードをinfrastructureへ配置し、domainはデコード済みの設定値だけを検証する。
- YAMLスキーマ処理専用のcodecパッケージを設け、domainとinfrastructureのどちらにも属さない境界として扱う。

Evaluation Criteria:

- `config.yaml`のスキーマバージョン、未知フィールド、値の妥当性を一貫して強制できること。
- ファイルI/O、XDGパス、ロック、原子的置換をdomainから分離できること。
- `map.yaml`や将来追加されるYAML処理へ例外が拡大しないこと。
- 責務境界と依存方向をコードレビューで明確に判定できること。
- 初期版に不要なパッケージや変換処理を増やさないこと。

Recommendation:

- `config.yaml`の厳格デコードと値検証だけをdomainへ配置する。domainはファイルを直接読み書きせず、受け取ったバイト列をドメイン設定へ変換する。YAMLエンコード、ファイルI/O、`map.yaml`を含むその他のYAML処理はinfrastructureへ配置する。

ADR Recommendation: Create ADR
ADR Recommendation Reason:

- Accepted済みADR-001にある外部技術の分離原則へ限定例外を追加し、複数の後続PRDが参照する設定境界へ影響する。
- 例外の拡大を防ぐため、適用対象と禁止範囲を実装前に固定する必要がある。

Human Decision Reason:

- `config.yaml`のYAML処理はdomainへ配置し、その他のYAML処理はinfrastructureへ配置する方針を人間が確定した。

Resulting ADR:

- `docs/decisions/adr-006-config-schema-ownership-boundary.md`

## Assumptions

- `docs/product.md` はAccepted済みの上位文脈として有効である。
- `docs/engineering/technology.md`、`docs/engineering/structure.md`、`docs/engineering/development-rules.md` の現行内容を候補抽出の入力とする。
- 初期版は単一ユーザー、単一マシンのローカル実行を対象とし、ネットワークアクセスを行わない。

## Open Questions

- Not applicable
