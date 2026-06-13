---
name: define-prd
description: ユーザーの課題を一問ずつ掘り下げ、Goal、Scope、Story、Acceptance Criteria、主要制約を統合した単一の prd.md を作成または更新する。
---

# PRD定義

ユーザー課題を、独立して成功判定できる1つの成果へ絞り、`prd.md` を作成する。本スキルでは実装方法や詳細設計は行わず、解くべき課題と要件の整理に集中する。

## 入力

- 新規作成: 解決したいユーザー課題・要件
- 更新: 対象PRDの3桁IDと変更したい方針

## 1. 現状を調査する

`docs/product.md`、`docs/decisions/adr-summary.md`、`docs/engineering/*.md`、関連するコード、既存PRD、`docs/architecture/*.md` を必要な範囲だけ調査する。コードや設定から判明する事実はユーザーに質問せず、自分で調べる。

## 2. 一問ずつ議論する

次の順で未確定事項を解消する。各質問では推奨回答と理由を示し、一度に1つだけ質問する。

1. 対象ユーザーとProblem（何が課題か）
2. Goalと成功判定（Success Metrics）
3. ScopeとOut of Scope
4. ユーザーが観測できる主要経路と失敗経路
5. セキュリティ、データ、互換性、運用上の制約
6. 前提（Assumptions）、未確定事項、Follow-up

## 3. リスクを分類する

PRD全体の変更リスクを `Low / Medium / High` で判定し、PRD内に記録する。認証・認可、データ移行、破壊的変更、公開API、外部連携、ファイルシステム信頼境界は原則 `High` とする。

## 4. PRDファイルを組み立てる

`docs/prds/prd-XXX-<slug>/prd.md` を次の構成で作成する。

- PRD ID
- Goal, Problem, Target User, User Value
- Success Metrics
- Scope, Out of Scope
- User Stories (価値単位)
- Acceptance Criteria (外部から検証可能な完成条件)
- Constraints (制約事項)
- Non-Functional Requirements (非機能要件)
- Risk Assessment (リスク評価)
- Assumptions (前提)
- Open Questions (未解決事項)

**※注意**: `Status: Draft/Accepted` や `Status Reason` などのメタデータヘッダーは記述しない。`backlog.md` や `requirements.md` は重複するため作成しない。

## 5. 生成する

書き込み前に主要要件、AC、リスク、未確定事項を要約して人間に提示し、生成の承認を得る。承認後にファイルを生成する。
完了時は、出力先、リスク、Blocking Open Questions、および次フェーズの `review-prd` を報告する。
