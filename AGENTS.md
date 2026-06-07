# AGENTS.md

## 回答方針

- ユーザーへの回答は、日本語で簡潔かつ丁寧に記述する。

## リポジトリ構成

- `cmd/context/`: CLIのエントリーポイント。
- `internal/cli/`: コマンド解析、入出力、終了コードなどのCLI境界。
- `internal/application/`: ユースケースの調整。
- `internal/domain/`: 外部技術に依存しないドメイン規則。
- `internal/infrastructure/`: ファイルシステムや永続化などの外部I/O。
- `docs/`: プロダクト、要件、設計、技術判断、開発プロセスの文書。

## 文書の所在

- `docs/product.md`: プロダクト全体の現在の意図。
- `docs/AI-driven-development.md`: AI駆動開発のプロセス。
- `docs/engineering/`: 技術スタック、構造、開発規則。
- `docs/decisions/`: 技術判断を記録したADR。
- `docs/prds/`: PRDごとの要件、設計、タスク。
- `docs/architecture/`: 実装済みシステムの現在の設計。

## 開発コマンド

開発コマンドの正本は `Taskfile.yml` とする。

- `task test`: Goのテストを実行する。
- `task lint`: すべてのLintを実行する。
- `task ci`: ローカルのCI品質ゲートを実行する。

## Skills

作業内容に対応するSkillは `.codex/skills/` にある。
