# Tasks

Status: Draft
Status Reason: <この状態である理由>
PRD ID: prd-XXX-<slug>

## Source

- PRD: ./prd.md
- Backlog: ./backlog.md
- Requirements: ./requirements.md
- Architecture Change: ./architecture-change.md

## Risk Assessment

- Flow: <Standard | Lightweight | Hotfix>
- Risk Level: <Low | Medium | High>
- Rationale: <適用フローとリスク判定の根拠>
- Human Confirmation: <Confirmed | Pending>

## Coordination

- Owner: <担当者またはAIエージェント>
- Parallel Work: <並行作業と統合順序。なければNot applicable>

## Implementation Order

1. T-001
2. T-002
3. T-003

## Tasks

<!-- 状態値と識別子の選択肢を1行で定義するため。 -->
<!-- textlint-disable preset-ja-technical-writing/sentence-length -->

### T-001: <Task Title>

Status: <Pending | In Progress | Blocked | Completion Candidate | Completed | Cancelled | Superseded>
Status Reason: <現在の状態である理由>
Replaced By: <置換先Task ID。該当しない場合はNot applicable>
Purpose: <目的>
Linked Stories: ST-001
Linked AC: AC-001
Linked Technical Requirements: TR-001
Linked Non-Functional Requirements: NFR-001
Dependencies: None
Shared Task Rationale: <複数Storyに共通する独立Taskの必要理由。該当しない場合はNot applicable>。

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->

#### Files to Change

- <変更予定または調査対象ファイル。実装中に変わる場合は更新する>

#### Test Tasks

- TT-001: <検証可能な振る舞い>
  - Status: <Pending | Completed | Cancelled | Superseded>
  - Status Reason: <現在の状態である理由>
  - Replaced By: <置換先Test Task ID。該当しない場合はNot applicable>
  - Type: <e2e | integration | API | component | unit | contract | regression | manual>
  - Linked AC: AC-001
  - Linked Technical Requirements: TR-001
  - Linked Non-Functional Requirements: NFR-001
  - Test First: <Yes | No>
  - Red Check Required: <Yes | No>
    <!-- 固定の状態値を1行で定義するため。 -->
    <!-- textlint-disable preset-ja-technical-writing/sentence-length -->
  - Red Check Result: <Not run | Failed as expected | Failed unexpectedly | Already passing | Not applicable>
  - Green Check Result: <Not run | Passed | Failed | Not applicable>
  - Not Run Reason: <未実行の場合の理由。該当しない場合はNot applicable>
  - Alternative Verification: <代替検証方法。該当しない場合はNot applicable>
  <!-- textlint-enable preset-ja-technical-writing/sentence-length -->

#### Implementation Notes

- Actual Files Changed: <実際に変更したファイル>
- Test Results: <実行したテストと結果>
- Follow-up: <追加で判明した後続対応。該当しない場合はNot applicable>

#### Code Review

- Status: <Pending | In Review | Blocked | Completed | Not applicable>
- Scope: <PRD全体レビューまたは限定レビューの対象差分>
- Findings: <Critical / High / Medium / Lowの指摘と解消状況。なければNot applicable>
- Fixes: <レビューで修正した内容とファイル。なければNot applicable>
- Verification: <再実行したテストと品質ゲート。未実行なら理由と代替検証>
- Remaining Risks: <Follow-upまたは受容した残存リスク。なければNot applicable>
- Human Decision: <Approved | Rejected | Pending>

#### Definition of Done

- <完了条件>
- 対応するテストが完了している
- 未実行テストがある場合、理由、代替検証方法、後続対応が記録されている
- 必要なドキュメント更新が完了している

## Documentation Updates

- `architecture-change.md` を実装結果に合わせて更新する
- 必要に応じて `docs/architecture/` を更新する
- 必要に応じて `docs/decisions/` を追加または更新する

## Quality Gates

- Tests: <Pass | Failed | Not run>
- Build: <Pass | Failed | Not run | Not applicable>
- Lint: <Pass | Failed | Not run | Not applicable>
- Type Check: <Pass | Failed | Not run | Not applicable>
- Not Run Reason / Alternative Verification: <該当しない場合はNot applicable>

## Code Review Summary

- Status: <Pending | In Review | Blocked | Completed | Not applicable>
- Scope: <対象PRD全体の差分。限定レビューだけの場合は未完了であることを明記>
- Critical / High Remaining: <件数と内容。なければNone>
- Medium / Low / Follow-up: <残存項目。なければNot applicable>
- Existing Changes Separation: <今回の変更と既存差分を区別した根拠>
- Human Decision: <Approved | Rejected | Pending>

## Architecture Update

- Status: <Pending | In Review | Blocked | Completed | Not applicable>
- Scope: <初回6ファイル作成、または今回の更新対象>
- Evidence: <実装差分、architecture-change、ADR、コード、設定などの調査根拠>
- Files Updated: <作成または更新したdocs/architecture/\*.md>
- Inconsistencies: <発見した不一致と解消状況。なければNot applicable>
- Unreflected Items: <未反映事項と理由。なければNot applicable>
- Human Decision: <Approved | Rejected | Pending>

## Release Check

<!-- リリース判定軸と固定値を1行で対応付けるため。 -->
<!-- textlint-disable preset-ja-technical-writing/sentence-length -->

- Scope: <対象PRDに対応するPR差分全体>
- Checked At: <確認日時またはコミット>
- Goal / Story / AC / Success Metrics: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- TR / NFR / ADR: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- Engineering Foundation: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- Tests / Regression: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- Migration / Configuration / External Services / Permissions / Operations: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- Monitoring / Logs / Rollback / Recovery: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- Architecture Consistency: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- Open Items Classification: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- Dependencies / External Code / Licenses: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- Security / Privacy / Accessibility / Legal / Contract / Policy: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
<!-- textlint-enable preset-ja-technical-writing/sentence-length -->
- Traceability / ID Integrity: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- Overall Result: <Pass | Blocked | Not applicable>
- Remaining Risks: <残存リスク。なければNot applicable>
- Follow-up: <移管先、理由、期限または対応条件。なければNot applicable>
- Human Decision: <Approved | Rejected | Pending>

## Post-Release Verification

- Result: <Pass | Failed | Pending | Not applicable>
- AC / Monitoring / Logs Checked: <確認内容>
- Rollback / Recovery Result: <実施結果。未実施ならNot applicable>

## Follow-up Transfer

- <Issueまたは次のPRD候補への参照、理由、重要度、対応条件。なければNot applicable>

## Assumptions

- <タスク分解時の前提>

## Open Questions

### Blocking

- <Accepted化または実装開始を妨げる未確定事項>

### Deferred

- <判断条件、判断工程、影響するTask>
