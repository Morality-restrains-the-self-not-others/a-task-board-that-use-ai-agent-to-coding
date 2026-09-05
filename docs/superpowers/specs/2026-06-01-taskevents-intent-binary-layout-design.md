# taskEvents 意图级二进制布局（仿 Python handlers 结构）

> 日期：2026-06-01  
> 状态：**已确认（v4）** — 用户 2026-06-01 定稿四项决策  
> 触发：用户认为 `bin/{event}/` 下单一二进制不足，应仿 `core/kafka/handlers/`  
> 关联：relocation-design v3.1、domain-to-event-coverage-audit、cmd-layout-design  
> **取代**：v3「每 event_type 一个二进制」→ **v4「每 intent 一个二进制」**  
> **门禁**：**G5 删 Python 前必须完成 v4 拆分**

---

## 0. 用户确认决策（2026-06-01）

| # | 决策 | 说明 |
|---|------|------|
| 1 | **D2 — 每 intent 独立 exe** | 含 `company_created` 三步：3 个独立二进制 + fan-out + 幂等/`DispatchRetryable` |
| 2 | **active/inactive 均纳入 runAll** | 启停由 **runAll / `enabled` 配置** 决定，**不再**用 Python `.active/.inactive` 文件名控制是否部署 |
| 3 | **命名** | `task-events-{event-kebab}-{intent-kebab}`（port + groupId + runAll 服务名） |
| 4 | **时序** | **G5 删 Python handlers 之前** 完成 v4 拆分与 G4.5 验收 |

---

## 1. Python 参照模型（只读对照）

```
core/kafka/handlers/{event}/{order}_{name}.active.py   # 历史：文件名标记 active
core/kafka/handlers/{event}/{order}_{name}.inactive.py # 历史：loader 跳过
```

Go v4 **镜像目录与 intent 命名**，但 **active/inactive 语义迁移到配置**：

| Python 时代 | Go v4 |
|-------------|-------|
| `.active.py` → loader 加载 | intent 目录 + exe **始终存在**；`enabled: true` |
| `.inactive.py` → loader 跳过 | 同上；默认 `enabled: false` |
| 单进程顺序多 handler | **多进程 fan-out**（D2） |

---

## 2. 目标概念

| 术语 | 定义 |
|------|------|
| **event slug** | 如 `user_created`（= Python handlers 子目录） |
| **intent slug** | 如 `0_create_company`、`1_set_default_deliverable_system` |
| **intent 二进制** | `bin/{event}/{intent}/task-events-{event-kebab}-{intent-kebab}` |

**订阅**：同一 `event_type` → 多个 intent 进程，**各独立 consumer group** → Redis Stream fan-out。

---

## 3. 目标布局（v4 定稿）

```
taskEvents/
  cmd/{event}/{intent}/main.go
  bin/{event}/{intent}/
    config.yaml                 # enabled, port, groupId
    task-events-{event}-{intent}
  internal/intents/{event}/{intent}/handler.go
```

### 3.1 示例

```
bin/user_created/
  0_create_company/          → task-events-user-created-0-create-company
  1_send_welcome_email/      → task-events-user-created-1-send-welcome-email
bin/company_created/
  1_set_default_deliverable_system/
  2_set_default_progress_system/
  3_create_default_workspace/   # D2：三个独立 exe，非 chain
```

---

## 4. 全量 intent 清单（18 个 exe，均进 runAll）

| event | intent | Python 文件后缀 | 默认 enabled | Go 实现 |
|-------|--------|-----------------|--------------|---------|
| user_created | 0_create_company | .active | **true** | 从 v3 usercreated 拆 |
| user_created | 1_send_welcome_email | .inactive | **false** | 新增 |
| company_created | 1_set_default_deliverable_system | .active | **true** | 从 companycreated 拆 |
| company_created | 2_set_default_progress_system | .active | **true** | 拆 |
| company_created | 3_create_default_workspace | .active | **true** | 拆 |
| workspace_created | 1_process_workspace_creation | .active | true | 从 v3 迁 |
| project_updated | 1_process_project_update | .active | true | 迁 |
| task_completed | 1_process_task_completion | .active | true | 迁 |
| ai_assistant_reply_completed | 1_persist_assistant_reply | .active | true | 迁 |
| billing_transaction_created | 1_process_billing_transaction | .active | true | 迁 |
| sse_message | 1_send_sse_message | .active | true | 迁 |
| cloud_server_started | 1_process_server_start | .active | true | 迁（行为部分等价） |
| cloud_server_stopped | 1_process_server_stop | .active | true | 迁 |
| cloud_server_start_auto | 1_process_server_start_auto | .active | true | 迁（auto_create 缺口保留） |
| cloud_platform_authorization_created | 1_process_cloud_platform_authorization | .active | true | 迁 |
| email_sent | 1_send_email | .inactive | **false** | notifications 拆 |
| invitation_created | 1_send_invitation_email | .inactive | **false** | 拆 |
| user_activated | 1_send_welcome_notification | .inactive | **false** | 拆 |

**非 intent 辅助代码**：`auto_create_resources.py` → `internal/intents/cloud_server_start_auto/` 库函数。

**进程数**：runAll `domain-events-intents` 组 **18 项**（默认启用的约 15，inactive 3 项 `enabled: false` 但仍列在 runAll，可 UI 单独启动）。

---

## 5. D2：`company_created` 三步 fan-out

### 5.1 语义变化（相对 Python）

Python：单进程 **顺序** 1→2→3。  
v4 D2：三进程 **并发** 收到同一 `COMPANY_CREATED`。

### 5.2 可行条件（已具备）

| intent | 幂等策略 |
|--------|----------|
| 1 deliverable | `ensureTenantDefaultDeliverable` — 已存在则 skip |
| 2 progress | `ensureTenantDefaultProgress` — 同上 |
| 3 workspace | `ensureDefaultWorkspace` — 已有 default ws 则 skip |

### 5.3 必须补充

- intent 3 在 1/2 未完成时 → **`DispatchRetryable`** + consumer 重试退避  
- **集成测试**：三 intent 同时订阅，`COMPANY_CREATED` 一条消息 → 最终 DB 与 Python 单进程结果一致  
- **禁止** G5 前若该测试未绿

---

## 6. 配置、命名与端口

### 6.1 命名规范（已确认）

```
服务名 / groupId / 二进制前缀：
  task-events-{event-kebab}-{intent-kebab}

示例：
  task-events-user-created-0-create-company
  task-events-company-created-1-set-default-deliverable-system
```

intent-kebab：`0_create_company` → `0-create-company`（数字前缀保留）。

### 6.2 `port_config.json` 结构

```json
"domainEvents": {
  "consumers": {
    "user_created": {
      "intents": {
        "0_create_company": {
          "enabled": true,
          "port": 8025,
          "groupId": "task-events-user-created-0-create-company",
          "events": ["USER_CREATED"]
        },
        "1_send_welcome_email": {
          "enabled": false,
          "port": 8125,
          "groupId": "task-events-user-created-1-send-welcome-email",
          "events": ["USER_CREATED"]
        }
      }
    }
  }
}
```

- 端口：**18020–18037** intent 健康检查（见 port-conflict-resolution 设计）；**废弃** 8020–8034、8005–8012
- **废弃** v3 扁平键 `billing_transaction_created` 等（迁为 `billing_transaction_created.intents.1_*`）

### 6.3 `bin/{event}/{intent}/config.yaml`

与 port_config merge；`enabled` 为 runAll / run.sh 是否启动的依据。

---

## 7. runAll（active + inactive 均登记）

**原则**：18 个 intent **全部**出现在 runAll；是否运行看 `enabled` + 用户 UI 操作，**不**因 historical inactive 而省略服务定义。

```yaml
- name: domain-events-intents
  services:
    - name: task-events-user-created-0-create-company
      build_command: "bash run.sh build user_created/0_create_company"
      start_command: "bash run.sh start user_created/0_create_company"
      # enabled 由 port_config + runAll 启动逻辑读取；false 时 start 组跳过，但服务仍可单独启
    - name: task-events-user-created-1-send-welcome-email
      # enabled: false 默认 — 仍在列表，UI 可手动启动
    # … 共 18 项
```

`run.sh start user_created`：仅启动该 event 下 **`enabled: true`** 的 intent；`start user_created/1_send_welcome_email` 可强制启单个。

---

## 8. 实施计划（G5 前置）

| 阶段 | 内容 | 验收 |
|------|------|------|
| **V4-0** | `intent_registry.go`、`LoadIntent(event,intent)` | unit |
| **V4-1** | 拆 `internal/intents/`；单 intent 事件先迁（billing/sse/…） | `go test ./...` |
| **V4-2** | 拆 `company_created` ×3（D2）+ fan-out 集成测试 | integration 绿 |
| **V4-3** | 补 `user_created/1_*`、notifications 三 inactive intent | exe 存在 |
| **V4-4** | `cmd/`/`bin/` 目录重组；删 v3 扁平 15 二进制 | `g45_verify` |
| **V4-5** | runAll 18 服务；health 8020–812x | health 全绿 |
| **G4.5** | 注册链 + 无双消费 | runbook 签字 |
| **G5** | 删 Python handlers + Django dispatch | **仅 V4+G4.5 后** |

---

## 9. 领域概念清单

| 类型 | v4 |
|------|-----|
| **Entity** | `EventIntentConsumer`（event + intent + enabled + groupId） |
| **Aggregate** | `IntentDeployment` = cmd + bin + internal/intents |
| **Domain Event** | 不变；intent 是消费侧切面 |

---

## 10. 价值流影响

**流**：`domain-events-consumer-split`

- 健康字段：`task-events-{event}-{intent}.health.status`（18 条）  
- 测试：per-intent integration；company 三 intent fan-out 为 **新 P0**  
- G5  blocked by v4 + G4.5（用户确认）

---

## 11. 与 v3 / cmd-layout 关系

| 文档 | 关系 |
|------|------|
| relocation v3.1 | event 级 → **升级为 intent 级** |
| cmd-layout | `cmd/{event}/{intent}/`；删域级 `cmd/accounts` 等 |
| bin-layout | `bin/{event}/{intent}/` 每 intent 一 exe |
| coverage-audit | 18 intent 覆盖原 15 event + company 拆细 |

---

## 12. 开放实现细节（非阻塞定稿）

- runAll 对 `enabled: false` 的默认组启动行为：skip + 可单启（实现时写清 README）  
- 端口分配表：实施 V4-0 时生成完整 18 端口矩阵 CSV  
