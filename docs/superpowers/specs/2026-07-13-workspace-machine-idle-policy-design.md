# 设计：工作空间机器节点启用策略、闲置回收与优先调度

- 日期：2026-07-13
- 状态：已采纳（goal-mode 自动决策，跳过确认门）
- 相关 URL：
  - 设置：`/tenant/{t}/settings/task-panel/`
  - 工作面板：`/tenant/{t}/work-panel`
- 架构版本：v19 🎯 target
- `python_api_approval`: scoped-down（**零新增 Python HTTP 接口**；全部落 Go）

## 1. 问题

1. **无工作空间级机器策略**：现有仅任务级 `auto_release_minutes`（启动时勾选）与终态释放（v17）；无法按工作空间配置「启用哪些机器节点 / 闲置多久回收」。
2. **闲置机器无法复用**：同工作空间内机器已启动但容器已停（闲置）时，新任务启动仍走全新 `start-vm`，浪费冷启动时间与费用。
3. **工作面板缺汇总**：`workspace-runtime-indicators` 仅按任务返回布尔指示，不展示「启动数 / 闲置数 / 回收间隙」。

## 2. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 每个工作空间可配置启用的机器节点（云平台授权 CPA） | 设置页可勾选并持久化 |
| S2 | 可配置闲置自动回收分钟数（0=关闭） | PUT 后 GET 回显；扫描器按阈值 stop |
| S3 | 闲置时新容器优先落在同工作空间闲置机器 | start-vm 路径返回 reuse 或内部接管 |
| S4 | work-panel 展示启动数、闲置数、回收间隙 | 汇总 API + 头部 UI |
| S5 | 无新 Python 公网/internal HTTP 接口 | 全部在 taskCloudService / taskEvents |

## 3. 方案对比（自动采纳 A）

| 方案 | 描述 | 取舍 |
|------|------|------|
| **A（采纳）** | `taskCloudService` 新表 `workspace_machine_policies`；汇总/策略 API；start-vm 闲置复用；taskEvents 定时闲置回收 | SSOT 在云服务；与现有 indicators/stop-vm 一致；无 Python 新接口 |
| B | 策略存 `taskProjectService.workspaces` | 跨服务读机器态需双写；违反「机器数据在 cloud」 |
| C | 全新建独立 machine-pool 服务 | 过度设计；当前 1 任务 1 配置可渐进演进 |

## 4. 领域概念（轻量）

| 概念 | 说明 |
|------|------|
| **WorkspaceMachinePolicy** | 工作空间机器策略聚合根：启用 CPA 列表、闲置回收分钟、是否优先复用闲置 |
| **MachineNodeRuntime** | 派生视图：由 `cloud_server_configs` + open history 判定 `started` / `idle` / `busy` |
| **IdleReuse** | 领域服务：为新任务选择同工作空间闲置节点并绑定 |
| **IdleRecycle** | 领域服务/定时 intent：超过阈值的闲置节点调用 stop |

### 状态判定

| 状态 | 条件 |
|------|------|
| **started（已启动）** | `last_runtime_status=Running`，或 `instance_id` 以 `mock-` 开头 |
| **starting（启动中）** | `last_runtime_status` ∈ {Starting, Pending, Initializing} |
| **busy** | started 且 `server_url` 非空（容器已 register-reachability） |
| **idle** | started 且非 busy（机器在跑、容器不在） |
| **stopped / released** | 非 started；云平台返回实例不存在（Released）时须清本地 `instance_id` 并关闭 open history |

> 注意：仅有 `instance_id` 或 open history **不足以**计入「已启动」——须与真实运行态对齐，避免「启动中 / 已释放」误入已启动过滤。

`idle_since`：取最近一次 `server_url` 清空或 history 无 container 活动时间；若无精确字段，MVP 用 `cloud_server_configs.updated_at`（清空 server_url 时刷新）或新增 `idle_since` 列（推荐）。

## 5. 数据模型（taskCloudService SSOT）

```sql
CREATE TABLE IF NOT EXISTS workspace_machine_policies (
  company_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  idle_recycle_minutes INTEGER NOT NULL DEFAULT 5,
  prefer_idle_reuse INTEGER NOT NULL DEFAULT 1,
  enabled_authorization_ids TEXT NOT NULL DEFAULT '[]', -- JSON 数组；空=不限制（沿用租户全部 active CPA）
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (company_id, workspace_id)
);

-- cloud_server_configs 增列
ALTER TABLE cloud_server_configs ADD COLUMN idle_since DATETIME DEFAULT NULL;
-- 当 server_url 从非空→空 时写入 idle_since；重新 register 时清空
```

同步：`db/table_ownership.yaml` 登记 `workspace_machine_policies` → `task-cloud-service`。

## 6. API（全部 Go / taskCloudService）

前缀：`/api/tenant/{tenant}/workspace/{workspace}/cloud/compute/`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `workspace-machine-policy/` | 读策略；无行则返回默认 `{idle_recycle_minutes:5, prefer_idle_reuse:true, enabled_authorization_ids:[]}` |
| PUT | `workspace-machine-policy/` | 更新策略（需租户成员；写操作记审计日志） |
| GET | `workspace-machine-summary/` | `{started_count, idle_count, busy_count, idle_recycle_minutes, prefer_idle_reuse}` |

扩展既有：

- `workspace-runtime-indicators/`：可选附带 summary 字段（或前端并行调 summary）。
- `start-vm` / `start-vm-auto`：
  1. 读策略；若 `enabled_authorization_ids` 非空且请求 `authorization_id` 不在列表 → **403**。
  2. 若 `prefer_idle_reuse`：查找同 workspace 闲置节点（匹配 platform/auth 优先）→ **reuse**：
     - 将闲置 config 的 `instance_id`/`public_ip`/平台字段迁到目标 `task_id`（或绑定映射表）；
     - 清空源任务机器绑定并关闭其 open history；
     - 返回 `{status, reuse:true, instance_id, public_ip, ...}`，跳过新建 ECS。
  3. 否则走现有启动路径。

网关：APISIX / task-gateway 已有 `/api/tenant/*/workspace/*/cloud/compute/*` 转发至 :8018；新 path 随同前缀自动可达（确认路由通配）。

Swagger：更新 `taskCloudService/openapi.yaml`。

## 7. 闲置回收执行链

**Intent**：`workspace_machine_idle/1_recycle_idle_nodes`  
**触发**：taskEvents 定时（建议每 60s）或基于内部 tick（与现有 intent 端口顺延）。

流程：

1. 列出所有 `idle_recycle_minutes > 0` 的 policy。
2. 对每个 workspace 查 idle 节点且 `now - idle_since >= minutes`。
3. 调用既有 stop 编排（与 v17 终态释放同路径：ECS → CLOUD_SERVER_STOPPED；relay/mock → stop）。
4. 幂等；记录结构化日志 `event=idle_recycle`。

**优先级**：终态释放 > 任务级 `auto_release` > 工作空间闲置回收（后两者可并存，先到先执行）。

## 8. 前端

### 8.1 设置页 `WorkspaceSettingsTaskPanel.vue`

每个工作空间行增加「机器节点」按钮 → 模态框：

- 多选：租户 active CPA 列表（调既有 CPA API）作为「启用的机器节点」
- 数字输入：闲置自动回收（分钟），0=关闭；展示文案「闲置 N 分钟后自动回收」
- 开关：优先在闲置节点启动新容器
- 保存 → PUT policy

拆分子组件 `WorkspaceMachinePolicyModal.vue`（避免继续膨胀已超 500 行的 SFC）。

### 8.2 工作面板 `WorkPanel.vue` / Header

在看板头部展示摘要条（`data-alias="workspace-machine-summary"`）：

- 主视图紧凑：空心/虚线环徽标 + 已启动/闲置数字（可点击过滤）；启动中 > 0 时展示琥珀色脉冲点 + 数字
- 详情（图例、回收策略、计数说明）通过「!」按钮 hover/focus 浮层展示
- 随现有 15s 轮询刷新 summary

## 9. 权限

| 操作 | 角色 |
|------|------|
| GET policy / summary / indicators | 租户工作空间成员（读） |
| PUT policy | 租户管理员或工作空间管理员（与云平台设置同级；若无细粒度则复用 CPA 写权限） |

不新增角色体系；复用现有 session/forward-auth。

## 10. 价值流影响（输入给 Step 4）

- 域：`cloud-integration` + `project-workspace`
- 新 stream：`workspace-machine-idle-policy`
- 字段：`task-cloud-service.workspace_machine_policies.*`、`task-cloud-service.cloud_server_configs.idle_since`
- 测试：Go 单测 + Playwright work-panel / settings

## 11. 🐍 Python 新增接口清单与 Go 替代评估

### 拟新增接口

| # | 方法 | 路径 | 归属 | 说明 |
|---|------|------|------|------|
| — | — | — | — | **无** |

### 选型结论

- **最终选择**: Go（taskCloudService + taskEvents）
- **python_api_approval**: `scoped-down`
- **Swagger**: taskCloudService OpenAPI 同步

## 12. 🏛️ 架构变更影响

- **迭代版本**: v19 🎯 target
- **迭代名称**: workspace-machine-idle-policy
- **作者**: claude
- **设计日期**: 2026-07-13 15:37
- **新增文件**:
  - 🆕 `docs/architecture/v19-application-integration-20260713-1537-claude.puml`
  - 🆕 `docs/architecture/v19-application-integration-20260713-1537-claude.archimate`（含 Plateau/Gap/WP）
  - 🆕 `docs/architecture/v19-application-integration-20260713-1537-claude.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] `workspace_machine_policies` DataObject
  - 🟢 [NEW] taskEvents intent idle recycle
  - 🟡 [MODIFIED] taskCloudService — policy/summary/reuse/start-vm 门禁
  - 🟡 [MODIFIED] Vue WorkPanel / WorkspaceSettingsTaskPanel

### .archimate 架构变迁要点

| 元素 | 内容 |
|------|------|
| Plateau v17 | Current — 终态释放已交付 |
| Plateau v19 | Target — 工作空间闲置策略与汇总 |
| Gap | 无 workspace 级启用节点/闲置回收/优先复用/面板汇总 |
| WorkPackage | 实现 policy API + recycle intent + 前端 |

## 13. 非目标（本迭代不做）

- 跨工作空间机器共享
- 多容器共驻同一 ECS（reuse 仍为 1 机器绑定 1 活跃任务）
- 替换任务级 `auto_release_minutes` UI
- Django 直连 task_cloud.db
