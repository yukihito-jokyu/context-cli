---
name: create-backlog
description: PRD連番IDからAccepted済みPRDを特定し、Scopeを成立させる全Storyをユーザー価値単位へ分解して、同じPRDディレクトリのbacklog.mdをDraftとして新規作成または更新する。PRDからBacklogを初めて作るとき、PRD変更を既存Backlogへ反映するとき、Storyの分割・優先度・依存関係を見直すときに使用する。
---

# Backlog作成

指定されたPRDのScopeを、独立して価値または進捗を確認できるStoryへ分解し、`backlog.md` のDraftを生成する。

## 必須条件

- 呼び出し時にPRDの3桁連番IDを受け取る。例: `001`
- 対話と本文には原則として日本語を使う。識別子、パス、既存文書の表記は維持する。
- 実行時は `docs/AI-driven-development.md` を参照しない。本スキルの記載規則に従う。
- Storyの内容を確定するために必要な情報が資料にない場合だけ、一問ずつ人間へ確認する。
- 内容の妥当性レビューとAccepted化は行わず、別セッションの `document-review` へ委ねる。

## 1. 対象PRDを特定する

入力されたIDを `XXX` とし、`docs/prds/prd-XXX-*/prd.md` を検索する。

- 該当が1つの場合: 対象PRDとして使う。
- 該当がない場合: 検索結果を報告して停止する。
- 複数該当する場合: 候補を列挙し、重複を自動解消せず停止する。

対象PRDのStatusが `Accepted` でなければ停止する。出力先は対象PRDと同じディレクトリの `backlog.md` に固定する。

## 2. 資料を読む

次の順で読む。

1. `docs/ai-driven-development/templates/backlog.md`
2. 対象の `prd.md`
3. 更新時は既存の `backlog.md`
4. PRDから参照される `docs/product.md`
5. Story分割との整合確認に必要な場合だけ `docs/engineering/*.md`
6. PRDまたは既存Backlogから直接参照されるAccepted成果物

テンプレートが存在しない、または確定した内容を表現できない場合は独自構成で生成せず、不足または不整合を報告して停止する。テンプレート自体は修正しない。

PRDの `Goal`、`Problem`、`Scope` は常に抽出する。資料から判明する内容を人間へ質問しない。資料間の矛盾は自動解消せず、箇所とStory分割への影響を示して人間の判断を得る。

## 3. Story候補へ分解する

PRDのScopeを成立させる全Storyを列挙する。直近で着手するStoryだけに限定しない。具体的な実装範囲は後続の `tasks.md` へ委ねる。

### User Story

- ユーザーが独立して価値または進捗を確認できる、検証可能な振る舞い単位にする。
- UI、API、DBなどの技術層や実装工程では分割しない。
- 1つのStory内で、利用者、達成したいこと、その価値を明確にする。
- PRD Scopeを超える価値を混ぜない。

### Technical Story

次をすべて満たす場合だけ作る。

- 特定のUser Storyを成立させるために必要である。
- User Story内の作業として扱うには、独立した依存関係、リスク、または検証がある。
- 単なる実装工程や技術層の分割ではない。

各Technical Storyには、対応するUser Story IDと必要理由を必ず記載する。ユーザー価値との接続を説明できない内部改善は現在のBacklogへ含めず、別PRD候補またはFollow-upとして提示する。

## 4. PriorityとDependenciesを決める

Priorityは次に固定する。

- `P0`: PRDの成果成立に不可欠で、他Storyの完了を阻害する。
- `P1`: PRDの成果成立に不可欠だが、P0への依存や順序上の緊急性がない。
- `P2`: Scope内だが、成果成立後でも提供できる。

独立して後回しにできるP2が見つかった場合は、PRDのScope過多を疑い、別PRD候補として人間へ確認する。自動的に除外またはPRD変更をしない。

<!-- 固定の定義または対応関係を1行で維持するため。 -->
<!-- textlint-disable preset-ja-technical-writing/no-doubled-conjunction -->

`Dependencies` には、他Storyが完了しなければ対象Storyを完了または検証できない関係だけを書く。単なる推奨実装順や同じ機能領域という関係は含めない。外部システムや別PRDへの依存はStory IDと混在させず、Story本文または `Open Questions` に記載する。

<!-- textlint-enable preset-ja-technical-writing/no-doubled-conjunction -->

## 5. Epicの要否を決める

Epicは、複数Storyを同じユーザー成果または利用段階としてまとめることで、Backlogや依存関係の理解が明確になる場合だけ使う。

- Story件数による固定基準を設けない。
- 分類名を付けるだけのEpicは作らない。
- Epic自体に独立した要件やAcceptance Criteriaを持たせない。
- Epicには所属するStory IDを列挙する。
- 不要な場合はテンプレートどおり `Not applicable` とする。

## 6. IDと更新方針を適用する

Story IDはPRD内で一意な `ST-XXX` の連番にする。一度使用したIDは再利用、振り直し、削除しない。

既存Backlogの更新では、今回影響するStoryだけを変更する。不要になったStoryは残し、`Cancelled` または `Superseded` として理由と置換先を記載する。Storyの分割・統合では新しいIDを発行し、旧Storyから新Storyへの参照を残す。

PRD変更に伴う更新では、影響するStoryと `requirements.md`、`architecture-change.md`、`tasks.md` などの下流成果物への影響を提示する。後続成果物は自動更新しない。

## 7. 生成承認を得る

生成前に次をまとめて提示する。

- User Story候補と分割理由
- Technical Story候補、対応User Story、必要理由
- 各StoryのPriorityとDependencies
- Epicの要否と、使う場合は所属Story
- AssumptionsとOpen Questions
- 別PRD候補またはFollow-up
- 更新時は変更対象と下流への影響

人間へ一問ずつ確認し、未確定事項を解消する。最後に「議論を終了してDraftを生成するか」を確認し、明示的な承認を得るまでファイルを書き換えない。

## 8. Draftを生成する

テンプレートの見出し構成と順序を維持する。該当しない項目も削除せず `Not applicable` と記載する。独自見出しは追加しない。

Statusは常に `Draft` とする。既存Backlogが `Accepted` または `Implemented` でも `Draft` へ戻し、`Status Reason` に変更理由と再レビューが必要な理由を書く。

着手順はPriorityとDependenciesで表し、具体的な実装範囲やTaskをBacklogへ書かない。AIが事実や方針を補完せず、人間判断が必要な事項は `Open Questions`、人間が暫定的に許容した前提だけを `Assumptions` に記載する。

## 9. 自己検証する

生成後に次を確認し、必要なら修正する。

- PRD Scopeを成立させるStoryが網羅されている。
- PRD Scopeを逸脱したStoryがない。
- User Storyが縦断的で検証可能な振る舞い単位になっている。
- Technical Storyが必要最小限で、対応User Storyと必要理由がある。
- Priorityが定義に従い、P2によるScope過多を見落としていない。
- Dependenciesが真の完了・検証依存だけになっている。
- Epicが必要な場合だけ使われている。
- IDが一意で、既存IDの再利用、振り直し、削除がない。
- 更新時に対象外の既存内容を変更していない。
- テンプレートの構成、Status、PRD ID、出力先が正しい。

## 10. 完了を報告する

次を簡潔に報告する。

- 作成または更新した `backlog.md` のパス
- 対象PRD IDとStatus
- User Story、Technical Story、Epicの件数
- 参照した資料
- 残るAssumptionsとOpen Questions
- 別PRD候補またはFollow-up
- 影響を受ける下流成果物
- 次工程として別セッションの `document-review` が必要であること
