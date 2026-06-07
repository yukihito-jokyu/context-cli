---
name: create-tasks
description: Accepted済みのPRD、Backlog、Requirements、Architecture Change、必要なADRとEngineering Foundationから、実装可能なTask、Test Tasks、実装順序、変更予定ファイル、品質ゲートを整理し、同じPRDディレクトリのtasks.mdをDraftとして新規作成または更新する。実装計画を作るとき、要件とテストの追跡関係を確認するとき、既存Taskを分割・置換・再計画するときに使用する。
---

# Tasks作成

Accepted済みの要件と設計を、追跡可能で検証可能な実装計画へ変換する。

## 必須条件

- 呼び出し時にPRDの3桁連番ID（例: `001`）を受け取る。
- 対話と本文には原則として日本語を使う。識別子、パス、既存文書の表記は維持する。
- 実行時は `docs/AI-driven-development.md` を参照しない。本スキルの記載規則に従う。
- 内容の妥当性レビュー、StatusのAccepted化、実装は行わない。

## 1. 対象PRDを解決する

PRD IDから `docs/prds/prd-XXX-*/` を検索し、対象ディレクトリを一意に特定する。

次をすべて満たす場合だけ続行する。

- `prd.md`、`backlog.md`、`requirements.md`、`architecture-change.md` が存在する。
- 4成果物のStatusがすべて `Accepted` である。
- `architecture-change.md` にBlocking Open Questionがない。
- `docs/engineering/technology.md`、`structure.md`、`development-rules.md` が存在し、適用対象のStatusが `Accepted` である。
- 必要なADRがすべて存在し、Statusが `Accepted` または `Implemented` である。
- 既存 `tasks.md` を更新する場合、そのPRD IDが対象PRDと一致する。

1つでも満たさない場合は、計画を推測せず理由と戻るべき成果物を報告して停止する。

## 2. 資料とコードを読む

次の順で読む。

1. `docs/ai-driven-development/templates/tasks.md`
2. 対象のAccepted済み `prd.md`、`backlog.md`、`requirements.md`、`architecture-change.md`
3. `docs/engineering/*.md`
4. 関連するAcceptedまたはImplemented済みADR
5. 存在する場合は `docs/architecture/*.md`
6. 関連PRDのAccepted済み成果物
7. 既存 `tasks.md`
8. 変更箇所、依存関係、既存テスト、品質ゲート、実行コマンドを確認するために必要なコードと設定

コードや設定は計画の実現可能性と変更候補を確認するために調査する。実装、リファクタ、設定変更は行わない。

テンプレートが存在しない、または確定事項を表現できない場合は独自構成で生成せず、不足または不整合を報告して停止する。テンプレート自体は修正しない。

## 3. 適用フローとリスクを評価する

ユーザー影響、データ変更、認証・認可、課金、外部連携、影響範囲、仕様の明確さ、障害時の運用影響を評価し、`Standard / Lightweight / Hotfix` の推奨と根拠を提示する。適用フローは人間が確定する。

軽量フローでも `tasks.md` を作る場合は、このスキルの追跡性と状態管理の規則を適用する。テストファーストは、標準フローでは原則必須、軽量フローではリスクがある変更だけ必須とする。

## 4. 要件の追跡範囲を確定する

対象となる全Story、AC、TR、NFRを抽出し、少なくとも1つのTask、Test Task、Quality Gate、Release Checkのいずれかへ紐づける。

- ユーザーから観測できるACは、原則としてTest Taskで検証する。
- TRは実装Taskと必要最小限のテストへ紐づける。
- NFRは、実装が必要ならTaskへ、検証だけが必要ならTest Task、Quality Gate、Release Checkへ紐づける。
- 対応不要と判断した要件は省略せず、その理由を記載する。
- 孤立したStory、AC、TR、NFRを残さない。

## 5. Taskへ分割する

Taskは、単独で実装、検証、完了判定、差分レビューができる単位にする。固定の件数や行数では分割しない。

各Taskは次を満たす。

- 目的と完了条件が1つにまとまっている。
- 1つ以上のStoryまたはTechnical Storyへ紐づく。
- 対応するAC、TR、NFRとTest Tasksを特定できる。
- 依存関係と実装順序を明示できる。
- 新たな重要設計判断なしで実装できる。

複数Storyに共通する基盤作業は独立Taskにできる。その場合は関連する全Story IDと、独立Taskが必要な理由を記載する。Storyに紐づかない環境整備や内部改善が必要なら、上流のBacklogへ戻りTechnical Storyを追加する。実装都合だけの未紐づけTaskは作らない。

並行作業を予定する場合は、担当範囲、競合し得るファイルまたは設計領域、依存関係、統合順序を記載する。

## 6. Files to Changeを調査する

コード、設定、テスト構成から、各Taskの変更予定または調査対象を `Files to Change` に記載する。これは確定リストではない。

変更箇所を特定できない場合は単に `TBD` とせず、特定のための調査をTaskまたはTask内の作業として明示する。クラス、関数、逐次的な実装手順まで固定しない。

## 7. Test Tasksを設計する

Test TasksはTask単位ではなく、検証可能な振る舞い単位で作る。対象はACのユーザー視点の振る舞い、TRの制約、NFR、バグの再現条件、保持すべき既存挙動である。

ユーザー価値に近いE2E、integration、API、component interactionを優先し、内部制約は必要最小限のunit、contract、boundaryテストで補う。実装詳細だけを固定するテストは避ける。

各Test Taskに、関連するAC、TR、NFR、テスト種別、Test First、Red Check要否を記載する。テスト環境の制約で実行できない可能性がある場合は、代替検証と後続対応を計画する。

## 8. 未確定事項を分類する

未確定事項は次のいずれかに分類する。

- Blocking: 解決するまでAccepted化または実装開始へ進めない事項
- Deferred: Draft時点では保留でき、判断条件、判断工程、影響するTaskが明確な事項

Blockingが残っていてもDraftは生成できる。Assumptionsには人間が暫定的に承認した前提だけを記載する。

## 9. IDと既存内容を維持する

Task IDは `T-001`、Test Task IDは `TT-001` のようにPRD内の種類別通し番号とする。一度使用したIDは再利用、振り直し、削除しない。

不要になったTaskまたはTest Taskは残し、Statusを `Cancelled` または `Superseded` にして理由と置換先を記載する。分割・統合では新しいIDを発行し、旧IDから参照を残す。

既存ファイルの更新では今回の変更テーマに関係する箇所だけを変更し、整理目的の削除、言い換え、再構成をしない。資料間の矛盾を自動解消せず、人間が正本と修正対象を確定するまで停止する。

## 10. Draft内容を組み立てる

テンプレートの見出し構成と順序を維持し、独自見出しを追加しない。該当しない項目も削除せず、理由を添えて `Not applicable` と記載する。

- Status: 常に `Draft`
- Status Reason: 新規計画または再計画の理由と、軽レビューが必要な理由
- Risk Assessment: 推奨フロー、リスク、根拠、人間の確認状態
- Implementation Order: 依存関係を反映した順序
- Tasks: 目的、追跡関係、依存、変更候補、Test Tasks、完了条件
- TaskとTest Taskの状態履歴: Status、Status Reason、置換先
- Documentation Updates: 実装結果を反映する文書
  <!-- 固定の定義または対応関係を1行で維持するため。 -->

- Code Review、Architecture Update、Quality Gates、Release Check、Post-Release Verification: 実行結果を後から記録できる初期状態。Release Checkには標準チェック項目、総合結果、残存リスク、Follow-up、人間判断の記録先を含める

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->

- Follow-up Transfer: 初期値は `Not applicable`
- Assumptions: 人間が承認した前提だけ
- Open Questions: BlockingとDeferredに分類

## 11. 生成承認を得る

書き込み前に次を要約する。

- 対象PRD、適用フロー、リスク
- Task構成、実装順序、依存関係
- Story、AC、TR、NFRとTask・Test Taskの対応
- Test First方針と品質ゲート
- Files to Changeの確度と必要な調査
- BlockingとDeferred
- 既存Taskを変更、分割、取消、置換する箇所

「この内容でtasks.md Draftを生成するか」を確認し、人間の明示的な承認を得るまでファイルを書き換えない。

## 12. Draftを生成する

承認後、対象PRDディレクトリの `tasks.md` を新規作成または更新する。別の出力先は人間が明示した場合だけ許可する。

生成後に次を自己検証し、必要なら修正する。

- テンプレートの構成と順序を維持している。
- StatusがDraftであり、Status Reasonが今回の計画を説明している。
- 全Story、AC、TR、NFRに対応先がある。
- Taskが実装、検証、完了判定、差分レビュー可能な粒度である。
- 共通Taskが関連する全Storyと必要理由を持つ。
- 実装順序とDependenciesが矛盾していない。
- Test Tasksが検証可能な振る舞い単位である。
- テストがユーザー価値に近いものを優先している。
- Task別Code ReviewとPRD全体のCode Review Summaryが初期状態で存在する。
- Architecture Updateが初期状態で存在する。
- Files to Changeが過不足なく、確定リストとして扱われていない。
  <!-- 固定の定義または対応関係を1行で維持するため。 -->

- Documentation Updates、Quality Gates、Release Check、Post-Release Verificationの記録先があり、Release Checkの全標準項目を個別判定できる。

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->

- BlockingとDeferredが正しく分類されている。
- IDの再利用、削除、リンク切れ、孤立IDがない。
- 既存内容を意図せず変更していない。
- 上流成果物、Engineering Foundation、ADR、現在architecture、コード、設定との矛盾がない。

内容の妥当性レビューは行わない。

## 13. 完了を報告する

次を簡潔に報告する。

- 対象PRD IDと出力先
- 適用フローとリスク
- Task数、実装順序、主要な依存関係
- 要件とテストの対応状況
- BlockingとDeferred
- 取消、置換、調査Task
- 参照した資料
- 次工程として、対象ファイルを指定した別セッションの `document-review`（軽）が必要であること
- Blockingがある場合、Accepted化と実装開始へ進めないこと
