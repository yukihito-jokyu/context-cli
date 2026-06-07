---
name: release-check
description: 標準フローでは実装、コードレビュー、architecture更新が完了したPRD、軽量・緊急フローではレビュー済み変更記録について、差分、テスト結果、運用準備を照合し、リリース可否候補を判定する。最終ゲート、Blocked修正後、軽量変更、hotfixの確認に使用する。
---

# リリースチェック

既存成果物、PR差分、検証結果を照合し、リリースを妨げる問題がないか判定する。コード、成果物、設定を修正せず、問題は責務を持つ工程へ戻す。

## 必須条件

- `standard` ではPRDの3桁連番IDを必須入力とする。
- `lightweight / hotfix` ではPRD IDを要求せず、implementationとcode-reviewで使用したIssue、PR説明、または作業記録を必須入力とする。
- 対話、記録、完了報告には原則として日本語を使う。識別子、パス、コード表記は維持する。
- 実行時は `docs/AI-driven-development.md` を参照しない。本スキルの記載規則に従う。
- 独立したリリースチェック文書を作らない。標準では対象PRDの `tasks.md`、軽量・緊急では指定された変更記録に結果を記録する。
- コード、テスト、設定、成果物を修正しない。
- リリース、デプロイ、本番操作、コミット、PR作成、pushを行わない。

## 1. 対象を解決する

`standard` ではPRD IDから `docs/prds/prd-XXX-*/` を検索し、対象ディレクトリを一意に特定する。対象が存在しない、複数存在する、またはIDと文書内のPRD IDが一致しない場合は停止する。

原則として対象PRDに対応するPR差分全体をチェック範囲とする。限定的なTaskやファイルだけを確認してPRD全体を `Pass` にしてはならない。

`lightweight / hotfix` では指定された変更記録に対応する差分全体を対象とする。記録と差分を一意に対応付けられない場合は停止する。

## 2. 開始条件を検査する

`standard` では次をすべて満たす場合だけチェックへ進む。

- 全実装Taskが `Completed` である。
- PRD全体のCode Review Summaryが `Completed` かつHuman Decisionが `Approved` である。
- `Critical / High` の未解決指摘がない。
- `architecture-change.md` が人間により `Implemented` へ確定されている。
- Architecture Updateが `Completed` かつHuman Decisionが `Approved` である。
- 新規作成または変更された `docs/architecture/*.md` が人間により `Accepted` へ確定されている。
- 必要なADRが `Accepted` または `Implemented` である。
- Blocking Open Questionがない。

不足時はチェック結果を作らず、不足条件、影響、戻るべき工程を提示して停止する。

<!-- 固定の定義または対応関係を1行で維持するため。 -->

`lightweight / hotfix` では、実装がCompletion CandidateまたはCompleted、必要なコードレビューがApproved、`Critical / High` が未解決でない、必要な文書更新とADRが完了、Blocking事項がない場合だけ進む。hotfixではRollbackとリリース後監視の準備も必須とする。

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->

## 3. 入力を読む

`standard` では次を読む。

1. `docs/product.md`
2. `docs/engineering/technology.md`、`structure.md`、`development-rules.md`
3. 対象PRDの `prd.md`、`backlog.md`、`requirements.md`、`architecture-change.md`、`tasks.md`
4. 関連するAcceptedまたはImplemented済みADR
5. 関連するAccepted済み `docs/architecture/*.md`
6. `tasks.md` の実変更ファイル、テスト結果、品質ゲート、コードレビュー、Architecture Update、未実行理由、Follow-up
7. 今回のPR差分またはPRDに対応するGit差分
8. 関連するコード、テスト、設定、依存関係、スキーマ、デプロイ・運用設定

会話履歴、一時的な要約、作成AIの自己評価を正本にしない。

<!-- 固定の定義または対応関係を1行で維持するため。 -->

`lightweight / hotfix` では、`docs/product.md`、Engineering Foundation、指定された変更記録、関連ADR、関連するarchitecture文書、差分、コード、テスト、設定、運用情報を読む。

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->

## 4. 調査範囲を決める

<!-- 固定の定義または対応関係を1行で維持するため。 -->

`standard` では `tasks.md`、`lightweight / hotfix` では指定された変更記録のActual Files ChangedとGit差分を起点に、リリース可否を判断するために必要な範囲を追跡する。

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->

- 関連コード、テスト、設定、依存関係を確認する。
- 必要に応じてDB変更、外部サービス、環境変数、権限、デプロイ、監視、ログ、運用手順まで確認する。
- Story、AC、TR、NFR、ADRと実装・テストの対応を確認する。
- 今回変更していない領域の全面監査は行わない。
- 無関係な既存問題は今回のリリースを妨げるか判定し、妨げない場合はFollow-up候補として分離する。
- 証跡が確認できない項目を推測で `Pass` にしない。

## 5. テスト結果を検証する

既存のテスト結果、実行時点、対象差分、未実行理由を最初に確認する。全テストを機械的に再実行せず、次の場合だけ関連テストまたは品質ゲートを再実行する。

- 最終テスト後にコード、テスト、設定が変更されている。
- 結果が古い、対象範囲が不明、または証跡が不足している。
- セキュリティ、データ移行、認証・認可など高リスク領域を変更している。
- 統合状態でのみ確認できる項目がある。

実行不能なら理由、代替検証、リリースへの影響を記録する。リリース判断に必要な検証を満たせない場合は `Blocked` とする。

## 6. チェック項目を判定する

次を一項目ずつ `Pass / Blocked / Not applicable` で判定し、根拠と未解決事項を記録する。

1. PRDのGoal、Story、ACが実装とテストで満たされ、Success Metricsを観測できる。
2. TR、NFR、ADRの制約に違反していない。
3. Engineering Foundationの適用規則に違反せず、未承認の例外がない。
4. 必要なテストと回帰確認が完了し、未実行項目の理由と対応が妥当である。
5. データ移行、設定、環境変数、外部サービス、権限、運用手順への影響が確認されている。
6. 変更リスクに応じた監視、ログ、切り戻しまたは復旧方法が確認されている。
7. `architecture-change.md`、`docs/architecture/*.md`、関連ADRが実装結果と一致している。
8. 未解決事項がリリース阻害または後続対応へ分類されている。
9. 新規依存、外部由来コード、ライセンス条件が確認されている。
10. 適用されるセキュリティ、プライバシー、アクセシビリティ、法令、契約、組織ポリシーが確認されている。
11. Story、AC、TR、Task、Test Taskのリンク切れ、孤立ID、対応漏れがない。

<!-- 固定の定義または対応関係を1行で維持するため。 -->

`lightweight / hotfix` では、PRD固有のStory、AC、TR、Task、Test Taskを `Problem / Expected Outcome / Constraints / 検証項目 / 変更単位` に読み替える。存在しないPRD成果物を理由に `Blocked` にしないが、正本に必要な情報がなければ `Blocked` とする。

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->

1つでも未解決の `Blocked` があれば総合結果を `Blocked` とする。すべての適用項目が `Pass` で、非適用項目に妥当な理由がある場合だけ総合結果を `Pass` 候補とする。標準フロー全体を `Not applicable` にはしない。

## 7. Blockedを処理する

このスキル自身では問題を修正しない。問題、影響、根拠、推奨する戻り先、再開条件を提示する。

- 実装またはテスト不足: `implementation` または `code-review`
- 現在状態のarchitecture文書不備: `update-architecture` と必要な `document-review`
- PRD、Requirements、設計、ADR、Engineering Foundationの問題: 対応する作成・レビュー工程
- リリース・運用準備の不足: 担当Taskの再開または追加Taskの確定

修正と必要な再レビューが完了した後、PR差分全体を対象にこのスキルを再実行する。

## 8. 人間の判断を得る

全項目の判定、根拠、残存リスク、Follow-up候補、総合結果候補をまとめる。`Blocked` または人間の判断が必要な項目だけを一問ずつ確認する。

人間がリリースを妨げないリスクを受容する場合は、理由、影響、対応期限または条件、移管先を記録する。セキュリティ、データ損失、法令・契約違反の重大な懸念は、単なるリスク受容で `Pass` に変更しない。

AIがリリース可否候補を提示し、「この判定でリリース可否を確定するか」を確認する。人間の明示確認後だけHuman Decisionを `Approved` または `Rejected` にする。

## 9. tasks.mdへ記録する

`standard` では `tasks.md` のRelease Check欄だけを更新する。

- Scope
- Checked At
- 各チェック項目のResult、Evidence、Open Item
- Overall Result
- Remaining Risks
- Follow-up
- Human Decision

総合結果が `Blocked` の場合は、戻り先と再開条件も記録する。他のTask、実装結果、コードレビュー、Architecture Updateの記録を変更しない。

`lightweight / hotfix` では指定された変更記録へ同じ項目を記録する。IssueまたはPR説明を直接更新できない場合は追記内容を提示し、人間による記録を確認するまで承認済みと扱わない。

## 10. リリース後検証へ引き渡す

`Pass` が承認された場合、リリース後に確認する主要AC、Success Metrics、監視、ログ、エラー率、移行結果を提示する。切り戻し・復旧条件と手順を確認し、リスクに対して不足する場合は `Blocked` とする。

このスキルはリリース実行とリリース後検証を担当しない。リリース後の結果は、標準では `tasks.md` のPost-Release Verification、軽量・緊急では指定された変更記録へ追記する。継続対応はIssueまたは次のPRD候補へ移管する。

リリース後検証とFollow-up移管が完了した時点で、AIがプロセス完了候補を提示し、人間が最終確定する。

## 11. 完了を報告する

対象モード、PRDまたは変更記録、チェック範囲、再実行した検証、各項目の判定、総合結果、Blocked、残存リスク、人間の判断、リリース後検証項目、次の工程を簡潔に報告する。
