# Project Bootstrap

Status: Completion Candidate
Status Reason: 計画した共通開発基盤の実装とローカル検証が完了し、独立したコードレビューへ引き渡せる状態になった。

## Source

- Technology: ./technology.md
- Structure: ./structure.md
- Development Rules: ./development-rules.md
- Related Decisions:
  - ../decisions/adr-001-layer-architecture-and-dependency-direction.md
  - ../decisions/adr-002-yaml-persistence-and-storage-boundaries.md

## Scope

- 単一Goモジュールと、Accepted済みのレイヤー境界を示すビルド可能な最小パッケージを作成する。
- Nix開発環境からビルド、テスト、Lint、文書を検証できるTaskfileを作成する。
- Go、npm、Nixの依存バージョンを既存の固定方法で管理する。
- LinuxとmacOSのビルドおよびテストと、LinuxのLintを実行する最小CIを作成する。

## Out of Scope

- `init`、`add`、`sync`などの機能コード。
- YAMLスキーマ、設定ファイル、永続化処理。
- CLI解析および対話UIライブラリの選定。
- デプロイ、リリース、署名、パッケージマネージャー配布。

## Planned Changes

- `go.mod`: GoモジュールとGoバージョンを定義する。
- `cmd/context/main.go`: ビルド可能な最小エントリーポイントを作成する。
- `internal/*/doc.go`: Accepted済みの責務境界をビルド可能なパッケージとして作成する。
- `Taskfile.yml`: 開発、ビルド、テスト、Lint、型検査、CI向けコマンドを統一する。
- `.golangci.yml`: golangci-lint v2の厳格なバグ、セキュリティ、設計、スタイル検査を設定する。
- `.github/workflows/ci.yml`: LinuxとmacOSの品質ゲートを作成する。
- `flake.nix`: `gotestsum`と`govulncheck`をNix開発環境へ追加する。
- `package.json`: Nix環境と同じpnpmバージョンを固定する。

## Verification Plan

- `nix flake check --no-update-lock-file`
- `nix develop --no-update-lock-file --command task ci`
- `nix develop --no-update-lock-file --command task test:errors`
- GitHub Actionsワークフローの構文と参照コマンドを静的確認する。

## Risks

- Nixpkgs更新時にGoまたは開発ツールのバージョンが変化する可能性がある。`flake.lock`を正本として固定する。
- 最小パッケージは責務境界だけを定義し、具体的な配置は各PRDの設計で追加する必要がある。
- GitHub Actions自体の実行結果はローカル検証だけでは保証できない。

## Rollback

- 今回追加したファイルを削除し、`flake.nix`と`package.json`の今回変更分だけを差し戻す。
- 既存の未コミット変更は変更または削除しない。

## Actual Files Changed

- `.github/workflows/ci.yml`
- `.golangci.yml`
- `Taskfile.yml`
- `cmd/context/main.go`
- `docs/engineering/bootstrap.md`
- `flake.nix`
- `go.mod`
- `internal/application/doc.go`
- `internal/cli/doc.go`
- `internal/domain/doc.go`
- `internal/infrastructure/doc.go`
- `package.json`

## Verification Results

- `nix flake check --no-update-lock-file`: 成功。aarch64-darwinの開発Shellとformatterを評価した。
- `nix develop --no-update-lock-file --command task ci`: 成功。gofmt、Prettier、`go vet ./...`、golangci-lint、govulncheck、textlint、actionlint、`go test ./...`、`go build`が成功した。
- `nix develop --no-update-lock-file --command task test:errors`: 成功。gotestsumから全パッケージを実行した。
- `nix develop --no-update-lock-file --command pnpm install --frozen-lockfile`: 成功。pnpm 11.4.0でロックファイルとの差分がないことを確認した。
- `GOOS=linux GOARCH=amd64 go build`: 成功。Linux amd64向けにクロスビルドした。
- `git diff --check`: 成功。

## Not Run Reasons

- GitHub Actions上のLinuxおよびmacOSジョブは、共有環境を操作しないため未実行。actionlintとローカルのクロスビルドで代替確認したが、各GitHubホスト上の成功はコードレビュー後のPull Requestで確認する必要がある。
- `nix flake check --all-systems`は、現在のaarch64-darwin環境で他システムのderivationを構築しないため未実行。Flakeは`eachDefaultSystem`を使用し、Linux向けGoビルドは別途確認した。
- 機能上の振る舞いが存在しないため、テストケースは追加していない。`go test ./...`とgotestsumの実行経路だけを確認した。

## Follow-up

- 別セッションで`code-review`をbootstrapモードで実行する。
- レビューがApprovedとなり、人間が最終確定した場合だけStatusを`Completed`へ変更する。
- 完了後に`create-engineering-foundation`のauditモードで文書、設定、コード、CIの整合性を確認する。
- `go.yaml.in/yaml/v3`は未使用依存を避け、YAML永続化を実装するPRDで追加する。

## Code Review

- Status: Pending
- Findings: Not applicable
- Fixes: Not applicable
- Verification: Not applicable
- Remaining Risks: Not applicable
- Human Decision: Pending
