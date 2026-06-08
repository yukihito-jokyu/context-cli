# Generic Document Review

## Recognition

- Target: 専用リファレンスが存在しない補助文書
- Template: use only when an explicit template exists
- Default importance: AI recommends from role and change risk
- Status required: only when the document defines one

## Inputs

- Required: target document
- Conditional: explicitly referenced upstream documents, template, related decisions
- Code/config: only when the document claims current implementation facts

## Review Points

- 文書の責務と対象読者が明確か
- 上流成果物および正本と矛盾していないか
- 必要情報が不足せず、不要な責務を含んでいないか
- Assumptions、Open Questions、Deferredが適切に分類されているか
- 参照先が存在し、到達可能か
- 下流利用者が文書から必要な判断を行えるか
- Statusを持つ場合、状態と理由が内容に一致しているか

## Completion

- 補助文書は汎用観点で完了できる
- プロセス上の主要成果物は専用リファレンスなしでAccepted化しない
