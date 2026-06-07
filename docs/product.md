# Product

Status: Accepted
Status Reason: 重レビューで配布対象、コマンド責務、正本とローカル編集の関係、成功条件を確定し、未解決事項がないことを確認した。

## Product Vision

開発リポジトリごとに必要なAI開発コンテキストを、信頼できる正本から迷わず安全に導入できる状態を実現する。

## Target Users

- 複数の開発リポジトリでAIコーディングエージェントを利用する個人開発者

## Core User Problems

- リポジトリごとに必要な指示ファイルやSkillを手作業で選択、コピー、更新する必要があり、導入漏れ、誤配置、意図しない上書きが発生する。
- AIコンテキストの知識がさまざまな開発リポジトリへ分散し、管理、再利用、更新が困難になる。

## Value Proposition

AIコンテキストの原本とプロジェクト別配布定義を単一のContext Repositoryで一元管理し、各リポジトリに必要な内容だけを対話的かつ安全に配布、再配布、同期できる。

## Product Goals

- Context RepositoryをAIコンテキストの唯一の正本として維持できる。
- リポジトリ固有および共通のSkillから、必要なものを選択して正しい場所へ配布できる。
- 既存ファイル、Skill、ローカル編集をユーザーの許可なく上書きしない。
- 個々の開発リポジトリに配布管理の知識を持たせず、追加済みのAIコンテキストを繰り返し配布、同期できる。

## Success Metrics

- 初期版の主要操作である設定、配布、再配布、同期の受入条件をすべて満たす。
- 選択したプロジェクトと配布物が、定義された優先順位と配置規則どおりに配布される。
- 選択済みの配布物だけを、Context Repositoryの現在の内容へ同期できる。
- 既存データやローカル編集を、ユーザーの許可なく上書きする事象を0件にする。
- 配布先で原本や配布定義を個別管理せずに、同じContext Repositoryから繰り返し配布できる。

## Current Focus

初期版として、ローカルのContext Repositoryを設定し、カレントディレクトリへ安全に配布、再配布、同期できる状態を完成させる。配布対象は、Context Repositoryに原本があるAGENTS.md、CLAUDE.md、共通・リポジトリ固有Skillとする。プロジェクト名を指定しない場合は、Context Repository内のプロジェクトから対話的に選択できるようにする。

## Product Principles

- Context Repositoryを唯一の正本とし、配布先でAIコンテキストの知識を管理しない。
- Context Repositoryは、AIコンテキストの原本とプロジェクト別配布定義を保持するローカルリポジトリとする。
- ユーザーが配布対象を明示的に選択できる。
- AGENTS.mdとCLAUDE.mdはContext Repository内の原本をそのまま配布し、内容を生成しない。
- 既存データとローカル編集は、明示的な許可なしに変更しない。
- 配布先の編集は正本へ還元せず、正本への取り込みは初期版の対象外とする。
- リポジトリ固有のSkillを同名の共通Skillより優先する。
- `context init` は使用するContext Repositoryを設定する。
- `context add` はプロジェクトと配布物を選択し、初回配布または再配布する。
- `context sync` は選択済みの配布物だけをContext Repositoryの現在の内容へ更新する。
- 配布物の選択と選択解除は `context add` で行う。
- 最初は個人開発者向けの単純なローカル運用を優先する。

## Constraints

- 実装言語はGo、コマンド名は `context` とする。
- Context Repositoryはローカルファイルシステム上にクローン済みであることを前提とする。
- `context init` ではContext Repositoryの場所と必要な構造を検証する。
- Context Repositoryの設定と配布先の管理情報は、配布先ではなくユーザー単位で管理する。
- 配布先は `context add` 実行時のカレントディレクトリとする。
- `context sync` は実行時のカレントディレクトリだけを対象とする。
- 配布先の既存ファイル、Skill、ローカル編集との自動マージは行わない。
- 初期段階では個人利用を対象とし、チーム向けの権限、承認、同期機能は扱わない。

## Out of Scope

- Context Repositoryの自動クローン、`git pull`
- 複数人向けの設定共有、権限、承認管理
- 配布物の自動マージ
- リモートリポジトリからの直接配布
- GUI
- `context sync` 時の新規Skillの自動追加

## Assumptions

- Context Repositoryはユーザーが事前にローカルへクローンし、利用可能な状態に保つ。
- Context Repositoryには、`context init` で検証可能な所定のディレクトリ構造が存在する。
- 配布先を一意に識別し、選択済みの配布物とローカル編集の有無を判断できる。
- Context Repositoryの設定変更は、現在の設定と変更内容を示したうえでユーザーに確認する。

## Open Questions

- なし。
