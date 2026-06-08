---
name: create-implementation-plan
description: Accepted済みprd.mdから設計差分、重要判断、実装単位、変更候補ファイル、テスト、品質ゲートを一つのimplementation-plan.mdへまとめる。実装前の技術設計と作業計画を同時に作る場合に使用する。
---

# 実装計画作成

要件、設計、Task、テスト計画を`implementation-plan.md`へ統合し、実装中の重複記録をなくす。

## 必須条件

- PRDの3桁IDを受け取る。
- 対象`prd.md`が`Accepted`で、Blocking Open Questionがない。
- `docs/engineering/*.md`の適用文書が`Accepted`である。

## 1. 実装コンテキストを調査する

PRD、Engineering Foundation、関連ADR、現在Architecture、コード、設定、テスト、品質ゲート、Git差分を読む。既存構造に沿う最小変更を優先する。

## 2. 重要判断を処理する

複数の合理的な選択肢があり、後からの変更コストが高い、複数PRDへ影響する、またはセキュリティ・データ・運用境界を決める判断だけADR対象とする。

ADR対象は選択肢、評価軸、推奨案を一問ずつ人間と確定し、`docs/decisions/adr-XXX-*.md`へ直接Draftを作る。永続的な`adr-candidates.md`は作らない。局所的で可逆な判断は計画の`Design Decisions`へ簡潔に記録する。

## 3. 計画を作る

`docs/prds/prd-XXX-<slug>/implementation-plan.md`を次の構成で作成する。

- Status、Status Reason、PRD ID
- Source
- Change Summary
- Requirements Coverage
- Design
- Design Decisions and ADRs
- Work Units
- Files to Change
- Test Strategy
- Quality Gates
- Compatibility, Migration, Rollback
- Security and Operations
- Risks and Follow-up
- Open Questions

`Requirements Coverage`は各ACと検証方法の対応だけを簡潔に示す。`Work Units`は実装順、依存、完了条件を持つが、詳細な状態履歴やTask別レビュー欄を持たせない。

テストはユーザー価値に近いE2E・integrationを優先し、重要な境界をunitで補う。変更予定ファイルは候補として記載し、関数単位の手順まで固定しない。

## 4. リスクに応じて深さを変える

- `Low`: 局所設計、主要テスト、標準品質ゲート
- `Medium`: 失敗経路、互換性、回帰範囲を追加
- `High`: 脅威、データ保全、移行・復旧、境界テスト、必要なADRを明示

文書構成は変えず、必要な節の深さだけを変える。

## 5. 生成する

設計、実装単位、テスト、ADR、リスクを要約し、人間の生成承認後に`Status: Draft`で作成する。`architecture-change.md`と`tasks.md`は作成しない。

## 6. 自己検証する

- 全ACに実装または検証の対応先がある
- 依存順に連続実装できる
- 新しい重要判断が未処理でない
- Scope外変更を含まない
- テストと品質ゲートがリスクに見合う
- 移行、復旧、運用の要否が明確である

完了時は計画、ADR、Blocking、次フェーズの`review-implementation-plan`を報告する。
