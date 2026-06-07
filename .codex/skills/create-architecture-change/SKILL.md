---
name: create-architecture-change
description: Accepted済みのPRD、Backlog、Requirements、必要なADRとEngineering Foundationから、PRDによるシステム変更案を設計し、同じPRDディレクトリのarchitecture-change.mdをDraftとして新規作成または更新する。実装前のAPI、DB、ドメイン、パッケージ、外部連携、運用、移行、互換性、セキュリティへの変更を整理するとき、初回PRDの初期設計を定義するとき、既存architectureとの差分を設計するときに使用する。
---

# Architecture Change作成

Accepted済みの要件と判断を、実装前の追跡可能な設計変更案へ変換する。

## 必須条件

- 呼び出し時にPRDの3桁連番ID（例: `001`）を受け取る。
- 対話と本文には原則として日本語を使う。識別子、パス、既存文書の表記は維持する。
- 実行時は `docs/AI-driven-development.md` を参照しない。本スキルの記載規則に従う。
- 内容の妥当性レビュー、StatusのAccepted化、`tasks.md` 作成、実装は行わない。
- 実装後の確定版更新はimplementationスキル、現在状態の `docs/architecture/*.md` 更新はarchitecture更新スキルへ委譲する。

## 1. 対象PRDを解決する

PRD IDから `docs/prds/prd-XXX-*/` を検索し、対象ディレクトリを一意に特定する。

次をすべて満たす場合だけ続行する。

- `prd.md`、`backlog.md`、`requirements.md` が存在する。
- 3成果物のStatusがすべて `Accepted` である。
- `docs/engineering/technology.md`、`structure.md`、`development-rules.md` が存在し、適用対象のStatusが `Accepted` である。
- RequirementsまたはAccepted済みADRが必要としている未作成ADRがない。
- 既存 `architecture-change.md` を更新する場合、そのPRD IDが対象PRDと一致する。

1つでも満たさない場合は、設計を推測せず理由と戻るべき成果物を報告して停止する。

## 2. 資料を読む

次の順で読む。

1. `docs/ai-driven-development/templates/architecture-change.md`
2. 対象のAccepted済み `prd.md`、`backlog.md`、`requirements.md`
3. `docs/engineering/*.md`
4. 対象PRD、Story、Requirementから参照されるAccepted済みADR
5. 存在する場合は `docs/architecture/*.md`
6. 関連PRDのAccepted済み成果物と設計変更案
7. 既存 `architecture-change.md`
8. 現在の構造、変更箇所、既存制約、実現可能性の確認に必要な範囲だけコード、設定、依存関係

実装方法の確定、クラス・関数単位の設計、詳細な変更ファイル一覧までは調査しない。これらは `tasks.md` 作成へ委譲する。

テンプレートが存在しない、または確定事項を表現できない場合は独自構成で生成せず、不足または不整合を報告して停止する。テンプレート自体は修正しない。

## 3. 追跡範囲を確定する

Scopeを成立させるすべての対象Storyと、対応するAC、TR、NFRを特定する。各設計変更がどのStoryまたはRequirementを満たすか追跡できるようにする。

次も確認する。

- 関連PRDとの競合可能性、依存関係、統合順序
- 既存architectureとの差分。初回PRDで未作成なら初期設計として必要な範囲
- 関連ADRの判断と制約
- API、DB、ドメイン、パッケージ、外部連携、インフラ、エラー処理、可観測性への影響
- 移行、互換性、廃止、切り戻し、復旧への影響
- セキュリティ、プライバシー、運用への影響
- 実装後に `docs/architecture/*.md` へ反映する対象

## 4. 設計粒度を制御する

Requirementsを満たすためのコンポーネント境界、責務、依存方向、データフロー、API、DB、ドメインなどの変更方針を記載する。

完成基準は、後続AIが新たな重要設計判断を追加せず `tasks.md` を作成できることである。ただし、クラス、関数、詳細な変更ファイル、逐次的な実装手順は含めない。

既存文書やコードから確認できない事実を補完してはならない。人間が暫定的に許容した前提だけをAssumptionsへ記載する。

## 5. ADRへ戻す事項を判定する

Accepted済みADRにない重要な設計判断が必要な場合は、`architecture-change.md` 内で決定せず作成を停止し、ADR候補抽出へ戻す。

次のいずれかに該当し、複数の合理的な選択肢がある判断をADR候補とする。

- システム横断または複数PRDへ影響する。
- 長期的な制約になる。
- 導入、移行、運用、撤回のコストが高い。
- 後から変更することが難しい。
- API、DB、設定、CLIなどの破壊的変更を伴う。

軽微で局所的かつ容易に変更できる判断だけを設計変更案へ直接記載する。判断に迷う場合は候補と根拠を提示し、人間に確認する。

## 6. Open Questionsを分類する

未確定事項は次のいずれかに分類する。

- Blocking: 解決するまでAccepted化または `tasks.md` 作成へ進めない事項
- Deferred: Draft時点では保留でき、判断条件と判断する工程が明確な事項

Blockingが残っていてもDraftは生成できる。Deferredには、何が判明したら判断するかと、どの工程で判断するかを必ず記載する。

## 7. Draft内容を組み立てる

テンプレートの見出し構成と順序を維持し、独自見出しを追加しない。該当しない項目も削除せず、理由を添えて `Not applicable` と記載する。

- Status: 常に `Draft`
- Status Reason: 新規設計案または変更テーマと、重レビューが必要な理由
- PRD ID: 対象ディレクトリと一致させる
- Source: 読み込んだPRD配下の主要成果物
- Target Stories: 対象Story ID
- Change Summary: Requirementsをどう満たす変更か
- 各変更カテゴリ: 現在状態との差分と変更方針
- Related Decisions: Accepted済みADRだけ
- Updates to Current Architecture: 実装後に反映する対象。現在状態として先取りしない
- Assumptions: 人間が明示的に許容した前提だけ
- Open Questions: BlockingとDeferredに分類する

既存ファイルの更新では今回の変更テーマに関係する箇所だけを変更し、整理目的の削除、言い換え、再構成をしない。資料間の矛盾を自動解消せず、人間が正本と修正対象を確定するまで停止する。

## 8. 生成承認を得る

書き込み前に次を要約する。

- 対象PRDとTarget Stories
- 主要な設計変更案と既存architectureとの差分
- RequirementsとADRへの追跡関係
- 影響範囲、移行、互換性、切り戻し
- BlockingとDeferred
- ADR候補抽出へ戻す事項
- 更新時に変更する既存箇所

「この内容でarchitecture-change.md Draftを生成するか」を確認し、人間の明示的な承認を得るまでファイルを書き換えない。

## 9. Draftを生成する

承認後、対象PRDディレクトリの `architecture-change.md` を新規作成または更新する。別の出力先は人間が明示した場合だけ許可する。

生成後に次を自己検証し、必要なら修正する。

- テンプレートの構成と順序を維持している。
- StatusがDraftであり、Status Reasonが今回の変更を説明している。
- Story、AC、TR、NFR、ADRとの追跡関係がある。
- 設計粒度が実装詳細へ踏み込んでいない。
- 重要判断を設計案内で暗黙に決定していない。
- BlockingとDeferredが正しく分類されている。
- 既存architectureとの差分が明確である。
- 互換性、移行、廃止、切り戻し、セキュリティ、運用への影響を評価している。
- 実装後に現在architectureへ反映する対象が明確である。
- 既存内容を意図せず変更していない。
- 上流成果物、既存architecture、コード、設定との矛盾がない。

内容の妥当性レビューは行わない。

## 10. 完了を報告する

次を簡潔に報告する。

- 対象PRD IDと出力先
- Target Stories
- 主要な設計変更
- BlockingとDeferred
- ADR候補または差し戻し事項
- 参照した資料
- 次工程として、対象ファイルを指定した別セッションの `document-review`（重）が必要であること
- Blockingがある場合、Accepted化と `tasks.md` 作成へ進めないこと
