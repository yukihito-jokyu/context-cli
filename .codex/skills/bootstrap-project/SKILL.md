---
name: bootstrap-project
description: Accepted済みEngineering Foundationと必要なADRに従い、プロジェクト開始時の初期ディレクトリ、依存関係、開発・テスト・Lint・型検査・CIの最小基盤を一度だけ計画・実装し、docs/engineering/bootstrap.mdへ結果を記録する。最初のPRD前の環境構築、未完了bootstrapの再開、失敗後の再実行に使用する。
---

# Bootstrap Project

Engineering Foundationを実行可能な開発基盤へ反映する、1回限りの独立工程を担当する。

## 必須条件

- PRD IDや通常の `tasks.md` を入力にしない。
- `docs/engineering/technology.md`、`structure.md`、`development-rules.md` が存在し、すべて `Accepted` であることを必須とする。
- BlockingなADR候補がなく、必要なADRが `Accepted` または `Implemented` であることを必須とする。
- `docs/AI-driven-development.md` は参照せず、このスキルの規則に従う。
- コミット、push、PR作成、デプロイ、本番・共有環境操作は、人間の明示指示なしに行わない。

## 1. 実行可否を確認する

次を読む。

1. `docs/ai-driven-development/templates/engineering/bootstrap.md`
2. `docs/product.md`
3. `docs/engineering/technology.md`
4. `docs/engineering/structure.md`
5. `docs/engineering/development-rules.md`
6. `docs/engineering/adr-candidates.md`
7. 関連するAcceptedまたはImplemented済みADR
8. 既存の `docs/engineering/bootstrap.md`
9. 設定、依存関係定義、CI、既存ディレクトリ、Git差分

既存のbootstrap記録が `Completed` の場合は原則停止する。追加の基盤変更は標準、軽量、または緊急変更フローへ委譲する。前回が `Blocked` または `In Progress` の場合だけ解除条件と現在差分を確認して再開する。

## 2. 最小計画を作る

対象は最初のPRDを安全に開始するための共通基盤に限定する。

- 初期ディレクトリと必須ファイル
- パッケージ管理と固定された依存関係
- 開発、ビルド、テスト、Lint、型検査の実行経路
- 最小CI
- 必須の設定例と秘密情報の除外

機能コード、機能固有のDB・API・UI、将来予測による抽象化、デプロイ、本番環境構築は含めない。

変更予定ファイル、実行順序、検証、リスク、外部操作、ロールバック方法を提示する。依存追加、外部接続、破壊的操作は人間の事前承認を得る。

## 3. bootstrap.mdを開始する

人間の実行承認後、テンプレートに従って `docs/engineering/bootstrap.md` を作成または更新し、Statusを `In Progress` にする。

記録する内容は次の通り。

- Source Engineering FoundationとADR
- ScopeとOut of Scope
- Planned Changes
- Verification Plan
- RiskとRollback
- Actual Files Changed
- Verification Results
- Not Run Reasons
- Follow-up

## 4. 実装・検証する

計画した順に最小範囲を実装する。既存差分を削除、巻き戻し、無関係に整形しない。

各変更後に、適用可能なセットアップ確認、ビルド、テスト、Lint、型検査、CI設定検証を行う。実行できない検証は理由、代替検証、残存リスクを記録する。品質ゲートを緩和して通過させない。

Engineering Foundation、ADR、コード、設定の矛盾や新しい重要判断を発見した場合は停止し、`Blocked`、影響、戻るべき成果物、解除条件を記録する。

## 5. 完了候補を提示する

実装と検証が完了したらStatusを `Completion Candidate` とし、変更ファイル、検証結果、未実行項目、残存リスク、Follow-upを提示する。このスキル自身は `Completed` にしない。

次に別セッションの `code-review` をbootstrapモードで実行する。レビューがApprovedとなり、人間が最終確定した場合だけ `bootstrap.md` を `Completed` にする。その後、`create-engineering-foundation` のauditモードで文書、設定、コード、CIの整合性を確認する。

## 6. 完了を報告する

次を簡潔に報告する。

- Statusと人間の確定結果
- 実変更ファイル
- 実行した検証と未実行理由
- Blocked、残存リスク、Follow-up
- code-reviewとauditへの引き渡し
