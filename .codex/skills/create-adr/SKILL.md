---
name: create-adr
description: 最重レビューでAcceptedになったADR候補を正式な技術判断として記録し、docs/decisions/adr-XXX-short-title.mdのDraftを生成する。PRDまたはEngineering Foundationのadr-candidates.mdからADRを作成するとき、複数のAccepted候補を個別ADRへ変換するとき、既存ADRをSupersedesする新しい判断を記録するときに使用する。
---

# ADR作成

Accepted済み候補の判断を補完・再解釈せず、正式なADR Draftとして記録する。

## 必須条件

- 呼び出し時にSourceと1つ以上のCandidate IDを受け取る。
  - Source: PRDの3桁連番ID（例: `001`）または `engineering-foundation`
  - Candidate ID: `ADC-001` 形式
- 対話と本文には原則として日本語を使う。識別子、パス、既存文書の表記は維持する。
- 実行時は `docs/AI-driven-development.md` を参照しない。本スキルの記載規則に従う。
- 判断の再議論、候補の採否、内容の妥当性レビュー、ADRのAccepted化は行わない。
- 複数Candidate IDを受け取っても、一候補につき1つのADRを作る。

## 1. Sourceを解決する

<!-- 固定の定義または対応関係を1行で維持するため。 -->

PRD IDを受け取った場合は `docs/prds/prd-XXX-*/adr-candidates.md`、`engineering-foundation` の場合は `docs/engineering/adr-candidates.md` を対象とする。

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->

次をすべて満たす場合だけ続行する。

- 対象ファイルが一意に存在する。
- 成果物全体のStatusが `Accepted` である。
- 指定したCandidate IDがすべて存在する。
- 各候補のStatusが `Accepted` である。
- 各候補の `ADR Recommendation` が `Create ADR` である。
- `Resulting ADR` が `Not applicable` であり、未生成である。

1つでも満たさない場合は、全件を生成せず理由を報告して停止する。

## 2. 資料を読む

次の順で読む。

1. `docs/ai-driven-development/templates/adr.md`
2. 対象の `adr-candidates.md`
3. Sourceに対応するAccepted済み上流成果物
4. `docs/engineering/*.md`
5. `docs/decisions/adr-*.md`
6. 存在する場合は `docs/architecture/*.md`
7. 候補から直接参照されるAccepted成果物
8. 現在実装の事実確認が必要な場合だけコード、設定、依存関係

テンプレートが存在しない、または確定事項を表現できない場合は独自構成で生成せず、不足または不整合を報告して停止する。テンプレート自体は修正しない。

## 3. 生成可能性を検証する

各候補について次を確認する。

- 1つの判断に限定されている。
- Acceptedになった採用判断を `Recommendation` と `Human Decision Reason` から一意に特定できる。
- 合理的な選択肢、評価結果、不採用理由をADRへ記録できる。
- Context、Consequences、Risks、Mitigationsを根拠資料から記述できる。
- 関連PRD、Story、Requirement、Architecture Change、既存ADRを追跡できる。
- 既存ADRとの重複、競合、Supersedes関係が解決済みである。
- ADR作成を妨げるOpen Questions、情報不足、資料間の矛盾がない。

不足や矛盾をAIが補完、推測、再決定してはならない。該当Candidate ID、問題、影響を報告して全件の生成を停止し、対象 `adr-candidates.md` の `document-review` へ戻す。

## 4. 採番とファイル名を決める

`docs/decisions/adr-*.md` の最大3桁番号を確認し、その次から指定候補順に連番を割り当てる。既存ADRがない場合は `001` から始める。欠番を再利用しない。

短縮名は候補タイトルから英小文字のkebab-caseで提案し、ファイル名を `adr-XXX-short-title.md` とする。

- 候補ごとにADR番号、タイトル、短縮名、出力先を提示する。
- 同じ出力先が既に存在する場合は、自動で短縮名や番号を変更せず停止する。
- 人間が生成を承認した時点で短縮名を確定する。

## 5. Draft内容を組み立てる

テンプレートの見出し構成と順序を維持し、独自見出しを追加しない。該当しない項目も削除せず `Not applicable` または `None` と記載する。

- Status: 常に `Draft`
- Status Reason: Accepted候補から正式ADRを生成し、最重レビューが必要であること
- Decision ID: 割り当てた `ADR-XXX`
- Date: Draft生成日。後続のStatus変更では更新しない
- Context: 判断が必要になった背景。解決策を先取りしない
- Decision: 人間がAcceptedにした採用判断
- Options: 候補の全選択肢、Pros、Cons、不採用理由
- Rationale: 評価軸、人間の確定理由、採用根拠
- Consequences / Risks / Mitigations: 根拠資料で確認できる内容
- Related項目: 候補と上流成果物の参照
- Supersedes: 置き換え対象。なければ `None`
- Superseded By: Draft生成時は `None`
- Assumptions: 人間が明示的に許容した前提だけ
- Open Questions: ADRの判断確定を妨げない後続確認だけ。なければ `Not applicable`

採用されなかったすべての選択肢に不採用理由を残す。既存ADRを置き換える場合も旧ADRを変更しない。

## 6. 生成承認を得る

書き込み前に全件をまとめて提示する。

- SourceとCandidate ID
- ADR番号、タイトル、短縮名、出力先
- Decisionの要約
- 不採用案と不採用理由
- 関連成果物と既存ADR
- Supersedes関係
- Assumptions、Open Questions

「この内容とファイル名でADR Draftを生成するか」を確認し、人間の明示的な承認を得るまでファイルを書き換えない。

## 7. 全件を一括生成する

承認後、すべてのADRファイルと対象 `adr-candidates.md` の更新を1つの変更単位として扱う。途中で一件でも生成または検証に失敗した場合は、部分的なADRや参照更新を残さず全件を生成前の状態へ戻す。

各ADRの生成後、対応する候補の `Resulting ADR` だけを生成したADRパスへ更新する。候補のStatus、Human Decision Reason、成果物全体のStatusは変更しない。

## 8. 自己検証する

生成後に次を確認し、必要なら修正する。

- 指定したCandidate IDと生成ADRが一対一で対応する。
- 採番がグローバルな最大番号の次から連続し、既存番号と衝突しない。
- ファイル名が確定した短縮名と一致する。
- DecisionがAccepted候補から逸脱していない。
- 全選択肢と不採用理由が残っている。
- 関連成果物、既存ADR、Supersedesを追跡できる。
- DateがDraft生成日である。
- テンプレート構成、Draft状態、出力先が正しい。
- 各候補の `Resulting ADR` が正しい。
- 対象外の候補、既存ADR、上流成果物を変更していない。

内容の妥当性レビューは行わない。

## 9. 完了を報告する

次を簡潔に報告する。

- SourceとCandidate ID
- 作成したADRのDecision IDとパス
- 更新した `adr-candidates.md`
- Supersedes関係
- Assumptions、Open Questions
- 参照した資料
- 次工程として、各ADRファイルを指定した別セッションの `document-review` が必要であること
