# 领域笔记 — 项目文件树与变动列表 Tab

- **日期**: 2026-08-22
- **限界上下文**: 任务协作 / 容器层工作区（既有）
- **本增量**: 无新实体、仓储、领域事件

## 已有概念（不改）

- **Layer**：可写叠层，ID 为 `layer_id`
- **LayerChanges**：相对父层/工作区的路径变动投影（`changes[]`、`displayCount`）
- **ProjectFileTree**：容器层文件懒加载投影

## 呈现层状态（非领域）

```
LayerFilesActiveTab = "tree" | "changes"
```

仅存在于 Vue 组件；切层时复位为 `changes`。

## 事件

纯查询/展示例外，见意图文档对照表。无 MQ 契约、无消费者、无幂等键需求。

## 端口

不新增端口。写操作仍走既有 layer git staged/commit 适配器。
