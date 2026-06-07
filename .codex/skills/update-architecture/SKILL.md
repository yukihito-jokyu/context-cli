---
name: update-architecture
description: 実装とコードレビューが完了したPRDについて、確定済みarchitecture-change、関連ADR、コード、設定を照合し、実装済みシステムの現在状態をdocs/architecture/*.mdへ反映する。最初のPRD後の標準6ファイル作成、後続PRD後の影響ファイル更新、architecture文書レビューへの引き渡しに使用する。
---

# アーキテクチャ更新

実装済みシステムの現在状態だけを `docs/architecture/*.md` へ反映する。変更履歴、未実装の予定、設計判断の正当化は記載しない。

## 必須条件

- 呼び出し時にPRDの3桁連番IDを必須入力として受け取る。
- 対話、記述、完了報告には原則として日本語を使う。識別子、パス、コード表記は維持する。
- 実行時は `docs/AI-driven-development.md` を参照しない。本スキルの記載規則に従う。
- 独立した更新レポートは作らない。結果はarchitecture文書と `tasks.md` に記録する。
- コード、設定、上流成果物、Engineering Foundationを変更しない。
- コミット、PR作成、push、デプロイ、本番操作は、人間の指示または事前承認なしに行わない。

## 1. 対象PRDを解決する

PRD IDから `docs/prds/prd-XXX-*/` を検索し、対象ディレクトリを一意に特定する。対象が存在しない、複数存在する、またはIDと文書内のPRD IDが一致しない場合は停止する。

## 2. 開始条件を検査する

次をすべて満たす場合だけ更新へ進む。

- `prd.md`、`backlog.md`、`requirements.md`、`tasks.md` が存在する。
- `architecture-change.md` が存在し、人間により `Implemented` へ確定されている。
- 全実装Taskが `Completed` である。
- PRD全体のCode Review Summaryが `Completed` かつHuman Decisionが `Approved` である。
- `Critical / High` の未解決指摘がない。
- 必要なADRが `Accepted` または `Implemented` である。
- `docs/engineering/technology.md`、`structure.md`、`development-rules.md` が存在し、適用対象が `Accepted` である。
- Blocking Open Questionがない。

条件を満たさない場合は、不足条件、影響、戻るべき工程を提示して停止する。

## 3. 入力を読む

次を読む。

1. `docs/product.md`
2. `docs/engineering/technology.md`、`structure.md`、`development-rules.md`
3. 対象PRDの全成果物
4. 関連するAcceptedまたはImplemented済みADR
5. 存在するすべての `docs/architecture/*.md`
6. `tasks.md` のActual Files Changed、Test Results、Code Review
7. 今回の実装差分とGit状態
8. 関連するコード、設定、スキーマ、API、依存関係、デプロイ・運用設定
9. `docs/ai-driven-development/templates/architecture/` の標準6テンプレート

会話履歴や一時的な要約を正本にしない。

## 4. 初回作成か更新かを判定する

標準ファイルは `overview.md`、`database.md`、`api.md`、`domain.md`、`package.md`、`operations.md` の6つに固定する。

- 6ファイルがすべて存在しない場合は初回作成とし、すべてを対応テンプレートから作成する。
- 6ファイルがすべて存在する場合は更新とし、実装差分と `architecture-change.md` から影響を受けるファイルだけを選ぶ。
- 一部だけ存在する場合は不完全な状態として停止し、不足ファイルと影響を報告する。
- 無関係なファイルを整理、表現統一、再構成の目的で変更しない。
- 該当しない領域もファイルを削除せず、本文を `Not applicable` とし、理由を残す。

## 5. 調査範囲を決める

`tasks.md` のActual Files Changedと今回のGit差分を起点に、現在状態を正確に記述するために必要な範囲だけ調査する。

- 関連コード、設定、テスト、依存関係を追跡する。
- 必要に応じてDB、API、ドメイン、パッケージ、デプロイ、監視、運用設定まで確認する。
- 実行時の構成や外部依存がコードだけでは分からない場合は、設定と公式資料を確認する。
- 未変更領域の全面的なarchitecture監査は行わない。
- 既存architectureの記述が疑わしい場合は、今回の変更との影響関係を確認し、不一致として扱う。

## 6. 現在状態を確定する

意図、要件、判断はPRD、Requirements、ADRを正とし、技術規範はEngineering Foundationを正とする。現在の技術状態はコード、設定、テスト、確定版 `architecture-change.md` を照合して確認する。

実装済み構造、名称、パス、確定済み設計の反映漏れと、参照情報の追加は記載漏れとして更新できる。

次を発見した場合は更新を停止する。

- コードと `architecture-change.md` のどちらが正しいか判断できない。
- 実装がADR、Requirements、Engineering Foundationに反している。
- Scope、要件、重要な設計判断、技術規範を変更する必要がある。
- 未承認の公開API、データ構造、外部依存、運用方式が実装されている。
- 単なる記載漏れでは説明できない設計差異がある。

停止時は、差異、対象箇所、影響範囲、正本の推奨候補、戻るべき工程を提示する。人間の判断と必要な再レビューが完了するまで再開しない。

## 7. architecture文書を更新する

各ファイルの見出し構成と順序は対応テンプレートに固定し、独自見出しを追加しない。

- 未実装の予定、候補、履歴、会話上の判断経緯を書かない。
- 現在のコードと運用状態を、後続AIが実装判断に使える粒度で簡潔に記述する。
- `Source` に確認したコード、設定、PRD、`architecture-change.md`、ADRを列挙する。
- `Related PRDs` に今回のPRDを追加し、既存参照を維持する。
- `Related ADRs` に適用されるADRを記載し、既存参照を根拠なく削除しない。
- 既存内容を整理目的で削除または再構成しない。

新規作成または内容変更したファイルは `Status: Review` とする。`Status Reason` に対象PRDの実装結果を反映し、別セッションのレビューが必要であることを書く。

変更していないファイルのStatusと内容は維持する。初回作成では6ファイルすべてを `Review` とする。

## 8. 自己検証する

更新後に次を確認する。

- 初回は標準6ファイルがすべて存在する。
- 更新時は影響ファイルだけが変更されている。
- 見出し構成と順序がテンプレートに一致する。
- 変更ファイルが `Review` で、Status Reasonが更新理由を示している。
- 現在状態だけが記載され、未実装の予定や履歴が混ざっていない。
- 6ファイル間に矛盾がない。
- コード、設定、`architecture-change.md`、ADR、Engineering Foundationと一致する。
- `Source`、`Related PRDs`、`Related ADRs` が維持されている。
- Not applicableの理由が明確である。
- 既存内容への意図しない変更がない。

内容の妥当性確認と `Accepted` 化は、別セッションの `document-review` に委譲する。

## 9. tasks.mdへ記録する

`tasks.md` のArchitecture Update欄だけを更新する。

- Status
- Scope
- Evidence
- Files Updated
- Inconsistencies
- Unreflected Items
- Human Decision

TaskのPurpose、Scope、追跡関係、実装結果、コードレビュー結果は変更しない。独立した更新レポートや詳細な会話ログは残さない。

## 10. 完了候補を提示する

<!-- 固定の定義または対応関係を1行で維持するため。 -->

初回作成または更新、調査根拠、変更ファイル、変更しなかったファイル、不一致、未反映事項、残存リスク、自己検証結果、レビュー対象を要約し、「architecture更新完了を確定するか」を人間へ確認する。

人間の明示確認後だけ、Architecture Update Statusを `Completed`、Human Decisionを `Approved` にする。architecture文書自体を `Accepted` にはしない。

承認されない場合は `In Review` または `Blocked` とし、理由と再開条件を記録する。

## 11. 後続工程へ引き渡す

変更した `docs/architecture/*.md` を、更新を担当したAIとは別セッションの `document-review` へ渡す。

- 初回作成時は6ファイルを一組でレビューする。
- 更新時は変更ファイルを対象とし、影響を受ける他ファイルとの整合性も確認させる。
- 対象architecture文書が人間により `Accepted` へ確定されるまで `release-check` へ進まない。

## 12. 完了を報告する

対象PRD、初回作成または更新、調査範囲、更新ファイル、不一致、未反映事項、残存リスク、`tasks.md` の記録状態、人間の確定結果、次の工程を簡潔に報告する。
