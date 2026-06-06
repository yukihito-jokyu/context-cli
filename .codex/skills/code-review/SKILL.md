---
name: code-review
description: 標準フローのAccepted済み成果物、軽量・緊急フローの変更記録、またはbootstrap記録とEngineering Foundationに照らして実装差分を独立レビューし、指摘確認、承認された修正、再検証、結果記録まで行う。PRD全体、高リスク変更、hotfix、bootstrap、修正後の再レビューに使用する。
---

# コードレビュー

実装AIから独立したレビュー状態で、今回の実装差分が要件、設計、技術規範を満たすかを確認し、人間が採用した指摘の修正と再検証まで行う。

## 必須条件

- `standard` ではPRDの3桁連番IDを必須入力とし、Task IDを任意入力とする。
- `lightweight / hotfix` ではPRD IDを要求せず、implementationで使用したIssue、PR説明、または作業記録を必須入力とする。
- `bootstrap` ではPRD IDを要求せず、`docs/engineering/bootstrap.md` を入力とする。
- 対話、指摘、完了報告には原則として日本語を使う。識別子、パス、コードの表記は維持する。
- 対象コードを実装したAIとは別のセッションで実行する。同一セッションの場合は原則停止する。
  <!-- 固定の定義または対応関係を1行で維持するため。 -->
  <!-- textlint-disable preset-ja-technical-writing/sentence-length,preset-ja-technical-writing/max-ten -->

- `$run-prd-workflow` 経由に限り、全Task完了後にコンテキスト圧縮を完了し、成果物、実装差分、テスト結果、レビュー基準だけを再読込して、実装時の推論、自己評価、会話要約を根拠にしない場合は、同一セッションを独立レビュー相当として許可する。圧縮前から連続してレビューする場合は停止する。

<!-- textlint-enable preset-ja-technical-writing/sentence-length,preset-ja-technical-writing/max-ten -->

- 実装AIの非公開の推論や自己評価を前提にせず、成果物、差分、テスト結果、コードから独立して判断する。
- 実行時は `docs/AI-driven-development.md` を参照しない。本スキルの記載規則に従う。
- 独立したレビューレポートファイルは作らない。標準では `tasks.md`、軽量・緊急では指定された変更記録、bootstrapでは `docs/engineering/bootstrap.md` へ結果を記録する。
- コミット、PR作成、push、デプロイ、本番操作は、人間の指示または事前承認なしに行わない。

標準フローではPRD全体、軽量フローでは高リスク変更、緊急フローとbootstrapでは全変更のコードレビューを必須とする。それ以外の軽量変更でもAIが実施を推奨できる。

## 1. 対象を解決する

`standard` ではPRD IDから `docs/prds/prd-XXX-*/` を検索し、対象ディレクトリを一意に特定する。

- PRD IDのみ: 対象PRDで今回実装された全Taskの差分をレビューする。
- PRD IDとTask ID: 指定Taskに関係する差分だけを限定レビューする。
- 限定レビューは高リスクTaskや途中確認に使い、それだけでPRD全体のコードレビュー完了とは扱わない。
- 指定Taskが存在しない、実装未着手、または差分を特定できない場合は理由を報告して停止する。

`lightweight / hotfix` では指定された変更記録とimplementation結果から対象差分を特定する。限定レビューではなく変更全体を対象とし、記録不足または差分を分離できない場合は停止する。

`bootstrap` では `docs/engineering/bootstrap.md` のActual Files ChangedとGit差分からbootstrap全体を対象にする。限定レビューだけで完了扱いにしない。

## 2. 入力と前提を検査する

`standard` では次を読む。

1. `docs/product.md`
2. `docs/engineering/technology.md`、`structure.md`、`development-rules.md`
3. 対象PRDの `prd.md`、`backlog.md`、`requirements.md`、`architecture-change.md`、`tasks.md`
4. 関連するAcceptedまたはImplemented済みADR
5. 存在する場合は `docs/architecture/*.md`
6. 対象差分、関連コード、設定、テスト、依存関係
7. 実行済みテスト、品質ゲート、未実行理由
8. Git状態、履歴、差分

`docs/engineering/*.md` は3ファイルを確認し、変更箇所に適用される規則をレビュー基準にする。

上流成果物またはEngineering Foundationが未Accepted、参照不能、相互矛盾している場合は推測で進めない。対象箇所、影響、正本の候補を提示して停止する。

初回PRDで `docs/architecture/*.md` が存在しない場合は、Requirements、ADR、`architecture-change.md` を現在の設計根拠とする。

<!-- 固定の定義または対応関係を1行で維持するため。 -->
<!-- textlint-disable preset-ja-technical-writing/sentence-length -->

`lightweight / hotfix` では、`docs/product.md`、Engineering Foundation、指定された変更記録、関連ADR、存在する場合は `docs/architecture/*.md`、対象差分、テスト結果、Git状態を読む。`Problem / Expected Outcome / Constraints / Risk Assessment`、hotfixでは `Rollback` が不足していれば停止する。

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->

<!-- 固定の定義または対応関係を1行で維持するため。 -->
<!-- textlint-disable preset-ja-technical-writing/sentence-length -->

`bootstrap` では、`docs/product.md`、Engineering Foundation、`docs/engineering/bootstrap.md`、関連ADR、対象差分、設定、依存関係、CI、検証結果、Git状態を読む。Engineering Foundationが未Accepted、bootstrap記録がCompletion CandidateまたはCompletedでない、または差分を分離できない場合は停止する。

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->

軽量・緊急モードではStory、AC、TR、NFRを `Problem / Expected Outcome / Constraints / 検証項目` へ読み替える。bootstrapではEngineering Foundation、bootstrap Scope、Verification Planをレビュー基準にする。

## 3. 差分基準を決定する

次の優先順で今回の実装差分を特定する。

1. 対象Taskの実装開始前として記録されたGit状態
2. 対象PRDまたはTaskに関連するコミット差分
3. `tasks.md` の `Actual Files Changed` と現在の作業ツリー
4. 人間が指定した基準コミットまたはブランチ

上位の方法で一意に決まらない場合だけ次の方法へ進む。最終的に特定できなければ、基準コミットを人間へ確認する。

- 既存の未コミット変更と今回の変更を区別する。
- 無関係な既存差分はレビュー対象から除外するが、今回の変更との相互作用は確認する。
- 差分を安全に分離できない場合はレビューを停止する。
- 生成物、lockfile、vendorコードも差分へ含めるが、生成元、依存変更、整合性を中心に確認する。

## 4. 開始概要を提示する

レビュー開始前に次を簡潔に提示する。

- 対象PRD、対象Task、全体レビューまたは限定レビュー
- 採用した差分基準と対象ファイル
- 関連Story、AC、TR、NFR、ADR
- 適用するEngineering Foundationの規則
- 実行済みテスト、品質ゲート、未実行項目
- 変更リスクと重点レビュー領域
- 既存差分との区別

読み取り専用の調査と検証は続けてよい。コードやテストの修正は、指摘一覧を提示した後に行う。

## 5. レビューする

少なくとも次を確認する。

- Story、AC、TR、NFRを実装が満たしているか
- ADR、`architecture-change.md`、`docs/architecture/*.md`と矛盾しないか
- `docs/engineering/*.md`の技術、配置、依存、実装、テスト、Git、運用規則に適合するか
- Scope外、不要、重複、意図しない変更がないか
- 公開API、互換性、データ、移行、設定、外部連携への影響が正しいか
- エラー処理、境界条件、再試行、整合性、並行性、復旧が妥当か
- テストがユーザー価値と主要な失敗・境界条件を検証しているか
- 回帰テストと品質ゲートに不足がないか
- 新規依存の目的、代替、保守状況、ライセンス、セキュリティ、サイズ、運用影響が確認されているか
- 既存の未コミット変更を破壊または混入していないか

認証・認可、個人情報、秘密情報、外部入力、ファイル操作、決済、公開APIを変更する場合は、脅威、悪用経路、データ露出、権限境界、対策、残存リスクを追加で確認する。

品質ゲートの失敗は今回の変更起因か既存問題かを分類する。品質ゲートを無断で緩和、無効化、除外してはならない。

## 6. 指摘を分類する

指摘は次の4段階にする。

- `Critical`: セキュリティ侵害、データ損失、重大な要件違反など。必ず修正する。
- `High`: AC未達、設計・ADR違反、重大な回帰、重要テスト不足など。原則修正する。
- `Medium`: 保守性、例外処理、限定的な不具合リスクなど。修正またはFollow-upへ送る。
- `Low`: 可読性、命名、軽微な改善。完了を妨げない。

各指摘に、対象箇所、根拠、影響、推奨対応を付ける。最初に全指摘の一覧を重要度順に提示し、人間が全体像を確認できるようにする。

`Critical / High`が未解決ならコードレビュー完了および後続工程への移行を禁止する。誤検知と確認できた場合だけ解除する。

## 7. 一問ずつ確認する

一覧提示後、判断が必要な指摘を重要度順に一問ずつ確認する。

- `Critical / High`: 一件ずつ必ず確認する。
- `Medium`: 共通原因または同じ修正方針ならまとめて確認できる。
- `Low`: 原則として一覧提示のみとし、完了を妨げない。
- 機械的な軽微修正は、一覧の確認後にまとめて修正できる。

各質問で推奨案と理由を提示する。人間の回答を修正承認として扱い、同じ内容の承認を重ねて求めない。

## 8. 承認された指摘を修正する

人間が採用した指摘について、コード、設定、テストと必要な `tasks.md` の記録を修正する。既存差分とScopeを保護し、指摘に必要な最小範囲へ限定する。

次は直接修正せず、レビューを停止して上流工程へ戻す。

- Scope、Story、AC、TR、NFRの変更
- ADRまたは重要な設計判断の変更
- `architecture-change.md`の設計方針の変更
- `docs/engineering/*.md`の規則変更
- Taskの目的、依存関係、追跡関係の変更
- 承認されていない新規依存、公開API、データ移行

矛盾箇所、必要な判断、影響する成果物、推奨する戻り先を提示する。必要な成果物が再レビューされるまで修正を続けない。

## 9. 再検証する

修正後に次を行う。

1. 指摘に直接対応するテストを実行する。
2. 影響範囲の回帰テストを実行する。
3. 適用されるビルド、Lint、型検査、セキュリティ検査をする。
4. 修正差分だけでなく今回の対象差分全体を再レビューする。
5. 新しい指摘または上流矛盾がないか確認する。

テストや品質ゲートを実行できない場合は、理由、代替検証、残存リスク、実行条件を記録する。検証不能が `Critical / High` の解消確認を妨げる場合は完了扱いにしない。

## 10. 結果を記録する

`standard` で、コードレビュー中に `tasks.md` で更新してよいのは次だけである。

- Code ReviewのStatus、Scope、Findings、Fixes、Verification、Remaining Risks、Human Decision
- Actual Files Changed
- Test Results、品質ゲート結果、未実行理由、代替検証
- 追加で判明したFollow-up
- 人間が明示確定した場合のTask StatusとStatus Reason

TaskのPurpose、Scope、Linked Stories、AC、TR、NFR、Dependencies、設計方針は変更しない。

限定レビューでは対象TaskのCode Reviewだけを更新し、PRD全体のCode Review Summaryを `Completed` にしない。全体レビューではTask別結果と全体結果を記録する。

<!-- 固定の定義または対応関係を1行で維持するため。 -->
<!-- textlint-disable preset-ja-technical-writing/sentence-length -->

`lightweight / hotfix` では指定された変更記録、`bootstrap` では `docs/engineering/bootstrap.md` に、Status、Scope、Findings、Fixes、Verification、Remaining Risks、Human Decision、実変更ファイル、Follow-upだけを記録する。

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->

独立したレビュー履歴や詳細な会話ログは残さず、重要な判断、解消結果、残存リスクだけを記録する。

## 11. 完了候補を提示する

次をすべて満たす場合だけコードレビュー完了候補を提示する。

- `Critical / High`が残っていない。
- 採用した`Medium`が解消済みである。
- 未対応項目がFollow-upまたは受容リスクに分類されている。
- 要件、ADR、設計、Engineering Foundationと整合している。
- 必要なテストと品質ゲートが成功しているか、許容可能な未実行理由が記録されている。
- 修正後の対象差分全体を再レビューしている。
- 既存差分との区別が維持されている。

次を要約し、「コードレビュー完了を確定するか」を人間へ確認する。

- 解消した指摘
- 残るMedium、Low、Follow-up、受容リスク
- 修正ファイルと既存差分
- テスト、品質ゲート、未実行理由
- 上流または後続成果物への影響
- 対象Taskまたは変更記録を `Completed` にできるか

人間の明示確認後だけ、Code Review Statusを `Completed`、Human Decisionを `Approved` にする。対象Taskまたは変更記録が完了条件を満たし、人間が同時に確定した場合だけStatusを `Completed` にできる。

承認されない場合はCode Review Statusを `Blocked` または `In Review` とし、理由と再開条件を記録する。

## 12. 後続工程へ引き渡す

`standard` のPRD全体コードレビュー完了後は、次の順に引き渡す。

1. 人間による `architecture-change.md` の `Implemented` 確定
2. architecture更新スキルによる `docs/architecture/*.md` 更新
3. 更新したarchitecture文書の `document-review`
4. `release-check`

`architecture-change.md`と実装の不一致を見つけた場合は、自動修正せず内容と影響を報告する。重要な設計変更なら上流へ戻し、実装結果の記録差だけならimplementationまたは対応する更新工程へ委譲する。

`lightweight / hotfix` は `release-check` へ引き渡す。`bootstrap` は `create-engineering-foundation` のauditモードへ引き渡し、通常のrelease-checkは行わない。

## 13. 完了を報告する

次を簡潔に報告する。

- 対象モード、PRD・Taskまたは変更記録
- 採用した差分基準
- 指摘件数と解消結果
- 修正ファイルと既存差分
- テスト、品質ゲート、未実行理由
- Follow-up、受容リスク、Blocked
- 対象記録の状態と人間の確定結果
- 上流へ戻した事項
- 次の工程
