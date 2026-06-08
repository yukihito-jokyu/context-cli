# Product Update

## Responsibility

- プロダクト全体の意図、対象ユーザー、価値、目標、原則、Current Focusを保持する。
- 個別PRDの実装詳細や一時的な解決策を含めない。

## Sources

- `docs/ai-driven-development/templates/product.md`
- 関連するAccepted済みPRD、ADR、現在の実装事実

## Update Rules

- Issueがプロダクト全体の方向、Scope、前提を変更する場合だけ更新する。
- 個別機能の判断はPRD以下へ置き、productには横断的な結論だけ反映する。
- 意味を変更した場合は`Draft`へ戻し、再レビュー理由を`Status Reason`へ記録する。

## Verification

- Goal、Target User、Value、Principles、Current Focusに矛盾がない。
- 既存PRDが新しい上位方針と両立するか影響を列挙する。
