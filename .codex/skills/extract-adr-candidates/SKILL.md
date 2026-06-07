---
name: extract-adr-candidates
description: Accepted済みのRequirements、またはDraft・Review・AcceptedのEngineering Foundationから重要な技術判断を抽出し、選択肢、評価軸、推奨案、ADR化要否をadr-candidates.mdのDraftとして記録する。PRDの設計開始前、Engineering Foundation作成後、既存方針との競合や新しい重要判断が見つかったときに使用する。
---

# ADR候補抽出

重要な技術判断候補をまとめて抽出し、最重レビューへ渡せる `adr-candidates.md` を生成する。

## 必須条件

- 呼び出し時に、PRDの3桁連番ID（例: `001`）または `engineering-foundation` を受け取る。
- 対話と本文には原則として日本語を使う。識別子、パス、既存文書の表記は維持する。
- 実行時は `docs/AI-driven-development.md` を参照しない。本スキルの記載規則に従う。
- 候補を一問ずつ確定せず、候補一覧と推奨案をまとめて提示する。
- 候補の採否、内容の妥当性確認、Accepted化、ADR作成は行わず、別セッションの `document-review` とADR作成スキルへ委ねる。

## 1. モードと出力先を決める

### PRDモード

入力IDを `XXX` とし、`docs/prds/prd-XXX-*/` を検索する。

- 該当ディレクトリが1つでなければ、検索結果を報告して停止する。
- `prd.md`、`backlog.md`、`requirements.md` がすべて存在し、`Accepted` の場合だけ続行する。
- 出力先を同じディレクトリの `adr-candidates.md` に固定する。

### Engineering Foundationモード

入力が `engineering-foundation` の場合に使用する。

<!-- 固定の定義または対応関係を1行で維持するため。 -->

- `docs/engineering/technology.md`、`structure.md`、`development-rules.md` がすべて存在し、各Statusが `Draft / Review / Accepted` のいずれかの場合だけ続行する。

- ADR判断以外のBlocking Open Question、参照不能、3ファイル間または上位文脈との未解決矛盾がある場合は停止する。
- 出力先を `docs/engineering/adr-candidates.md` に固定する。

PRDモードの未Accepted、参照不能、または正本間に未解決の矛盾がある場合は、理由を報告して停止する。

## 2. 資料を読む

次の順で読む。

1. `docs/ai-driven-development/templates/adr-candidates.md`
2. 対象モードのAccepted成果物
3. 更新時は既存の `adr-candidates.md`
4. `docs/engineering/*.md`
5. `docs/decisions/adr-*.md`
6. 存在する場合は `docs/architecture/*.md`
7. 対象成果物から直接参照されるAccepted成果物
8. 現在実装の事実確認が必要な場合だけコード、設定、依存関係

Engineering Foundationモードでは、2と4は同じ3ファイルを指すため一度だけ読む。

テンプレートが存在しない、または確定事項を表現できない場合は独自構成で生成せず、不足または不整合を報告して停止する。テンプレート自体は修正しない。

資料から判明する内容を人間へ質問しない。資料間の矛盾は自動解消せず、対象箇所、影響、正本候補を提示して停止する。

## 3. 候補を抽出する

次のいずれかに該当する判断を候補にする。

- 後から変更するコストが高い。
- 複数の合理的な選択肢がある。
- セキュリティ、データ整合性、可用性、運用へ重大な影響がある。
- 複数PRD、複数モジュール、またはシステム全体に影響する。
- 技術的負債または重大なリスクを意図的に受け入れる。
- `docs/engineering/` の原則や依存方向を変更する。

容易に戻せる設定、細かなライブラリ選択、実装中に局所的に決められる事項は原則として候補にしない。設定、コード、Git履歴、または `architecture-change.md` で十分な判断をADRへ昇格させない。

各候補の境界を1つの判断に限定する。複数の独立した判断を一候補へまとめない。

## 4. 候補を記述する

各候補に次を記載する。

- Candidate ID: ファイル内で一意な `ADC-XXX`
- Title
- Status: 初期値は `Proposed`
- Decision Needed: 判断が必要な理由
- Decision Timing: `Now / Before Implementation / Deferred`
- Related PRDs
- Related Stories
- Related Requirements
- Related Architecture Changes
- Related ADRs: 適用、補足、競合、または置き換え対象との関係
- Options: 合理的な選択肢
- Evaluation Criteria: 比較に使う評価軸
- Recommendation: 推奨案と根拠
- ADR Recommendation: `Create ADR / Do not create ADR`
- ADR Recommendation Reason
- Human Decision Reason: 初期値は `Not applicable`
- Resulting ADR: 初期値は `Not applicable`

関連しない参照項目は削除せず `Not applicable` とする。判断期限を推測しない。今決める必要がない場合は `Deferred` を推奨し、再検討条件を理由へ含める。

## 5. 既存ADRとの関係を判定する

- 既存ADRで判断済み: 新規候補にせず、適用したADRと適用可能性を完了報告へ記載する。
- 既存ADRの判断内の補足: ADR候補にせず、後続の設計文書で扱う事項として提示する。
- 既存ADRの判断変更: 新規候補とし、置き換え対象を明記する。既存ADRは変更しない。
- 判断境界が異なる: 別候補とし、既存ADRとの関係を記載する。
- 重複または競合を判定できない: 自動解消せず、差異と影響を人間へ確認する。

## 6. IDと更新履歴を守る

`ADC-XXX` は出力ファイル内の通し番号とする。一度使用したIDは再利用、振り直し、削除しない。

レビュー後のStatusには次を使う。

- `Accepted`: ADRを作成する候補
- `Rejected`: ADR化しない候補
- `Deferred`: 判断を延期する候補

更新時は今回影響する候補だけを変更する。既存候補を整理目的で削除、再構成、改番しない。判断境界が変わる分割・統合では新しいIDを発行し、旧候補に理由と置換先を残す。

このスキルはレビュー結果を先取りしてStatusを `Accepted / Rejected / Deferred` に変更しない。既存の確定結果も上流変更との明確な矛盾がない限り維持する。

## 7. 生成承認を得る

生成前に次をまとめて提示する。

- 全候補のCandidate ID案、判断境界、判断理由
- 判断期限
- 関連成果物とID
- 選択肢、評価軸、推奨案
- ADR化推奨の有無と理由
- 既存ADRとの重複、適用、競合、置き換え候補
- ADR候補にしなかった重要事項と扱い先
- Assumptions、Open Questions、Blocking事項

不足情報がある場合だけ一問ずつ確認する。最後に「議論を終了してDraftを生成するか」を確認し、人間の明示的な承認を得るまでファイルを書き換えない。

## 8. Draftを生成する

テンプレートの見出し構成と順序を維持する。該当しない項目も削除せず `Not applicable` と記載し、独自見出しは追加しない。

成果物全体のStatusは常に `Draft` とする。既存ファイルが `Accepted` でも `Draft` へ戻し、`Status Reason` に抽出テーマと再レビューが必要な理由を書く。

AIが選択肢、事実、評価結果を根拠なく補完しない。人間判断が必要な事項は `Open Questions`、人間が暫定的に許容した前提だけを `Assumptions` に記載する。

## 9. 自己検証する

生成後に次を確認し、必要なら修正する。

- ADR化基準を満たさない局所判断を候補にしていない。
- 各候補が1つの明確な判断である。
- 選択肢と評価軸が推奨案へ不当に偏っていない。
- 判断期限とDeferred条件が明確である。
- 関連PRD、Story、Requirement、Architecture Change、既存ADRを追跡できる。
- 既存ADRとの重複、競合、置き換え関係を確認している。
- Candidate IDが一意で、旧IDと確定結果を失っていない。
- テンプレート構成、Draft状態、出力先が正しい。
- 対象外の既存内容を変更していない。

内容の妥当性レビューは行わない。

## 10. 完了を報告する

次を簡潔に報告する。

- 作成または更新した `adr-candidates.md` のパス
- 対象モードと入力成果物のStatus
- 候補件数とCandidate ID
- 既存ADRの適用、重複、競合、置き換え候補
- ADR候補にしなかった重要事項と扱い先
- Assumptions、Open Questions、Blocking事項
- 参照した資料
- 次工程として、対象ファイルを指定した別セッションの `document-review` が必要であること

レビュー後は `Accepted` のCandidate IDをADR作成スキルへ渡す。ADR生成後、同スキルが `Resulting ADR` を更新する。
