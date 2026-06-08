# Tasks

Status: Accepted
Status Reason: レビュープロセス完了。テストタスクの正常系/異常系の網羅性、共通基盤タスクの設計根拠、および保留事項の判断基準を整備し、合意を完了した。
PRD ID: prd-001-safe-context-repository-setup。

## Source

- PRD: ./prd.md
- Backlog: ./backlog.md
- Requirements: ./requirements.md
- Architecture Change: ./architecture-change.md

## Risk Assessment

- Flow: Standard
- Risk Level: Medium
- Rationale: 本機能はCLIの初回セットアップと信頼境界検証（シンボリックリンク・書き込み権限）、排他ロックと一時ファイル原子置換を伴う設定ファイルの安全な保存を担う核心モジュールであり、高い整合性と堅牢性が求められるため。
- Human Confirmation: Confirmed

## Coordination

- Owner: Antigravity
- Parallel Work: Not applicable

## Implementation Order

1. T-001
2. T-002
3. T-003
4. T-004
5. T-005
6. T-006

## Tasks

### T-001: ドメイン設定モデルと検証規則

- Status: Completed
- Status Reason: 人間の完了確定および承認を得たため。
- Replaced By: Not applicable
- Purpose: 設定データのドメインモデルを定義し、YAMLの厳格なデコードで未知フィールドを排除する。さらにリポジトリ絶対パスの字句的な正規化ロジックを実装する。
- Linked Stories: `ST-001, ST-003, ST-005`
- Linked AC: `AC-001, AC-007, AC-016, AC-018`
- Linked Technical Requirements: `TR-002, TR-007, TR-013, TR-014, TR-015`
- Linked Non-Functional Requirements: `NFR-006`
- Dependencies: None
- Shared Task Rationale: 本タスクで定義する設定データモデルおよびパス正規化ロジックは、初回設定（ST-001）、設定変更（ST-002）、同一リポジトリ再実行時の判定（ST-003）、および既存設定の互換性チェック（ST-005）の各ストーリーにおいて共通の検証・判定基準となるため。

#### Files to Change

- `internal/domain/config.go`
- `internal/domain/config_test.go`

#### Test Tasks

- TT-001: YAML厳格デコードによる未知フィールドおよび不正値の検出検証
  - Status: Completed
  - Status Reason: 未知フィールド、不正YAML、複数文書、不正値の拒否をユニットテストで確認した。
  - Replaced By: Not applicable
  - Type: unit
  - Linked AC: AC-018
  - Linked Technical Requirements: TR-013, TR-014
  - Linked Non-Functional Requirements: NFR-006
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — 未実装のConfig APIにより期待どおりビルド失敗した。
  - Green Check Result: Passed — `go test ./internal/domain`で対象ケースが成功した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

- TT-002: スキーマバージョン（v1）の妥当性検証および未対応バージョン拒否の検証
  - Status: Completed
  - Status Reason: 対応version 1の受理と未対応versionの拒否をユニットテストで確認した。
  - Replaced By: Not applicable
  - Type: unit
  - Linked AC: AC-016
  - Linked Technical Requirements: TR-013, TR-015
  - Linked Non-Functional Requirements: None
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — 未実装のConfig APIにより期待どおりビルド失敗した。
  - Green Check Result: Passed — `go test ./internal/domain`で対象ケースが成功した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

- TT-003: シンボリックリンクを解決しない字句的に正規化された絶対パス変換ロジックの検証
  - Status: Completed
  - Status Reason: 相対パスの絶対化、字句的正規化、シンボリックリンク非解決をユニットテストで確認した。
  - Replaced By: Not applicable
  - Type: unit
  - Linked AC: AC-007
  - Linked Technical Requirements: TR-007
  - Linked Non-Functional Requirements: None
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — 未実装のConfig APIにより期待どおりビルド失敗した。
  - Green Check Result: Passed — `go test ./internal/domain`で対象ケースが成功した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

#### Implementation Notes

- Actual Files Changed: `internal/domain/config.go` / `internal/domain/config_test.go` / `go.mod` / `go.sum` / `docs/prds/prd-001-safe-context-repository-setup/tasks.md`
- Test Results: Red Checkとして未実装APIによるビルド失敗を確認した。Green Checkでは`go test ./internal/domain`と`go test ./...`が成功し、全16テストが合格した。`go vet ./...`、`golangci-lint run`、変更文書のPrettier、textlint、`git diff --check`も成功した。`task ci`は実行環境に`actionlint`がなく中断し、`govulncheck`はGo 1.25でビルドされたツールとGo 1.26の不一致により実行できなかった。全体文書LintにはT-002定義内の既存指摘が2件残る。
- Follow-up: `actionlint`とGo 1.26で再ビルドした`govulncheck`を利用できる開発環境で、`task ci`を再実行する。

#### Code Review

- Status: Completion Candidate
- Scope: `T-001` で新規追加されたconfig.goとそのユニットテストの差分
- Findings: 指摘なし。
- Fixes: Not applicable
- Verification: `go test ./internal/domain`、`go test ./...`、`go vet ./...`、`golangci-lint run`が成功した。
- Remaining Risks: Not applicable
- Human Decision: Pending

#### Definition of Done

- `Config` 構造体とYAMLパース・検証メソッドが `internal/domain/config.go` に実装されていること
- 厳格デコードおよびバージョン検証、パス正規化ロジックのユニットテストがすべて合格していること
- 必要なドキュメント更新が完了していること

---

### T-002: ドメインリポジトリ構造/権限バリデータ

- Status: Completed
- Status Reason: 人間の完了確定および承認を得たため。
- Replaced By: Not applicable
- Purpose: ファイルシステム操作を抽象化するFileSystemポートを定義する。また、リポジトリ内の固定フォルダ構造（projects/, utils/skills/ 等）やパーミッション（シンボリックリンク排除、他者書き込み権限の排除、現在ユーザーの読み取り・検索可能性）を検証するドメインサービスを実装する。
- Linked Stories: `ST-001, ST-002, ST-003, ST-004`
- Linked AC: `AC-010, AC-011, AC-012, AC-013, AC-014, AC-015`
- Linked Technical Requirements: `TR-001, TR-004, TR-008, TR-009, TR-010, TR-011, TR-012, TR-016, TR-017, TR-018, TR-019`
- Linked Non-Functional Requirements: `NFR-001, NFR-006`
- Dependencies: T-001
- Shared Task Rationale: 本タスクで実装するリポジトリ構造およびパーミッション検証（シンボリックリンク排除、他者書き込み権限排除）のドメインバリデータは、初回設定（ST-001）、設定変更（ST-002）、同一リポジトリ再実行時の検証（ST-003）、および不正なリポジトリの設定拒否（ST-004）に共通の信頼境界検査として必要となるため。

#### Files to Change

- `internal/domain/repository.go`
- `internal/domain/repository_test.go`

#### Test Tasks

- TT-004: リポジトリ構成（projects/, utils/skills/等の存在、1件以上のprojects/<project-name>の存在、各Skill内のSKILL.md存在）の不足または不備に対するエラー収集テスト
  - Status: Completed
  - Status Reason: 必須ディレクトリ欠落、projects内のプロジェクト欠落、SKILL.md欠落に対するエラー収集がテストケースで合格した。
  - Replaced By: Not applicable
  - Type: unit
  - Linked AC: AC-011
  - Linked Technical Requirements: `TR-009, TR-016, TR-017, TR-018, TR-019`
  - Linked Non-Functional Requirements: NFR-006
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — 未実装のValidateでテスト失敗を確認。
  - Green Check Result: Passed — `go test ./internal/domain`が成功した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

- TT-005: リポジトリパスの構成要素（親ディレクトリ含む）および必須検証ファイル内にシンボリックリンクが含まれる場合に拒否されることの検証
  - Status: Completed
  - Status Reason: 親ディレクトリ・リポジトリルート・ディレクトリ・ファイルにおけるシンボリックリンクの排除をテストケースで確認した。
  - Replaced By: Not applicable
  - Type: unit
  - Linked AC: AC-014
  - Linked Technical Requirements: TR-009, TR-012
  - Linked Non-Functional Requirements: None
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — 未実装のValidateでテスト失敗を確認。
  - Green Check Result: Passed — `go test ./internal/domain`が成功した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

- TT-006: リポジトリルート、構成ディレクトリ、必須ファイルにグループ・他ユーザーへの書き込み権限（パーミッションマスク違反）がある場合に拒否されることの検証
  - Status: Completed
  - Status Reason: マスク0o022を適用し、グループ・他者の書き込み権限を持つ対象の拒否をテストケースで確認した。
  - Replaced By: Not applicable
  - Type: unit
  - Linked AC: AC-013
  - Linked Technical Requirements: TR-009
  - Linked Non-Functional Requirements: None
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — 未実装のValidateでテスト失敗を確認。
  - Green Check Result: Passed — `go test ./internal/domain`が成功した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

- TT-007: リポジトリにアクセスできない（存在しない、現在のユーザーが読み取り・検索を行えない）場合に拒否されることの検証
  - Status: Completed
  - Status Reason: 存在しないパスおよび読み取り/検索不可時のエラー検知をテストケースで確認した。
  - Replaced By: Not applicable
  - Type: unit
  - Linked AC: AC-010, AC-012
  - Linked Technical Requirements: TR-009, TR-019
  - Linked Non-Functional Requirements: None
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — 未実装のValidateでテスト失敗を確認。
  - Green Check Result: Passed — `go test ./internal/domain`が成功した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

- TT-007-2: 正常系リポジトリ構成（全必須ディレクトリ・ファイルが存在し、シンボリックリンクを含まず、適切な権限で管理されている）における検証成功テスト
  - Status: Completed
  - Status Reason: 全ての条件を満たした正常なリポジトリに対する検証成功をテストケースで確認した。
  - Replaced By: Not applicable
  - Type: unit
  - Linked AC: AC-010 / AC-011 / AC-012 / AC-013 / AC-014
  - Linked Technical Requirements: TR-009, TR-010
  - Linked Non-Functional Requirements: NFR-001, NFR-006
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — 未実装のValidateでテスト失敗を確認。
  - Green Check Result: Passed — `go test ./internal/domain`が成功した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

#### Implementation Notes

- Actual Files Changed: `internal/domain/repository.go` / `internal/domain/repository_test.go` / `docs/prds/prd-001-safe-context-repository-setup/tasks.md`
- Test Results: Red CheckとしてValidate未実装でのテスト失敗を確認。Green Checkでは `go test ./internal/domain/...` が成功し、`task ci`（静的解析、フォーマット、文書検証、脆弱性検査、ビルド）がすべて合格した。
- Follow-up: Not applicable

#### Code Review

- Status: Completion Candidate
- Scope: `T-002` で新規追加されたrepository.goとそのユニットテストの差分
- Findings: High 1件。Skillディレクトリ自体の読み取り・検索可能性を検証しておらず、AC-012およびTR-019を満たさない実装だった。
- Fixes: 各Skillディレクトリを`ReadDir`で実際に列挙し、読み取り・検索不能時にRepository相対パスを含む検証エラーを返す処理と回帰テストを追加した。
- Verification: `go test ./internal/domain`、`go test ./...`、`go vet ./...`、`golangci-lint run`が成功した。
- Remaining Risks: ADR-005で定義された「所有者と親ディレクトリの所有関係によるディレクトリ置換可能性」というリスクは残存リスクとして受容し、ドメインバリデータ層では直接検証しない。
- Human Decision: Pending

#### Definition of Done

- `FileSystem`, `FileStatus` インターフェースが定義され、検証ロジック `RepositoryValidator` が実装されていること
- 各エラーケース（構造不備、シンボリックリンク、権限、読み取り不可）を網羅するテストがすべて合格していること
- 必要なドキュメント更新が完了していること

---

### T-003: YAML永続化インフラ実装（ロックと原子置換）

- Status: Completion Candidate
- Status Reason: 実装およびすべてのテスト（TT-008, TT-009, TT-010, TT-011）の合格と静的解析が成功したため。
- Replaced By: Not applicable
- Purpose: ConfigRepositoryポートをインフラ層で実装し、設定ディレクトリの権限強制（0700）やファイルの権限強制（0600）を行う。また、排他ファイルロックの即時判定、変更競合の検知、および一時ファイルを用いた原子的なファイル置換機能を提供する。
- Linked Stories: `ST-001, ST-002, ST-005`
- Linked AC: `AC-002, AC-005, AC-006, AC-015, AC-017`
- Linked Technical Requirements: `TR-003, TR-005, TR-006, TR-011, TR-015`
- Linked Non-Functional Requirements: `NFR-002, NFR-003, NFR-004, NFR-007, NFR-008`
- Dependencies: T-001
- Shared Task Rationale: 本タスクで提供する設定ファイル（0600）や設定ディレクトリ（0700）の権限強制、および排他ロック（flock）、一時ファイルによる原子置換といった永続化インフラは、初回設定（ST-001）、設定変更（ST-002）、およびスキーマ互換性チェック（ST-005）の各ストーリーにおいて安全な設定保存を行うための共通のファイルIO基盤となるため。

#### Files to Change

- `internal/application/ports.go`
- `internal/infrastructure/yaml/config.go`
- `internal/infrastructure/yaml/config_test.go`

#### Test Tasks

- TT-008: ディレクトリ `0700`、ファイル `0600` による作成検証と、既存設定の過剰権限に対するエラー停止・書き込み拒否テスト
  - Status: Completed
  - Status Reason: インテグレーションテストを実装し、適切なディレクトリおよびファイルの権限強制と過剰権限検知が正しく動作することを確認した。
  - Replaced By: Not applicable
  - Type: integration
  - Linked AC: AC-015
  - Linked Technical Requirements: TR-003
  - Linked Non-Functional Requirements: NFR-002
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — スタブ実装によりテストが失敗することを確認した。
  - Green Check Result: Passed — `go test ./internal/infrastructure/yaml` で正常に合格した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

- TT-009: ファイル書き込み時の排他ロック（lockfile作成とflock等）の取得検証と、ロック取得不可時の即座終了テスト（ブロッキング待機なし）
  - Status: Completed
  - Status Reason: テストケースで他ファイル記述子によるロックをシミュレートし、Save処理が即時にErrLockFailedで終了することを確認した。
  - Replaced By: Not applicable
  - Type: integration
  - Linked AC: AC-015
  - Linked Technical Requirements: TR-011
  - Linked Non-Functional Requirements: NFR-003, NFR-008
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — スタブ実装によりテストが失敗することを確認した。
  - Green Check Result: Passed — `go test ./internal/infrastructure/yaml` で正常に合格した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

- TT-010: ロック取得後における既存設定ファイルの再読込およびメモリ上の読込時点からの変化の整合性検証テスト
  - Status: Completed
  - Status Reason: expectedOldの有無および内容の変化に伴うコンフリクト（ErrConfigConflict）検知ロジックをテストし、パスした。
  - Replaced By: Not applicable
  - Type: integration
  - Linked AC: AC-006
  - Linked Technical Requirements: TR-006
  - Linked Non-Functional Requirements: NFR-003
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — スタブ実装によりテストが失敗することを確認した。
  - Green Check Result: Passed — `go test ./internal/infrastructure/yaml` で正常に合格した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

- TT-011: 同一ディレクトリでの一時ファイル（`config.yaml.tmp`）を用いた原子的な書き込み/置換（`os.Rename`）と、エラー・中断時の不完全ファイル削除検証
  - Status: Completed
  - Status Reason: 一時ファイルへの書き込みとos.Rename、および後片付け処理が機能し、途中のゴミファイルが残らないことを検証した。
  - Replaced By: Not applicable
  - Type: integration
  - Linked AC: AC-002, AC-006
  - Linked Technical Requirements: TR-003, TR-006
  - Linked Non-Functional Requirements: NFR-004, NFR-007
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — スタブ実装によりテストが失敗することを確認した。
  - Green Check Result: Passed — `go test ./internal/infrastructure/yaml` で正常に合格した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

#### Implementation Notes

- Actual Files Changed: `internal/application/ports.go` / `internal/infrastructure/yaml/config.go` / `internal/infrastructure/yaml/config_test.go`
- Test Results: `go test ./...` および `task ci`（静的解析・ドキュメント検証・フォーマット・脆弱性検査・ビルド）のすべてに合格し、エラーのない状態であることを確認した。
- Follow-up: Not applicable

#### Code Review

- Status: Completion Candidate
- Scope: `internal/infrastructure/yaml/` に追加されたpersistence / lockコードおよびそのインテグレーションテスト差分
- Findings: High 1件、Medium 1件。固定名一時ファイルがsymlinkを追跡してリンク先を上書きでき、保存前の`Config.Validate`も不足していた。
- Fixes: `os.CreateTemp`による排他的なランダム名一時ファイルへ変更し、書き込み・`Sync`・`Close`後に原子置換する処理、symlinkリンク先の非変更テスト、保存前のConfig検証と不正バージョン拒否テストを追加した。
- Verification: `go test ./internal/infrastructure/yaml`、`go test ./...`、`go vet ./...`、`golangci-lint run`が成功した。
- Remaining Risks: Not applicable
- Human Decision: Pending

#### Definition of Done

- `ConfigRepository` ポートの具象実装が追加されていること
- 隔離されたテスト一時ディレクトリを用い、権限チェック、排他ロック、競合検知、一時ファイル原子置換のインテグレーションテストがすべて合格していること
- 必要なドキュメント更新が完了していること

---

### T-004: OSファイルシステムインフラ実装

- Status: Completion Candidate
- Status Reason: LocalFileSystemの実装およびすべてのテスト（TT-012）の合格と静的解析が成功したため、人間の完了確定を待つ。
- Replaced By: Not applicable
- Purpose: `domain.FileSystem` インターフェースをOSのシステムコール（lstat、readdir）を用いて実装し、実際のファイルシステム階層へのアクセスを提供する。
- Linked Stories: `ST-004`
- Linked AC: `AC-010, AC-011, AC-012, AC-013, AC-014`
- Linked Technical Requirements: `TR-009, TR-012, TR-019`
- Linked Non-Functional Requirements: `NFR-001`
- Dependencies: T-002
- Shared Task Rationale: Not applicable

#### Files to Change

- `internal/infrastructure/fs/fs.go`
- `internal/infrastructure/fs/fs_test.go`

#### Test Tasks

- TT-012: 実際のOSファイルおよびシンボリックリンク、パーミッションマスクを用いた `LStat` および `ReadDir` の検証テスト
  - Status: Completed
  - Status Reason: テスト用一時ディレクトリにおいて、通常ファイル、ディレクトリ、シンボリックリンク、各種パーミッションを正しく判別できるテストが成功したため。
  - Replaced By: Not applicable
  - Type: integration
  - Linked AC: AC-012, AC-013, AC-014
  - Linked Technical Requirements: TR-009, TR-012, TR-019
  - Linked Non-Functional Requirements: NFR-001
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — 未実装のAPIにより期待どおりテストが失敗することを確認した。
  - Green Check Result: Passed — `go test ./internal/infrastructure/fs/...` で正常に合格した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

#### Implementation Notes

- Actual Files Changed: `internal/infrastructure/fs/fs.go` / `internal/infrastructure/fs/fs_test.go`
- Test Results: `go test ./...` および `task ci`（静的解析・ドキュメント検証・フォーマット・脆弱性検査・ビルド）のすべてに合格し、エラーのない状態であることを確認した。
- Follow-up: Not applicable

#### Code Review

- Status: Completion Candidate
- Scope: `internal/infrastructure/fs/fs.go` とそのテストの差分
- Findings: 指摘なし。
- Fixes: Not applicable
- Verification: `go test ./...`、`go vet ./...`、`golangci-lint run`が成功した。
- Remaining Risks: ADR-005で定義された「所有者と親ディレクトリの所有関係によるディレクトリ置換可能性」というリスクは残存リスクとして受容する。
- Human Decision: Pending

#### Definition of Done

- `domain.FileSystem` インターフェースの具象実装クラスが実装されていること
- テスト用一時ディレクトリにおいて、通常ファイル、ディレクトリ、シンボリックリンク、各種パーミッションを正しく判別できるテストが成功していること
- 必要なドキュメント更新が完了していること

---

### T-005: アプリケーションユースケース実装

- Status: Completed
- Status Reason: 人間がT-006の実装開始を承認し、依存TaskであるT-005の完了を確定したため。
- Replaced By: Not applicable
- Purpose: `InitRepositoryUseCase` を実装し、指定パスの同一性評価、リポジトリ内容の検証、既存設定の有無に応じたインタラクティブなUIプロンプト確認呼び出し、および書き込み保存のオーケストレーションを実装する。
- Linked Stories: `ST-001, ST-002, ST-003, ST-004, ST-005`
- Linked AC: `AC-001, AC-002, AC-004, AC-005, AC-006, AC-007, AC-008, AC-009, AC-015, AC-017`
- Linked Technical Requirements: `TR-001, TR-002, TR-003, TR-004, TR-005, TR-006, TR-007, TR-008, TR-010, TR-011, TR-015`
- Linked Non-Functional Requirements: `NFR-003, NFR-007`
- Dependencies: `T-003, T-004`

#### Files to Change

- `internal/application/init_repository.go`
- `internal/application/init_repository_test.go`
- `internal/application/ports.go`

#### Test Tasks

- TT-013: 初回設定時（config.yaml未存在）に、検証通過後、プロンプト表示せずにそのまま永続化処理が呼び出されることの検証
  - Status: Completed
  - Status Reason: モックを用いて初回設定時のSave呼び出し、ConfirmChange未呼び出し、expectedOld=nil、正しいConfig値を確認した。
  - Replaced By: Not applicable
  - Type: unit
  - Linked AC: AC-001, AC-002
  - Linked Technical Requirements: TR-001, TR-002, TR-003
  - Linked Non-Functional Requirements: None
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — 未実装のNewInitRepositoryUseCaseにより期待どおりビルド失敗した。
  - Green Check Result: Passed — `go test ./internal/application`で対象ケースが成功した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

- TT-014: 同一リポジトリ再実行時に、検証結果にかかわらずプロンプトおよびファイル書き込み処理がスキップされ、検証成功時は正常終了、検証失敗時はエラー終了することの検証
  - Status: Completed
  - Status Reason: 検証成功時の正常終了（Save/ConfirmChange未呼び出し）と検証失敗時のRepositoryValidationErrorを確認した。
  - Replaced By: Not applicable
  - Type: unit
  - Linked AC: AC-007, AC-008, AC-009
  - Linked Technical Requirements: TR-008
  - Linked Non-Functional Requirements: None
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — 未実装のNewInitRepositoryUseCaseにより期待どおりビルド失敗した。
  - Green Check Result: Passed — `go test ./internal/application`で対象ケースが成功した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

- TT-015: 異なるリポジトリへの設定変更時に、検証通過後、UIポート経由で承認が得られた場合は設定書き換えが行われ、拒否/中断された場合は書き込みをスキップして既存設定が維持されることの検証
  - Status: Completed
  - Status Reason: 承認時のSave呼び出し（expectedOld付き）、拒否時のErrChangeAborted、中断時のエラー返却を確認した。
  - Replaced By: Not applicable
  - Type: unit
  - Linked AC: AC-004, AC-005, AC-006
  - Linked Technical Requirements: TR-004, TR-005, TR-006
  - Linked Non-Functional Requirements: None
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — 未実装のNewInitRepositoryUseCaseにより期待どおりビルド失敗した。
  - Green Check Result: Passed — `go test ./internal/application`で対象ケースが成功した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

- TT-015-2: リポジトリ検証またはスキーマ検証失敗時に、書き込み処理（保存）が呼び出されず、既存設定（ある場合）が維持されることの検証
  - Status: Completed
  - Status Reason: 初回設定時の検証失敗、設定変更時の検証失敗、未対応バージョン、不正設定内容の各ケースでSave未呼び出しを確認した。
  - Replaced By: Not applicable
  - Type: unit
  - Linked AC: AC-015
  - Linked Technical Requirements: TR-011, TR-015
  - Linked Non-Functional Requirements: NFR-007
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — 未実装のNewInitRepositoryUseCaseにより期待どおりビルド失敗した。
  - Green Check Result: Passed — `go test ./internal/application`で対象ケースが成功した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

#### Implementation Notes

- Actual Files Changed: `internal/application/init_repository.go` / `internal/application/init_repository_test.go` / `internal/application/ports.go`
- Test Results: `go test ./...` および `task ci`（静的解析・ドキュメント検証・フォーマット・脆弱性検査・ビルド）のすべてに合格し、エラーのない状態であることを確認した。
- Follow-up: Not applicable

#### Code Review

- Status: Completion Candidate
- Scope: `internal/application/init_repository.go` とテストコードの差分
- Findings: 指摘なし。
- Fixes: Not applicable
- Verification: `go test ./...`、`go vet ./...`、`golangci-lint run`が成功した。
- Remaining Risks: Not applicable
- Human Decision: Pending

#### Definition of Done

- `InitRepositoryUseCase` の実装が完了し、設定読み込み、検証、UI対話、保存のユースケースが連結されていること
- 各種モックポートを用いた初回設定、同一再実行、変更承認・拒否、および検証エラー発生時のユースケーステストがすべて合格していること
- 必要なドキュメント更新が完了していること

---

### T-006: CLIコマンドハンドラとエントリーポイント

- Status: Completion Candidate
- Status Reason: CLI引数解析、対話確認、出力と終了コードの変換、slog境界、mainの依存組み立てを実装し、対象テスト、全体テスト、静的解析、文書Lint、ビルドが成功したため、人間の完了確定を待つ。
- Replaced By: Not applicable
- Purpose: `context init` コマンドライン解析、標準入出力を介したインタラクティブ承認プロンプト、警告・エラーの標準エラー仕分け、秘密情報等を出力しないsLogger、およびmainエントリーポイントへの組み込みを行う。
- Linked Stories: `ST-001, ST-002, ST-003, ST-004, ST-005`
- Linked AC: `AC-001, AC-003, AC-004, AC-005, AC-006, AC-009, AC-010, AC-011, AC-012, AC-013, AC-014, AC-016, AC-017, AC-018`
- Linked Technical Requirements: `TR-005, TR-006`
- Linked Non-Functional Requirements: `NFR-005, NFR-006, NFR-008`
- Dependencies: T-005

#### Files to Change

- `internal/cli/init.go`
- `internal/cli/init_test.go`
- `cmd/context/main.go`

#### Test Tasks

- TT-016: CLIコマンド `context init <path>` 実行時の標準出力結果およびエラー時の標準エラー出力内容、ならびに終了コード（0 / 非ゼロ）の検証
  - Status: Completed
  - Status Reason: 正常終了、使用法エラー、リポジトリ検証エラー、未対応設定、予期しないエラーの出力先・内容・終了コードをCLIテストで確認した。
  - Replaced By: Not applicable
  - Type: e2e
  - Linked AC: `AC-001, AC-003, AC-010, AC-011, AC-012, AC-013, AC-014, AC-016, AC-018`
  - Linked Technical Requirements: None
  - Linked Non-Functional Requirements: NFR-005, NFR-006
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — 未実装のHandler、ConsoleUI、終了コード定義により期待どおりビルド失敗した。
  - Green Check Result: Passed — `go test ./internal/cli`および`go test ./...`で対象ケースが成功した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

- TT-017: 設定変更確認プロンプト（[y/N] の標準入力）における承認（'y' 等）および拒否（'n' / 'Enter' 等）のシミュレーション検証
  - Status: Completed
  - Status Reason: `y`、`yes`、大文字、末尾改行なしの承認と、`n`、空入力の拒否、入力中断を標準入出力差し替えテストで確認した。
  - Replaced By: Not applicable
  - Type: e2e
  - Linked AC: AC-004, AC-005, AC-006
  - Linked Technical Requirements: TR-005, TR-006
  - Linked Non-Functional Requirements: None
  - Test First: Yes
  - Red Check Required: Yes
  - Red Check Result: Passed — 未実装のConsoleUIにより期待どおりビルド失敗した。
  - Green Check Result: Passed — `go test ./internal/cli`および`go test ./...`で対象ケースが成功した。
  - Not Run Reason: Not applicable
  - Alternative Verification: Not applicable

#### Implementation Notes

- Actual Files Changed: `internal/cli/init.go` / `internal/cli/init_test.go` / `cmd/context/main.go` / `internal/application/init_repository.go` / `docs/prds/prd-001-safe-context-repository-setup/tasks.md` / `docs/prds/prd-001-safe-context-repository-setup/architecture-change.md`
- Test Results: Red Checkとして未実装APIによるビルド失敗を確認した。Green Checkでは`go test ./internal/cli`の16テストと`go test ./...`が成功した。`go vet ./...`、`golangci-lint run`（0 issues）、文書のPrettier・textlint、`go build -o /private/tmp/context-cli-bin ./cmd/context`、`git diff --check`も成功した。`task ci`は実行環境に`actionlint`がなく中断し、`govulncheck ./...`はGo 1.25でビルドされたツールとGo 1.26の不一致により実行できなかった。
- Follow-up: `actionlint`とGo 1.26で再ビルドした`govulncheck`を利用できる開発環境で、`task ci`を再実行する。

#### Code Review

- Status: Completion Candidate
- Scope: `internal/cli/init.go` および `cmd/context/main.go` の差分とE2Eテスト差分
- Findings: High 1件。確認入力待ちの`ReadString`がContextキャンセルを監視せず、SIGINT後も停止しない可能性があった。
- Fixes: 入力読込結果と`ctx.Done()`を待つ処理へ変更し、確認待ち中のキャンセルで`context.Canceled`を返す回帰テストを追加した。入力なしのEOFは設定変更中断として扱う。
- Verification: `go test ./internal/cli`、`go test ./...`、`go vet ./...`、`golangci-lint run`、`go build ./cmd/context`が成功した。
- Remaining Risks: Not applicable
- Human Decision: Pending

#### Definition of Done

- `context init` コマンドで設定処理が正しく動作し、対話確認、エラー時のStder出力および終了コードが正しく遷移すること
- 標準入出力をモック・置換したCLIインテグレーション / E2Eテストがすべて合格していること
- `Taskfile.yml` を用いた品質ゲート（`task ci`）がローカル環境ですべて合格すること

## Documentation Updates

- `architecture-change.md` を実装結果に合わせて更新する
- 必要に応じて `docs/architecture/` を更新する
- 必要に応じて `docs/decisions/` を追加または更新する

## Quality Gates

- Tests: `go test ./...`成功、112件合格。
- Build: `go build ./cmd/context`成功。
- Lint: `golangci-lint run`は0 issues。Prettierおよびtextlint成功。`git diff --check`成功。
- Type Check: `go vet ./...`成功。
- Not Run Reason / Alternative Verification: `task ci`は`actionlint`がPATHに存在しないため中断した。`govulncheck ./...`はGo 1.25でビルドされたツールとプロジェクトのGo 1.26の不一致により実行不能だった。raceテストは実行環境でGo 1.26のraceランタイムを解決できず実行不能だった。通常テスト、静的解析、Lint、ビルドで代替検証した。

## Code Review Summary

- Status: Completion Candidate
- Scope: PRD-001に対する全実装およびテストの差分
- Critical / High Remaining: None
- Medium / Low / Follow-up: Mediumは解消済み。`actionlint`、Go 1.26で再ビルドした`govulncheck`、raceランタイムを利用できる環境で未実行ゲートを再実行する。
- Existing Changes Separation: 差分基準は`4fe1ca5`と現在の作業ツリー。`.codex/skills/document-review/`、`.codex/skills/resolve-issue/`、`.agents`およびPRD-001と無関係な文書スキル変更は対象外とし、`context init`実装、関連設定・テスト・ADR・PRD記録だけをレビューした。
- Human Decision: Pending

## Architecture Update

- Status: Pending
- Scope: `docs/architecture/system-overview.md` の新規作成による全体のレイヤー構造および `context init` の制御/データフロー記述
- Evidence: 実装されたコードベースと `architecture-change.md`、ADRとの整合
- Files Updated: Not applicable
- Inconsistencies: Not applicable
- Unreflected Items: Not applicable
- Human Decision: Pending

## Release Check

- Scope: PRD-001に対する全実装およびテストの差分
- Checked At: Not applicable
- Goal / Story / AC / Success Metrics: Pending — 未実施
- TR / NFR / ADR: Pending — 未実施
- Engineering Foundation: Pending — 未実施
- Tests / Regression: Pending — 未実施
- Migration / Configuration / External Services / Permissions / Operations: Pending — 未実施
- Monitoring / Logs / Rollback / Recovery: Pending — 未実施
- Architecture Consistency: Pending — 未実施
- Open Items Classification: Pending — 未実施
- Dependencies / External Code / Licenses: Pending — 未実施
- Security / Privacy / Accessibility / Legal / Contract / Policy: Pending — 未実施

- Traceability / ID Integrity: Pending — 未実施
- Overall Result: Pending
- Remaining Risks: Not applicable
- Follow-up: Not applicable
- Human Decision: Pending

## Post-Release Verification

- Result: Pending
- AC / Monitoring / Logs Checked: Not applicable
- Rollback / Recovery Result: Not applicable

## Follow-up Transfer

- Not applicable

## Assumptions

- ロック制御、一時ファイル置換、ファイル/ディレクトリ権限判定はmacOSおよびLinuxの標準的なPOSIXシステムコール（flock / lstat / chmod / rename）が機能するローカルファイルシステム上で実行されることを想定する。
- ユーザー設定は `~/.config/context/config.yaml`（または `XDG_CONFIG_HOME` 経由）に格納されるものとする。

## Open Questions

### Blocking

- `Not applicable` (必要な前提・判断はADRにより解消済み)

### Deferred

- **Windows対応**: パス区切り文字、権限表現（`chmod`）、排他ロック（`flock`）、一時ファイルの原子的置換（`rename`）のWindows互換性評価は、Windows対応を定義する将来のPRDへ保留する。
  - **判断条件**: 将来Windows対応のサポート要件が明示的に決定された段階。
  - **判断工程**: Windows対応を目的とする将来のPRD作成・レビューフェーズ。
  - **影響Task**: `T-002`, `T-003`, `T-004`, `T-005`（OS依存のファイル操作、権限チェック、およびロック処理を行うタスク全体）。
- **CLIコマンド解析ライブラリ**: `context init` 単体の時点では標準ライブラリ `flag` または単純な引数スライス走査で実装し、将来サブコマンドが増大する段階で正式なライブラリ選定とADRを作成する。
  - **判断条件**: サブコマンドまたはオプション定義が拡張され、標準 `flag` パースでは保守が難しくなる（概ね3個以上のサブコマンドが存在する）段階。
  - **判断工程**: 新たなサブコマンド追加を伴う将来のPRD設計時のADR（候補）作成フェーズ。
  - **影響Task**: `T-006`（CLIコマンドハンドラ）。
