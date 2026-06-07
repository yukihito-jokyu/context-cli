---
name: create-requirements
description: PRD連番IDからAccepted済みPRDとBacklogを特定し、StoryごとのAcceptance CriteriaとTechnical Requirements、PRD共通のNon-Functional Requirementsを整理して、同じPRDディレクトリのrequirements.mdをDraftとして新規作成または更新する。Backlogから要件を初めて作るとき、PRD・Backlog変更を既存要件へ反映するとき、例外・境界条件・技術制約・非機能要件を見直すときに使用する。
---

# Requirements作成

Accepted済みPRDとBacklogから、設計と検証へ渡せる要件を抽出し、`requirements.md` のDraftを生成する。

## 必須条件

- 呼び出し時にPRDの3桁連番IDを受け取る。例: `001`
- 対話と本文には原則として日本語を使う。識別子、パス、既存文書の表記は維持する。
- 実行時は `docs/AI-driven-development.md` を参照しない。本スキルの記載規則に従う。
- 資料から判明する内容は質問せず、要件を確定するために不足する情報だけを一問ずつ確認する。
- 内容の妥当性レビューとAccepted化は行わず、別セッションの `document-review` へ委ねる。

## 1. 対象成果物を特定する

入力されたIDを `XXX` とし、`docs/prds/prd-XXX-*/prd.md` を検索する。

- 該当が1つの場合: 対象PRDとして使う。
- 該当がない場合: 検索結果を報告して停止する。
- 複数該当する場合: 候補を列挙し、重複を自動解消せず停止する。

対象PRDと同じディレクトリの `backlog.md` を特定する。PRDとBacklogの両方が `Accepted` の場合だけ続行し、片方でも存在しない、参照不能、または未Acceptedなら理由を報告して停止する。

出力先は同じディレクトリの `requirements.md` に固定する。

## 2. 資料を読む

次の順で読む。

1. `docs/ai-driven-development/templates/requirements.md`
2. 対象の `prd.md`
3. 対象の `backlog.md`
4. 更新時は既存の `requirements.md`
5. 要件に関係する `docs/engineering/*.md`
6. PRD、Backlog、既存Requirementsから直接参照されるAccepted成果物とADR
7. 既存インターフェースや制約の事実確認が必要な場合だけコードと設定

テンプレートが存在しない、または確定内容を表現できない場合は独自構成で生成せず、不足または不整合を報告して停止する。テンプレート自体は修正しない。

<!-- 固定の定義または対応関係を1行で維持するため。 -->

PRDのGoal、Problem、Scope、Constraintsと、Backlogの全Story、Priority、Dependencies、Technical StoryのSupportsとReasonを抽出する。資料間の矛盾は自動解消せず、箇所と要件への影響を示して人間の判断を得る。

## 3. Storyごとの要件を抽出する

Backlogにある全Storyを `Story Requirements` へ記載する。実装対象のStoryだけに限定しない。

### Acceptance Criteria

- Storyを満たすための、独立して判定可能なユーザー視点の振る舞いごとに分ける。
- 正常系に加え、ユーザーが認識できる主要な失敗と境界条件を含める。
- Storyの成果やユーザー体験に影響する入力不正、権限不足、外部依存失敗などを含める。
- 発生可能性と影響がともに小さい例外は網羅せず、リスクに応じて選定する。
- 自然文を標準とし、複雑な状態遷移だけGiven / When / Thenを使う。
- 技術的な検証条件、設計判断、実装方式を含めない。

Technical Storyではユーザー視点のACを無理に作らない。利用者または運用者から観測できる完成条件がある場合だけACを作り、なければ `Not applicable` とする。

### Technical Requirements

- Story固有で、満たさなければAC達成またはTechnical Storyの完了を保証できない技術的制約・保証条件だけを記載する。
- 内部的な復旧、整合性、再試行、ログなど、Story固有の保証を必要に応じて含める。
- 技術選定、具体的な実装方法、設計判断を含めない。
- 複数Storyに共通する性能、セキュリティ、可用性、運用などの制約はNFRへ移す。

Technical StoryにはTRを必須とし、Backlogの対応User Story IDと必要理由を要件内容から追跡できるよう維持する。

## 4. Non-Functional Requirementsを抽出する

NFRはPRD全体へ適用する共通要件として記載する。対象を限定する場合だけ `Applies To` にStory IDを列挙する。

性能、セキュリティ、可用性、データ整合性、アクセシビリティ、法令・契約・組織ポリシー、運用、監視などの適用可能性を確認する。該当しないカテゴリは削除せず `Not applicable` と明記する。

数値や基準を根拠なく補完しない。人間判断が必要な未確定値は `Open Questions` に置く。法的判断や専門判断をAIが確定しない。

## 5. IDを管理する

IDは種類ごとにPRD内で一意な通し番号にする。

- Acceptance Criteria: `AC-001`
- Technical Requirements: `TR-001`
- Non-Functional Requirements: `NFR-001`

Storyごとに番号をリセットしない。一度使用したIDは再利用、振り直し、削除しない。

不要になった要件は元のStory配下またはNFR一覧に残し、IDの後ろへ `[Removed]`、`[Cancelled]`、または `[Superseded]` を付ける。その直下に理由と、存在する場合は置換先IDを記載する。分割・統合では新しいIDを発行し、旧IDから新IDへの参照を残す。

## 6. ADR候補を抽出する

TRまたはNFRを満たすために重要な設計判断が必要な場合、Requirements本文へ解決策を書かずADR候補として提示する。

次のいずれかに該当する判断を候補にする。

- 後から変更しにくい。
- 複数の合理的な選択肢がある。
- セキュリティ、データ整合性、可用性、運用に影響する。
- 複数PRDまたはシステム全体に影響する。
- 技術的負債を意図的に受け入れる。

候補一覧、関連TRまたはNFR、選択肢、推奨案、ADR化を推奨する理由をまとめて提示する。ADRの作成や確定は行わない。

## 7. 更新範囲を制御する

既存Requirementsの更新では、今回のPRDまたはBacklog変更に影響される要件だけを変更する。整理目的で既存内容を削除、再構成、改番しない。

更新によって影響する `architecture-change.md`、ADR、`tasks.md`、テストなどを提示する。下流成果物は自動更新しない。

## 8. 生成承認を得る

生成前に次をまとめて提示する。

- StoryごとのAC候補と主要な失敗・境界条件
- StoryごとのTR候補
- Technical Storyの対応User Storyと必要理由
- 共通NFRと、限定適用する場合のStory ID
- AssumptionsとOpen Questions
- ADR候補
- PRD・Backlogとの網羅性または逸脱の懸念
- 更新時は変更対象と下流成果物への影響

未確定事項は一問ずつ確認する。最後に「議論を終了してDraftを生成するか」を確認し、人間の明示的な承認を得るまでファイルを書き換えない。

## 9. Draftを生成する

テンプレートの見出し構成と順序を維持する。該当しない項目も削除せず `Not applicable` と記載する。独自見出しは追加しない。

Statusは常に `Draft` とする。既存Requirementsが `Accepted` または `Implemented` でも `Draft` へ戻し、`Status Reason` に変更理由と再レビューが必要な理由を書く。

AIが事実や方針を補完しない。人間判断が必要な事項は `Open Questions`、人間が暫定的に許容した前提だけを `Assumptions` に記載する。Blocking Open Questionが残っていてもDraftは生成できるが、生成前に下流作業を開始できないことを明示する。

## 10. 自己検証する

生成後に次を確認し、必要なら修正する。

- PRD ScopeとBacklogの全Storyが要件へ引き継がれている。
- Scope外の要件や解決策が混入していない。
- 各ACがユーザー視点で独立して判定可能である。
- 主要な失敗・境界条件がリスクに応じて扱われている。
- TRがStory固有の制約・保証に限定されている。
- Technical StoryにTR、対応User Story、必要理由がある。
- 共通要件がNFRへ置かれ、限定適用時だけStory IDがある。
- NFRの各カテゴリについて適用または `Not applicable` が明記されている。
- AC、TR、NFRのIDがPRD内で一意かつ連続運用され、旧IDが失われていない。
- Story、AC、TR、NFRの参照が追跡可能である。
- 設計判断がRequirements本文へ混入せず、必要なADR候補が提示されている。
- 更新時に対象外の既存内容を変更していない。
- テンプレートの構成、Status、PRD ID、出力先が正しい。

内容の妥当性確認は行わない。

## 11. 完了を報告する

次を簡潔に報告する。

- 作成または更新した `requirements.md` のパス
- 対象PRD IDと、PRD・BacklogのStatus
- Story、AC、TR、NFRの件数
- 参照した資料
- 残るAssumptions、Open Questions、Blocking事項
- ADR候補
- 影響を受ける下流成果物
- 次工程として別セッションの `document-review` が必要であること
