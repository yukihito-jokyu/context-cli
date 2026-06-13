---
name: review-prd
description: 統合されたprd.mdをプロダクト価値、Scope、Acceptance Criteria、制約、リスクの観点から独立レビューし、合意した修正を反映してAccepted候補にする。PRD Draftの確定や方針変更後の再レビューに使用する。
---

# PRDレビュー

`prd.md`が、不要な下流文書を作らずに実装計画の正本として使えるかを確認する。

## 必須条件

- 対象PRDの3桁IDまたは具体的な`prd.md`パスを受け取る。
- 作成直後の推論を根拠にせず、PRD、上流文書、コードを読み直す。
- 同一セッションで実行する場合は、作成時の会話要約をレビュー根拠にしない。

## 1. 資料を読む

対象`prd.md`、`docs/product.md`、`docs/engineering/*.md`、関連ADR、既存Architecture、関連コードを必要な範囲で読む。Statusや参照先の存在も確認する。

## 2. レビューする

次を重点確認する。

- Problem、Goal、Target User、User Valueが一貫している
- Success Metricsが実装完了後に観測可能である
- Scopeが1つの独立成果に収まっている
- StoryとACがScope、正常系、主要失敗系を覆う
- ACが解決策ではなく観測可能な結果を表す
- セキュリティ、データ、互換性、運用制約がリスクに見合う
- 不要な要件、将来要件、実装詳細が混入していない
- BlockingなOpen Questionが残っていない

指摘は`Blocking / Major / Minor`に分類し、対象箇所、影響、推奨修正を一覧で示す。

## 3. 一問ずつ解消する

方針判断が必要な`Blocking`と`Major`を一度に1つずつ確認する。各質問で推奨案と理由を示す。合意した修正は対象`prd.md`だけへ反映する。

意味を変えない機械的修正は一覧確認後にまとめて行う。別成果物を自動修正しない。

## 4. 確定する

全体を再検査し、次を満たす場合だけ`Accepted`を提案する。

- 未解決のBlockingがない
- 対応対象のMajorが解消済み
- Deferredに判断条件がある
- PRD内のStory、AC、制約が相互に追跡できる
- 実装計画を推測なしで開始できる

人間の明示承認後だけ`Status: Accepted`へ変更する。独立レビュー報告書は作らず、`Status Reason`には主要な確定事項と残存リスクだけを短く記録する。

完了時は指摘、修正、残存リスク、次フェーズの`create-implementation-plan`を報告する。
