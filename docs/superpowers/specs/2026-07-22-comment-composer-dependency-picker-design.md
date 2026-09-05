# 设计：添加评论时选择执行依赖（composer dependency picker）

- **日期**: 2026-07-22
- **状态**: approved（goal-mode 自动采纳）
- **迭代名**: `comment-composer-dependency-picker`
- **作者**: claude
- **python_api_approval**: n/a（扩展既有 Go TTS / taskCloudService）

## 1. 问题

任务详情「添加评论」区仅提交正文；`execution_mode` 默认 `wait_previous`，前序依赖由前端同步绑定时隐式取「上一条」。用户无法在发评时声明：

1. 是否等待前序评论执行完成；
2. 若等待：等**全部前序**，还是等**指定评论**。

## 2. 选定方案

| 选择 | 值 | 持久化 |
|------|-----|--------|
| 不等待 | `execution_mode=independent` | TTS `comments.execution_mode`；绑定同值 |
| 等待 · 全部前序 | `execution_mode=wait_previous` + `depends_on_comment_ids=[]` | TTS JSON 列；绑定 `depends_on_comment_id=""`，调度释义为「全部前序 completed」 |
| 等待 · 指定 | `execution_mode=wait_previous` + `depends_on_comment_ids=[...]` | TTS JSON；绑定以逗号拼接写入既有 `depends_on_comment_id`（兼容单列） |

### UI（composer）

布局顺序（自上而下）：**执行依赖选择器 → 评论输入框 → 提交按钮**（依赖须在输入前，避免先写后选）。

```
○ 不等待前序评论
○ 等待前序完成
   ○ 前面全部评论
   ○ 指定评论  [多选 checkbox：顶层 user/ai 评论摘要]
```

默认：等待 · 全部前序（与历史默认 `wait_previous` 对齐，并强化为真·全部前序）。

### 调度语义变更（taskCloudService）

- `independent` → 立即推进（不变）
- `wait_previous` + 无 depends → **全部**序位更早的 binding 均为 `completed` 才启动（原：仅上一条）
- `wait_previous` + depends 列表 → 列表中每个 comment 的 binding 均为 `completed`（缺失则阻塞）

### 非目标

- 云上真正多 ECS；计费拆分；看板卡片 composer 完整多选（可复用组件，本期任务详情优先）

## 3. 事件

| 意图 | 事件 | 发布点 |
|------|------|--------|
| 创建评论带依赖 | 沿用 CommentCreated 载荷扩展 `depends_on_comment_ids`（若已有）/ 创建响应回传 | TTS |
| 绑定推进 | CommentContainerBindingAdvanced（`depends_on_comment_id` 可含逗号列表） | Cloud |
