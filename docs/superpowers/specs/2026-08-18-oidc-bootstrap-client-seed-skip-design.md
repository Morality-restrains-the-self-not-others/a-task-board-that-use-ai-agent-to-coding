# GitLab SH-1 SSO：OIDC bootstrap client 被 data_migrate_log 一次性跳过

- **日期**: 2026-08-18
- **作者**: cursor
- **迭代**: oidc-bootstrap-client-seed-skip
- **状态**: accepted（2026-08-18 闸门已落地：`goMigrateStep.Always` + `010_oidc_bootstrap_clients` 每次 migrate 必跑）
- **关联**: [v85 可插拔多区域 gitService](./2026-08-18-pluggable-multi-region-gitservice-design.md) 交付缺口；ADR-0002、ADR-0014
- **意图**: `docs/intents/backend/pluggable_multi_region_gitservice.intent.md`（补 OIDC seed 验收）
- **python_api_approval**: n/a（零新增 Python 接口；落点 Go `taskAuth` migrate CLI）
- **架构**: **不升级版本** — 无新组件/数据流；v85 已声明「taskAuth 多 GitLab OIDC client」，本次修 seed 闸门

---

## 🔍 Trace 日志分析 (traceId: `661ed44b12cea27653bfa34773e50087`；先前 `7bc6fededff5e2b9777e10343d6d018d`)

- **Grafana Trace Dashboard**: [打开](http://10.2.150.68:3000/d/distributed-trace-view?var-trace_id=661ed44b12cea27653bfa34773e50087&var-tempo_trace_id=661ed44b12cea27653bfa34773e50087)
- **Grafana 日志搜索**: [打开](http://10.2.150.68:3000/explore?orgId=1&left={"datasource":"loki","queries":[{"refId":"A","expr":"{job=~\".+\"} |= \"661ed44b12cea27653bfa34773e50087\"","queryType":"range"}]})
- **时间范围**: 2026-08-18 21:16:04.881 +08 → 同毫秒级结束（`duration_ms: 0`）
- **涉及服务**: task-auth（HTTP :8003，经 APISIX `api.daydaymoney.com`）

### Loki / Tempo 查询尝试

| 步骤 | 查询条件 | 时间范围 | 结果 |
|------|---------|---------|------|
| 主查询 | `{job=~".+"} \| json \| trace_id` | 1h | **1 条** `task-auth` `GET /api/oidc/authorize` status=400 |
| 回退 1 | `{job=~".+"} \|= "<ID>"` | 24h | 同上 1 条 |
| D2 Loki 健康 | `/ready` + `loki_ingester_memory_chunks` | 即时 | Loki ready，chunks=223 |

先前窗口（trace `7bc6fed…`）Loki 空库；本窗口采集已恢复。文件 sink 与 Loki 一致。

### 日志摘要

```
GET /api/oidc/authorize  status=400  duration_ms=0
service=task-auth
trace_id=661ed44b12cea27653bfa34773e50087
ts=2026-08-18T21:16:04.881+08:00
```

同路径先前 400：`20:14:12` `trace_id=7bc6fededff5e2b9777e10343d6d018d`；`20:13:31` `af5d6874723012b4ad301c035d8b675d`。

浏览器错误体与 `handleOidcAuthorize` 在 `loadOidcClient` 失败时写出的 JSON 一致：

`{"error":"unauthorized_client","error_description":"client not found","trace_id":"..."}`

### 关键发现

- 请求已到达 taskAuth OIDC authorize（GitLab OmniAuth `client_id` / `redirect_uri` 拼装正确）。
- 1ms 即 400 → 未进入登录页/发码，失败点是 **client 查找**。
- authorize 在 `redirect_uri` 校验前对「client 不存在」返回 JSON（不 302 到 GitLab），故页面直接显示 JSON。

### 根因假设（已被 DB 证实）

`auth_oidc_client` **没有** `gitlab-git-service-tencent-sh-1` 行。conf 有、GitLab 有、DB 无。

---

## 1. 对当前架构的理解

根据 `docs/architecture/` **v85 ✅ current**：

- **视图**: `enterprise-landscape`、`application-integration`
- **业务层**: Identity & Access；Git 托管按区域商品化
- **应用层**: taskAuth（OIDC Provider）、taskBill（区域开通）、gitService 现网 + SH-1、taskFE
- **技术层**: GitLab CE 现网 `:8012` / `gitlab.${baseDomain}`；GitLab CE SH-1 `:8014` / `gitlab-tencent-sh-1.${baseDomain}`；边缘 nginx；OIDC issuer = `${subdomains.gateway}`（`api.daydaymoney.com`）

📋 架构版本历史（节选）：

- v85 (2026-08-18 15:50) ✅ current — 可插拔多区域 gitService（含「taskAuth 多 GitLab OIDC client」）
- v84 archived — 评论运行态推送
- v83/v82/v78 🎯 正交 target 积压

本次需求是 **v85 交付缺口的缺陷分析与 seed 闸门修复**，不改拓扑。

---

## 2. 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | Nodes 108 / Edges 937 / Files 17；语言 js/ts/python/bash；branch `main`；last_updated 2026-08-18T19:30:41 |
| 关键发现 | 根图未索引 Go `taskAuth` 符号 |
| 决策影响 | 爆炸半径以 Grep 为准 |
| skip 理由 | `CRG unavailable for Go impact` — 索引面偏前端/脚本 |

| 符号/路径 | 爆炸半径 | 设计动作 |
|-----------|----------|----------|
| `loadOidcClient` / `handleOidcAuthorize` | `taskAuth/src/oidc_handlers.go`、`oidc_db.go` | 行为保持；失败语义已正确 |
| `seedOidcBootstrapClients` / `ensureOidcClient` | `oidc_bootstrap.go`、`oidc_db.go` | **保留幂等 INSERT**；改调用闸门 |
| `RunGoDataMigrate` skip-by-`step_key` | `data_migrate_go.go`；`db/task-auth/migrate.sh` | **conf 驱动 seed 每次 migrate 必跑** |
| `main` HTTP 路径 | `taskAuth/src/main.go` | **禁止**启动时 `RunGoDataMigrate`（ADR-0002 已满足，保持） |
| `bootstrapClients` YAML | `conf/auth/task-auth/config.yaml` | SSOT 已含第二 client，无需再改键名 |

同类搜索（Go seed + `data_migrate_log` 跳过）：仅 `taskAuth` 的 `GoDataMigrateSteps` 把 **conf 驱动、可追加的 OIDC client** 做成一次性 key。其它服务的 skip 针对 SQL 文件名，模式不同，不并案改 DDL 闸门。

---

## 3. 问题陈述

复现路径：

1. 打开 `https://gitlab-tencent-sh-1.daydaymoney.com/users/sign_in`
2. 点击 taskAuth SSO
3. 跳转 `https://api.daydaymoney.com/api/oidc/authorize?client_id=gitlab-git-service-tencent-sh-1&redirect_uri=https://gitlab-tencent-sh-1.daydaymoney.com/users/auth/openid_connect/callback&...`
4. 页面 JSON：`unauthorized_client` / `client not found`

**已接通的半截**：SH-1 GitLab OmniAuth 使用独立 `GITLAB_OIDC_CLIENT_ID=gitlab-git-service-tencent-sh-1`；taskAuth conf `bootstrapClients` 已追加该 client 与公网 callback。

**未接通的半截**：`auth_oidc_client` 在 03:38:56 由 `010_oidc_bootstrap_clients` **只 seed 了当时的 3 个 client**。15:01 写入 YAML 的第四个 client **从未 INSERT**。

### 运行时证据（2026-08-18）

| 源 | 事实 |
|----|------|
| `auth_oidc_client` | 仅 `gitlab-git-service`、`ai-provider`、`chrome-extension` |
| `data_migrate_log` | `010_oidc_bootstrap_clients` applied_at=`2026-08-18 03:38:56` |
| conf mtime | `conf/auth/task-auth/config.yaml` 15:01 已含 `gitlab-git-service-tencent-sh-1` |
| taskAuth 进程 | 17:35 启动（**晚于** conf），HTTP 路径不跑 Go seed |
| `RunGoDataMigrate` | `SELECT 1 FROM data_migrate_log WHERE step_key=?` 命中则 **continue，不调用** `seedOidcBootstrapClients` |
| 即使再点 9999 | `db/task-auth/migrate.sh` → `go run ./src migrate` **仍会 skip** 该 step |

因此：不是「忘了重启 taskAuth」，也不是 redirect_uri 白名单错误（那种会返回 `redirect_uri not allowed`）。是 **把可追加的 conf seed 当成一次性 DDL step**。

`ensureOidcClient` 本身已是幂等 INSERT-if-missing，并对 `managed_by=bootstrap` 自愈 `redirect_uris`、对 `admin` 不覆盖（OPT-20260808-025）。闸门跳过导致它根本没机会跑。

---

## 4. 选定方案

### 4.1 决策锁定

| # | 决策 | 选择 |
|---|------|------|
| D1 | HTTP 启动是否 seed OIDC | **否**（ADR-0002 / 元规则 40） |
| D2 | migrate 时是否因 `010_oidc_bootstrap_clients` 已存在而 skip | **否** — 该步每次 `taskAuth migrate` / 9999 必跑 |
| D3 | `ensureOidcClient` 语义 | **不变**（缺则 INSERT；bootstrap 自愈 URI；admin 不覆盖） |
| D4 | 架构图 | **不新增 v86** |
| D5 | 运维立刻通 SSO | 代码合入后跑一次 migrate；**不**在业务进程内热插入 |

### 4.2 代码

`goMigrateStep` 增加 `Always bool`（或等价：seed 函数不参与 skip）。`010_oidc_bootstrap_clients` 设 `Always: true`：

- 每次 `RunGoDataMigrate` 调用 `seedOidcBootstrapClients`
- 仍可 `INSERT`/`UPDATE` log 行并写入 `checksum`（建议为 bootstrap `client_id` 排序后的哈希），便于审计「上次 seed 的 conf 集合」
- SQL 类 step 继续 skip-if-logged

`db/task-auth/migrate.sh` 注释改为：Go OIDC seed **每次执行、表级幂等**，不以 step_key 跳过。

### 4.3 验收后操作

```bash
# 9999「初始化全部数据库」或：
cd taskAuth && go run ./src migrate
```

预期：`auth_oidc_client` 出现 `gitlab-git-service-tencent-sh-1`，`redirect_uris`=`["https://gitlab-tencent-sh-1.daydaymoney.com/users/auth/openid_connect/callback"]`。然后再点 GitLab SH-1 SSO，应进入 taskAuth 登录页（或已登录则 302 回 GitLab callback）。

### 4.4 拒绝的方案

| 方案 | 拒绝原因 |
|------|----------|
| HTTP `main` 调 `seedOidcBootstrapClients` | 违反 ADR-0002 |
| 删 `010_oidc_bootstrap_clients` 日志行当一次性 workaround | 下一轮加 client 会再踩；且无回归测试 |
| 只手工 INSERT 一行、不改闸门 | SSO 能通，v85「追加 conf 即多 client」仍然假 |
| 新 SQL 文件写死 client | 密钥与 redirect 的 SSOT 是 YAML，不是 SQL |

---

## 5. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|----------|----------------|--------|--------------|---------|
| migrate 同步 conf bootstrap OIDC clients | — | `taskAuth migrate` / 9999 `migrate.sh` | `auth_oidc_client` INSERT/自愈 | 运维配置同步，无跨聚合副作用；**无对应 MQ 事件** |
| 用户走 GitLab SH-1 SSO authorize | （既有 OIDC 授权码流，非新事件） | `handleOidcAuthorize` | 登录页 / 发码 | 本次不改授权码语义 |

---

## 6. Domain Concept Inventory

| 概念 | 说明 |
|------|------|
| Bounded Context | Identity（taskAuth OIDC Provider） |
| Key Entity | `OidcClient`（`auth_oidc_client`，identity=`client_id`） |
| Candidate Aggregate | OidcClient（redirect 白名单、secret hash、`managed_by`） |
| Domain Events | 无新增；seed 非领域事件 |
| 约束 | `managed_by=admin` 不被 conf 覆盖；`bootstrap` 行 conf 自愈 |

---

## 7. 价值流影响

现有 `conf/value-stream.yaml`：

- `gitlab-oidc-redirect-uri-public-ssot` — 管 redirect 白名单，**不覆盖**「新 client_id 未入库」
- `oidc-playwright-e2e-login` — 现网 `gitlab-git-service` 实例；SH-1 未纳入

建议 Step 4 增量（本设计只标识，不改 YAML）：

| stream / step | 变更 |
|---------------|------|
| 新 step `oidc-bootstrap-clients-conf-resync` | `taskAuth/src` 单测：log 已有 010 时追加 cfg client，`RunGoDataMigrate` 仍 INSERT |
| `oidc-playwright-e2e-login`（可选） | SH-1 入口冒烟，或与现网分 step |
| fields | `task-auth.auth_oidc_client.client_id`（三段名；描述：bootstrapClients 同步后必须存在） |

---

## 8. 测试计划（TDD）

红：`data_migrate_log` 已有 `010_oidc_bootstrap_clients` + cfg 含未入库 `client_id` → 今日 `RunGoDataMigrate` 后 `loadOidcClient` 仍 nil。

绿：同上路径 INSERT 成功；已有三 client 不变；`managed_by=admin` 行 redirect 不被覆盖（复用 `oidc_managed_by_test.go` 语义）。

另：`python3 dataMigrate/check_no_startup_migrate.py` 仍绿（HTTP `main` 不调 `RunGoDataMigrate`）。

---

## 9. 🏛️ 架构变更影响

- **不新建** v86 四类伴生文件
- current 仍为 v85
- VERSION_HISTORY 不追加拓扑变更；本缺口记在本规格 + 意图变更记录

---

## 10. 观测性附录

本窗口 Loki 已可检索 `661ed44b12cea27653bfa34773e50087`（job=`task-auth`）。先前 `7bc6fed…` 窗口 chunks=0 已恢复。SSO 修复不依赖采集。剩余：`handleOidcAuthorize` client-not-found 路径未打 `client_id` 结构化 WARN（`oidc_handlers.go` 已超 500 行，拆分后补）。
