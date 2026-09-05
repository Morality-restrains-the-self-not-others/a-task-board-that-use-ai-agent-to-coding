# 创建设计：Git/子 Git 不可用时自动运行不启动服务器

**日期**: 2026-07-18  
**状态**: 已采用（goal-mode 自动选型，跳过用户确认）  
**范围**: 任务创建/更新开启 `auto_run` 时的异步 `start-vm` 触发；页面元素 `data-testid="task-nested-repos-clone-error"` 对应「无法获取子 Git 仓库列表」场景

## 目标（成功标准）

1. 关联项目的 Git 仓库或子 Git 仓库**无法获取**（未授权、父仓不可见、探测失败等）时，即使任务 `auto_run=true`，也**不得**调用云侧 `start-vm` / `start-vm-auto`。
2. `auto_run` 标志仍可保存为 `true`（软跳过，符合「即使打开自动运行也不自动启动」）。
3. 创建/更新响应暴露跳过原因，便于前端提示；服务端打结构化日志。
4. 子仓列表为空且无 error（无 `.gitmodules`）**不**阻止自动启动。
5. 单元测试覆盖：授权失败 → 零次 cloud start；成功探测 → 仍触发 start。

## 方案对比与决策

| 方案 | 说明 | 取舍 |
|------|------|------|
| A. 硬门禁 400，拒绝 `auto_run=true` | 与现有 `validateAutoRunPrerequisites` 同级 | 拒绝：与「即使打开自动运行」语义冲突 |
| B. 软跳过：保存 `auto_run`，跳过 `scheduleTaskAutoRun` | 复用 `GET /api/internal/nested-git-repos/` 探测 | **采用**：行为与文案一致，无新公开 API |
| C. 仅前端拦截 | 用户可绕过 | 拒绝：门禁必须在 taskTaskService |

**架构影响**: 无新服务、无新公开 HTTP 契约（沿用 internal nested-git-repos）→ **不更新** ArchiMate 三件套。

## 行为

```
创建/更新 auto_run=true
  → validateAutoRunPrerequisites（云/镜像/模版，硬门禁不变）
  → 对关联项目每个 repo_url 调用 internal nested-git-repos
  → 任一 error 非空 或 探测运输失败 → scheduleTaskAutoRun(StartSkipReason=…)
       → ensure 【自动运行】评论（正文含「未启动服务器：{reason}」）
       → 不调用 start-vm / start-vm-auto
  → 全部 OK（含 empty nested）→ scheduleTaskAutoRun 如常（评论 + start-vm）
```

> 2026-08-17：软跳过**不得**省略自动执行评论；仅跳过启服，并在评论中写明原因。

跳过时响应附加（非破坏性字段）：

```json
{
  "auto_run_start_skipped": true,
  "auto_run_start_skip_reason": "无法获取子 Git 仓库列表：未检测到可用授权。…",
  "code": "AUTO_RUN_GIT_ACCESS_SKIPPED"
}
```

## 落点

| 文件 | 职责 |
|------|------|
| `taskTaskService/src/auto_run.go` | `probeGitAccessForAutoRun` + 软跳过接线 |
| `taskTaskService/src/auto_run_test.go` | 红绿测例 |
| `taskTaskService/src/task_handlers.go` | create/update 在 schedule 前探测 |
| `front_project/.../autoRunGateHints.js`（可选） | 跳过原因文案 |
| `docs/intents/backend/...` | 意图与测试意图 |

## 角色权限（摘要）

不新增权限点；internal nested 调用沿用现有服务间路径，使用创建/更新操作者的 `user_id` 做 token 探测。

## NFR（L2）

- 安全：fail-closed（探测失败亦跳过启动）。
- 可用性：跳过原因中文可读；不阻塞任务保存。
- 性能：每个关联仓一次 internal GET；超时沿用 projectHTTP 15s。
