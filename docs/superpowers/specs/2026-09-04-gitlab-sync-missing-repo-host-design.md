# 从 GitLab 同步项目 —「missing repo host」修复

- **日期**: 2026-09-04 15:10
- **作者**: cursor
- **状态**: approved（goal-mode 自动采纳）
- **页面**: `/tenant/:id/projects` →「从 GitLab 同步项目」模态
- **可见症状**: `无法获取 GitLab 授权：missing repo host`（红框 `role="alert"`）

## 架构理解（口头确认）

根据当前架构设计稿（v129 ✅ current）：

- 应用层：taskFE → APISIX → taskProjectService；OAuth 换票经 taskGitOauth `/api/internal/gitsite/{host}/oauth/access-for-user/`
- 业务层：租户项目列表 → GitLab 远程仓导入
- **本次为既有 handler 参数接线缺陷**，不新增/移除服务组件或数据流 → **不更新** `docs/architecture/` target

📋 架构版本历史最近：v129 runAll 金丝雀平滑重启 ✅ current。本迭代在 v129 基线上做 Bug 修复。

## 🕸️ Code Review Graph 分析

- CRG `update --brief` 已执行（软依赖，增量 OK）
- Codegraph explore：`handleGitlabRemoteRepos` → `findProviderByGitlabHost` → `fetchGitAccessToken`
- 唯一将可能为空的 `gitlab_host` 传入 `fetchGitAccessToken` 的调用点：`gitlab_import_handlers.go:94`
- 其它调用方均传入完整 `repoURL` / `apiRepoURL` — **单例缺陷**

## 问题分析

| 项 | 结论 |
|----|------|
| 触发 | 打开「从 GitLab 同步」→ `loadRemoteRepos` → `GET .../gitlab-remote-repos/tenant_id/{tid}/` **无** `gitlab_host` |
| 后端 | `findProviderByGitlabHost("")` 按设计 fallback 到首个 `gitlab:*` provider，并返回 `gitlab_website` |
| 缺陷 | 换票仍用原始空 `gitlabHost` 调 `fetchGitAccessToken` → `extractHostNetloc("")` → `"missing repo host"` |
| 文案 | `无法获取 GitLab 授权：` + tokenErr → 用户所见 |
| 前端 | 首屏不知 host，依赖后端 fallback（注释已写明）；OAuth start 已带 `Accept: application/json`（前序修复） |

无用户粘贴 `data-traceId`；Loki 排障门禁不适用。根因由代码路径静态确认。

## 方案（选定）

在 `handleGitlabRemoteRepos` 中，resolve provider 之后：

```text
tokenRepoURL = gitlab_host 若非空，否则 entry.WebsiteOrigin（再否则 entry.Host）
fetchGitAccessToken(uid, tokenRepoURL, ...)
```

**拒绝方案**：

| 方案 | 拒绝原因 |
|------|----------|
| 仅改前端传 `gitlab_host` | 首屏打开时尚不知 host；且破坏「空 host → fallback」契约 |
| 改 `fetchGitAccessToken` 允许空 host | 破坏 gitsite 路径寻址；其它调用方依赖 host 校验 |

## 业务意图 → 事件对照

| 业务意图 | 事件 | 例外理由 |
|---------|------|---------|
| 列出用户可同步的 GitLab 仓库 | — | 纯查询；无服务端新状态变更 |

## 价值流影响

- 触及：租户项目列表 → GitLab 同步导入（既有流）
- 无新 stream / 无字段变更
- 测试：`gitlab_import_handlers_test.go`（新增）覆盖空 `gitlab_host` 换票用 WebsiteOrigin

## 🐍 Python 新增接口

not_applicable — 无新增 Python/Go HTTP 接口（仅修既有 handler 内部参数）。

## 角色权限（Step 2 摘要）

- **主体**: 已登录租户成员（`X-Auth-User-Id`）
- **资源**: 租户级 GitLab 远程仓列表（只读）
- **变更**: 权限边界不变；仅修正换票 host 解析

## NFR（Step 5 摘要）

| 维度 | 级别 | 说明 |
|------|------|------|
| 正确性 | L3 | 鉴权路径；空 host 不得伪造成功 |
| 幂等 | L0 | 只读 GET |
| 路径分片 | L0 | 已有 `tenant_id` |

## 🏛️ 架构变更影响

无（Bug 修复，无组件/数据流增删改）。

## 验收

- `GET gitlab-remote-repos` 无 `gitlab_host` 时，换票请求命中 `/api/internal/gitsite/<WebsiteOrigin-host>/...`，**不**返回 `missing repo host`
- 显式 `gitlab_host` 仍按该 host 寻址
- 单测覆盖上述两路径
- Search：`fetchGitAccessToken(` across taskProjectService — 仅 `handleGitlabRemoteRepos` 曾传空 host；其余均传完整 repoURL（单例，已修）
