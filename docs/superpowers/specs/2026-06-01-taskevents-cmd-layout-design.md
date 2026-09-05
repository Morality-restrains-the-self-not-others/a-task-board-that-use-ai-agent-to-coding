# taskEvents `cmd/` 目录结构整理

> 日期：2026-06-01  
> 状态：**已确认方向 — 无域级残留**  
> 触发：用户检阅 `taskEvents/cmd` 目录混乱  
> 用户决策：**不应再有域级目录；域级应全部拆为事件级**  
> 关联：relocation-design v3.1、bin-layout-correction、G4.5 runbook

---

## 1. 现状检阅

### 1.1 当前 `cmd/` 树（混乱来源）

```
taskEvents/cmd/
├── accounts/main.go              # ❌ 域级 legacy · Django HTTP dispatch
├── billing/main.go               # ❌
├── cloud/main.go                 # ❌
├── notifications/main.go         # ❌
├── projects/main.go              # ❌
├── realtime/main.go              # ❌
└── event/                        # ❌ 多余嵌套
    ├── _template/main.go
    └── user_created/ …（15 个事件级 main）
```

### 1.2 同类问题：`bin/` 根目录

```
bin/task-events-accounts          # ❌ 域级产物（与 bin/user_created/ 布局不一致）
bin/task-events-billing
…（6 个）
bin/user_created/task-events-user-created   # ✅ 事件级
```

### 1.3 两类 main 的差异（为何域级应消失）

| | 域级 `cmd/accounts` | 事件级 `cmd/event/user_created` |
|--|---------------------|----------------------------------|
| 订阅 | 多 event_type | 单一 event_type |
| 执行 | Django `/dispatch/` | `internal/handlers/*` 本地 Go |
| 端口 | 8005–8012 | 8020–8034 |
| 终态 | **删除** | **保留** |

15 个事件二进制 **已全部实现**；域级进程仅 G4.5 迁移期双轨残留，**不应再保留目录占位**。

---

## 2. 用户确认的目标

> 「不应该还有域级的，域级应该都拆为事件级。」

据此：

- **不采用** `cmd/_legacy/` 隔离方案  
- **终态 `cmd/` 仅含** 15 个事件入口 + 脚手架  
- 域级 `cmd/{accounts,…,notifications}` **直接删除**（非搬迁）  
- 同步删除 `run.sh` 域级 build/start、`bin/task-events-*` 根文件、runAll `domain-events` 组  

---

## 3. 目标布局

```
taskEvents/
  cmd/
    README.md                       # 目录说明 + slug ↔ port 索引
    _template/main.go               # 新事件脚手架
    user_created/main.go            # 与 bin/user_created/ 同名
    company_created/main.go
    billing_transaction_created/main.go
    sse_message/main.go
    …（共 15 个，无 event/ 中间层）
  bin/
    user_created/
      config.yaml
      task-events-user-created
    …（15 个子目录，无 bin 根级 task-events-accounts）
  internal/handlers/{event}/        # 不变
```

**构建**（统一一种）：

```bash
go build -o "bin/${slug}/task-events-${slug//_/-}" "./cmd/${slug}"
# 或经 run.sh build user_created
```

---

## 4. 删除清单（与 G5 对齐，可提前至 G4.5 末）

### 4.1 源码

| 路径 | 说明 |
|------|------|
| `cmd/accounts/` | 域级 main |
| `cmd/billing/` | |
| `cmd/cloud/` | |
| `cmd/notifications/` | 已拆为 email_sent / invitation_created / user_activated |
| `cmd/projects/` | 已拆为 4 个 projects 事件 |
| `cmd/realtime/` | 已拆为 sse_message |
| `cmd/event/` | 整层上移后删除空目录 |

### 4.2 构建 / 运维

| 路径 | 说明 |
|------|------|
| `run.sh` 中 `DOMAINS`、`build_domain`、`start_domain`、`stop_domains` | 域级命令 |
| `bin/task-events-{accounts,…,notifications}` | 6 个根级可执行文件 |
| `runAll.yaml` → `domain-events` 组 | 仅保留 `domain-events-bin` |
| `port_config.json` → `domainEvents.consumers.accounts` 等域级键 | 可选 G5；或 `enabled: false` |

### 4.3 Django / Python（G5，可与 cmd 删除同批）

- `Saas_project/core/kafka/handlers/`  
- `core/task_events/*/dispatch` internal URLs  

---

## 5. 迁移步骤

| # | 动作 | 验证 |
|---|------|------|
| 1 | `git mv cmd/event/{slug} → cmd/{slug}`（15 个） | |
| 2 | `git mv cmd/event/_template → cmd/_template` | |
| 3 | 删除 `cmd/event/`、`cmd/accounts` … `cmd/realtime` | |
| 4 | 新增 `cmd/README.md` | |
| 5 | 精简 `run.sh`：仅 `EVENTS` + `build_event`/`start events` | `./run.sh build all` |
| 6 | 删除 `bin/task-events-*` 根文件；`.gitignore` 已覆盖 | |
| 7 | 从 `runAll.yaml` 移除 `domain-events` 组 | runAll 校验 |
| 8 | `./scripts/g45_verify.sh` + `go test ./...` | 全绿 |
| 9 | G5：Python handlers + Django dispatch | pytest / E2E |

**预估**：1 个 PR，~45 分钟，无 handler 逻辑变更。

---

## 6. 领域概念清单

| 类型 | 终态 |
|------|------|
| **Bounded Context** | taskEvents 消费侧仅「单事件进程」 |
| **Entity** | `EventConsumerBinary`（slug、port、groupId）— **无** LegacyDomainConsumer |
| **Aggregate** | `PerEventDeployment` = `cmd/{slug}` + `bin/{slug}` + handler |
| **Domain Event** | 15 种 event_type，1:1 映射到 `cmd/{slug}` |

---

## 7. 价值流影响

**流**：`domain-events-consumer-split`

| 维度 | 影响 |
|------|------|
| 行为 | 运行面仅事件级 consumer；`task-events-accounts.health` 等域级 health 字段 **废弃**，改为 `task-events-user-created.health` 等 |
| 测试 | `test_task_events_*_dispatch.py` 在 G5 删除或改为 Redis 注入；Go integration 已覆盖 |
| value-stream.yaml | 后续 step 可将 `task-events-accounts.*` 字段迁移为事件级 slug（step 3 价值流映射时做） |

---

## 8. 与原方案 A 的差异

| 原方案 A | 用户确认后 |
|----------|------------|
| `cmd/_legacy/` 暂存域级至 G5 | **直接删除**域级 cmd |
| G4.5 双轨可并存 | **仅** `domain-events-bin` / 事件级 |
| `cmd/event/` 保留 | **扁平** `cmd/{slug}/` |

---

## 9. 下一步

用户已确认方向 → 建议 **直接实施**（`/7-build` 或单 PR），无需再保留 legacy 目录。
