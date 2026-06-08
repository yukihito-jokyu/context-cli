---
name: update-nix
description: このリポジトリのNix Flake開発環境を新規作成または更新し、開発ツールの追加・削除、Flake inputの追加・更新、Goツールチェーン設定、整形、ロック更新、検証を行う。「xxをNix開発環境へ追加したい」「nixpkgsを更新したい」「Goのバージョンを変えたい」「開発環境からツールを削除したい」ときに使用する。
---

# Update Nix

このリポジトリの `flake.nix` と関連ファイルを、要求に必要な最小差分で更新する。

## 管理対象

- `flake.nix`
- `flake.lock`
- `.envrc`
- `.gitignore`
- `scripts/validate.sh`

Goライブラリは `go.mod` の責務とし、このスキルでは追加しない。

## 基準構成

`flake.nix` が存在しない場合は次の方針で作成する。

- `nixpkgs` は `github:NixOS/nixpkgs/nixos-unstable` を使う。
- 複数systemの出力には `flake-utils.lib.eachDefaultSystem` を使う。
- `devShells.default` は `pkgs.mkShellNoCC` で定義する。
- 初期ツールは `go`、`gopls`、`delve`、`golangci-lint`、`go-task`、`git`、`nixfmt-rfc-style` とする。
- `formatter` に `pkgs.nixfmt-rfc-style` を指定する。
- `CGO_ENABLED=0` とする。
- Gitルートを `WORKSPACE_ROOT` とし、`.toolchain/` 配下へ `GOCACHE`、`GOENV`、`GOPATH`、`GOMODCACHE` を分離する。
- `$GOPATH/bin` は `PATH` の末尾へ追加し、Nix管理ツールを優先する。
- `.envrc` は `use flake` とする。
- `.toolchain/` は `.gitignore` へ追加する。
- CLIのエントリーポイントが未確定なら `packages.default` を作らず、追加予定のTODOコメントを残す。
- `unfree` は許可しない。必要な場合はライセンスと影響を説明し、事前確認する。

## 更新手順

1. Git差分、`flake.nix`、`flake.lock`、`.envrc`、`.gitignore`、本スクリプトを読む。
2. ツール名とNixpkgs属性名が異なる可能性があれば `nix search nixpkgs <名前>` などで確認する。
3. 候補が一意なら選択する。複数候補から用途を判断できない場合だけ、候補と推奨を提示して確認する。
4. 要求に必要な最小差分を編集する。既存の無関係な変更を戻さない。
5. `nix fmt` を実行する。
6. `scripts/validate.sh` を実行する。追加ツールの確認が必要なら、許可されたツール名を引数で渡す。
7. 失敗を修正できる限り修正し、再検証する。変更を勝手に巻き戻さない。
8. 変更内容、ロック更新の有無、検証結果、未解決事項を報告する。

## ロック更新

Nixpkgs内のパッケージを追加・削除するだけなら `flake.lock` を更新しない。

次の場合だけ更新する。

- Flake inputを追加・削除した場合
- input更新を明示的に依頼された場合
- 固定済みnixpkgsに必要なパッケージがなく、更新が必要な場合

特定inputだけを更新する場合は `nix flake update <input名>` を使う。全inputの一括更新は明示依頼がある場合だけ行う。

## 事前確認

通常のツール追加・削除は、確認なしで編集から検証まで行う。次は事前確認する。

- Flake inputの追加・削除
- 全inputの一括更新
- 対応OS・CPUの変更
- `CGO_ENABLED` など開発環境全体へ影響する変更
- 既存ツールのメジャーバージョン変更
- unfreeパッケージの許可

追加ツールが一部のsystemで利用できない場合は、全対象で評価可能な構成を原則とする。除外が必要なら理由をコメントし、対象systemを黙って削除しない。

## 検証スクリプト

`scripts/validate.sh` はGitルートへ移動し、ロックファイルを暗黙更新せずに次を確認する。

- Nixフォーマット
- Flake check
- 全systemのFlake評価
- 初期ツールのバージョンコマンド
- 引数で指定された追加ツールのバージョンコマンド

任意のコマンド文字列を実行しない。未対応ツールを検証する場合は、`scripts/validate.sh` の `case` に副作用のない確認コマンドを追加してから実行する。
