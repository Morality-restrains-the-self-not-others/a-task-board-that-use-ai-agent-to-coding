# DaydaymoneyGrafana 插件：登录 + 报警捕获规则 + task2app 建任务

**日期：** 2026-07-05  
**状态：** 已实现

## 目标

让 DaydaymoneyGrafana 可作为 Grafana App 插件安装，控制台支持：

1. 账号 + 访问令牌登录 task2app
2. 配置报警捕获规则（默认 Error，可自定义 Loki 查询或消息正则）
3. 命中规则后通过 `POST /api/tenant/{tenant}/workspace/{ws}/todos/` 自动建任务

## 方案

- **登录**：`POST /api/accounts/users/login-with-access-token/`，body `{username, access_token}`，返回 session `token`；后续请求 `Authorization: Token {token}`
- **配置存储**：Grafana 插件 `jsonData`（登录态与 `captureRules[]`；管理员配置页）
- **捕获引擎**：保留 Loki 轮询 + 全局 JS 错误；按 `captureRules[]` 逐条匹配后建任务
- **项目/分支解析**：拉取租户项目列表，用 `projectMatchRegex` 对日志行匹配项目名；`branchMatchRegex` 提取工作分支；`mergeTargetBranch` 写入 `branch_strategy.merge_target_branch_name`

## 捕获规则字段

| 字段 | 说明 |
|------|------|
| `id` | 规则 UUID |
| `name` | 展示名 |
| `enabled` | 是否启用 |
| `matchMode` | `error_default` / `custom_loki` / `custom_regex` |
| `lokiQuery` | custom_loki 时完整 LogQL |
| `messageRegex` | custom_regex 时对日志行的 JS 正则 |
| `tenantId` | 租户 ID |
| `ownerMemberId` | 责任人 CompanyMember.id |
| `workspaceId` | 工作空间 ID |
| `projectMatchRegex` | 项目名匹配正则（对日志行或项目名） |
| `branchMatchRegex` | 从日志提取分支的正则（首捕获组） |
| `mergeTargetBranch` | 合并目标分支 |

## 默认规则

安装后预置一条 `error_default` 规则，Loki 查询：`{job="runall"} | json | level=~"(?i)error|fatal"`。

## 兼容

- 已移除 `errorReportUrl` / `ingestToken` 旧版回退；须登录 task2app 后按捕获规则建任务。
