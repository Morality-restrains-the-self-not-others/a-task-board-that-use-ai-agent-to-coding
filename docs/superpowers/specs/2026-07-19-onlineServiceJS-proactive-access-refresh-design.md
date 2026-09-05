# onlineServiceJS 主动 refresh-access（对齐 go_relay）

- **日期**: 2026-07-19
- **状态**: 已批准并实现（/goal 自动采纳推荐方案）
- **迭代**: onlineServiceJS-proactive-access-refresh
- **关联**: OPT-20260719-036；失败经验 `57_container_auto_run_steps_409_missing_access_token.md`
- **作者**: claude

## 问题陈述

任务 `task_13571260012264162867` 在 access TTL（**1 小时**）过后约 9.5h，平台 `container-auto-run-steps` 曾因 by-scope `TOKEN_EXPIRED` 映射为 409。审计显示容器仅在启动时各调用一次：

| 时间 | 事件 |
|------|------|
| 10:58:13 | `token_issued` |
| 11:00:32 | `exchange_refresh` + `refresh_access` |
| 之后至 21:29 | **无** 再次 `refresh_access` |

用户疑问：容器侧为什么没有 refresh-access？机制是否有问题？

## 架构理解（基线）

- 当前 tip：**v41 ✅ current**（`application-integration`）
- 凭据路径：容器 `onlineServiceJS` → SaaS `server-container-token/*` → **taskCredentialService**；平台转发走 Cloud `container-target` → by-scope
- 本迭代为 **onlineServiceJS 行为补齐**，不新增微服务、不改拓扑 → **不写架构 target**

📋 版本历史（最近）：v41 current — 任务子树状态；v40 archived — 排队自动执行节奏。

## 根因判定：机制缺口（非偶发故障）

### 现状对比

| 运行时 | 启动换票 | TTL 将尽主动续签 | 触发点 |
|--------|----------|------------------|--------|
| **go_relayToTrae** | ✅ exchange + refresh-access | ✅ `ensureFreshAccessToken`（skew=5m） | 每次 push 前 |
| **onlineServiceJS**（本任务 `stream_backend=trae`） | ✅ `runBootstrapTokenExchangeOnly` | ❌ **无** | 仅 bootstrap / 403·401 回退 |

### onlineServiceJS 实际行为

1. 启动：`exchange-refresh` → 立刻 `refresh-access` → 写入 `ACCESS_TOKEN` env + 落盘 `container_refresh_token.json`（含 `expires_at`）。
2. 心跳：`server-container-token/heartbeat/` **只上报存活**，不读 `expires_at`、不续签。
3. `expires_at` 已落盘，但 **没有任何 timer / 心跳钩子** 读取并触发 `runRefreshAccessOnly`。
4. 容器 auth 按 **env 字符串比对**，不强制平台 TTL → 进程可「假活」数小时，平台侧 `expires_at` 已过期。

结论：**不是「refresh 失败」**，而是 **trae 路径故意/遗漏未实现主动续签**（OPT-036 已记录）。与「服务未满 1 天」无关；access 固定 **1h** TTL。

## 设计目标

1. 长跑 `onlineServiceJS` 在 access 剩余寿命 ≤ skew 时主动 `refresh-access`，更新 env + 落盘 + 平台 DB（经既有 API）。
2. 与 go_relay 语义对齐：skew 默认 **5 分钟**；失败不杀进程，保留当前 token，打日志（硬失败仍靠后续 401/心跳诊断）。
3. 不单方在 credential `EnsureAccessByScope` 对「非空过期」换新（上一轮已改为原样转发）；**续签责任在持有 refresh 的容器进程**。

## 推荐方案（⭐）

在 `onlineServiceJS` 增加 **proactive access refresh 调度**：

| 项 | 约定 |
|----|------|
| 触发 | 独立 `setInterval`（建议 60s 轮询）或挂在心跳 tick 前 |
| 条件 | 已有 refresh_token；已解析 `expires_at`；`now >= expires_at - skew` |
| skew | 默认 5m；可用 env `TRAE_ACCESS_TOKEN_REFRESH_SKEW_SEC` 覆盖 |
| 动作 | `runRefreshAccessOnly` → 更新 `process.env.ACCESS_TOKEN` → `writePersistedRefreshToken`（含新 expires_at） |
| expires 未知 | 若落盘无 `expires_at`：可选「每 50 分钟保守刷新一次」或「跳过主动续签仅依赖 401 回退」——推荐 **保守周期刷新**（避免静默过期） |
| 并发 | 单飞锁，避免与 bootstrap / 手动回退重叠 |
| 可观测 | `token-refresh.log` / 既有 `appendTokenRefreshLog`：proactive begin/OK/fail |

**不做**：

- 不加长平台 1h TTL（安全窗口另议）
- 不在 Cloud/credential 对过期非空 access 单方 mint（防 env 401）
- 不新增 Python/Django 接口（复用既有 `refresh-access`）

## 备选（不推荐作主方案）

| 方案 | 优点 | 缺点 |
|------|------|------|
| A. 仅加长 TTL 至 24h | 改动小 | 泄漏窗口大；仍无主动续签则终将过期 |
| B. 心跳失败时再 refresh | 少定时器 | 心跳可能仍带旧 token「成功」，过期后才痛 |
| C. 平台 by-scope 对过期单方 refresh | 平台侧自愈 | 与容器 env 立即不一致 → 全线 401 |

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|--------|--------------|---------|
| 容器主动续签 access | （既有）`refresh_access` audit | taskCredentialService `RefreshAccess` | 审计表；无跨聚合新副作用 | 复用既有命令；无新 MQ 事件 |
| 查询 auto-run-steps | — | — | — | 只读转发，无对应事件 |

## Domain Concept Inventory（轻量）

- **BC**: Container Credentials（taskCredentialService + 容器 runtime）
- **Entity**: `ContainerToken`（access / refresh / expires_at）
- **命令**: `RefreshAccess`（已有）；新增仅为 **调度触发**

## Value Stream 影响

- 触及：云平台 / 任务容器连接与转发（心跳、container-target、L0 代理）
- 字段：`taskCredentialService.container_tokens.container_access_token` / `*_expires_at`（既有列，无新列）
- 测试：onlineServiceJS 单测（timer/skew）；可选 Playwright：长 TTL mock 下 by-scope 仍 200

## 验收标准

1. 单元：expires 在 skew 内 → 调用 refresh-access mock 一次；未到期不调用。
2. 集成：启动换票后将 expires 拨到过去+skew，等待一轮调度 → DB `refresh_access` 审计新增、`expires_at` 更新。
3. 回归：bootstrap 403/401 回退路径不变；`TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE` 仍跳过。

## 🏛️ 架构变更影响

- **不需要**新版本架构文件（无组件/数据流增删改；行为补齐属运行时修复）。
- 关闭缺口：OPT-20260719-036（落地后标记 completed）。

## 风险

| 风险 | 缓解 |
|------|------|
| 续签失败 | 保留旧 token；日志 + 连续失败计数；不 fail-closed 杀监听（与 go_relay soft skip 一致） |
| 时钟漂移 | skew≥5m；UTC 解析与 credential 一致 |
| 旧镜像未部署 | 文档注明需重建/推送容器镜像；平台侧过期转发兜底仍有效 |

## 批准门

- 无新增 Python 接口 → 跳过 Python 专项审批。
- `/goal` 已自动批准并落地：`proactiveAccessRefresh.mjs` + `server.mjs` 调度。

## 实现摘要（2026-07-19）

| 文件 | 变更 |
|------|------|
| `onlineServiceJS/src/proactiveAccessRefresh.mjs` | skew/保守续签判定 + `startProactiveAccessRefreshLoop` |
| `onlineServiceJS/src/proactiveAccessRefresh.test.mjs` | 单测 |
| `onlineServiceJS/src/bootstrap.mjs` | `readPersistedTokenStore`；导出 `runRefreshAccessOnly` |
| `onlineServiceJS/src/server.mjs` | 心跳旁启动主动续签循环 |

Env：`TRAE_ACCESS_TOKEN_REFRESH_SKEW_SEC`（默认 300）、`TRAE_ACCESS_TOKEN_REFRESH_POLL_SEC`（默认 60）、`TRAE_SKIP_PROACTIVE_ACCESS_REFRESH`。
