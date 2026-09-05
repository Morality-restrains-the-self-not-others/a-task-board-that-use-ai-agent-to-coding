# 意图：代理步骤命令正文放到折叠区外

## 背景与目标

任务详情执行日志里，每条代理步骤是默认折叠的 `<details>`。纯文本命令（`llm_response.content` 落到 `plainSubtitle`，indigo `<pre>`）原先写在折叠体内，折叠时只能看到「步骤 N · ✅」和「复制JSON」，必须展开才能看见 `bash sed …` 这类具体命令。

目标：把命令/纯文本正文放到 **summary（折叠区外）**，折叠时即可扫到具体命令；展开后仍只展示一份，不重复。

## 范围与边界

- 范围内：`TaskDetailAgentStepCard` / `TaskDetailAgentStepCardHeader` 将 `plainSubtitle` 渲染到 `summary`；`TaskDetailAgentStepCardDetails` 不再渲染同一份 indigo `<pre>`。
- 范围外：不改步骤数据契约、不改 `agentStepPlainBodyForPre` 的「标题已含 tool_calls.command 则不再进 pre」规则；不恢复已下线的「工具调用」预览条；html/markdown 富文本仍只在展开后展示。

## 约束与风险

- 长命令用既有 `max-h-48 overflow-auto`，避免单步撑满视口。
- 点命令 `<pre>` 不得误触发行折叠（`@click.stop`）；展开/收起仍点标题行。
- 「复制JSON」仍是剪贴板只读动作，不发写接口。

## 验收标准

1. 步骤 `open=false` 且有 `plainSubtitle` 时，`[data-testid="layer-agent-step-command-preview"]` 在 `summary` 内可见，文案含命令全文。
2. 同一卡片内该 testid 只有一处；折叠体内不再出现 indigo 命令 `<pre>`。
3. `tool_results` / 完整 JSON / 富文本仍仅在展开后可见。
4. 结构化 `tool_calls` 不额外出现「工具调用」预览条（既有行为）。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 代理步骤命令放到折叠区外 | — | — | — | — | 纯前端展示，无服务端事实变更 |

## 实施计划

1. 先写 `TaskDetailAgentStepCard.commandOutsideFold.test.js`（折叠可见命令、不重复、富文本仍在体内）。
2. `plainSubtitle` 改传到 Header 的 summary；Details 删除对应 `<pre>`。
3. 跑通该测例及既有 accordion / collapsedToolCalls 回归。

## 变更记录

- 2026-09-02：任务详情页元素调整——将 indigo 命令 `<pre>` 从步骤折叠体移到 summary。
