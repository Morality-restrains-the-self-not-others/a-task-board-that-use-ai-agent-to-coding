# [运行时] 任务详情代理步骤折叠时箭头仍呈展开态

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-19
- 最后修改：2026-07-19
- 维护者：Trae AI 团队

## 现象

- 页面：`/tenant/.../workspace/.../task-detail/.../` → 代理步骤列表
- 步骤卡片 `<details>` 为折叠（`open=false`），但 summary 内 `▶` 仍旋转约 90°（像已展开）

## 根因

外层「代理步骤」`<details class="group" open>` 与内层步骤卡 `<details class="group">` 嵌套。Tailwind 匿名 `group-open:rotate-90` 会匹配**任意**带 `group` 且 `open` 的祖先；外层常开导致子步骤箭头永远旋转。

## 解决方案

使用命名 group 隔离：

| 层级 | group 类 | 箭头变体 |
|------|----------|----------|
| 外层代理步骤 | `group/agent-steps` | `group-open/agent-steps:rotate-90` |
| 内层步骤卡 | `group/agent-step` | `group-open/agent-step:rotate-90` |

涉及：`TaskDetailAgentStepsSection.vue`、`TaskDetailAgentStepsHeader.vue`、`TaskDetailAgentStepCard.vue`、`TaskDetailAgentStepCardHeader.vue`。

## 验证

```bash
cd taskFE/app
npx vitest run src/components/task-detail/TaskDetailAgentStepsSection.accordion.test.js
# 用例：isolates step chevron group-open from always-open parent details
bash scripts/runall-lifecycle.sh build
```

折叠步骤箭头应指向右（无 rotate）；仅该步骤 `open` 时旋转。
