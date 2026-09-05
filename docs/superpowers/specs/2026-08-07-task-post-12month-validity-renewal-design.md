# 任务帖定价模型重构 — 0.33元/帖/12个月 + 单帖独立到期续存

> **变更记录 (2026-08-16)**：购买默认定价曾调整为 **0.55 元/帖/12个月**。续存只消耗 1 次创建帖配额，**不另扣费**（不以 `server_start_renewal` 标价记账）。
> **变更记录 (2026-08-18)**：购买默认定价一度改回 0.33 元，同日再次改为 **0.55 元/帖/12个月**。下文「续存同价」为当时决策原文，续存仍不另扣费。

> **迭代**: task-post-12month-validity-renewal
> **作者**: claude
> **日期**: 2026-08-07
> **状态**: ✅ delivered（G1–G6 全部完成，测试全绿）
> **架构版本**: v15 enterprise-landscape / v67 application-integration（基于 v14/v66 target 线）
>
> **交付验证**（2026-08-07）：
> - taskTaskService：`ok src 17.740s` + domain；OpenAPI drift gate OK（6 mounted / 6 public）
> - taskBill：`ok src 20.880s`；taskEvents：全包通过
> - taskFE：305 测试文件 / 1572 用例全绿（含 v15 存续期 2 新增用例）
> - runAll.yaml 已注册 task_post_renewed(18057) / task_post_expired(18058) / task_post_expiry_scan(18059)

---

## 1. 背景与目标

### 1.1 业务诉求（用户原话）

> 我想把任务帖子设置为 0.33元/帖/12个月，然后续存的话，0.33元/帖子/12个月。

### 1.2 需求澄清结论（用户已确认）

| # | 决策点 | 结论 |
|---|--------|------|
| 1 | 12 个月的含义 | **帖子存续期**：帖子（内容+执行历史）存续 12 个月，到期下架/归档 |
| 2 | 到期时间计算 | **单帖独立到期**：每帖各自 `expires_at`，续存只延展指定帖 |
| 3 | 存量配额 | **存量统一新规则**：存量帖上线日起统一按 12 个月到期 |
| 4 | 服务器启动扣费 | **剥离**：帖子存续期内的执行不再按次扣费 |
| 5 | 续存扣费 | **续存 = 消耗 1 次创建帖次数** → 帖子 +12 个月 |

### 1.3 新定价模型（目标态）

```
购买：0.33元/次 → 账户获得「创建帖次数」
创建新帖：消耗 1 次创建帖次数 → 帖子获得 12 个月存续期（单帖独立 expires_at = now + 12M）
存续期内：帖子可正常查看/执行，服务器启动不再按次扣费
帖子到期：下架/归档 —— 列表置灰、详情只读、不可执行/编辑/评论
续存：消耗 1 次创建帖次数 → expires_at = max(now, 当前到期日) + 12M（不缩短已购时长）
```

---

## 2. 现状分析

### 2.1 计费现状（实查 DB + 代码）

`task_bill.billing_unit` 实况：

| unit_type | price | unit | 现状用途 |
|-----------|-------|------|----------|
| `server_start` | **33 分** | 次 | 任务执行费（服务器启动扣费） |
| `gitlab_disk` | 800 分 | GB/月 | 磁盘（已有按月有效期+续费模式） |
| `gitlab_traffic` | 100 分 | GB | 流量 |
| `post_creation` | 0 分 | 次 | 帖子创建（0 元，未实际计费） |
| `recharge` | 1 分 | 次 | 充值 |

**现有任务帖计费链路**：
1. **购买**：`billing_resource_order`（resource_type=`task_post`，0.33元/帖）→ 支付 → `billing_account.task_post_quota += N`（**纯计数器，无到期时间**）
2. **创建帖子**：taskTaskService 创建任务时调 `/api/internal/taskbill/consume-task-post-quota/` → 消耗 1 配额（`consumeAndRecordTaskPostQuota`，orders.go:608）→ 记录 consumption 流水 + outbox
3. **服务器启动**：taskBill `/api/internal/taskbill/charge-server-start/` → 优先扣配额、余额不足时从账户余额扣 0.33元（charge.go:240 `chargeServerStart`）；余额不足返回 402 **阻断服务器启动**
4. **配额消耗顺序**：FEFO — 优先消耗带 `expires_at` 的赠送 grant（`billing_resource_grant`，最早到期先耗），再扣主计数器（orders.go:531/568）

### 2.2 可复用基建（重要）

| 基建 | 位置 | 说明 |
|------|------|------|
| `billing_resource_grant.expires_at` | taskBill | 已支持 per-grant 到期 + FEFO 消耗（赠送配额在用） |
| `diskExpiresAtFromMonths` | gitlab_resources.go:209 | `max(now, current_expires_at) + months` 续费算法（**本设计续存算法直接复用此模式**） |
| `server_start_renewal` 单位类型 | product_pricing_public.go:28 | 公开定价 API 已预留「任务续存价」字段（当前可空） |
| TASK_POST_CREATED 事件 | taskEvents taskpostcreation | 帖子创建事件已存在 |

### 2.3 关键代码位置

| 文件 | 作用 |
|------|------|
| taskBill/src/orders.go:393 | task_post 订单发放 → 配额计数器 +N |
| taskBill/src/orders.go:608 | `consumeAndRecordTaskPostQuota` 创建扣费 |
| taskBill/src/charge.go:240 | `chargeServerStart` 服务器启动扣费（**剥离目标**） |
| taskBill/src/resource_pricing.go | 定价读取/更新/API 组装 |
| taskBill/src/product_pricing_public.go | 公开定价 API（含 renewal 预留字段） |
| taskTaskService/src/task_handlers.go:92 | 创建任务 → consumeTaskPostQuota |
| taskTaskService/src/bill_client.go:24 | 扣费 HTTP 客户端 |
| taskCloudService/src/bill_client.go:91 | `chargeServerStart` 客户端（**dormant，仅测试引用**） |
| dataMigrate/taskTaskService/001_schema.sql | task_tasks 表结构（**无 expires_at，需新增列**） |

### 2.4 服务器启动扣费现状核实

`taskCloudService/src/bill_client.go` 的 `chargeServerStart`/`checkServerStartBalance` 客户端**当前无运行时调用点**（仅 `bill_client_test.go` 引用），`taskEvents` 侧也无扣费调用。活跃扣费路径仅为「创建任务消耗配额」。这意味着「剥离服务器启动扣费」的实际改动面较小：主要是 taskBill 端 `charge-server-start/` 接口语义变更（改 no-op 或改为帖子有效期校验），不涉及复杂下线回滚。

---

## 3. 🕸️ Code Review Graph 分析

**CRG unavailable**：`.code-review-graph/graph.db` 存在但已过期（2026-08-06 08:31 构建，34 小时前；head 不匹配），且知识图谱仅覆盖 12 个文件（taskBill/taskTaskService 不在图中，`callers_of consumeTaskPostQuota` 返回 not_found）。本设计基于直接源码探查（见 §2.3），未依赖图查询。建议实施阶段（/4-build 或 /5-tdd）重建图后补充爆炸半径验证。

---

## 4. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|--------|--------------|---------|
| 创建帖子（消耗 1 创建帖次数） | `TASK_POST_CREATED`（已有） | taskEvents taskpostcreation handler | 站内通知/Webhook；**扩展**：事件载荷携带 `expires_at` | — |
| 续存帖子（消耗 1 创建帖次数） | `TASK_POST_RENEWED`（**新增**） | 续存接口（taskTaskService → taskEvents） | 站内通知；前端刷新到期时间 | — |
| 帖子到期下架 | `TASK_POST_EXPIRED`（**新增**） | 到期扫描（taskEvents 定时） | 站内通知/归档落库 | — |
| 购买创建帖次数（支付成功） | `BILLING_TRANSACTION_CREATED`（已有事件族） | taskBill orders.go | 流水查询/管理后台 | — |
| 查询帖子存续期状态（只读） | — | — | — | 纯查询，无状态变更，无对应事件 |

> 设计硬门禁满足：所有改变系统事实的业务意图均有 MQ 事件投递；纯查询意图已注明例外理由。

---

## 5. 价值流影响

基于 `conf/value-stream.yaml` 盘点：

**受影响 streams**：
- `billing-account-event-driven-init` — `task-bill.billing_account.task_post_quota`（语义从「执行配额」变为「创建帖次数」）
- 任务创建流（task-management domain）— 创建任务后新增存续期字段写入

**字段影响**（三节段命名）：

| service.table.column | 变更 | 说明 |
|---------------------|------|------|
| `task-tasks.task_tasks.post_expires_at` | 🆕 新增列 | 帖子存续到期时间（DATETIME NULL，NULL=未启用存续期） |
| `task-bill.billing_unit.price` | 🟡 语义变更 | `server_start` 33分：执行费 → 创建帖费（12个月）；`server_start_renewal` 33分 新单位 |
| `task-bill.billing_account.task_post_quota` | 🟡 语义变更 | 执行配额 → 创建帖次数 |
| `task-bill.billing_usage.unit_id` | 🟡 扩展 | 记录续存 consumption 流水 |

**测试影响**：新增/更新 —— taskBill 续存扣费单测、taskTaskService 到期校验与续存接口单测、FE 定价/续存 UI 回归。

---

## 6. 详细设计

### 6.1 数据模型

#### 6.1.1 task_tasks 新增列（dataMigrate/taskTaskService 新 migration）

```sql
ALTER TABLE task_tasks ADD COLUMN post_expires_at DATETIME NULL;
-- 存量回填：上线日起统一 +12 个月（用户已确认「存量统一新规则」）
UPDATE task_tasks SET post_expires_at = DATE_ADD(NOW(), INTERVAL 12 MONTH) WHERE post_expires_at IS NULL;
CREATE INDEX idx_tasks_post_expires ON task_tasks(post_expires_at);
```

- **到期判定**：`expired = post_expires_at IS NOT NULL AND post_expires_at < now`（不新增 status 枚举值，由时间戳推导，避免状态漂移）
- **续存延展**：`post_expires_at = max(now, post_expires_at) + 12 个月`（复用 gitlab_resources.go `diskExpiresAtFromMonths` 同款算法，不缩短已购时长）

#### 6.1.2 taskBill billing_unit（价格不变，语义变更）

| unit_type | price | unit（改） | 语义 |
|-----------|-------|-----------|------|
| `server_start` | 33 分 | `帖/12个月` | 创建帖费用（创建+续存同价） |
| `server_start_renewal` | 33 分 | `帖/12个月` | 续存费用（**公开定价 API 已预留此类型**，仅补 seed 行） |

- `billing_account.task_post_quota` 保留列名（最小改动），注释/API 文案改为「创建帖次数」

### 6.2 API 设计（全部 Go 服务，不触发 Python 新增接口门禁）

#### 6.2.1 taskTaskService（公网）

> 路径遵循 `/api/${serviceName}/${funcName}/tenant_id/{tid}/workspace_id/{wid}/...` 键值对规范（`api-url-path-design.mdc`），任务域 funcName = `todos`（存量 `/api/tenant/*/workspace/*/todos/*` 的迁移目标，见 `2026-08-04-api-path-convention-audit.md`）。

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/{task_id}/renew/` | **新增**续存接口：校验权限（owner/workspace 成员）→ 调 taskBill internal 续存扣费（消耗 1 创建帖次数）→ `post_expires_at = max(now, cur) + 12M` → 返回新到期时间；配额不足返回 402 |
| POST | `/api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/` | **修改**创建接口：现有 consumeTaskPostQuota 调用点扩展，成功后写入 `post_expires_at = now + 12M`（tenant_id/workspace_id 路径段为权威身份，同时注入 `X-Auth-Tenant-Id` / `X-Workspace-Id` 头） |
| GET | `/api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/`（列表）、`/api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/{task_id}/`（详情） | **修改**：响应携带 `post_expires_at` / `post_expired` 只读字段 |
| — | 执行/编辑入口 | **修改**：帖子已过期（expired）时拒绝执行/编辑/评论（403 + 提示续存） |

> 存量裸路径（`/api/tasks/{task_id}/...` 位置参数形式）按「存量兼容」规则保留但不再扩展；本迭代全部新路径与前端调用一律走上述键值对规范形式。

#### 6.2.2 taskBill（internal + 公网）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/internal/taskbill/consume-task-post-quota/` | **修改**：返回体增加 `expires_at`（帖子到期时间），供 taskTaskService 写入 |
| POST | `/api/internal/taskbill/consume-task-post-renewal/` | **新增**续存扣费：扣 1 创建帖次数（FEFO）+ 记录 `server_start_renewal` consumption 流水 + outbox；幂等键 `task_post_renewal:{task_id}` |
| POST | `/api/internal/taskbill/charge-server-start/` | **剥离**：改为校验帖存续期有效则 no-op 成功（不再扣费）；随迁移期后删除 |
| GET | `/api/public/product-pricing/` | **修改**：`task_points`（33）语义=创建帖费；`task_renewal_points`（33，来自 server_start_renewal）替代 `task_renewal_points_per_month` 字段 |

#### 6.2.3 taskEvents

| 变更 | 说明 |
|------|------|
| 🆕 `taskpostrenewed` handler | 消费 `TASK_POST_RENEWED` → 站内通知/Webhook |
| 🆕 到期扫描 | 每日定时（复用既有调度机制）扫描 `post_expires_at < now AND post_expires_at IS NOT NULL` 的帖子 → 发布 `TASK_POST_EXPIRED`（通知用户续存） |
| 🟡 `taskpostcreation` handler | 事件载荷扩展 `expires_at` |

### 6.3 前端（taskFE）

| 页面 | 变更 |
|------|------|
| Pricing.vue | 任务帖展示「0.33 元/帖/12个月」，说明文案「每帖 12 个月存续期，到期可续存」 |
| OrderCreate.vue | 任务帖购买文案「创建帖次数」，单价说明更新 |
| 帖子列表/详情 | 展示到期时间 + 过期置灰态；过期帖显示「续存」按钮（调 renew API） |
| 创建任务弹窗 | 提示「消耗 1 帖（12 个月存续期）」 |
| 到期提示 | 收到 TASK_POST_EXPIRED 通知后引导续存 |

### 6.4 存量迁移（上线批次）

1. **DB 迁移**（dataMigrate）：task_tasks 加列 + 存量帖回填 `NOW() + 12M`（上线日起统一 12 个月）
2. **billing_unit seed**：`server_start_renewal` 33 分；`server_start` unit 更新为「帖/12个月」
3. **存量配额**：`task_post_quota` 保留（已购次数继续作为创建帖次数使用）
4. **用户沟通**：站内公告/系统通知告知新规则（存量帖统一 12 个月到期、执行不再按次扣费）

### 6.5 到期后行为细节

- **状态**：`expired`（时间戳推导）—— 列表置灰、详情只读（可看内容与执行历史）、不可执行/编辑/评论
- **续存恢复**：续存成功 → `expires_at` 延展 → 自动恢复可执行
- **宽限策略（建议）**：到期后保留 30 天只读宽限期，期内续存不缩短时长；宽限期后仍不续存 → 归档（不可见，仅管理后台可查）。*此点可在实施时按运营策略调整*

---

## 7. 🏛️ 架构变更影响

- **迭代版本**: v15 🎯 target（enterprise-landscape）/ v67 🎯 target（application-integration）
- **迭代名称**: 任务帖定价模型重构 — 0.33元/帖/12个月 + 单帖续存
- **作者**: claude
- **设计日期**: 2026-08-07

**新增文件**（每个视图三类伴生格式，缺一不可）：
- 🆕 `docs/architecture/v15-enterprise-landscape-<ts>-claude.puml`（基于 v14）
- 🆕 `docs/architecture/v15-enterprise-landscape-<ts>-claude.archimate`（含 Plateau/Gap/WP 架构变迁视图）
- 🆕 `docs/architecture/v15-enterprise-landscape-<ts>-claude.mermaid.md`
- 🆕 `docs/architecture/v67-application-integration-<ts>-claude.puml`（基于 v66）
- 🆕 `docs/architecture/v67-application-integration-<ts>-claude.archimate`（含 v66→v67 迁移与核心数据流视图）
- 🆕 `docs/architecture/v67-application-integration-<ts>-claude.mermaid.md`

**变更明细**：
- 🟢 [NEW] `server_start_renewal` 计费单位（taskBill）；`TASK_POST_RENEWED`/`TASK_POST_EXPIRED` 事件（taskEvents）；taskTaskService 续存接口
- 🟡 [MODIFIED] taskBill — server_start 计费语义（执行费→创建帖费）、`charge-server-start/` 剥离为 no-op、消费接口返回 expires_at
- 🟡 [MODIFIED] taskTaskService — `task_tasks.post_expires_at`、创建任务写存续期、到期校验、renew 接口
- 🟡 [MODIFIED] taskEvents — 载荷扩展 + 到期扫描 + 新事件 handler
- 🟡 [MODIFIED] taskCloudService — bill_client `chargeServerStart` 退役（dormant 代码清理）
- 🔴 [DEPRECATED] `charge-server-start/` 计费路径（剥离）；`task_renewal_points_per_month` 旧字段

### .archimate 架构变迁要点

| 元素类型 | 内容 |
|----------|------|
| **Plateau v66** | Current 基线 — 按次计费（server_start 33分/次 + 配额 FEFO） |
| **Plateau v67** | Target — 帖子存续期 12 个月 + 创建帖次数 + 续存 |
| **Gap** | 帖子无存续期概念、执行按次扣费、无续存入口 |
| **WorkPackage** | WP1 DB 迁移与存量回填 → WP2 taskBill 计费语义改造 → WP3 taskTaskService 存续期/renew → WP4 taskEvents 事件与扫描 → WP5 FE 展示与续存 UI |
| **视图** | `架构变迁 v66→v67 — 任务帖存续期` + `v67 Target — 创建/续存/到期数据流`（均含 sourceConnection 连线） |

---

## 8. 🐍 Python 新增接口清单与 Go 替代评估

**未触发**：本设计全部新增/修改接口均位于 Go 服务（taskTaskService 公网 renew、taskBill internal renewal 扣费、taskEvents handler），无 Django/Flask 新增 endpoint。Django saas-backend 已完全退役（v57，2026-07-30）。`python_api_approval: not_applicable`。

---

## 9. 风险与开放问题

| # | 风险/问题 | 影响 | 缓解 |
|---|----------|------|------|
| 1 | 收入模型变更（按次→按帖按年） | 高执行频率用户收入下降 | 产品侧评估；12 个月存续期内不限次执行 |
| 2 | 存量帖统一 12 个月到期 | 存量用户可能流失 | 30 天宽限只读 + 站内通知引导续存 |
| 3 | `charge-server-start/` 剥离后服务器启动门禁 | 余额校验移除后可能出现滥用 | 改为帖子存续期校验（expired 帖不可启动），云资源本身仍有 budget 管控 |
| 4 | 到期扫描任务部署 | 无扫描则过期帖永不下架 | 惰性校验（执行/查看入口）+ 定时扫描双保险 |
| 5 | server_start 计费下线对账单/流水展示影响 | 历史 consumption 流水含义变化 | 历史流水只读保留，新流水标注新单位 |

**实施决策（/goal 自动决策，2026-08-07）**：
- 宽限期：**不设宽限期** —— 到期即过期（`post_expires_at < now`），前端置灰/徽标提示 + 续存按钮即时恢复；风险 2 的"30 天宽限"由前端文案引导替代
- 存续期内执行：**不限次** —— `charge-server-start/` 已 no-op（G2），云资源 budget 管控兜底（风险 3 缓解已落地）
- 到期扫描：**每日 ticker（24h）+ 惰性校验双保险**（风险 4 缓解已落地）；扫描入口 `/api/internal/tasks/expire-posts/`（taskTaskService，requireInternalSecret）
- 续存规则：`expires_at = max(now, 当前到期日) + 12M`（不缩短已购时长）—— 若已到期按 now+12M
