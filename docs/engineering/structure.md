# ディレクトリ構造

Status: Accepted
Status Reason: Cobra + pflagを用いたCLIコマンド定義と、internal配下の独立した非公開パッケージ群によるシンプルな構造への移行を人間が承認した。

## 構造の原則

- CLIのコマンド解析、フラグ、入出力制御は `pkg/cmd/` に配置する。
- 外部プロジェクトに非公開としたい共通機能や特定ドメインのヘルパーモジュールは `internal/` 配下に独立したパッケージとして配置する。
- パッケージは変更理由が同じ責務でまとめ、コードが存在しない段階で空のディレクトリを作らない。

## ディレクトリ構造

```text
.
├── cmd/
│   └── context/                  # エントリーポイント（依存注入と実行開始）
├── pkg/
│   └── cmd/                      # CLIコマンド定義（Cobra）および Factory
└── internal/                     # 外部非公開のプロジェクト専用モジュール群
    ├── config/                   # ユーザー設定の探索、検証、安全な永続化
    └── repository/               # Context Repositoryの構造検証とパス正規化
```

## ディレクトリの責務

- `cmd/context/`: `main` パッケージを置き、`Factory` の作成、Cobraの `RootCmd` の呼び出し、プロセス実行開始（`Execute()`）を担当する。
- `pkg/cmd/`: 各種コマンド（`root`, `init` 等）の定義、フラグ・位置引数のパース、および `Factory` を介したI/Oフローの制御を担当する。
- `internal/config/`: `Store` が設定保存先の探索、既存設定の読み込み、期待値を用いた比較更新を統括する。YAMLスキーマ、ファイルシステム境界、判定可能な設定エラーもこのパッケージに閉じる。
- `internal/repository/`: `Validator` がContext Repositoryのパスを字句的に正規化し、必要な構造を検証する。

## モジュール境界と依存方向

- `cmd/` および `pkg/cmd/` は、必要に応じて `internal/` 配下の各パッケージをインポートして利用する。
- `internal/` 配下の各パッケージは、互いに独立した存在であり、循環依存を起こしてはならない。また、上位のCLIコマンド定義である `pkg/cmd/` に依存してはならない。
- 依存関係（入出力ストリーム、設定のローダー等）は、すべて `Factory`（`pkg/cmd/factory.go`）を経由して各コマンドに引き渡す。
- `Factory.Config` は `internal/config.Open` で生成した永続設定を遅延注入し、コマンドは設定ファイルやOS環境へ直接アクセスしない。
