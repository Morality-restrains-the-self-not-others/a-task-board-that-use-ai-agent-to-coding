# 角色权限分析：Fork 确认弹窗副本数量

**日期**: 2026-08-23  
**设计**: `docs/superpowers/specs/2026-08-23-fork-copy-count-design.md`

## 权限影响分析

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| 确认弹窗数量选择（纯 UI） | 能打开任务详情的工作区成员 | Workspace / Task | 无写 | 页面级鉴权 | ✅ 充分 | — |
| `POST .../todos/` × N | 工作区可创建任务的用户 | Workspace | write | `hasWorkspaceAccess` + 任务帖配额 `consumeTaskPostQuota` | ✅ 充分 | 每份独立扣配额；配额不足在第 k 份 402，保留 1..k-1 |
| `auto_run=true` × N | 同上 | Workspace / Cloud | write + 启服 | 既有 `validateAutoRunPrerequisites` / OAuth / Git 身份 | ✅ 充分 | UI 警示 N 份云资源；不另开权限点 |
| Idempotency-Key `${batch}:${i}` | 客户端 | 请求 | — | `claimTaskCreateDedup` | ✅ 充分 | 键含序号，避免 10s fork 短窗把 N 份折成 1 |

## 角色建模

不引入新角色或权限粒度。

## IDOR / 越权

数量选择不改变资源 ID。每份 POST 仍校验 `tenant_id` + `workspace_id` + 工作区访问。禁止跨租户批量派生。

## 滥用面

前端 cap 99 是产品上限，不是安全上限（脚本仍可多次 Fork）。缓解：既有任务帖配额与 auto_run 计费。本增量不新增后端硬顶。
