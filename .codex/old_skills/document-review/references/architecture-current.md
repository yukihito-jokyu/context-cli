# Current Architecture Review

## Recognition

- Paths: `docs/architecture/*.md`
- Templates: matching files under `docs/ai-driven-development/templates/architecture/`
- Default importance: `中`
- Status required: yes

## Inputs

- Required: all applicable current architecture documents and matching templates
- Conditional: implemented PRD artifacts and Accepted ADRs
  <!-- 固定の定義または対応関係を1行で維持するため。 -->
  <!-- textlint-disable preset-ja-technical-writing/max-comma -->

- Code/config: implementation, runtime configuration, schema, API, operations, package structure

<!-- textlint-enable preset-ja-technical-writing/max-comma -->

## Review Points

- 実装後の現在状態として正しいか
- 未実装の予定、履歴、一時的な変更案が混ざっていないか
- overview、domain、API、database、package、operations間で矛盾していないか
- Source、Related PRDs、Related Decisionsが残っているか
- Not applicableの理由が現在状態に照らして明確か
- 実装、設定、ADR、確定版architecture-changeと一致しているか
- 単なる記載漏れと設計判断の変更を区別しているか
- 新規作成または更新されたファイルがReviewで、Status Reasonに対象PRDと再レビュー理由があるか
- 更新対象外のファイルが整理目的で変更されていないか

## Completion

- 初回作成時は標準6ファイルを一組で確認する
- 更新時も影響を受ける他ファイルとの整合性を確認する
- Allowed transition: `Review -> Accepted`
