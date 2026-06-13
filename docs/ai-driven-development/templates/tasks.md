# タスク定義 (Tasks)

仕様書ID: spec-XXX-<slug>

## 情報源

- 仕様書: ./spec.md
- バックログ: ./backlog.md
- 要件定義: ./requirements.md
- アーキテクチャ変更履歴: ./architecture-change.md

## リスク検討

- 適用フロー: <Standard | Lightweight | Hotfix>
- リスクレベル: <Low | Medium | High>
- 根拠: <適用フローとリスク判定の根拠>
- 人間の確認: <Confirmed | Pending>

## 作業調整

- 担当者: <担当者またはAIエージェント>
- 並行作業: <並行作業と統合順序。なければNot applicable>

## 実装順序

1. T-001
2. T-002
3. T-003

## 個別タスク定義

### T-001: <タスクタイトル>

ステータス: <Pending | In Progress | Blocked | Completion Candidate | Completed | Cancelled | Superseded>
理由: <現在の状態である理由>
置換先: <置換先Task ID。該当しない場合はNot applicable>
目的: <目的>
関連ストーリー: ST-001
関連完成条件: AC-001
関連技術要件: TR-001
関連する非機能要件: NFR-001
依存関係: None
独立タスク根拠: <複数Storyに共通する独立Taskの必要理由。該当しない場合はNot applicable>。

#### 変更予定ファイル

- <変更予定または調査対象ファイル。実装中に変わる場合は更新する>

#### テストタスク

- TT-001: <検証可能な振る舞い>
  - ステータス: <Pending | Completed | Cancelled | Superseded>
  - 理由: <現在の状態である理由>
  - 置換先: <置換先Test Task ID。該当しない場合はNot applicable>
  - テスト種別: <e2e | integration | API | component | unit | contract | regression | manual>
  - 関連完成条件: AC-001
  - 関連技術要件: TR-001
  - 関連する非機能要件: NFR-001
  - テスト先行 (TDD): <Yes | No>
  - レッドチェック必須: <Yes | No>
  - レッドチェック結果: <Not run | Failed as expected | Failed unexpectedly | Already passing | Not applicable>
  - グリーンチェック結果: <Not run | Passed | Failed | Not applicable>
  - 未実行理由: <未実行の場合の理由。該当しない場合はNot applicable>
  - 代替検証: <代替検証方法。該当しない場合はNot applicable>

#### 実装ノート

- 実際の変更ファイル: <実際に変更したファイル>
- テスト結果: <実行したテストと結果>
- 追加課題: <追加で判明した後続対応。該当しない場合はNot applicable>

#### コードレビュー

- ステータス: <Pending | In Review | Blocked | Completed | Not applicable>
- レビュー範囲: <仕様書全体レビューまたは限定レビューの対象差分>
- 指摘事項: <Critical / High / Medium / Lowの指摘と解消状況。なければNot applicable>
- 修正内容: <レビューで修正した内容とファイル。なければNot applicable>
- 検証状況: <再実行したテストと品質ゲート。未実行なら理由と代替検証>
- 残存リスク: <Follow-upまたは受容した残存リスク。なければNot applicable>
- 人間の意思決定: <Approved | Rejected | Pending>

#### 完了条件

- <完了条件>
- 対応するテストが完了している
- 未実行テストがある場合、理由、代替検証方法、後続対応が記録されている
- 必要なドキュメント更新が完了している

## ドキュメント更新

- `architecture-change.md` を実装結果に合わせて更新する
- 必要に応じて `docs/architecture/` を更新する
- 必要に応じて `docs/decisions/` を追加または更新する

## 品質ゲート

- テスト: <Pass | Failed | Not run>
- ビルド: <Pass | Failed | Not run | Not applicable>
- 静的解析 (Lint): <Pass | Failed | Not run | Not applicable>
- 型チェック: <Pass | Failed | Not run | Not applicable>
- 未実行理由/代替検証: <該当しない場合はNot applicable>

## コードレビューサマリー

- ステータス: <Pending | In Review | Blocked | Completed | Not applicable>
- レビュー範囲: <対象の仕様書全体の差分。限定レビューだけの場合は未完了であることを明記>
- 残存重要指摘: <件数と内容。なければNone>
- 軽微な残存事項: <残存項目。なければNot applicable>
- 既存差分との分離: <今回の変更と既存差分を区別した根拠>
- 人間の意思決定: <Approved | Rejected | Pending>

## アーキテクチャ更新

- ステータス: <Pending | In Review | Blocked | Completed | Not applicable>
- 更新範囲: <初回6ファイル作成、または今回の更新対象>
- エビデンス: <実装差分、architecture-change、ADR、コード、設定などの調査根拠>
- 更新ドキュメント: <作成または更新したdocs/architecture/\*.md>
- 不一致事項: <発見した不一致と解消状況。なければNot applicable>
- 未反映事項: <未反映事項と理由。なければNot applicable>
- 人間の意思決定: <Approved | Rejected | Pending>

## リリース判定

- レビュー範囲: <対象仕様書に対応するPR差分全体>
- 確認日時/コミット: <確認日時またはコミット>
- ゴール / ストーリー / 完成条件 / 成功指標: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- 技術要件 / 非機能要件 / ADR: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- 開発基盤: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- テスト / 回帰確認: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- 移行 / 設定 / 外部サービス / 権限 / 運用: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- 監視 / ログ / ロールバック / 復旧: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- アーキテクチャ整合性: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- 残存課題分類: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- 依存関係 / 外部コード / ライセンス: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- セキュリティ / プライバシー / アクセシビリティ / 法的要件: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- トレーサビリティ / ID整合性: <Pass | Blocked | Not applicable> — <根拠と未解決事項>
- 総合結果: <Pass | Blocked | Not applicable>
- 残存リスク: <残存リスク。なければNot applicable>
- 後続引き継ぎ: <移管先、理由、期限または対応条件。なければNot applicable>
- 人間の意思決定: <Approved | Rejected | Pending>

## リリース後検証

- 結果: <Pass | Failed | Pending | Not applicable>
- 完成条件 / 監視 / ログの確認: <確認内容>
- ロールバック/復旧結果: <実施結果。未実施ならNot applicable>

## 後続タスクの移管

- <Issueまたは次の仕様書候補への参照、理由、重要度、対応条件。なければNot applicable>

## 前提条件

- <タスク分解時の前提>

## 未解決事項

### 開発ブロック事項 (Blocking)

- <仕様書のAccepted化または実装開始を妨げる未確定事項>

### 保留事項 (Deferred)

- <判断条件、判断工程、影響するTask>
