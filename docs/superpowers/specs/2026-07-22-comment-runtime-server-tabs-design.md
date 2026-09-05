# 设计：服务器运行状态按评论容器 Tab 展示

**Date**: 2026-07-22  
**Iteration**: comment-runtime-server-tabs  
**Based on**: v51 评论级执行细节

## 问题

任务详情「服务器运行状态」面板仍是任务级单一视图，无法对应当前评论关联的运行中镜像/容器。用户期望用 Tab 切换查看各评论关联的服务器情况。

## 选定方案（自动采纳）

**前端 Tab 壳 + 任务级 runtime 数据复用（MVP）**

1. 新建 `buildCommentRuntimeServerTabs`：从顶层评论筛出与执行/容器相关的项（AI、含 container_agent 子评论、或当前 active 执行评论）。
2. 新建 `ServerConfigCommentRuntimeTabs`：Tab 栏 + 上下文条（评论摘要、关联镜像、当前执行标记）+ 内嵌现有 `ServerConfigServerRuntimePanel`。
3. 本期仍只有任务级实例 API：所有 Tab 共享同一 runtime 查询/Workbench/停止；非当前执行 Tab 显示「共享任务级实例」提示。
4. 无候选评论且服务器运行中：回退单 Tab「任务级服务器」。
5. 服务器未运行：不强制 Tab，仍渲染原面板（零 Tab 或空态由组件处理为直接展示面板）。

### 非目标

- 每评论独立 `server-runtime-status` / stop-vm（属 OPT-20260722-038）
- 新后端 API

## 组件树

```
ServerConfig.logic (runtime section)
└── ServerConfigCommentRuntimeTabs
    ├── tablist（评论）
    └── ServerConfigServerRuntimePanel（共享 props）
```

## 数据契约

```ts
type CommentRuntimeServerTab = {
  commentId: string
  label: string
  imageId: string
  imageLabel: string
  isActiveExecution: boolean
  sharesTaskServer: boolean // MVP 恒 true
}
```

## 业务意图 → 事件对照

**无对应事件**：纯前端展示重组，无服务端状态变更。

## 架构影响

扩展 v51 `CommentExecutionContext` 的 UI 消费面；不新增 Go 服务/表。架构制品：见同迭代 `v52-application-integration-*`（轻量标注 Vue Tab 壳）。

## 验收

- 有关联评论且服务器运行中：面板顶部出现评论 Tab
- 切换 Tab 更新上下文条（评论/镜像），runtime 操作仍可用
- 相关 vitest 通过
