# Design: ztree 推送 unauthorized（OAuth 已授权）

**Date**: 2026-07-22  
**Status**: adopted (goal-mode auto)  
**Task**: `task_13762772779307981876`

## 🕸️ CRG

- semantic_search：embedding/图未命中 OAuth 节点（graph head 落后）；改用日志 + 定向读源码。
- 爆炸半径：`taskCloudService` prepare → `taskGitOauth` access-for-user；Django summary 路径不受影响。

## Trace 证据

| trace_id | 首错 |
|---|---|
| `60345685-9a51-49b6-be9b-de9489fb307d` | `cloud_prepare_git_push` → prepare **502** |
| `83b6bca2-d57e-41f2-97af-3e886ae3eb7f` | 同上 |

`task-git-oauth.log`：`bridge secret rejected` + `access-for-user` **401** `unauthorized`。

## 根因（选定方案）

`taskCloudService.gitOauth.bridgeSecret` 为空且无 env 回退 → 不带 `X-GitOauth-Bridge-Secret` → gitOauth 拒收。UI「OAuth 已授权」经 Django（有 secret）查 summary，故绿标与推送失败并存。

## 方案对比（自动采用 A）

| 方案 | 说明 | 决策 |
|---|---|---|
| A | 对齐 bridgeSecret + 回退链 + 错误文案 | **采用**（与 taskProjectService 一致，改动小） |
| B | 关闭 gitOauth RequireBridgeSecret | 拒绝（削弱内网 API 鉴权） |
| C | 推送改走 Django prepare | 拒绝（Gateway 路径已是目标架构） |

## 验收

1. Cloud → access-for-user 带正确 bridge header，不再 401。
2. prepare 成功则 `use_oauth_access_push=true` 并转发容器。
3. 单测覆盖 header 发送与 unauthorized 文案映射。

## 架构

无 ArchiMate 视图变更（配置/鉴权对齐，不新增组件）。
