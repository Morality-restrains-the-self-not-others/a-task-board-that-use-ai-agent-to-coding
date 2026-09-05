# 启动日志默认折叠 + zTree「推送并创建PR」设计

- **日期**: 2026-07-12
- **作者**: goal-mode / 0-auto-flow（自动采纳）
- **状态**: approved（goal-mode 跳过用户门）
- **相关页面**: `?relayToTrae=true` 任务详情

## 问题 / 意图

1. 直接启动「启动日志」可折叠，但默认展开占视口；需求改为**默认折叠**（保留展开能力）。
2. zTree 层节点仅有「推送」文案；推送后 GitHub PR 已异步化，主响应不含 `github_pull_request`，前端无法在完成后跳转 PR 页。

## 成功标准

1. 启动日志默认折叠；点击「展开日志 / 折叠日志」可切换；折叠时正文隐藏，标题与引导条仍可见。
2. 当层存在未推送提交（既有 `canPush`：工作区干净且 `ahead > 0`）时，显示按钮文案 **「推送并创建PR」**。
3. 点击后完成推送并等待 PR 创建（或明确跳过/失败）；若有 `html_url`，在 zTree 节点显示 **PR** 按钮，点击后打开 PR 审查页（不自动 `window.open`）。

## 方案（已采纳）

### A. 启动日志默认折叠

| 项 | 改动 |
|----|------|
| `ServerConfig.logic.vue` | `relayToTraeLogsExpanded = ref(false)` |
| `ServerConfigRelayDirectPanel.vue` | prop `logsExpanded` default `false` |
| 意图 019 | 改为默认折叠；与模拟启动对齐 |
| 单测 / E2E | 断言默认折叠 |

### B. 推送并创建 PR + 跳转

| 层 | 改动 |
|----|------|
| UI | `LayerGraphZtreeNode.vue`：按钮文案「推送」→「推送并创建PR」；title 同步 |
| 前端请求 | `onLayerGraphLayerPush` 请求体增加 `wait_for_pr: true` |
| Django forward | `forward_container_layer_git_push`：若 `wait_for_pr`，同步执行 PR follow-up，并对 `queued` job **轮询** `get_external_job` 直至终态（超时约 50s），将最终结果写入响应 `github_pull_request`；否则保持现有异步 `schedule_pr_follow_up` |
| 前端完成态 | 有 `html_url` 时写入 `git_remote.pr_html_url` 并显示 PR 按钮；点击后再 `window.open`；无 url 时沿用 skipped / compare_url |

### 不做

- 不新增独立「仅推送」按钮（本迭代将原推送按钮语义升级为推送并创建 PR）。
- 不改容器 `git/push` 协议；PR 仍在 SaaS 侧完成。
- 不新增 Application Component；无 ArchiMate 拓扑变更（接口行为扩展）。

## 架构变更影响

- 🟡 [MODIFIED] Django `forward_container_layer_git_push` — 可选同步等待 PR
- 🟡 [MODIFIED] Vue zTree / ServerConfig 启动日志默认态
- **说明**: 无新服务节点；完整 ArchiMate 三件套不强制本次交付。

## Domain 概念（轻量）

- 操作: PushAndCreatePullRequest（应用服务）
- 输入: layer_id, target_branch, wait_for_pr
- 输出: push 结果 + github_pull_request（含 html_url / skipped / compare_url）

## 价值流影响

- 新/改步骤: relay-startup-logs-default-collapsed；ztree-push-and-create-pr
- 测试: vitest + Django 单测 + 意图文档
