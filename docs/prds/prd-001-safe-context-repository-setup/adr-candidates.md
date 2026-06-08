# ADR Candidates

Status: Accepted
Status Reason: 最重レビューで判断時系列、追跡関係、信頼境界の選択肢と残存リスクを修正し、3件の候補を推奨案でADR化する判断として確定した。
Source: prd-001-safe-context-repository-setup。

## Candidates

### ADC-001: Context Repository構造の宣言方式

<!-- 判断時期の固定値を1行で定義するため。 -->

Status: Accepted
Decision Needed: Accepted済みRequirementsで固定したContext Repositoryの必須構造と、原本およびプロジェクト別配布定義の配置について、採用理由、代替案、拡張性の制約を実装開始前にADRへ正式記録する必要がある。
Decision Timing: Now。

Related PRDs:

- prd-001-safe-context-repository-setup

Related Stories:

- ST-001
- ST-004

Related Requirements:

- AC-001
- AC-010
- AC-011
- AC-012
- AC-015
- TR-001
- TR-009
- TR-010
- TR-011
- TR-016
- TR-017
- TR-018
- TR-019

Related Architecture Changes:

- Not applicable

Related ADRs:

- Not applicable

Options:

- Context Repository内の固定パスと固定ファイル名で必須構造を規約化する。
- Context Repository内のマニフェストで、原本とプロジェクト別配布定義の配置を宣言する。
- 初期版に必要な最小構造だけを固定し、追加の配置規則またはマニフェスト導入は後続要件で判断する。

Evaluation Criteria:

- 初期版の配布対象であるAGENTS.md、CLAUDE.md、共通・リポジトリ固有Skillを一意に特定できること。
- 構造不備、必須対象の欠落、不正な定義を設定前に検出できること。
- 利用者がContext Repositoryを作成、確認、修正しやすいこと。
- 検証規則とエラーを自動テストで再現できること。
- 初期版に不要な構文、互換性管理、移行処理を増やさないこと。
- 後続の配布対象または配置規則の追加に対応できること。

Recommendation:

- 初期版に必要な最小構造として、Repositoryルートの`projects/`と`utils/skills/`、1件以上の`projects/<project-name>/`、各プロジェクトの`skills/`、各Skillの`SKILL.md`を固定する。`projects/<project-name>/AGENTS.md`、`CLAUDE.md`、`README.md`は任意とし、`projects/<project-name>/skills/`と`utils/skills/`は空を許容する。必須ディレクトリは実際に列挙し、必須ファイルは実際に開いて読み取れることを検証する。追加の配置規則やマニフェスト導入は後続要件で判断する。

ADR Recommendation: Create ADR
ADR Recommendation Reason:

- Context Repositoryは全配布操作の唯一の正本であり、その構造は後続の`context add`と`context sync`を含む複数PRDへ影響する。
- 宣言方式を後から変更すると、Repository、検証処理、配布処理、テスト、互換性へ広範な変更が必要になる。
- 固定構造とマニフェストには複数の合理的な選択肢があり、初期版で受容する拡張性の制約を記録する必要がある。
- 推奨内容はAccepted済みRequirementsのTR-016〜TR-019で必須化されており、ADRでは要件を変更せず判断理由と制約を正式記録する。

Human Decision Reason:

- Accepted済みRequirementsのTR-016〜TR-019と整合し、初期版の配布対象を一意かつ検証可能に特定しながら、マニフェストの構文、互換性管理、移行処理を増やさずに実装できるため、推奨案を採用してADR化する。追加の配置規則またはマニフェストは、後続要件で必要性が生じた場合に改めて判断する。

Resulting ADR:

- `docs/decisions/adr-003-context-repository-structure-specification.md`

### ADC-002: Context Repositoryの同一性の判定方式

<!-- 判断時期の固定値を1行で定義するため。 -->

Status: Accepted
Decision Needed: Accepted済みRequirementsで固定した、Context Repositoryの同一性を判定する方式について、採用理由、代替案、Repository移動や別名パスに関する制約を実装開始前にADRへ正式記録する必要がある。
Decision Timing: Now。

Related PRDs:

- prd-001-safe-context-repository-setup

Related Stories:

- ST-002
- ST-003
- ST-004

Related Requirements:

- AC-004
- AC-007
- AC-008
- AC-009
- AC-010
- AC-013
- AC-014
- AC-015
- TR-007
- TR-008
- TR-009
- TR-010
- TR-011
- TR-012

Related Architecture Changes:

- Not applicable

Related ADRs:

- `docs/decisions/adr-002-yaml-persistence-and-storage-boundaries.md`: 判定に使用するContext Repositoryの識別子を`config.yaml`へ保存する既存の永続化判断を適用する。識別子の意味は判断境界に含まれていない。

Options:

- シンボリックリンクを解決せず、字句的に正規化した絶対パスの文字列で同一性を判定する。
- シンボリックリンクを解決した実体パスで同一性を判定する。
- デバイス番号とinodeなど、ファイルシステムが提供する識別情報で同一性を判定する。

Evaluation Criteria:

- macOSとLinuxで一貫した判定規則を説明できること。
- シンボリックリンクを設定対象として受け入れない安全要件と整合すること。
- 保存済み識別子を利用者が確認し、診断できること。
- パス表記の差による不要な設定変更確認を抑制できること。
- Repositoryの移動、置換、マウント差異に対する挙動を明確にできること。
- 外部ランタイムやネットワークアクセスを必要としないこと。

Recommendation:

- シンボリックリンクを解決せず、字句的に正規化した絶対パスの文字列で同一性を判定する。判定後もRepositoryを再検証し、同一パス上の内容または安全性が変化した場合は設定を書き換えずエラー終了する。

ADR Recommendation: Create ADR
ADR Recommendation Reason:

- Repositoryの識別方式は設定変更の確認、冪等性、安全性、保存形式に影響し、後続の配布と同期でも共通して使用される可能性が高い。
- 実体パスまたはファイルシステム識別情報へ後から変更すると、既存設定との互換性と同一性の意味が変わる。
- 字句的パスを採用することで受容するRepository移動や別名パスの制約を記録する必要がある。
- 推奨内容はAccepted済みRequirementsのTR-007とTR-008で必須化されており、ADRでは要件を変更せず判断理由と制約を正式記録する。

Human Decision Reason:

- Accepted済みRequirementsのTR-007とTR-008に整合し、macOSとLinuxで説明可能かつ利用者が診断しやすい識別子を、外部ランタイムなしで実装できるため、推奨案を採用してADR化する。Repositoryの移動またはシンボリックリンク経由の別名パスは別Repositoryとして扱われ、設定変更の確認が必要になる制約を受容する。

Resulting ADR:

- `docs/decisions/adr-004-context-repository-identity-determination.md`

### ADC-003: Context Repositoryのファイルシステム信頼境界

<!-- 判断時期の固定値を1行で定義するため。 -->

Status: Accepted
Decision Needed: Context Repositoryを唯一の正本として安全に使用するため、どの権限状態とパス構成を信頼可能として受け入れるかを実装前に固定する必要がある。
Decision Timing: Now。

Related PRDs:

- prd-001-safe-context-repository-setup

Related Stories:

- ST-001
- ST-002
- ST-003
- ST-004

Related Requirements:

- AC-010
- AC-012
- AC-013
- AC-014
- AC-015
- TR-001
- TR-004
- TR-008
- TR-009
- TR-010
- TR-011
- TR-012

Related Architecture Changes:

- Not applicable

Related ADRs:

- `docs/decisions/adr-002-yaml-persistence-and-storage-boundaries.md`: ユーザー設定側の権限、ロック、原子的更新に関する判断を適用する。Context Repository自体の信頼境界は判断境界に含まれていない。

Options:

- 現在の利用者が所有するRepositoryだけを受け入れる。他者が書き込める親ディレクトリ配下を拒否し、現在の利用者による読み取りとディレクトリ検索を必須とする。グループと他ユーザーへの書き込み、およびパス構成要素と必須対象のシンボリックリンクも拒否する。
- 現在の利用者が読み取りとディレクトリ検索を行え、グループと他ユーザーに書き込みが許可されず、パス構成要素と必須対象にシンボリックリンクがないRepositoryだけを受け入れる。
- グループ共有による書き込みを許可し、他ユーザーへの書き込みとシンボリックリンクだけを拒否する。
- 読み取り可能性と必須構造だけを検証し、書き込み権限とシンボリックリンクを受け入れる。

Evaluation Criteria:

- 検証後に別の利用者またはリンク先の変更によって配布元が意図せず置き換わるリスクを抑制できること。
- Repositoryの所有者または親ディレクトリの権限による置換可能性を抑制できること。
- 初期版の個人開発者、単一ユーザー、ローカル利用という前提に適合すること。
- macOSとLinuxで検証可能な規則として定義できること。
- 正当なローカルRepositoryを過度に拒否しないこと。
- 検証失敗の対象と復旧行動を利用者へ説明できること。
- 設定時だけでなく同じRepositoryへの再実行時にも一貫して再検証できること。

Recommendation:

- 現在の利用者が読み取りとディレクトリ検索を行え、グループと他ユーザーに書き込みが許可されず、パス構成要素と必須対象にシンボリックリンクがないRepositoryだけを受け入れる。検証失敗時は設定を書き込まず、既存設定を維持する。

ADR Recommendation: Create ADR
ADR Recommendation Reason:

- Context Repositoryは後続の全配布内容の正本であり、信頼境界の誤りは不正な内容の配布とデータ整合性へ重大な影響を与える。
- 複数の合理的な権限モデルがあり、安全性とローカル運用の柔軟性のトレードオフを記録する必要がある。
- この判断はRepository検証、再検証、エラー処理、OS別テストへ横断的に影響する。

Human Decision Reason:

- 初期版の個人利用、単一ユーザー、ローカル実行という前提に適合し、Context Repositoryへの意図しない書き込みとシンボリックリンクによる参照先変更を拒否できるため、推奨案を採用してADR化する。所有者と親ディレクトリを検証する厳格案は、正当なローカルRepositoryを拒否する可能性と実装複雑度が増すため初期版では採用しない。所有者と親ディレクトリによる置換可能性は、下流の設計および実装レビューで継続確認する残存リスクとして受容する。

Resulting ADR:

- `docs/decisions/adr-005-context-repository-filesystem-trust-boundary.md`

## Assumptions

- `docs/product.md`、`prd.md`、`backlog.md`、`requirements.md`、Engineering Foundation、ADR-001、ADR-002の現行内容を候補抽出の入力とする。
- 初期版は個人開発者による単一ユーザー、単一マシンのローカル実行を対象とし、ネットワークアクセスを行わない。
- Context Repositoryは利用者が事前にローカルへクローンし、利用可能な状態に保つ。

## Open Questions

- Not applicable
