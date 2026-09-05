# 测试意图：代理步骤命令正文放到折叠区外

## 对应意图
`042_agent_step_command_outside_fold.intent.md`

## 测试目标

验证纯文本命令在步骤折叠时仍出现在 summary 内；不在折叠体重复；富文本与 tool_results 仍随展开显示。

## 测试分层

- 单元：`taskFE/app/src/components/task-detail/TaskDetailAgentStepCard.commandOutsideFold.test.js`
- 回归：`TaskDetailAgentStepCard.collapsedToolCalls.test.js`、`TaskDetailAgentStepsSection.accordion.test.js`

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 卡片 `open=false`，`plainSubtitle` 为 bash 命令 | 挂载步骤卡 | summary 内 `layer-agent-step-command-preview` 可见且含命令；`details.open===false` |
| T1b | 卡片 `open=true`，同上 | 挂载步骤卡 | 命令仍在 summary 且全卡仅一份 |
| T2 | 折叠且有命令 | 挂载步骤卡 | 命令 preview testid 仅一处；折叠体 Details 根节点不含该 pre |
| T3 | `open=false`，`bodyMode=html` 且无 `plainSubtitle` | 挂载步骤卡 | 无 command-preview；`agent-step-rich-frame` 因折叠不可见 |
| T4 | 有 `toolResultText` 且折叠 | 挂载步骤卡 | 文案不含 `tool_results.result` |
| T5 | 有 `tool_calls` 且折叠 | 挂载步骤卡 | 不出现「工具调用」预览条 |

## 数据与环境

- Vitest + jsdom；`@vue/test-utils`；无需网络。

## 通过标准

```
cd taskFE/app && npx vitest run src/components/task-detail/TaskDetailAgentStepCard.commandOutsideFold.test.js src/components/task-detail/TaskDetailAgentStepCard.collapsedToolCalls.test.js src/components/task-detail/TaskDetailAgentStepsSection.accordion.test.js
```

全绿。
