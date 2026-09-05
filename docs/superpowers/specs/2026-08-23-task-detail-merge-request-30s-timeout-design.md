# 任务详情「一键合并」请求超时（30 秒）根因分析与修复设计

- **日期**: 2026-08-23
- **作者**: cursor
- **状态**: 🔧 implemented（代码已落地，待精准编译重启后公网验收）
- **traceId**: `ab3e126c-2618-47c4-ba4f-675948d61ad9`
- **关联**: taskFE、taskGateway、taskGitOauth、GitLab `gitlab-tencent-sh-1.daydaymoney.com`

## 1. 现象

任务详情评论区「一键合并」约 30 秒后弹窗：`请求超时（30 秒），请检查网络后重试`（`data-traceId` 如上）。不是页面加载失败，是合并写操作被前端默认 30s Abort。

## 2. Trace 分析

- Loki 全 job 查询 1h/24h/7d 均为 0 条；D2 确认 Loki 无 ingest（`memory_chunks=0`）。
- 18:00 整点 `truncate-ram-work-logs.sh` 清空 `logs/task-git-oauth.log`（事故 17:57）。
- **替代证据**：
  - APISIX：`POST /api/git-oauth/merge-request-merge/...`，`upstream_latency=29955ms`，`status=499`
  - Tempo：仅 task-auth forward-auth ~6ms
  - MySQL 审计 `merge_request_merge`：`PUT .../merge_requests/4/merge` →
    `Client.Timeout exceeded while awaiting headers`
- 目标 MR：`https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad/-/merge_requests/4`

**根因**：GitLab SH-1 同步 PUT merge 等响应头超过 30s；前端 `apiFetch` 默认 30s 与 `gitDo` 30s 同时打满。弹窗文案误导为「请检查网络」。

**后续核对（用户）**：GitLab UI `merged_at` = **2026-08-23 17:17:11 +08**。平台唯一审计是 17:57:30 的 PUT 失败。因此 17:57 是对**已合并 MR** 的再点击；GET 当时未把 `merged_at` 认成 `merged`，仍发出 PUT 并空等 30 秒。

## 3. 已落地修复

| 层 | 改动 |
|----|------|
| taskGitOauth | 合并 PUT 60s / 头 45s；状态 GET 30s / 头 15s；超时返回 504 中文；阶段日志 token/status/merge/reconcile |
| taskGitOauth | GET 认 `merged_at` / `merge_commit_sha` / `merged` 为已合并；PUT/超时失败后**再 GET**，若已合并则 200 `noop` |
| taskFE | `mergeGitPullRequest` timeout 90s；Abort 文案改为「请打开 MR 确认是否已合并」 |
| taskFE | 「一键合并」仅 `state===open` 时显示；已合并 / 未知 / 已关闭均不可点 |

超时阶梯：FE 90s > git merge PUT 60s > git GET 30s ≪ APISIX read 120s。无新服务、无架构版本 bump（v104 仍为 current）。

## 4. 测试

- Go：`TestMergeRequestMergeGitTimeoutReturns504Chinese`、`TestIsGitAPITimeout`、`TestParseMergeRequestStateMergedAtOverridesOpened`、`TestMergeRequestMergeReconciles405WhenGitLabAlreadyMerged`、`TestMergeRequestMergeReconcilesTimeoutWhenGitLabAlreadyMerged`
- FE：`taskDetailGitPrReply.merge-timeout.test.js`；`CommentGitPrReply` 已合并/未知隐藏按钮

## 5. 业务意图 → 事件对照

| 业务意图 | 事件 | 例外 |
|---------|------|------|
| 合并成功 | `GIT_MERGE_REQUEST_MERGED`（已有） | — |
| 已合并再点（noop） | `GIT_MERGE_REQUEST_MERGED`（`noop=true`） | 未改变 Git 事实，审计 `noop_already_merged` |
| 合并超时且复查仍未合并 | 无 | 仅审计 failed |

## 6. Code Review Graph

根图 Nodes 108，未索引 `handleMergeRequestMerge`。爆炸半径以手工调用链为准。

## 7. 架构

纯超时/文案修复，不更新 `docs/architecture/`。基线仍为 v104 current。
