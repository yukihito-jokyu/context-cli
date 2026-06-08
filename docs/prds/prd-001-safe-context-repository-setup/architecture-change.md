# Architecture Change

Status: Review
Status Reason: PRD-001の全実装TaskがCompletedまたはCompletion Candidateとなり、context initの実装結果を設計へ反映したため、Implemented確定前の確認を求める。
PRD ID: prd-001-safe-context-repository-setup。

## Source

- PRD: ./prd.md
- Backlog: ./backlog.md
- Requirements: ./requirements.md

## Target Stories

- ST-001: 検証済みContext Repositoryを初回設定する
- ST-002: 確認してContext Repositoryの設定を変更する
- ST-003: 同じContext Repositoryへの再実行を安全に完了する
- ST-004: 安全に利用できないContext Repositoryの設定を拒否する
- ST-005: 未対応スキーマの既存設定を安全に扱う

## Related PRDs and Coordination

- Related PRD: Not applicable
- Conflict Risk: Not applicable
- Integration Order: Not applicable

## Change Summary

本変更では、個人開発者がContext Repositoryを安全に登録・変更できるようにするため、サブコマンド `context init <path>` の処理フローと構成コンポーネントを実装する。具体的には、リポジトリ構造と権限の検証（シンボリックリンク排除、グループ・他者書き込み拒否等）、既存設定（`config.yaml`）のスキーマ整合性チェック、更新時の排他ロックと原子的更新を担う各レイヤーを新規に実装する。

## API Changes

- CLIインターフェース（サブコマンド）の追加:
  - コマンド: `context init <path>`
  - 引数: `path`（必須。設定対象のローカルContext Repositoryパス）
  - 振る舞い:
    - 指定パスの同一性、検証結果を評価。
    - 設定変更時は既存設定と変更予定を表示し、標準入力での承認確認を求める（インタラクティブ）。
    - 成功時は検証結果と設定内容を標準出力へ出力し、終了コード `0` で終了。
    - 失敗時はエラー内容を標準エラーへ出力し、非ゼロの終了コードで終了。

## Database Changes

- Not applicable（データベースは使用せず、YAMLファイルへ永続化する）

## Domain Changes

- **設定データモデル (`internal/domain/config.go`)**:
  - `Config` 構造体: 永続化対象となる全体設定データモデル。
    - `Version` (int): スキーマバージョン（初期値: `1`）。
    - `RepositoryPath` (string): 正規化された絶対パス文字列。
  - `config.yaml`のYAML厳格デコード（未知フィールド拒否）および妥当性検証ルールを定義。
- **リポジトリ構造・権限バリデータ (`internal/domain/repository.go`)**:
  - `FileSystem` インターフェース: 外部I/Oに依存しないように定義するファイルシステム操作の抽象化。
    - `LStat(ctx context.Context, path string) (FileStatus, error)`
    - `ReadDir(ctx context.Context, path string) ([]FileEntry, error)`
  - `FileStatus` インターフェース: ファイル情報（ディレクトリ/レギュラーファイル/シンボリックリンク判定、他者書き込み権限の有無、読み取り/検索可能性）を表す。
  - `RepositoryValidator` サービス: `FileSystem` を用いて、対象リポジトリの構造と権限が信頼境界（ADR-003, ADR-005）を満たすかを検証する。
    - 検出したすべての構造不備・権限エラーを収集し、リポジトリ相対パスとエラーメッセージのリストとして返却する。

## Package Changes

- `internal/domain/` パッケージ配下: `config.go`, `repository.go` の新規追加。
- `internal/application/` パッケージ配下: `init_repository.go` (UseCase), `ports.go` (Persistence/UI抽象) の新規追加。
- `internal/infrastructure/fs/` パッケージ配下: OSのファイル操作を用いた `domain.FileSystem` の具象実装を追加。
- `internal/infrastructure/yaml/` パッケージ配下: domainの`config.yaml`デコードを利用し、排他ロックと一時ファイル原子置換を伴う `Config` 永続化の具象実装を追加。
- `internal/cli/` パッケージ配下: `init.go` (initコマンドハンドラ、プロンプト処理) の新規追加。
- `cmd/context/` パッケージ配下: 設定永続化、ローカルファイルシステム、対話UI、ユースケースを組み立て、割り込み可能なCLIを起動するエントリーポイントを実装。

## External Integration and Infrastructure Changes

- **設定ファイル永続化 (`internal/infrastructure/yaml`)**:
  - 設定ディレクトリ: `XDG_CONFIG_HOME/context/` （未設定時は `~/.config/context/`）。新規作成時は権限 `0700`。
  - 設定ファイル: `config.yaml`。権限 `0600`。既存のディレクトリ/ファイルの権限がこれより広い場合は暗黙の変更をせずエラー終了（NFR-002）。
  - 排他制御: ファイル書き込み時は同一ディレクトリ内にロックファイル（例: `config.yaml.lock`）を作成し、排他ロック（`flock` 等）を取得。ロック取得失敗時は待機せず即座にエラー終了。
  - 原子的更新: 同一ディレクトリ内に一時ファイル（例: `config.yaml.tmp`）を書き出し、`os.Rename` を用いて原子的に置換。

## Error Handling Changes

- domain/applicationレイヤーでは判定可能な型/値（カスタムエラー型等）でエラーを生成・伝播。
- cliレイヤーの境界でエラーを一括して人間可読な形式（英語メッセージ）に変換し、標準エラーに出力して非ゼロコードでプロセスを終了。
- エラーおよびログには設定内容全体、ファイル内容、秘密情報、不要なユーザーパスを含めない（NFR-006）。

## Observability Changes

- `log/slog` を使用し、CLI向けのカスタムログハンドラを構成。
- CLI向けログハンドラは時刻と属性を出力せず、レベルとメッセージだけを標準エラーへ出力する。
- 診断ログ（デバッグレベル）は明示的なデバッグ指定時のみ標準エラーに出力。
- エラーと警告は標準エラーへ出力。通常の処理結果は標準出力へ出力。

## Migration Plan

- 初回リリースのため、既存データの移行処理は不要。

## Compatibility and Deprecation

- Breaking Changes: Not applicable (初回リリースのため破壊的変更なし)
- Migration Path: Not applicable
- Deprecation Conditions: Not applicable
- Rollback / Recovery: 設定処理の中断・失敗時は一時ファイルを確実に削除し、既存の設定ファイルを完全に維持した状態で終了する（NFR-007）。

## Security and Privacy Impact

- **信頼境界の強制 (ADR-005)**:
  - 指定されたパスから親ディレクトリを含む全てのパス構成要素について `lstat` を行い、シンボリックリンクが含まれる場合は拒否する。
  - 構成要素またはリポジトリ内ファイルにグループ・他者への書き込み権限（`0022` などのマスクに反する権限）がある場合は拒否する。
- **設定ファイルの安全保護 (NFR-002)**:
  - 設定ディレクトリは `0700`、ファイルは `0600` で作成・検証し、他者からの読み書きを防止。

## Impact

- 影響範囲: bootstrap直後のため、本変更による他機能への影響はありません。

## Related Decisions

- [ADR-001: レイヤー構造と依存方向](file:///Users/yukihito/Documents/github_projects/context-cli/docs/decisions/adr-001-layer-architecture-and-dependency-direction.md)
- [ADR-002: YAML永続化方式と分割境界](file:///Users/yukihito/Documents/github_projects/context-cli/docs/decisions/adr-002-yaml-persistence-and-storage-boundaries.md)
- [ADR-003: Context Repository構造の宣言方式](file:///Users/yukihito/Documents/github_projects/context-cli/docs/decisions/adr-003-context-repository-structure-specification.md)
- [ADR-004: Context Repositoryの同一性の判定方式](file:///Users/yukihito/Documents/github_projects/context-cli/docs/decisions/adr-004-context-repository-identity-determination.md)
- [ADR-005: Context Repositoryのファイルシステム信頼境界](file:///Users/yukihito/Documents/github_projects/context-cli/docs/decisions/adr-005-context-repository-filesystem-trust-boundary.md)

## Updates to Current Architecture

- 実装完了後、新規作成されるドキュメント `docs/architecture/system-overview.md` にて全体のレイヤー構成と `context init` の制御/データフローを記述する。

## Assumptions

- Context Repositoryはローカルにクローン済みである。
- 単一ユーザーによるローカル実行であり、複数プロセスによる頻繁な並行書き込み競合は想定しない（ただし排他ロックによる防御は行う）。

## Open Questions

### Blocking

- Not applicable (必要な前提・判断はADRにより解消済み)

### Deferred

- **Windows対応**: パス区切り文字、権限表現（chmod）、排他ロック（flock）、一時ファイルの原子的置換（Rename）のWindows互換性評価は、Windows対応を定義する将来of将来のPRDへ保留する。
- **CLIコマンド解析ライブラリ**: `context init` 単体の時点では標準ライブラリ `flag` または単純な引数スライス走査で実装し、将来サブコマンドが増大する段階で正式なライブラリ選定とADRを作成する。
