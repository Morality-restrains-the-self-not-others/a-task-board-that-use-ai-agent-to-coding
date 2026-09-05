# 从 GitLab 同步 — 来源须选自已购买区域

- **日期**: 2026-09-04 15:20
- **作者**: cursor
- **状态**: approved（`/goal` 自动采纳）
- **页面**: https://www.daydaymoney.com/tenant/:id/projects →「从 GitLab 同步项目」
- **元素**: `来源：https://gitlab.daydaymoney.com`（`GitlabSyncProjectsModal` 标题旁）

## 架构理解（口头确认）

根据当前架构（v129 ✅ current）：

- 共有多视图基线；最近交付含 v129 金丝雀重启、v128 阿里云 GitLab 区域目录
- 应用层：taskFE → APISIX → taskProjectService（`gitlab-remote-repos`）/ taskBill（`gitlab-resources`）/ taskGitOauth
- 业务层：租户已购买 `billing_tenant_gitlab_resource`；区域元数据含 `gitlab_web_url`
- 技术层：多区域 GitLab 实例（非单一 `gitlab.daydaymoney.com`）

📋 版本历史：v129 ✅ current。本次在 v129 上做 **application-integration** 变更 → **v130 target**。

## 🕸️ Code Review Graph 分析

- CRG `update --brief` OK；Codegraph：`GitlabSyncProjectsModal` ← `useGitlabProjectSync` ← `Projects.vue`
- Navbar 已有 `fetchGitResources` → `GET /api/tenant/{tid}/billing/gitlab-resources/`
- 后端 `handleGitlabRemoteRepos` 已支持 `?gitlab_host=`；空 host 时 fallback 首个 `gitlab:*`（即平台默认站）— **正是错误「来源」的根因**
- `findProviderByGitlabHost` 可解析完整 URL（剥 scheme）

## 问题分析

| 项 | 结论 |
|----|------|
| 现状 | 打开模态即 `GET .../gitlab-remote-repos/...` **不传** `gitlab_host` → 后端 fallback → 展示平台站 `gitlab.daydaymoney.com` |
| 期望 | 先拉租户**已购买** GitLab 列表，用户选一个后再拉仓库 / 启 OAuth（「跳转」指对该实例授权与同步） |
| 非目标 | 不改计费开通；不新增 Python API；不改 DLT/事件（纯查询） |

## 方案（选定）

**前端 composable + 模态 UI**：

1. `openModal` 先 `GET /api/tenant/{tid}/billing/gitlab-resources/`（与 Navbar 同源）
2. 过滤：`gitlab_web_url` 非空（排除 `pending_node` 未挂载）
3. UI：
   - **0** → 提示先购买/开通，链到价格/订单；**不**调用 remote-repos、不展示平台默认 URL
   - **1** → 自动选中并加载仓库；标题旁展示该 URL
   - **N** → `<select>` 选区域（`region_name` + URL）；切换即带 `gitlab_host` 重载仓库并清空勾选
4. `loadRemoteRepos`：`?gitlab_host=${encodeURIComponent(selectedWebUrl)}`
5. OAuth `startGitlabOAuth`：继续用选中的 `gitlabWebsite` 作 `repo_url`

**拒绝方案**：

| 方案 | 原因 |
|------|------|
| 后端去掉空 host fallback | 破坏其它调用方；本需求是业务选站，不是禁 fallback |
| 只改文案、仍打默认站 | 用户仍连错实例 |
| 新建 list API | 已有 `gitlab-resources` 列表 |

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 列出已购 GitLab | — | — | — | 纯查询 |
| 按实例列远程仓 | — | — | — | 纯查询 |
| 批量/合并建项目 | （既有） | 既有路径 | 既有 | 本次不改写路径 |

## 价值流影响

- 触及：租户项目列表 → GitLab 同步导入
- 字段：只读消费 `billing.billing_tenant_gitlab_resource` + `billing_gitlab_region.gitlab_web_url`（已有）
- 测试：`useGitlabProjectSync.test.js`、`GitlabSyncProjectsModal.unit.test.js`、Playwright `Projects.gitlab-sync`

## 🐍 Python 新增接口

not_applicable — 无新增 Python/Go HTTP 接口；复用既有 billing + projects API。

## 🏛️ 架构变更影响

- **迭代版本**: v130 🎯 target
- **迭代名称**: gitlab-sync-purchased-region-select
- **变更明细**: 🟡 taskFE 同步模态：先读 taskBill 已购区域再带 `gitlab_host` 调 taskProjectService
- **文件**（见 VERSION_HISTORY v130）：每个视图 `.puml` + `.diff.archimate` + `.full.archimate` + `.mermaid.md`

## NFR / 权限摘要（goal 压缩）

- 权限：与 Navbar 同 API；须已登录 + 租户上下文；无新写权限
- 幂等：只读列表 L0；建项目路径不变
- 路径分片：`tenant_id` 已在 URL
- 前端防重放：区域切换为只读刷新；OAuth 仍为整页跳转（既有 Anti-Replay-OK）
