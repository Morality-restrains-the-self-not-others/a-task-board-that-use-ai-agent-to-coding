# 意图：任务/项目数据 GET 与 GitLab 探活分离

- 日期：2026-08-31
- 设计：`docs/superpowers/specs/2026-08-31-task-detail-get-15s-timeout-design.md`
- 索引：B-086

## 背景与目标

任务详情 GET 在租户 GitLab 不可达时固定约 15s 才 200。根因是数据 GET 同步全量 GET project，后者内嵌 GitLab 探活。目标：**页面数据立即返回**；**探活标记用独立请求**，不挡住首屏渲染。

## 范围与边界

- 范围内：`handleGetProject` 去掉同步 GitLab enrich；`loadProjects` 不再 GET project；任务详情前端 paint 后独立 `POST validate-git-repos`；项目详情确认只靠该 POST 填探活标记。
- 范围外：修复 GitLab 主机；Loki；在任务 GET 内做 lite 同步调用。

## 约束与风险

- 两阶段均为纯查询：无领域事件。
- 探活失败不得清空或重拉任务 JSON。
- 探活 SSOT：已有 `POST /api/projects/validate-git-repos/`，不新造第三条探活协议（除非磁盘占用需另 GET）。

## 验收标准

1. GitLab 挂起时任务详情 GET **不**接近 15s，首屏可渲染。
2. GET project 同步路径 **不**调用 `validateGitReposForUser` / disk-size enrich。
3. 探活仅独立 POST；任务页 `taskDetailLoading` 不等待该 POST。
4. 探活失败：数据仍在；错误 UI 带 `data-traceId`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 打开任务/项目详情（数据 GET） | — | handleGetTask / handleGetProject | — | 纯查询 |
| GitLab 探活标记（独立 POST） | — | validate-git-repos | — | 纯查询 |

## 变更记录

- 2026-08-31：trace `544a7a87-…` 头脑风暴；同日评审改为两阶段（数据立即 + 探活独立）。
- 2026-08-31：总体设计批准（不 bump 架构版本）。
