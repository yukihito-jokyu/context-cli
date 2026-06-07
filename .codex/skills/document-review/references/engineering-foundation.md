# Engineering Foundation Review

## Recognition

- Paths: `docs/engineering/technology.md`, `structure.md`, `development-rules.md`
- Templates: matching files under `docs/ai-driven-development/templates/engineering/`
  <!-- 固定の定義または対応関係を1行で維持するため。 -->

- Default importance: initial creation or policy change `重`; typo, reference, or explanatory correction `中`

<!-- textlint-enable preset-ja-technical-writing/sentence-length -->

- Status required: yes

## Inputs

- Required: all three templates, all three engineering documents, Accepted `docs/product.md`
- Conditional: Accepted ADRs and existing external canonical technical documents
  <!-- 固定の定義または対応関係を1行で維持するため。 -->
  <!-- textlint-disable preset-ja-technical-writing/max-comma -->

- Code/config: dependencies, runtime config, repository structure, lint/type/test config, CI

<!-- textlint-enable preset-ja-technical-writing/max-comma -->

## Review Points

- product.mdの意図、制約、対象環境と整合しているか
- 3ファイルの責務が混在せず、相互に矛盾していないか
- 実装AIが技術、配置先、依存可否、検証方法を判断できるか
- レビューAIが適用規則と例外を判断できるか
- 機能固有の判断や不要な先行設計を含んでいないか
- 現在の慣行、規範、技術的負債を区別しているか
- 自動検証可能な規則が設定・CI・実行コマンドと対応しているか
- 技術選定、依存方向、セキュリティなどの重要判断がADR候補になっているか
- 外部正本を参照する場合、適用範囲と到達可能な参照先が明確か
- 文書、設定、コードの矛盾について正本が確定しているか

## Completion

- 3ファイルを必ず一組として整合確認する
- Blocking Open Questionがない
- 必要なADRがAcceptedである
- Allowed transition: affected documents `Review -> Accepted`
- 一部だけ先にAccepted化しない
