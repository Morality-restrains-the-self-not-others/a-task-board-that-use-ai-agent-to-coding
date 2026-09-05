# 设计文档：relayToTrae Token 生命周期与云主机启动对齐

- **日期**: 2026-07-10 18:58
- **作者**: claude
- **状态**: ✅ 已批准（goal-mode 自动采用方案 A+A1，2026-07-10 19:03）
- **关联页面**: `http://183.250.1.132:4000/tenant/850256677331562496/workspace/861623708318031872/task-detail/task_12590983282794675865/?relayToTrae=true`
- **关联 trace**: `web-1783680933775-fsytjos9uqt`（启动后 UI 拉 layers/jobs 热路径）

---

## 1. 问题概述

用户在 `?relayToTrae=true` 任务详情页点击「启动」后，本地模拟启动的 Token 生命周期与云主机 UserData 拉起容器不一致：

| 阶段 | 云主机（真实） | 当前 relayToTrae（模拟） |
|------|----------------|--------------------------|
| 签发 | Go `POST /v1/token/init` → bootstrap `ACCESS_TOKEN` | 同左（tcg / Django → CRED） |
| 换票 | **容器内** `onlineServiceJS` 执行 `exchange-refresh` + `refresh-access` | **父进程** `go_relayToTrae` 预换票 |
| 子进程 | 无 skip | 强制 `TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE=1`，容器跳过换票 |

结果：模拟路径无法复现远端容器换票失败（如 `TOKEN_ACCESS_INVALID`）、换票时序、以及 bootstrap 对 `business_api_endpoint` 的依赖，削弱「模拟启动 = 真实容器启动」的验收价值。

用户目标：**调整 relayToTrae 的 Token 颁发/换票方式，与启动云服务器时一致**。

---

## 2. Trace 日志分析

- **Grafana Trace Dashboard**: [打开](http://localhost:3000/d/distributed-trace-view?var-trace_id=web-1783680933775-fsytjos9uqt&var-tempo_trace_id=web-1783680933775-fsytjos9uqt)
- **Grafana 日志搜索**: [打开](http://localhost:3000/explore?orgId=1&left={"datasource":"loki","queries":[{"refId":"A","expr":"{job=~\".+\"} |= \"web-1783680933775-fsytjos9uqt\"","queryType":"range"}]})
- **时间范围**: 2026-07-10 ~18:55:33+08
- **涉及服务**: task-container-gateway, task-auth, task-cloud-service, go-relay / onlineServiceJS

### 日志摘要（用户粘贴 + Loki）

1. `onlineServiceJS`：`token-exchange: skipped (TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE)`
2. 注册可达地址、心跳、层图推送调度正常
3. `BOOTSTRAP_PHASE=task_detail_begin` → 拉任务详情（2 仓）→ 拉克隆凭证
4. 同窗口可见 `[relayToTrae] token-exchange: performing exchange` → `exchange-refresh OK` → `refresh-access OK`（父进程）
5. 再次 `start: bash .../run.sh`，子进程再次 skip 换票
6. UI trace `web-1783680933775-fsytjos9uqt`：tcg `auth_validate` → `cloud_resolve` → upstream layers/jobs 200（热路径正常，与 Token 对齐问题正交）

### 关键发现

- 父进程换票 + 子进程 skip 是**刻意设计**（见 `docs/design-relayToTrae-token-expiry-fix.md`），当时为修复「双端都 skip 导致过期」；现与「模拟=真实」目标冲突。
- Go 签发 token 形如 `tok_<snowflake>`（`len≈17`），与日志 `initial_access_token len=17` 一致，属正常。

### 根因假设

模拟启动的换票责任落在 `go_relayToTrae`，而非 `onlineServiceJS`，导致与云主机 UserData 路径语义分叉。

---

## 3. 架构理解（基线）

根据当前架构设计稿：

- 共有多版本视图；**current 应用集成**为 **v13**（`v13-application-integration-20260710-1505-claude.puml`）；企业全景 current 仍为 v1（后续迭代多在 application-integration）
- 应用层关键组件：Vue → APISIX → taskContainerGateway → taskAuth / taskCloudService；**taskCredentialService** 为容器 Token SSOT；**onlineServiceJS** 为容器业务进程；**go_relayToTrae** 为本地模拟侧车
- 技术层：CRED `:8015` 持有 `container_tokens`
- 上次交付架构版本：**v13** ✅ shipped（容器执行热路径零 Django）

📋 架构版本历史（近几版）：

- v13 (2026-07-10) ✅ shipped — Container Exec Hot-Path Zero-Django
- v12 🎯 target — relay clear-logs path scope
- v11 ✅ shipped — Vendor Cloud Test Credentials SSOT

本次需求将在 **v13** 基础上设计 **v14**：调整 go_relay ↔ onlineServiceJS ↔ CRED 的换票责任边界（数据流变更，需更新 `application-integration`）。

---

## 4. 目标行为（与云主机对齐）

```text
[签发] tcg/Django → CRED POST /v1/token/init
         → bootstrap ACCESS_TOKEN（未 exchange）
[注入] go_relay 将 bootstrap ACCESS_TOKEN + TASK/BUSINESS API env 传给 onlineServiceJS
         → 不设置 TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE（或显式 =0）
[换票] onlineServiceJS bootstrap：
         exchange-refresh(access, business_api_endpoint)
         → refresh-access(refresh)
         → 落盘 refresh_token（已有 container_refresh_token.json）
[侧车] go_relay status-push / 主动续期：
         在子进程换票成功后，读取落盘 refresh_token（或监听换票完成信号）
         → 自行 refresh-access 同步 state.AccessToken / RefreshToken / ExpiresAt
```

云主机无 go_relay；对齐点是 **「谁执行首次 exchange」= 容器进程**。

---

## 5. 方案对比

| 方案 | 描述 | 优点 | 缺点 | 推荐 |
|------|------|------|------|------|
| **A. 子进程换票（云对齐）** | 去掉父进程 `performTokenExchange` 与 `TRAE_SKIP=1`；父进程启动后同步 refresh | 与 UserData 路径一致；能复现换票类故障 | 需处理 status-push 的 token 同步时序 | ⭐ |
| B. 维持父换票 | 仅文档说明差异 | 改动小 | 不满足「真实模拟」 | ❌ |
| C. 双模式开关 | 默认 A，env 可回退父换票 | 可回滚 | 两套语义，测试面翻倍 | 次选 |

**采用方案 A。**

### 5.1 父进程 status-push 同步策略（方案 A 子设计）

| 子方案 | 做法 | 取舍 |
|--------|------|------|
| **A1. 读落盘 refresh + refresh-access** | 子进程已写 `runtimeDir()/container_refresh_token.json`；父进程轮询/等日志后读取并 `refresh-access` | 复用现有落盘；与独立模式一致 | ⭐ |
| A2. 解析子进程 stdout「exchange-refresh OK」后再调 CRED | 不依赖文件 | 日志格式耦合 |
| A3. 子进程 HTTP 回调父进程上报 token | 显式契约 | 新增接口面，超出最小对齐 |

**采用 A1**：启动后等待换票完成（超时可配置，默认与 bootstrap token timeout 对齐，如 15–30s），读取 refresh 文件 → `refreshAccessToken` → 写入 `state`，再开始/继续 status-push。

若超时未换票成功：start 返回失败（或标记 `state.Error`），与云主机「换票失败则容器不可用」一致，**禁止**静默用 bootstrap token 做 status-push（exchange 后 bootstrap access 已清空，必 401）。

### 5.2 selected_image（Docker）路径补齐（2026-07-10 追记）

子进程 `run.sh` 路径已实现 A1；**selected_image** 原先在 `container_image.go` 跳过 host token wait，导致 status-push 仍用已作废 bootstrap token → `TOKEN_ACCESS_INVALID`。

**启动前状态清理（2026-07-14）**：每次 `docker run` 前清空 `RELAY_ONLINE_STATE_BASE`（默认 `go_relayToTrae/.relay_online_state`）下全部 per-task 残留；root 属主删不掉时用刚 pull 的业务镜像做 `docker run --entrypoint rm`。见 `docs/intents/container/010_relay_clear_state_on_container_start.intent.md`。

补齐方式（仍属 A1）：
1. `docker run -v <hostStateRoot>:/app/onlineProject_state`，并设置 `ONLINE_PROJECT_STATE_ROOT=/app/onlineProject_state`
2. 启动后 `waitForChildContainerTokens` 读取落盘 `container_refresh_token.json`
3. 同步 `state.AccessToken` / `RefreshToken` 后再 `registerTaskLocked` + status-push

---

## 6. Domain Concept Inventory（供 /5-ddd）

| 概念 | 说明 |
|------|------|
| **Bounded Context** | Container Runtime Credential（CRED）；Local Relay Sidecar（go_relay）；Container Agent（onlineServiceJS） |
| **Entity** | `ContainerToken`（access/refresh/expires，SSOT 在 CRED） |
| **Aggregate** | Task-scoped ContainerTokenSession（tenant/workspace/task） |
| **Domain Events（候选）** | `ContainerBootstrapTokenIssued`、`ContainerRefreshExchanged`、`RelaySidecarTokenSynced` |
| **不变式** | 首次 `exchange-refresh` 一次性；模拟与云主机均由 **容器进程** 执行首次 exchange |

---

## 7. 价值流影响（输入给 /3-value-stream）

| 问题 | 评估 |
|------|------|
| 影响的现有流 | `relay-to-trae-direct-start`、`relay-token-exchange-no-proxy-consistency`、`relay-register-token-exchange-guard` 及相关 Playwright |
| 是否新流 | 建议增量：`relay-token-lifecycle-cloud-parity`（断言子进程出现 `exchange-refresh OK` 且无 `TRAE_SKIP` skip 行） |
| 字段 | `taskCredentialService.container_tokens.container_access_token` / `container_refresh_token`（语义不变，写入方从 go_relay 改回 onlineServiceJS） |
| 测试 | 更新 `go_relayToTrae` 单测（`TestBuildChildEnv*`、token exchange 启动路径）；Playwright relay start；onlineServiceJS e2e 去掉「仅 skip」假设 |

---

## 8. 变更范围（实现预览，批准后执行）

| 组件 | 变更 |
|------|------|
| `go_relayToTrae/src/process.go` | 启动路径**不再**调用 `performTokenExchange`；`buildChildEnv` **不再**注入 `TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE=1` |
| `go_relayToTrae/src/process.go` / 新辅助 | 启动后等待子进程换票 → 读 `container_refresh_token.json` → `refresh-access` → 更新 state |
| `go_relayToTrae/src/token.go` | 保留 `performTokenExchange` 供单测/可选工具，或标记仅测试；生产启动路径不用 |
| `trae-agent/onlineServiceJS` | **默认行为不变**（无 skip 时自行换票）；e2e 仍可用显式 `TRAE_SKIP=1` 测旁路 |
| 文档 | 更新 `machine_container.md` / 废止「父进程预换票 + SKIP」为模拟默认；修订 `design-relayToTrae-token-expiry-fix.md` 结论 |
| 架构 | v14 `application-integration`：go_relay ⇄ CRED 的「启动前 exchange」改为「启动后 sync via refresh」；OSJS → CRED exchange 标为与云主机同路径 |

### 不在范围

- 不改 CRED `IssueToken` / `ExchangeRefresh` 语义
- 不改云主机 UserData 路径（已与 Go init 对齐的修复保持）
- 不新增 Python HTTP 接口

---

## 9. 🐍 Python 新增接口清单与 Go 替代评估

**触发判定**：本次**不新增**任何 Python HTTP 接口 → **不触发** Python 接口专项审批。

| # | 方法 | 路径 | 归属 | 说明 |
|---|------|------|------|------|
| — | — | — | — | 无 |

选型：全部在 **Go（go_relayToTrae）+ Node（onlineServiceJS）** 落地。

---

## 10. 🏛️ 架构变更影响

- **迭代版本**: v14 🎯 target
- **迭代名称**: relayToTrae Token 生命周期云主机对齐
- **作者**: claude
- **设计日期**: 2026-07-10 19:10
- **新增文件**:
  - 🆕 `docs/architecture/v14-application-integration-20260710-1910-claude.puml`
  - 🆕 `docs/architecture/v14-application-integration-20260710-1910-claude.archimate`（含 Plateau/Gap/WP 架构变迁视图）
  - 🆕 `docs/architecture/v14-application-integration-20260710-1910-claude.mermaid.md`
- **已有文件（未修改）**: v13 current application-integration
- **变更明细**: 🟡 go_relay / onlineServiceJS；🔴 父进程预换票+SKIP 默认路径

---

## 11. 验收标准

1. relay「启动」日志中出现 `[onlineServiceJS] token-exchange: POST .../exchange-refresh/` 且 **无** `skipped (TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE)`（除非测试显式设置）
2. CRED 中该 task 在换票后 `container_refresh_token` 非空、bootstrap access 已清空（与云主机一致）
3. go_relay status-push 在换票同步完成后使用 **refresh-access 后的** access，不出现因 bootstrap 已失效导致的持续 401
4. 单测：`buildChildEnv` 不含 `TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE=1`；启动路径覆盖「子换票 → 父 sync」
5. Playwright / 既有 relay start 用例更新断言

---

## 12. 风险与回滚

| 风险 | 缓解 |
|------|------|
| 父进程在子换票完成前 status-push → 401 | 换票完成前不 register / 不 push；超时失败 |
| refresh 文件路径与 REPO_ROOT/runtimeDir 不一致 | 契约化路径（与 onlineServiceJS `containerRefreshTokenStorePath` 一致），单测锁定 |
| 旧文档仍写「保留 TRAE_SKIP」 | 同步修订 design-relayToTrae-token-expiry-fix 与 skill |

回滚：恢复父进程 `performTokenExchange` + `TRAE_SKIP=1`（git revert）。

---

## 13. 审批结果

- **python_api_approval**: n/a（无新增 Python 接口）
- **design_approval**: approved（goal `/goal` 自动决策，方案 A+A1）
- **下一步**: 直接实现 + 单测 + v14 架构制品（跳过交互式 worktree 选择）
