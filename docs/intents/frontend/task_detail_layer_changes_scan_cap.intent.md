# 任务详情变动文件「目录扫描已达上限」治理

## 意图

消除或显著降低任务详情「文件变动」琥珀提示「目录扫描已达上限，部分文件可能未列入」；截断提示节点须挂载产生该状态的请求 `data-traceId`，便于排障。

## 现象

- 元素：`[data-testid="task-detail-layer-changes-truncated-hint"]`
- 文案：`目录扫描已达上限，部分文件可能未列入；可刷新重试。`
- 含义：容器侧父层 diff 在建立文件索引时触达 `MAX_DIFF_ENTRIES`（4000），`truncated=true`；**不是**列表分页的 `has_more`。

## 根因

`collectIndex` 对工作区做全树 walk，**未**跳过 `node_modules` / `.venv` 等噪声目录，额度被噪声占满后真实变动路径可能未进入对比集合。

## 方案（分阶段）

| 阶段 | 内容 | 状态 |
| --- | --- | --- |
| A. 跳过噪声目录 | `collectIndex` 复用 `shouldSkipListingDirName`（与文件树列表一致） | ✅ |
| B. 可观测 | 截断提示挂载最近一次成功拉取的 `data-traceId`（`fetch_trace_id`） | ✅ |
| C. git-first | 优先用双端 git status +（HEAD 分叉时）tree-diff；对象互不可见时双端 `ls-tree`；目录 walk 仅作无 git 回退 | ✅ OPT-018/019 |

## 变更相对旧版

| 项 | 旧 | 新 |
| --- | --- | --- |
| 索引扫描 | 含 node_modules 等；有 git 仍全量 walk | 跳过噪声；有 git 时 `collectPairChanges` 走 status/tree-diff |
| 截断提示 DOM | 无 `data-traceId` | 有则挂最近成功请求 traceId |
| `truncated` | 大仓易触顶 | 有 git 时一般为 false；仅无 git 回退 walk 仍可能触顶 |
| 提示文案 | 不变（仍提示刷新；滚动加载仅针对 `has_more`） | 不变 |

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 例外理由 |
|---------|--------|---------|
| 父层 diff 扫描跳过噪声 | — | 容器只读对比，无平台业务状态变更 |
| 截断提示挂 data-traceId | — | 前端可观测属性 |
| 父层 diff git-first | — | 容器只读算法，无平台业务状态变更 |

## 验收

```bash
cd trae-agent/onlineServiceJS && node --test \
  src/layerParentDiffCollect.test.mjs \
  src/layerParentDiff.pagination.test.mjs \
  src/layerParentDiffGit.test.mjs
cd task2app/front_project/app && npx vitest run \
  src/components/task-detail/TaskDetailExecLayerChangesHints.test.js \
  src/composables/taskDetail/taskDetailZTreeExecLogState.test.js
```

容器镜像需含 `layerParentDiffGit.mjs` / `layerParentDiffCompare.mjs` 后远端任务才走 git-first；前端 SPA 硬刷新后即可看到 `data-traceId`。
