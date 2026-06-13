---
name: define-prd
description: ユーザー課題を一問ずつ掘り下げ、Goal、Scope、Story、Acceptance Criteria、主要制約を統合した単一のprd.mdを作成または更新する。新機能の開始、既存PRDの方針変更、実装前の要求整理に使用する。
---

# PRD定義

ユーザー課題を、独立して成功判定できる1つの成果へ絞り、`prd.md`だけで実装計画を開始できる状態にする。

## 入力

- 新規作成: 解決したいユーザー課題
- 更新: 対象PRDの3桁IDと変更したい方針

## 1. 現状を調査する

`docs/product.md`、`docs/engineering/*.md`、関連するコード、既存PRD、ADR、`docs/architecture/*.md`を必要な範囲だけ読む。コードベースから判明する事項はユーザーへ質問しない。

## 2. 一問ずつ議論する

次の依存順で未確定事項を解消する。各質問では推奨回答と理由を示し、一度に1つだけ質問する。

1. 対象ユーザーとProblem
2. Goalと成功判定
3. ScopeとOut of Scope
4. ユーザーが観測できる主要経路と失敗経路
5. セキュリティ、データ、互換性、運用上の制約
6. 前提、未確定事項、Follow-up

実装方法、パッケージ構成、変更ファイルはこのフェーズで固定しない。複数の独立成果が混在する場合はPRDを分割する。

## 3. リスクを分類する

`Low / Medium / High`を判定する。認証・認可、課金、秘密情報、データ移行、破壊的変更、公開API、外部連携、ファイルシステム信頼境界は原則`High`とする。リスクはレビューとテストの深さを決めるために使い、文書数は増やさない。

## 4. PRDを組み立てる

`docs/prds/prd-XXX-<slug>/prd.md`を次の構成で作成する。

- Status、Status Reason、PRD ID
- Goal、Problem、Target User、User Value
- Success Metrics
- Scope、Out of Scope
- User Stories
- Acceptance Criteria
- Constraints
- Non-Functional Requirements
- Risk Assessment
- Assumptions
- Open Questions

Storyは価値単位、ACは外部から検証可能な振る舞いとして記述する。技術制約は必要な理由とともに記述し、実装詳細を先取りしない。該当しない定型項目は追加しない。

## 5. 生成する

書き込み前に境界、主要AC、リスク、未確定事項を要約し、人間の生成承認を得る。承認後に`Status: Draft`で作成または更新する。

重複する`backlog.md`と`requirements.md`は作成しない。ADR候補、設計、Task、テスト計画は次フェーズへ送る。

## 6. 自己検証する

- Goalが1つで成功判定できる
- 全ScopeがStoryとACで表現される
- 正常系、主要失敗系、境界条件がある
- Out of Scopeと制約が矛盾しない
- Open Questionの分類がBlockingまたはDeferredとして明確である
- コードと既存文書に既知の矛盾がない

完了時は出力先、リスク、Blocking、次フェーズの`review-prd`を報告する。
