# 2026-09-01 镜像市场「厂商门户（SSO）」入口消失 — 诊断设计

- **Status:** approved（2026-09-01：restore-apply + approve-move-only）
- **Date:** 2026-09-01
- **Page:** https://www.daydaymoney.com/tenant/882297276515512320/image-market
- **python_api_approval:** not_applicable（无新增 Python HTTP 接口）

## 当前架构理解（口头确认）

根据 `docs/architecture/` **v124 ✅ current**（2026-08-31 clone-run / conf-local）：

- 共有 2 个 current 视图：`enterprise-landscape`、`application-integration`（本迭代主题是部署机密面，**不是**厂商 SSO）。
- 业务层：平台运维 clone + overlay + `up.sh`。
- 应用层：runAll、confload、taskGateway、taskAuth、其余 Go / taskFE。
- 技术层：`DEPLOY_ROOT`、daydaymoney-deploy、`conf-local/`（含网关 TLS 与 OIDC 签名钥）、GitHub Release。
- 上次更新：v124（2026-08-31 18:55）。

厂商门户 SSO 路径（未在 v124 变更，沿用既有应用集成）：

```
taskFE ImageMarket
  → GET /api/ai-provider/vendor-status/   （APISIX ai-provider-auth + taskAuth forward-auth）
  → 若渲染 SSO：GET /api/accounts/sso/ai-provider/vendor/  （taskAuth sso_bridge → provider 换票）
```

本次需求在 v124 上诊断根因，并已批准把申请入口从门户收回镜像市场（架构 **v125 🎯 target**）。

📋 架构版本历史：v122–v124 为二进制部署 / conf-local / clone-run；v125 为 ImageMarket 恢复厂商申请入口。

## 🔍 Trace 日志分析

用户未粘贴 `data-traceId`。用现网 Referer 反查 Loki（`http://10.2.150.68:3100`，过去 6h）。

- **Grafana Trace Dashboard:** [打开](http://10.2.150.68:3000/d/distributed-trace-view?var-trace_id=12927e80-9f00-4dc5-87d7-7ce3d02342a8&var-tempo_trace_id=12927e80-9f00-4dc5-87d7-7ce3d02342a8)
- **Grafana 日志搜索:** [打开](http://10.2.150.68:3000/explore?orgId=1&left={"datasource":"loki","queries":[{"refId":"A","expr":"{job=~\".+\"} |= \"12927e80-9f00-4dc5-87d7-7ce3d02342a8\"","queryType":"range"}]})
- **时间范围:** 2026-09-01 13:51:39 +08
- **涉及服务:** task-gateway、task-auth、ai-provider

### 日志摘要

| 时刻 | 服务 | 事件 |
|------|------|------|
| 13:51:39.190 | task-auth | `GET /api/internal/gateway/forward-auth/` **200**（0ms） |
| 13:51:39.197 | ai-provider | `GET /api/ai-provider/vendor-status/` **200**（6ms） |
| 13:51:39 | task-gateway | 同源 Referer `…/tenant/882297276515512320/image-market`，upstream `:8010` 200，**body 109 bytes** |

网关注入（脱敏）：

- `X-User-Id`: `882297272207962112`
- `X-User-Email`: `sso-882297272207962112@sso.invalid`（微信/无邮箱登录的合成邮箱）
- `X-Gateway-Auth-Verified`: `1`
- 网关密钥头**非空**（与同日项目列表空密钥 401 **不是同一故障**）

Go `encoding/json` 对 map 按键名字母序序列化后，109 字节**精确等于**：

```json
{"has_email":false,"is_vendor":false,"status":"none","vendor":null,"vendor_application_review_enabled":false}
```

### 关键发现

- vendor-status **没有失败**；SSO 按钮消失是前端按契约**故意不渲染**。
- `has_email=false`：合成邮箱 `@sso.invalid` 不算可绑定邮箱（`applicantHasBindableEmail`）。
- `status=none`：该 saas 用户无 `ai_provider_vendor` 档案。
- `vendor_application_review_enabled=false`：运营已关闭厂商申请审核。

### 根因假设（已用运行时数据验证）

**不是回归删按钮、也不是 APISIX 空密钥。** 当前会话是微信类合成邮箱账号；即使审核已关，ImageMarket 仍要求「有真实邮箱」才画「厂商门户（SSO）」。点 SSO 也会被 taskAuth `sso_bridge` 拒绝签发 `vendor_bridge` 并跳转个人资料绑邮箱。

按源码，右上角应出现替代链「绑定邮箱后进入厂商门户」（`/profile/?sso_error=email_required#rg=profile.email_binding`），而不是 SSO。

## 🕸️ Code Review Graph 分析

- 根图 `.code-review-graph/graph.db` 存在（2026-08-31，18 files / 114 nodes）。
- `code-review-graph search "showVendorSsoButton"` → 0 nodes。
- **CRG unavailable for this symbol:** 索引面未覆盖 `taskFE/app/src/views/ImageMarket.vue` 与 `taskAiProvider` Go 符号。诊断以 Loki + 源码为准。

## 问题分析（产品契约）

`ImageMarket.vue`：

```js
showVendorSsoButton =
  vendorStatus === 'qualified'
  || (!vendorApplicationReviewEnabled && hasEmail)
```

| 条件 | 本账号实测 | 结果 |
|------|------------|------|
| qualified | `none` | 否 |
| 审核关 + 有邮箱 | 审核关 + `has_email=false` | 否 |
| 审核关 + 无邮箱 | 是 | 渲染「绑定邮箱后进入厂商门户」 |

2026-08-17 设计已把「申请认证」迁到 `provider.*`；镜像市场**不再**给未获准用户画 SSO/申请按钮。未绑邮箱时连 SSO 也不画，避免点进去立刻被 bridge 打回。

同日项目列表 401（`7af49e7f-…`，APISIX 空 `gatewayInternalSecret`）**未**打到本路径：vendor-status 已 200 且密钥头有值。

## 用户期望对照（2026-09-01 补充）

期望：**提交申请后，镜像市场应出现 SSO 链接。**

现网状态机与期望不一致：

| 步骤 | 用户以为 | 现网 |
|------|----------|------|
| 在哪申请 | 镜像市场 | 已迁到 [provider.daydaymoney.com](https://provider.daydaymoney.com/)（2026-08-17） |
| 提交后立刻 SSO | pending 即出现 | **只有 `qualified`（审核通过）或「审核关闭 + 真实邮箱」才画 SSO**；`pending` 不画 |
| 本账号 | 已走完申请 | Loki：`status=none`、无 vendor 行；合成邮箱会被 `requireVendorApplicant` **400 拒绝申请** |

因此不是「申请通过了但按钮丢了」，而是 **申请没有落库**（缺真实邮箱），镜像市场右上按契约为空（或只剩绑邮箱链）。

当前运营已关审核。对已绑邮箱用户，**不必申请**，刷新即可出现 SSO。本账号卡在第 0 步：微信合成邮箱。

**禁止**：让 `pending` 未审核账号直接 SSO（2026-08-07 已关的自动建号后门）。

## 方案（已否决备选；最终见「选定方向」）

| # | 方案 | 改动 | 架构 |
|---|------|------|------|
| A | **先绑邮箱**（与关审核现状对齐） | 无后端。用户绑定真实邮箱后刷新；审核已关 → 应出现 SSO，无需再申请 | 不更新 |
| B | **镜像市场漏斗 CTA**（推荐） | none：主按钮「绑定邮箱」或「去厂商门户申请」；pending：「审核中」；qualified / 关审核+邮箱：SSO。申请仍在 provider，不把表单搬回镜像市场 | 不更新 |
| C | **pending 也画 SSO** | 拒绝。等于未获准即可进门户 | 不更新 |
| D | **无邮箱仍画 SSO** | 点击由 `sso_bridge` 重定向绑邮箱 | 不更新 |

## 选定方向（已批准）

**申请只放镜像市场，从厂商门户撤掉申请面板**（AskQuestion `restore-apply` + `approve-move-only`，2026-09-01）。

- **Logic-Rollback-OK:** restore ImageMarket vendor apply form; remove provider ApplyPanel; SSO still only after qualified
- **SSO 时机不变:** 提交 → `pending`（审核中、无 SSO）→ 运营通过 → `qualified` 才出现「厂商门户（SSO）」。关审核 + 真实邮箱可直达 SSO。
- **邮箱硬墙不变:** 合成邮箱不能提交、不能画 SSO。现网该账号须先绑定真实邮箱；当前审核已关，绑完即可 SSO、不必再填申请。
- **代价:** 未加入任何租户的账号无法申请。

### 镜像市场四态（恢复）

| vendor-status | 审核开 | 审核关 |
|---------------|--------|--------|
| 无邮箱 | 「绑定邮箱后进入厂商门户」 | 同左 |
| none + 有邮箱 | 申请表单（公司/联系人/双证/短信） | 「厂商门户（SSO）」直达 |
| pending | 「审核中」disabled，无 SSO | （关审核不应长期 pending） |
| rejected | 「申请被驳回，重新申请」+ 表单 | 同左或 SSO（关审核 + 邮箱） |
| qualified | 「厂商门户（SSO）」 | SSO |

现网该账号：无邮箱 + none + 审核关 → 即使恢复表单，**仍先看到绑邮箱**；绑完后因审核已关会直接出 SSO，不必再填申请。

### 接口落点

复用既有 Go：`GET /api/ai-provider/vendor-status/`、`POST /api/ai-provider/vendor-application/`、证照 upload-url、用户级短信。**不新增 Python 接口。** 前端从 `taskAiProvider/frontend` 的申请契约对齐（字段/错误/`data-traceId`），抽共享或在 taskFE 重做薄表单，禁止复制过期 Django 路径。

### 反模式

- 不从 git 拷回已删的 `VendorApplicationForm` 旧文件与门户新实现并存两套校验。
- 不恢复 30s vendor-status 轮询（元规则 51）。提交成功后本页重载状态即可。

## 🏛️ 架构变更影响

- **迭代版本:** v125 🎯 target
- **迭代名称:** ImageMarket 恢复厂商申请入口
- **作者:** cursor
- **设计日期:** 2026-09-01 14:05
- **新增文件**（每个视图四类伴生格式）:
  - 🆕 `docs/architecture/v125-application-integration-20260901-1405-cursor.puml`
  - 🆕 `docs/architecture/v125-enterprise-landscape-20260901-1405-cursor.puml`
  - 🆕 `docs/architecture/v125-application-integration-20260901-1405-cursor.diff.archimate`（增量变迁）
  - 🆕 `docs/architecture/v125-enterprise-landscape-20260901-1405-cursor.diff.archimate`
  - 🆕 `docs/architecture/v125-application-integration-20260901-1405-cursor.full.archimate`（全量拓扑）
  - 🆕 `docs/architecture/v125-enterprise-landscape-20260901-1405-cursor.full.archimate`
  - 🆕 伴生 `.mermaid.md`（每个视图）
- **已有文件（未修改）:** `docs/architecture/v124-*-*.puml` (current)
- **变更明细:** 🟢 镜像市场申请过程 / 🟡 ImageMarket 与 taskAiProvider 入口 / 🔴 门户申请面板

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | Plateau v124 → Gap「申请在门户」→ WP → Plateau v125；镜像市场 → 网关 → ai-provider；门户 ApplyPanel 废弃连线 |
| **`.full.archimate`** | 叠加 v124 全量 + 本迭代申请/SSO 拓扑 |

> 老文件未被修改。目标架构将在 `/10-ship` 执行时切换为 current。

## 验收（选定 restore-apply）

1. 镜像市场 none+有邮箱+审核开：可见申请表单，提交后变 pending，**仍无 SSO**。
2. 运营通过后 qualified：出现「厂商门户（SSO）」。
3. 合成邮箱：无表单无 SSO，有绑邮箱链；绑完 + 审核关 → SSO。
4. `provider.*` **不再**展示申请认证面板。
5. `ImageMarket.vendorStatus.test.js` 覆盖申请/审核中/SSO 四态；禁止 pending 出现 SSO href。
6. vendor-status / 申请失败节点带 `data-traceId`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 打开镜像市场看厂商门户入口 | — | — | — | 纯查询 vendor-status，无领域事实变更 |
| 提交/重提厂商申请 | VendorApplicationSubmitted（既有，无新 MQ 订阅） | taskAiProvider vendor-application | AdminPortal 同步审核 | 与既有 vendor-application-kyc-docs 一致：无 MQ 消费者 |
| 绑定邮箱后进入厂商门户 | — | — | — | 邮箱绑定走既有个人资料写路径；本迭代不新增 |

## Domain Concept Inventory

- **Bounded Contexts:** 厂商门户（taskAiProvider）、主站身份（taskAuth）、租户控制台壳（taskFE）
- **Key Entities:** SaasUser（含合成邮箱）、Vendor、MarketplaceSettings
- **Candidate Aggregates:** Vendor（按 saas_user_id / email）、单行 MarketplaceSettings
- **Domain Events:** 无新增

## 价值流影响

- 意图改为：申请入口回到镜像市场；门户撤申请面板。见 `docs/intents/frontend/provider/vendor_apply_on_provider_portal.intent.md`。
- 见 `docs/superpowers/plans/2026-09-01-image-market-vendor-apply-value-stream.md` 与 `conf/value-stream.yaml` 流 `image-market-vendor-apply`。

## 权限影响分析

见 `.claude/skills/2-role-permission/permission-analysis-2026-09-01.md`。申请是主站账号级；不新增租户 region。合成邮箱 400；SSO 仅 qualified / 审核关+真实邮箱。

## 安全审查结论

绿灯。无新 API；IDOR 由网关 `X-User-Id` 约束。
