# ADR-003: Context Repository構造の宣言方式

Status: Accepted
Status Reason: 最重レビューにおいて、上流要件（prd-001/requirements.md）および開発規則との完全な整合性が確認され、承認されたため。
Decision ID: ADR-003
Date: 2026-06-07。

## Context

Accepted済みRequirementsで固定したContext Repositoryの必須構造と、原本およびプロジェクト別配布定義の配置について、採用理由、代替案、拡張性の制約を実装開始前にADRへ正式記録する必要がある。

この判断は、`docs/prds/prd-001-safe-context-repository-setup/adr-candidates.md`の`ADC-001`を正式化する。

## Decision

初期版に必要な最小構造として、Repositoryルートの`projects/`と`utils/skills/`、1件以上の`projects/<project-name>/`、各プロジェクトの`skills/`、各Skillの`SKILL.md`を固定する。`projects/<project-name>/AGENTS.md`、`CLAUDE.md`、`README.md`は任意とし、`projects/<project-name>/skills/`と`utils/skills/`は空を許容する。必須ディレクトリは実際に列挙し、必須ファイルは実際に開いて読み取れることを検証する。追加の配置規則やマニフェスト導入は後続要件で判断する。

## Options

### Option A: 固定パスと固定ファイル名による規約化（最小構造の固定）

Pros:

- マニフェストの構文解析、互換性管理、移行処理を増やす必要がない。
- 配布対象の特定と検証ロジックを単純に保てる。

Cons:

- ユーザーが任意のパスに配布ファイルを自由に配置するようなカスタマイズは行えない。

Rejection Reason:

- Not applicable

### Option B: マニフェストファイルによる配置の宣言

Pros:

- 柔軟なディレクトリ構造やファイル配置のカスタマイズが可能。

Cons:

- マニフェストのファイル形式、構文、バージョン管理、スキーマ検証、移行処理などの実装複雑度が増大する。
- 初期版の開発速度を阻害する。

Rejection Reason:

- 初期版に必要な配布対象の一意特定に対して、マニフェストの導入は構文解析や移行処理などの余分な複雑性を導入するため。

### Option C: 初期固定構造とマニフェストの両立

Pros:

- 将来の拡張性を最初から保証できる。

Cons:

- 未確定のユースケースに対して過剰な抽象化と実装オーバーヘッドが発生する。

Rejection Reason:

- 初期段階では過剰設計となり、実際必要になるかも不明な拡張性のために実装コストを支払うのは適切ではないため。

## Rationale

Accepted済みRequirementsのTR-016〜TR-019と整合し、初期版の配布対象を一意かつ検証可能に特定しながら、マニフェストの構文、互換性管理、移行処理を増やさずに実装できるため。追加の配置規則またはマニフェストは、後続要件で必要性が生じた場合に改めて判断する。

## Consequences

- Repository構造のバリデーションは、この固定構造（`projects/`, `utils/skills/`等）の存在と読み取り可能性を厳格にチェックする。
- ディレクトリやファイルの追加配置は行えない。
- 将来的な配置規則の変更は、Repositoryおよび検証・配布・同期処理への広範な影響を伴う可能性がある。

## Risks

- 将来的に構造を変更した際、既存のContext Repositoryやコードベースに対する互換性維持が難しくなる可能性がある。

## Mitigations

- 新しい配置規則が必要となった場合は、後続のADRにてマニフェスト導入などの移行パスを評価・決定する。

## Related PRDs

- docs/prds/prd-001-safe-context-repository-setup/prd.md

## Related Stories

- prd-001-safe-context-repository-setup/ST-001
- prd-001-safe-context-repository-setup/ST-004

## Related Requirements

- prd-001-safe-context-repository-setup/AC-001
- prd-001-safe-context-repository-setup/AC-010
- prd-001-safe-context-repository-setup/AC-011
- prd-001-safe-context-repository-setup/AC-012
- prd-001-safe-context-repository-setup/AC-015
- prd-001-safe-context-repository-setup/TR-001
- prd-001-safe-context-repository-setup/TR-009
- prd-001-safe-context-repository-setup/TR-010
- prd-001-safe-context-repository-setup/TR-011
- prd-001-safe-context-repository-setup/TR-016
- prd-001-safe-context-repository-setup/TR-017
- prd-001-safe-context-repository-setup/TR-018
- prd-001-safe-context-repository-setup/TR-019

## Related Architecture Changes

- None

## Supersedes

- None

## Superseded By

- None

## Assumptions

- 初期版は個人開発者による単一ユーザー、単一マシンのローカル実行を対象とし、ネットワークアクセスを行わない。
- Context Repositoryは利用者が事前にローカルへクローンし、利用可能な状態に保つ。

## Open Questions

- Not applicable
