# Structure

Status: Accepted
Status Reason: 人間が、`config.yaml`の厳格デコードと値検証をdomain、その他のYAML処理をinfrastructureへ配置する境界を承認した。

## Structure Principles

- CLIフレームワーク、対話UI、永続化方式をドメイン規則から分離する。
- 依存逆転を適用し、外部機能のインターフェースは利用規則を所有する側で定義する。
- パッケージは変更理由が同じ責務でまとめ、機能が存在しない段階で空の階層を作らない。
- 汎用的な便利関数ではなく、プロダクト上の責務が分かる境界を優先する。

## Directory Structure

```text
.
├── cmd/
│   └── context/                  # bootstrapで作成予定
├── internal/
│   ├── cli/                      # bootstrapで作成予定
│   ├── application/              # bootstrapで作成予定
│   ├── domain/                   # bootstrapで作成予定
│   └── infrastructure/           # bootstrapで作成予定
├── docs/
│   ├── engineering/              # 実在
│   └── ai-driven-development/    # 実在
├── flake.nix                     # 実在
├── package.json                  # 実在
└── Taskfile.yml                  # bootstrapで作成予定
```

## Directory Responsibilities

- `cmd/context/`: `main` パッケージを置き、依存の組み立てとプロセス開始だけを担当する。
- `internal/cli/`: コマンド解析、対話UI、引数と入出力、終了コード、CLI向けログ表示を担当する。
- `internal/application/`: `init`、`add`、`sync` などのユースケースを調整し、ユースケース固有のI/Oポートを定義する。
- `internal/domain/`: 配布物、選択状態、競合、検証などのプロダクト規則と、純粋なドメイン能力のインターフェースを定義する。
- `internal/infrastructure/`: ファイルシステム、YAML、XDG設定パス、ロック、原子的置換、時刻などのI/Oポートを実装する。
- `docs/engineering/`: 実装とレビューが参照する技術規範を置く。
- テストファイルは原則として対象パッケージと同じディレクトリへ置く。

## Module Boundaries

- `cmd/context` はビジネス規則を持たず、具象実装を組み立ててCLIを起動する。
- `internal/cli` はユーザー入力をapplicationの入力へ変換し、applicationの結果を表示と終了コードへ変換する。
- `internal/application` はユースケースの順序と整合性を管理するが、CLI、対話UI、YAML、OS固有APIへ依存しない。
- `internal/domain` はCLI、対話UI、`log/slog`、OSファイルAPIへ依存しない。例外として、全体設定のドメイン契約を一箇所で強制するため、`config.yaml`の厳格デコードと値検証に必要なYAMLライブラリへ依存できる。
- `internal/infrastructure` はapplicationまたはdomainで定義されたI/Oポートを実装するが、具象ユースケースやcliへ依存しない。
- ログが必要な場合は利用側で必要最小限のインターフェースを定義し、infrastructureまたはcliで `log/slog` に接続する。

## Dependency Direction

- 許可する主要な依存方向は `cli → application → domain` とする。
- `infrastructure` はapplicationが定義するI/Oポートとdomainの型にのみ依存する。
- `cmd/context` は依存の組み立てのため、cli、application、infrastructureを参照できる。
- domainから外側の層への依存は禁止する。
- infrastructureからapplicationの具象ユースケースまたはcliへの依存は禁止する。
- 循環依存は禁止する。
- 層をまたぐデータ型は、利用規則を所有する側へ配置する。

## Placement Rules

- コマンド、フラグ、プロンプト、標準入出力、終了コードは `internal/cli/` に置く。
- ユースケースの調整、トランザクション境界、中断処理は `internal/application/` に置く。
- 配布可否、競合判定、管理対象の識別など、外部技術に依存しない規則は `internal/domain/` に置く。
- `config.yaml`の厳格デコードと値検証は `internal/domain/` に置く。ファイルの読み書き、YAMLエンコード、`map.yaml`を含むその他のYAMLデコード、XDGパス解決、権限検証、ファイルロック、原子的置換は `internal/infrastructure/` に置く。
- 新規コードは、最も具体的な既存責務へ配置する。責務が一致しない場合は、依存方向を維持した新しいパッケージを定義する。
- 機能固有の構造は、関連PRDの `architecture-change.md` で設計してから追加する。

## Naming Rules

- Goパッケージ名は短い小文字とし、責務を表す単数形を優先する。
- Goファイル名は小文字のスネークケースを使用する。
- 公開識別子はGoの慣例に従い、名前だけで責務が分かる表現を使用する。
- `utils`、`common`、`helpers` など、責務が曖昧な汎用パッケージを作成しない。
- インターフェースは利用側で定義し、実装名ではなく必要な能力を表す。

## Generated / Vendor Files

- `go.sum`、`pnpm-lock.yaml`、`flake.lock` は手動編集せず、それぞれの依存管理コマンドで更新する。
- ロックファイルは関連する依存定義と同じ変更としてレビューする。
- 生成ファイルを追加する場合は、生成元、更新コマンド、レビュー対象を文書化する。
- vendoringは現時点では使用しない。導入する場合は理由と更新方法をADRまたは関連設計で決定する。

## Extension Guidelines

- 新しい外部技術はinfrastructureまたはcliの境界へ閉じ込め、domainへ漏らさない。`config.yaml`の厳格デコードに使用するYAMLライブラリだけは、前述の限定的な例外とする。
- 新しいサブコマンドは、既存ユースケースで表現できないユーザー成果がある場合に追加する。
- パッケージ追加時は、責務、依存先、利用者、テスト方法を説明できることを必須とする。
- 下位ディレクトリは、複数の実装が存在する、独立した変更理由がある、または依存制約を強制する必要がある場合に追加する。
- データ量または同時実行要件がYAMLの前提を超えた場合は、ストレージ境界を維持したままADRで代替案を評価する。

## Related Decisions

- `cli → application → domain` を基本とするレイヤー構造は、初回PRD開始前にADRで確定する。
- CLIコマンド解析と対話UIの配置境界は、関連PRDでライブラリを選定する際のADR候補とする。

## Assumptions

- `cmd/context/` と `internal/` 配下の主要パッケージはbootstrap工程で作成する。
- 初回bootstrapでは、空ディレクトリではなくビルドまたはテスト可能な最小単位だけを作成する。
- 初期版の機能数とチーム規模では、単一Goモジュールで十分である。

## Open Questions

### Blocking

- レイヤー構造の選択肢、却下理由、依存規則をADRで確定する。

### Deferred

- `internal` 配下を機能単位で細分化する条件は、最初のPRDのユースケースと変更頻度を確認して決定する。
- ファイルシステムの大文字小文字やマウント方式による同一性判定は、管理情報の要件を定義するPRDで決定する。
- 複数ファイル操作のロールバック責務と配置は、`add` または `sync` の要件策定時に決定する。
