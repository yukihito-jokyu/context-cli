# Engineering Foundation Update

## Responsibility

- `technology.md`: 技術スタック、実行環境、保存方式、技術制約。
- `structure.md`: ディレクトリ責務、モジュール境界、依存方向、配置規則。
- `development-rules.md`: 実装、テスト、セキュリティ、品質ゲートの規範。

## Sources

- 対応するEngineering Foundationテンプレート
- Accepted済みADR
- 実際のコード、設定、CI

## Update Rules

- 3ファイル間で同じ判断を重複説明せず、各責務へ分配する。
- 重要で不可逆な技術判断は本文だけで確定せず、ADRとの関係を明示する。
- 規範を変更した場合は対象ファイルを`Draft`へ戻す。事実同期だけなら既存Statusを維持できる。

## Verification

- 技術、構造、開発規則が相互に矛盾しない。
- コード、Taskfile、CI、依存関係と一致する。
- 例外には範囲、理由、リスク、期限または解除条件がある。
