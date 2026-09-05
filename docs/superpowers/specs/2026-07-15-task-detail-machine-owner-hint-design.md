# 设计：任务详情镜像区展示机器节点所属任务

- **日期**: 2026-07-15
- **作者**: cursor-grok
- **状态**: approved（goal-mode 自动采纳，跳过确认门）
- **迭代名**: `task-detail-machine-owner-hint`
- **相关页面**: `https://www.daydaymoney.com/tenant/{t}/workspace/{w}/task-detail/{taskId}/`
- **目标元素**: 任务详情「镜像」卡片（`#detail-image` 所在 `div.p-6.bg-white.border…`）
- **架构变更**: 无（不新增服务/限界上下文；仅扩展既有查询响应字段 + 前端展示）
- **python_api_approval**: n/a（零新增 Python/Django HTTP；扩展既有 Go `server-runtime-status`）

---

## 1. 问题 / 意图

容器已在机器节点上运行时，用户在镜像配置区无法一眼看到「当前容器所在机器节点隶属于哪个任务」。在闲置复用/历史共享实例场景下，机器绑定任务可能与当前浏览任务不一致，造成运维误解。

### 与用户表述对齐

> 如果容器已经运行，可以在这里显示一下容器运行的机器节点是隶属于哪个任务的

| 短语 | 本设计解释 |
|------|------------|
| 容器已经运行 | 当前任务 CSC `server_url` 非空，或前端已有 `serverUrl` / `containerPageUrl` / mock 容器运行态 |
| 机器节点隶属于哪个任务 | 同 `instance_id` 上非终态释放的 CSC 绑定任务（可多任务共享时列出全部） |
| 在这里显示 | 任务详情「镜像」卡片内、镜像选择器下方 |

---

## 2. 成功标准（SMART）

| # | 标准 | 可验证方式 |
|---|------|------------|
| S1 | 容器未运行时镜像区不展示所属任务提示 | 前端单测 |
| S2 | 容器运行且有 `instance_id` 时展示所属任务 ID（本任务标注「本任务」） | 前端单测 + Go 单测 |
| S3 | 同实例多绑定（共享）时列出全部非终态任务 ID | Go 单测 |
| S4 | 非本任务绑定可点击跳转对应 task-detail | 前端单测（href） |
| S5 | 无新 Python HTTP；Swagger 字段同步 | OpenAPI |
| S6 | 意图文档登记 | `docs/intents/frontend/task_detail/` |

---

## 3. 方案对比（自动采纳 A）

| 方案 | 描述 | 结论 |
|------|------|------|
| **A. 扩展既有 `server-runtime-status`，附带 `machine_bound_tasks` / `machine_owner_task_ids`；镜像区读该响应展示** | 复用已有轮询；无新公网路由 | **采纳** |
| B. 新建公网 `instance-bindings` 代理 | 多一次请求；权限面扩大 | 拒 |
| C. 仅前端用当前 `task.id` 展示 | 无法揭示共享/他任务归属 | 拒 |

### 自主决策

1. **所属任务集合**：`listInstanceBindings` 中 `terminal_released=0` 且 `instance_id` 匹配的 `task_id`。
2. **容器运行判定（后端）**：当前任务 CSC `server_url` 非空 → `container_running=true`。
3. **容器运行判定（前端展示门闩）**：`container_running` 或 `serverUrl`/`containerPageUrl`/`isMockContainerRunning` 任一为真，且存在 `instance_id`。
4. **展示文案**：`机器节点所属任务：` + 任务 ID 列表；当前任务加「本任务」；他任务链到 `/tenant/{t}/workspace/{w}/task-detail/{id}/`。
5. **不拉任务标题**：避免跨服务耦合；标题可后续增强。
6. **无新领域事件**：纯查询展示；书面例外。

---

## 4. 接口契约（扩展）

`GET …/cloud/compute/server-runtime-status/?task_id=`

新增可选字段（有 instance 时附带）：

```json
{
  "container_running": true,
  "machine_owner_task_ids": ["task_a", "task_b"],
  "machine_bound_tasks": [
    { "task_id": "task_a", "has_container": true, "terminal_released": false },
    { "task_id": "task_b", "has_container": false, "terminal_released": false }
  ]
}
```

---

## 5. 前端落点

- `ServerConfig.logic.vue` 镜像卡：插入 `ServerConfigMachineOwnerHint.vue`
- 数据源：已有 `serverRuntimeStatusResponse`
- 保持 `ServerConfig.logic.vue` 不再大幅膨胀：展示逻辑放进独立小组件 + 纯函数

---

## 6. 测试

- Go：`TestServerRuntimeStatusIncludesMachineOwnerWhenContainerRunning`
- Vue：`ServerConfigMachineOwnerHint.test.js`
- 纯函数：`machineOwnerHint.js` 单元测试
