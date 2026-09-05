# 交付物详情：展示上层交付物

- **日期**: 2026-07-14
- **状态**: approved（goal-mode 自动采用最优方案）
- **范围**: `task2app/front_project` 交付物详情身份区 UI；复用既有 `GET .../todos/{id}/` 的 `parent_task`；不新增 HTTP 接口
- **架构影响**: 无（纯前端展示既有字段；不改服务边界/数据所有权）

## 1. 问题与目标

工作面板「交付物详情」身份区已展示「交付物类别」，但未展示任务树中的上层交付物（`parent_task`）。非顶层交付物缺少对齐上下文。

目标：

1. 在「交付物类别」旁展示「上层交付物」。
2. **最顶层**（无 `parent_task`）不展示该字段。
3. 非顶层展示上层交付物的可读名称（标题），并提供跳转至上层详情，便于工作对齐。

## 2. 方案对比与选型

| 方案 | 描述 | 优劣 | 结论 |
|------|------|------|------|
| A. 仅展示 parent_task ID | 与 fork_from 一致 | 无法回答「是什么」 | 否 |
| B. 前端按需 GET 上层任务取 title | 有 parent 时再请求一次详情 | 零后端改动；标题可读 | ⭐ 采用 |
| C. Go 详情响应嵌套 parent 摘要 | 扩展 `taskToJSON` | 少一次 RTT；需改 Go+Swagger | 备选（后续优化） |

**采用方案 B**。

## 3. UI 结构

在 `TaskDetailTaskIdentityPanel` 身份区 grid 中，紧邻「交付物类别」增加：

```
[交付物类别]  [上层交付物]   ← 仅当 parent_task 有值时渲染
```

- 标签文案：`上层交付物`
- `data-testid="task-parent-deliverable"`
- 只读：可点击链接 → 上层任务详情（路由同 `forkSourceTaskRoute` 模式）
- 展示文案：优先 `parentTaskTitle`；加载中显示「加载中…」；失败回退 `ID:{id}`

## 4. 数据与行为

| 项 | 约定 |
|----|------|
| 父子字段 | API `parent_task`（DB `parent_task_id`） |
| 顶层判定 | `parent_task` 为空 / null / 缺省 → 不渲染区块 |
| 标题解析 | `useTaskDetail`：有 parent 时 `GET .../todos/{parentId}/` 取 `title` |
| 可选加速 | 若父组件已传入 `parentTaskTitle`（如 work-panel 列表缓存），跳过二次请求 |
| 编辑态 | 本期**不**提供修改上级；编辑时仍只读展示（与自动运行类似） |
| 权限 | 复用既有任务读权限；拉取失败不阻断详情页其它区域 |

## 5. 领域概念清单（轻量）

| 概念 | 说明 |
|------|------|
| Bounded Context | 任务协作 / 工作面板交付物内容树 |
| Entity | Task（Todo）；父子 `parent_task` |
| Aggregate | 任务为根；上层交付物为只读导航引用，不引入新聚合 |

## 6. 价值流影响（摘要）

- 影响流：工作面板 → 打开交付物详情 → 确认对齐上下文
- 字段：`taskTaskService.tasks.parent_task_id`（只读展示）；标题来自同表 `title`
- 测试：`TaskDetailTaskIdentityPanel` 单测 + 意图 `024`

## 7. 🏛️ 架构变更影响

- **无需**新建 architecture target 文件（无组件/数据流/服务边界变更）
- Python 新增接口：无（`python_api_approval: n/a`）

## 8. 改动文件清单（预期）

| 文件 | 变更 |
|------|------|
| `TaskDetailTaskIdentityPanel.vue` | 展示上层交付物 |
| `useTaskDetail.js` | 解析 parent、拉取标题、构造路由 |
| `TaskDetail.vue` | 透传 props |
| `TaskDetailTaskIdentityPanel.*.test.js` | 单测 |
| `docs/intents/.../024_*.intent.md` + `.test.intent.md` | 意图 |

## 9. 验收标准

1. 无 `parent_task` → 不出现「上层交付物」区块。
2. 有 `parent_task` → 类别旁可见「上层交付物」，展示上层标题（或可接受回退）。
3. 点击可进入上层任务详情（同租户/工作空间）。
4. work-panel 弹窗与独立任务详情页行为一致。
