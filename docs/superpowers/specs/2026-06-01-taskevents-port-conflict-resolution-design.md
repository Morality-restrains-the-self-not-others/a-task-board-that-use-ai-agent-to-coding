# taskEvents 端口冲突治理设计

> 日期：2026-06-01  
> 状态：**已确认并实施（方案 C）** — 用户选定 18020–18037 高端口段  
> 触发：G4.5 验收暴露两类端口问题 — (1) 本机 **8021** 被外部进程占用；(2) v3 扁平 `port_config` 导致 **company_created** 多 intent 误绑同一端口  
> 关联：`2026-06-01-taskevents-intent-binary-layout-design.md` §6.2、`task2app/conf/port_config.json`、`intent_registry.go`

---

## 0. 问题摘要

| 类型 | 现象 | 根因 |
|------|------|------|
| **配置冲突** | intent 2/3 监听 8026，`EADDRINUSE` | v3 扁平键 `company_created.port=8026` 被 `LoadIntent` 误套到所有 intent（已在代码层限制「仅 primary intent 继承 legacy」，但 **port_config 未迁嵌套**） |
| **主机冲突** | SSE intent 无法 bind 8021，`g45` 只能 SKIP | 本机未知进程长期占用 8021；`lsof` 不可见（权限/系统服务） |
| **遗留噪音** | `port_config` 仍含 8005–8012 域级 consumer | G5 已删域级 `cmd/*`，配置未同步清理 |
| **门禁绕过** | `check_event_health.sh` 对不可 bind 端口 SKIP | 临时措施，掩盖真实部署失败 |

**目标**：单一配置源、18 intent 端口互不重叠、启动前可诊断冲突、runAll/health 与 `port_config` 一致。

---

## 1. 领域概念清单（供 DDD 参考）

| 概念 | 说明 |
|------|------|
| **Bounded Context: taskEvents 编排** | `run.sh`、runAll、`bin/{event}/{intent}/` |
| **Bounded Context: 端口配置** | `port_config.json` + `intent_registry.go` + bin overlay |
| **Entity: IntentConsumer** | 一个 `{event}/{intent}` 二进制，属性：port、groupId、enabled、subscribed events |
| **Value Object: PortAllocation** | 8020–8099 主 intent 块；8120–8199 同 event 附加 intent 块 |
| **Domain Event** | 无新业务事件；属基础设施变更 |

---

## 2. 价值流影响

读取 `value-stream.yaml` 中 **领域事件 / task-events-*** 相关步骤：

| 影响 | 说明 |
|------|------|
| **受影响 stream** | 领域事件消费健康检查（仍引用 `task-events-accounts` 8005 等 **已废弃** 域级服务名） |
| **字段变更** | 健康字段应改为 intent 级，如 `task-events-sse-message-1-send-sse-message.health.status`；端口从 8005–8012 迁移到 8020–8099 / 8120+ |
| **测试** | `g45_verify.sh`、`check_event_health.sh`；Go integration 不变（不绑 HTTP 端口） |
| **状态** | 域级 consumer 步骤应标记 **deprecated**，新增 `domain-events-intents` 组 18 项 |
| **跨 stream** | 无新依赖；仅运维/编排层 |

完整 value stream 切片在 **Step 3 `/3-value-stream-价值流`** 更新 YAML。

---

## 3. 方案对比

### 方案 A — 仅清理配置 + 启动诊断（最小改动）

- 删 `port_config` 中 8005–8012 与 v3 扁平 event 键
- 写入 v4 嵌套 `consumers.{event}.intents.{intent}`
- `run.sh start` 前 `port_preflight`：不可 bind 则 **失败并打印占用提示**
- **8021 不动**；冲突机器靠 `bin/.../config.yaml` 本地 overlay 改端口

| 优点 | 缺点 |
|------|------|
| 改动面小 | 8021 主机冲突需每环境手工 overlay |
| 与 v4 设计 §6.2 一致 | 无统一「避坑端口」策略 |

### 方案 B — 重划端口块 + 迁 SSE（推荐）

在方案 A 基础上：

1. **保留主 intent 8020–8034**（与现 runAll health URL 一致，除 SSE）
2. **附加 intent 统一规则**：`8120 + event_index*10 + intent_order`  
   - 例：`user_created/1_*` → **8125**（保持）  
   - `company_created/2_*` → **8126**，`3_*` → **8136**（保持，与 registry 一致）
3. **SSE 默认端口 8021 → 8041**（仍在 8020–8099 主块，避开常见占用 8021）
4. `intent_registry.go` 与 `port_config` **双写同步**（registry 为默认，port_config 可覆盖）

| 优点 | 缺点 |
|------|------|
| 解决已知 8021 主机冲突 | runAll / 文档 / 监控 URL 需批量改 SSE → 8041 |
| 配置与代码单一真相 | 开发者本地若已习惯 8021 需知悉 |

### 方案 C — 高端口段 18020–18037

全部 intent 迁到 `18020+`，与 Django 8xxx 完全隔离。

| 优点 | 缺点 |
|------|------|
| 极少与本地服务冲突 | 与 v4 已定 8020–8034 **推翻**；runAll/文档全量改 |
| | 防火墙/习惯成本 |

**推荐：方案 B** — 在不大改主块的前提下，只动 SSE 默认端口 + 完成 port_config 嵌套迁移 + 启动 preflight。

---

## 4. 目标端口表（方案 B）

### 4.1 主 intent（8020–8041，8040–8049 预留）

| event/intent | port | enabled |
|--------------|------|---------|
| billing_transaction_created/1_process_billing_transaction | 8020 | true |
| sse_message/1_send_sse_message | **8041** | true |
| email_sent/1_send_email | 8022 | false |
| invitation_created/1_send_invitation_email | 8023 | false |
| user_activated/1_send_welcome_notification | 8024 | false |
| user_created/0_create_company | 8025 | true |
| company_created/1_set_default_deliverable_system | 8026 | true |
| workspace_created/1_process_workspace_creation | 8027 | true |
| project_updated/1_process_project_update | 8028 | true |
| task_completed/1_process_task_completion | 8029 | true |
| ai_assistant_reply_completed/1_persist_assistant_reply | 8030 | true |
| cloud_server_started/1_process_server_start | 8031 | true |
| cloud_server_stopped/1_process_server_stop | 8032 | true |
| cloud_server_start_auto/1_process_server_start_auto | 8033 | true |
| cloud_platform_authorization_created/1_process_cloud_platform_authorization | 8034 | true |

**8021**：标记 `deprecated`，文档注明「历史 SSE 端口，勿再使用」。

### 4.2 附加 intent（8120+）

| event/intent | port |
|--------------|------|
| user_created/1_send_welcome_email | 8125 |
| company_created/2_set_default_progress_system | 8126 |
| company_created/3_create_default_workspace | 8136 |

规则：`8136` 保留（已部署）；新 event 附加 intent 优先 `812{n}` 连续分配，避免与 8136 撞车时可跳号。

---

## 5. `port_config.json` 目标结构

```json
"domainEvents": {
  "transport": "redis",
  "consumers": {
    "sse_message": {
      "intents": {
        "1_send_sse_message": {
          "enabled": true,
          "host": "127.0.0.1",
          "port": 8041,
          "groupId": "task-events-sse-message-1-send-sse-message",
          "events": ["SSE_MESSAGE"]
        }
      }
    },
    "company_created": {
      "intents": {
        "1_set_default_deliverable_system": { "port": 8026, "...": "..." },
        "2_set_default_progress_system": { "port": 8126, "...": "..." },
        "3_create_default_workspace": { "port": 8136, "...": "..." }
      }
    }
  }
}
```

**删除**：

- `consumers.accounts` … `consumers.notifications`（8005–8012）
- 扁平键 `consumers.billing_transaction_created` 等（8020–8034 单层）

**加载优先级**（不变）：`intent_registry` 默认 → `port_config` intents → `bin/{event}/{intent}/config.yaml` overlay。

**Legacy 回退**：仅当 `consumers.{event}` **无** `intents` 子键时，且当前 intent 为该 event 的 **PrimaryIntent**，才读扁平 `port/groupId`（兼容过渡期）；G5 后扁平键删除，回退路径可删。

---

## 6. 启动与门禁

### 6.1 `run.sh start_intent`

```
port_preflight(event, intent):
  1. 读取最终 port（LoadIntent）
  2. 尝试 bind 127.0.0.1:port（Python/socket 探测）
  3. 失败 → 打印「port 被占用 / 不可 bind」+ 建议改 bin overlay 或 port_config
  4. 非零 exit（不再 silent skip 后 health FAIL）
```

### 6.2 `check_event_health.sh`

- 移除「不可 bind 则 SKIP」逻辑
- 依赖 preflight 保证进程已监听；health 仅 curl + 重试

### 6.3 `g45_verify.sh`

- 启动前 `kill` + `lsof` 清理 **8041**（替换 8021 在清理列表中）
- 全绿标准：integration + 全部 **enabled** intent health OK

---

## 7. 代码与文件变更清单

| 文件 | 变更 |
|------|------|
| `task2app/conf/port_config.json` | 嵌套 intents；删 8005–8012 与扁平 8020–8034 |
| `taskEvents/config/intent_registry.go` | SSE port 8021→8041 |
| `taskEvents/bin/README.md` | 更新端口表 |
| `runAll.yaml` | SSE health URL → 8041 |
| `taskEvents/scripts/check_event_health.sh` | 端口表 + 去 SKIP |
| `taskEvents/scripts/g45_verify.sh` | 清理端口列表 |
| `taskEvents/run.sh` | `port_preflight` |
| `value-stream.yaml` | 域级 health 字段 → intent 级（Step 3） |

**不在范围**：handler 业务逻辑、Redis groupId 变更（groupId 不变则无需 re-consume）。

---

## 8. 测试计划

| 层级 | 内容 |
|------|------|
| Unit | `TestLoadIntentCompanyFanOutPorts`、新增 `TestLoadIntentSSEPort8041` |
| Integration | 现有 Redis 测试不变 |
| Gate | `g45_verify.sh` 全绿（含 SSE 8041） |
| Manual | 本机 `lsof -i :8041` 启动后可见 LISTEN |

---

## 9. 风险与回滚

| 风险 | 缓解 |
|------|------|
| 8041 也被占用 | bin overlay 改 port；preflight 报错明确 |
| runAll 旧 URL 缓存 | 一次性改 runAll + README |
| 双写 registry/port_config 漂移 | 单 PR 同批改；CI 可加「registry 与 port_config 端口一致」脚本（可选） |

回滚：恢复 `intent_registry` 8021 + 扁平 port_config（不推荐，仅应急）。

---

## 10. 实施顺序（建议）

1. **P0** — `port_config` 嵌套 + 删 8005–8012（消除配置层冲突）
2. **P0** — SSE 8041：`intent_registry` + port_config + runAll + scripts
3. **P1** — `run.sh port_preflight` + 去掉 health SKIP
4. **P1** — `g45_verify` 全绿验收
5. **P2** — value-stream.yaml intent 级 health 字段（/value-stream 步骤）

---

## 11. 用户确认（2026-06-01）

- **选定方案 C**：全部 18 intent 连续映射 **18020–18037**
- **已实施**：`intent_registry.go`、`port_config.json`（嵌套 intents，删 8005–8012）、runAll、scripts、`run.sh port_preflight`
- **验收**：`g45_verify.sh` 全绿（含 SSE 18021）

---

## 变更日志

- 2026-06-01：初稿
- 2026-06-01：用户确认方案 C 并完成实施
