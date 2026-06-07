---
name: implementation
description: 標準フローではAccepted済みPRD成果物とtasks.md、軽量・緊急フローでは指定されたIssue・PR説明・作業記録に従い、変更単位でテスト、実装、検証、自己レビュー、進捗記録を行う。実装を開始または再開するとき、特定Taskだけを実装するとき、Blockedになった作業を解除後に再開するときに使用する。
---

# 実装

Accepted済みの意図、要件、設計、技術規範を変更せず、Task単位でコードへ反映する。

## 必須条件

- 呼び出し時に実行モードを確定する。
  - `standard`: PRDの3桁連番IDを必須入力とし、Task IDは任意とする。
  - `lightweight / hotfix`: PRD IDを要求せず、人間が指定したIssue、PR説明、またはリポジトリ内の作業記録パスを必須入力とする。
- 標準入力と軽量・緊急入力を混在させない。入力が曖昧なら開始前に人間へ確認する。
- 対話、進捗記録、報告には原則として日本語を使う。識別子、パス、コードの表記は維持する。
- 実行時は `docs/AI-driven-development.md` を参照しない。本スキルの記載規則に従う。
- 上流成果物の要件、Scope、設計判断を実装中に補完または変更しない。
- `docs/architecture/*.md` は更新しない。実装後にarchitecture更新スキルへ委譲する。
- コミット、PR作成、push、デプロイ、本番操作は、人間の指示または事前承認なしに行わない。

## 1. 対象と変更単位を解決する

`standard` では、PRD IDから `docs/prds/prd-XXX-*/` を検索し、対象ディレクトリを一意に特定する。

Task IDが指定された場合は、そのTaskだけを対象とする。未指定の場合は `Implementation Order` と依存関係に従い、依存Taskが `Completed` の最初の `Pending` Taskを選ぶ。

- `Completed / Cancelled / Superseded` は対象外とする。
- `Completion Candidate` は未完了として扱い、依存Taskの開始条件を満たさない。
- 同順位で複数の実行可能Taskがある場合は、候補、依存、リスク、推奨理由を提示し、人間が対象を確定する。
- 指定Taskが開始不能な場合は、理由と先に必要な対応を報告して停止する。

<!-- 固定の定義または対応関係を1行で維持するため。 -->

`lightweight / hotfix` では、指定された変更記録から `Problem / Expected Outcome / Constraints / Risk Assessment` を読み、1つの検証可能な変更単位を対象にする。`hotfix` では `Rollback` も必須とする。不足項目がある場合は実装せず、記録の補完を求める。以下の「Task」は、軽量・緊急モードではこの変更単位を指す。

軽量・緊急モードではStory、AC、TR、NFRを必須にせず、それぞれユーザー影響、Expected Outcome、Constraints、リスクに応じた品質・運用条件へ読み替える。

## 2. Definition of Readyを検査する

`standard` では、次をすべて満たす場合だけ実装へ進む。

- `prd.md`、`backlog.md`、`requirements.md`、`architecture-change.md`、`tasks.md` が存在し、すべて `Accepted` である。
- `docs/engineering/technology.md`、`structure.md`、`development-rules.md` が存在し、適用対象が `Accepted` である。
- 必要なADRが存在し、`Accepted` または `Implemented` である。
- Blocking Open Questionがない。
- 対象Taskの依存Taskがすべて `Completed` である。
- 必要な権限、開発環境、テスト環境が利用可能である。
- 上流成果物、技術規範、現在のコードと設定に未解決の矛盾がない。

1つでも満たさない場合は推測で進めず、該当箇所、影響、戻るべき成果物または解除条件を報告して停止する。

`lightweight / hotfix` では、次を満たす場合だけ進む。

- 変更記録に必須項目があり、人間が適用フローを確定している。
- `docs/engineering/*.md` が存在し、適用対象が `Accepted` である。
- 必要なADRが `Accepted` または `Implemented` である。
- Blockingな未確定事項がない。
- 必要な権限、開発環境、テスト環境が利用可能である。
- 正本、技術規範、現在のコードと設定に未解決の矛盾がない。

## 3. 正本と作業状態を読む

`standard` では次の順で読む。

1. `docs/product.md`
2. `docs/engineering/*.md`
3. 対象PRDの `prd.md`、`backlog.md`、`requirements.md`、`architecture-change.md`、`tasks.md`
4. 関連するAcceptedまたはImplemented済みADR
5. 存在する場合は `docs/architecture/*.md`
6. 対象Taskに関係するコード、設定、テスト、依存関係、品質ゲート
7. Git差分と作業ツリーの状態

初回PRDで `docs/architecture/*.md` が存在しない場合は、Requirements、ADR、`architecture-change.md` を設計根拠とする。

<!-- 固定の定義または対応関係を1行で維持するため。 -->

`lightweight / hotfix` では、`docs/product.md`、`docs/engineering/*.md`、指定された変更記録、関連ADR、存在する場合は `docs/architecture/*.md`、関連コード・設定・テスト、Git差分の順に読む。

セッション再開時も正本と現在の差分を読み直し、会話履歴や一時要約だけを根拠に再開しない。

## 4. 開始前の概要と承認境界

開始前に、対象Task、関連Story・AC・TR・NFR、変更予定または調査対象、Test Tasks、品質ゲート、リスク、既存差分との関係を簡潔に提示する。

Definition of Readyを満たす通常Taskは個別承認なしで開始できる。ただし、次は影響、選択肢、復旧方法を提示し、人間の事前承認を得る。

- データ移行、データ削除、破壊的変更
- 認証・認可、権限、課金、秘密情報
- 本番操作、デプロイ、外部公開
- 新しい外部依存、ライブラリ、外部サービス
- 重要な外部仕様またはセキュリティ・プライバシー境界

承認済みScopeを超える操作が必要になった場合は停止する。

## 5. 既存差分を保護する

- 既存の未コミット変更を削除、上書き、巻き戻し、無関係に整形しない。
- 対象Taskと無関係な差分は残したまま作業する。
- 同じファイルでも編集箇所を安全に分離できる場合は、既存変更を維持して実装する。
- 競合する、意図を判別できない、または安全に分離できない場合は停止して人間へ確認する。
- テストは既存差分を含む作業ツリーで実行したことを記録する。
- 完了報告では今回の変更と既存差分を区別する。

## 6. Task Statusを更新する

`standard` では対象Taskの開始時に `Pending` から `In Progress` へ変更し、`Status Reason` に開始理由を記録する。

状態遷移は次に限定する。

```text
Pending -> In Progress
In Progress -> Blocked
In Progress -> Completion Candidate
Completion Candidate -> Completed
Pending / In Progress -> Cancelled | Superseded
```

- `Completion Candidate` は、実装、テスト、自己レビューが完了し、AIが完了根拠を提示できる状態である。
- `Completed` は人間が最終確定した後だけ設定する。
- `Cancelled / Superseded` は上流成果物の更新と必要な再レビュー後だけ設定する。
- すべての遷移で `Status Reason` を更新する。

<!-- 固定の定義または対応関係を1行で維持するため。 -->

`lightweight / hotfix` では、変更記録に `In Progress / Blocked / Completion Candidate / Completed` の状態、理由、実変更ファイル、テスト結果、未実行理由、Follow-upを記録する。IssueまたはPR説明を直接更新できない場合は、完了報告で追記内容を提示し、人間が記録するまで完了扱いにしない。

## 7. Test Firstで実装する

標準フローでは原則テストファーストとし、軽量フローではリスクがある変更だけ必須とする。厳密なTDDを全Taskへ強制しない。

各Taskで次を行う。

1. Test Tasksと対象AC・TR・NFRを確認する。
2. 検証可能な振る舞い単位でテストを書く。
3. Red Checkが必要かつ可能なら、対象仕様の未実装を理由に失敗することを確認する。
4. TaskのScope内で最小限の実装する。
5. 対象テストを実行し、Greenを確認する。
6. 必要ならTask範囲内でリファクタする。
7. 関連する回帰テストと品質ゲートを実行する。
8. 差分を自己レビューする。
9. `tasks.md` の許可範囲だけを更新する。

ユーザー価値に近いE2E、integration、API、component interactionを優先し、内部制約は必要最小限のunit、contract、boundaryテストで補う。

<!-- 固定の定義または対応関係を1行で維持するため。 -->
<!-- textlint-disable preset-ja-technical-writing/no-doubled-conjunction -->

テストを実行できない場合は、未実行理由、代替検証、残存リスク、後で実行または追加すべきテストを記録する。品質ゲートを緩和または無効化してはならない。

<!-- textlint-enable preset-ja-technical-writing/no-doubled-conjunction -->

## 8. Scopeと判断の逸脱を止める

次を発見した場合は実装を停止する。

- `docs/engineering/*.md`、Requirements、ADR、`architecture-change.md`、`tasks.md`、コード間の矛盾
- AC、TR、NFR、Scope、設計方針、公開APIを変更する必要
- 新しい重要設計判断またはADR候補
- 承認されていない新規依存、危険な回避策、品質ゲート緩和

矛盾箇所、影響範囲、正本の推奨候補、戻るべき成果物を提示し、人間が修正方針を確定する。必要に応じて `grill-me` と `document-review` を実行し、必要な成果物が再びAcceptedになってから再開する。

新しい要求は、不具合修正、要件の明確化、新規Scopeに分類する。新規Scopeは原則としてFollow-up Storyまたは別PRD候補へ分離する。

## 9. 失敗とBlockedを管理する

失敗時は原因を分析し、安全かつScope内の別手段があれば再試行する。同じ原因で3回失敗した場合は原則停止する。

次は回数を待たず `Blocked` とする。

- 権限、環境、外部サービスの不足または障害
- 上流判断、仕様、設計の不足
- 安全に解消できない既存差分との競合

`Status Reason` とImplementation Notesへ、原因、試行内容、結果、解除条件、次の候補を記録する。Scope拡大、品質低下、危険な回避策で突破しない。依存Taskへは進まず、独立Taskを継続できる場合だけ候補として提示する。

## 10. 記録可能な範囲を守る

`standard` で、Accepted済み `tasks.md` のStatusをDraftへ戻さず更新してよいのは次だけである。

- Task StatusとStatus Reason
- Actual Files Changed
- Test Taskの実行状態、Red Check、Green Check
- Test Results、未実行理由、代替検証
- 追加で判明したFollow-up

TaskのPurpose、Scope、Linked Stories、AC、TR、NFR、Dependencies、設計方針は直接変更しない。変更が必要なら上流成果物へ戻る。

`lightweight / hotfix` では指定された変更記録の状態、実変更ファイル、テスト結果、未実行理由、代替検証、Follow-up、hotfixのRollback結果だけを更新する。Problem、Expected Outcome、Constraints、Risk Assessment、Rollback方針を変える場合は実装を止め、人間と変更記録を再確定する。

## 11. Taskを自己レビューする

各Taskの実装後に、次を確認する。

- Story、AC、TR、NFR、ADR、`architecture-change.md`への適合
- 不要な変更、Scope外変更、既存挙動の破壊
- セキュリティ、プライバシー、データ整合性、エラー処理
- テスト不足、回帰不足、未実行項目
- `docs/engineering/*.md`への適合
- 既存差分の保持

問題がなければTaskを `Completion Candidate` とし、差分、テスト結果、未実行項目、残存リスク、関連Story・AC・TRを提示する。人間が明示的に確定した場合だけ `Completed` にする。

## 12. 次のTaskへ進む

この節のTask選択と依存関係は `standard` だけに適用する。

- Task ID指定時は、そのTaskの完了候補提示またはBlocked記録で終了する。
- Task未指定時は、停止条件に該当しない限り、次の独立した実行可能Taskへ進む。
- `Completion Candidate` に依存するTaskへは進まない。
- 高リスクTask、重要な統合境界、上流矛盾、想定外変更では停止し、人間確認を得る。
- セッション中断時は、進捗、テスト結果、未解決事項、次のTaskを `tasks.md` の許可範囲へ記録する。

`lightweight / hotfix` は1つの変更単位の完了候補提示またはBlocked記録で終了する。

## 13. 実装結果を設計へ反映する

この工程は `standard` だけで行う。全実装Taskが `Completion Candidate` または `Completed` になった時点で、`architecture-change.md` を設計案から実装結果の確定候補へ更新する。

- 各Task完了時は設計案との差異を収集し、全Task完了候補時にまとめて反映する。
- 実装結果に合わせて既存セクションを更新し、独自見出しを追加しない。
- 重要な設計判断が変わった場合は更新で正当化せず、Requirements、ADR、`architecture-change.md` の再検討へ戻る。
- 内容更新後はStatusを `Review` とし、Status Reasonに実装結果の反映と確認が必要な理由を記載する。
- `Implemented` への変更は提案だけを行い、人間の最終確定に委ねる。

`docs/architecture/*.md` の現在状態更新はarchitecture更新スキルへ委譲する。

## 14. コードレビューへ引き渡す

作成AIとは別のレビューAIによるコードレビューへ、次を渡す。

- 対象PRDとTask一覧
- 今回の変更差分と既存差分の区別
- 関連Story、AC、TR、NFR、ADR
- 実行したテストと品質ゲート
- 未実行項目、代替検証、残存リスク
- Follow-upとBlocked
- 更新した `tasks.md` と `architecture-change.md`

`lightweight / hotfix` では、最後の項目を指定された変更記録へ読み替える。高リスク変更とすべてのhotfixはコードレビューを必須とする。

`standard` では、コードレビュー完了後に人間が `architecture-change.md` の `Implemented` 化を確定する。その後、architecture更新スキル、更新した `docs/architecture/*.md` のdocument-review、release-checkの順に進む。`lightweight / hotfix` はコードレビュー後にrelease-checkへ進む。

## 15. 完了を報告する

次を簡潔に報告する。

- 対象モード、PRD・Taskまたは変更記録
- Statusと人間の確定待ち
- 変更ファイルと既存差分
- テスト、品質ゲート、未実行理由
- Blocked、Follow-up、残存リスク
- `architecture-change.md` の更新状態。軽量・緊急ではNot applicable
- コードレビューへ渡す情報
- コミット可能な単位と、コミット未実施であること
