# Context Repositoryの安全な設定

Status: Accepted
Status Reason: 重レビューで成功条件と安全性の検証対象を明確化した。上流文書の未更新表記は、判断済みADRを正本とするFollow-upとして受容した。
PRD ID: prd-001-safe-context-repository-setup。

## Source Product

- ../../product.md

## Goal

個人開発者が、ローカルのContext Repositoryを検証したうえで初回設定または安全に変更できる。

## Problem

複数の開発リポジトリでAIコーディングエージェントを利用する個人開発者は、Context Repositoryを設定するとき、不正な構造や権限のRepositoryを設定したり、既存設定を意図せず変更したりする危険がある。

## Target User

複数の開発リポジトリでAIコーディングエージェントを利用する個人開発者。

## Success Metrics

- Metric: 初回設定と設定変更の正常系、および構造不備、権限不備、未対応スキーマ、確認拒否として列挙した全経路で自動テストが成功し、失敗時または確認拒否時の既存設定変更が0件である。
  - Evaluation Method: 自動テストにより、成功時だけ検証済みパスが保存され、列挙した各失敗時または確認拒否時には既存設定が変更されないことを確認する。
  - Observable Timing: 実装完了時。
- Metric: 利用者が主要な設定操作の結果と設定状態を実際の操作で確認できる。
  - Evaluation Method: 隔離した一時環境で、初回設定、設定変更の承認と拒否、不正なRepositoryの拒否を人間が操作し、表示内容と設定ファイルの状態が期待どおりであることを確認する。
  - Observable Timing: 実装完了後の動作確認時。

## Scope

- ローカルのContext Repositoryの初回設定。
- 設定済みContext Repositoryから別のRepositoryへの設定変更。
- 設定変更前の現在の設定と変更内容の表示、および明示確認。
- 同じContext Repositoryが設定済みの場合に、確認や書き込みを行わず正常終了する冪等な再実行。
- 設定前のパスの存在、所定構造、初期版の配布に必要な原本とプロジェクト別配布定義の読み取り可能性、権限、シンボリックリンク、および既存設定のスキーマ互換性の検証。
- 検証失敗または設定変更の確認拒否時における既存設定の維持。

## Out of Scope

- Context Repositoryの設定解除。
- Context Repositoryのクローンまたは更新。
- `context add` による配布または再配布。
- `context sync` による同期。

## User Value

個人開発者は、誤ったRepositoryや意図しない設定変更を避けながら、以後の配布と同期に使用するContext Repositoryを迷わず設定できる。

## Assumptions

- Context Repositoryは利用者が事前にローカルへクローンし、利用可能な状態に保つ。
- 初回設定では保存前の明示確認を必須とせず、検証結果と設定対象を表示して保存する。

## Open Questions

- Not applicable
