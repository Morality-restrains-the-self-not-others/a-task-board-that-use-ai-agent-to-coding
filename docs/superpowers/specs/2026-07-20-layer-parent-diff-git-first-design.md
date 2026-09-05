# 设计：父层 diff 优先 git 生成变动集（OPT-20260720-018）

## 目标

`getLayerParentDiffFiles` 在有 git 工作区时不再全量 `collectIndex` walk，避免大型仓库触 `MAX_DIFF_ENTRIES` 误报「目录扫描已达上限」。

## 方案（采用）

对每个配对 `(workP, workC)`：

1. 若两端均为 git 工作区：候选路径 = 子层 status ∪ 父层 status ∪（HEAD 不同时先 `git diff --name-status`；对象互不可见则双端 `ls-tree -r HEAD` 比 blob）
2. 对候选路径按磁盘存在性/内容分类为 added/removed/modified；`truncated=false`
3. 任一端非 git → 回退现有 `collectIndex` + `compareIndices`（可仍产生 truncated）

共享 `.git`（symlink）时 HEAD 通常相同，靠双端 status 覆盖「父脏子净」。独立 clone 分叉见 OPT-019。

## 非目标

- 不改 HTTP 路由/契约字段（仍返回 `changes`/`truncated`/`has_more`）
- 不改前端文案
- 不做架构拓扑变更（无新服务）

## 业务事件

| 意图 | 事件 | 例外 |
|------|------|------|
| 容器内只读 diff 算法 | — | 无平台业务状态变更 |

## 验收

见 `docs/intents/frontend/task_detail_layer_changes_scan_cap.intent.md` 更新后的验收命令。
