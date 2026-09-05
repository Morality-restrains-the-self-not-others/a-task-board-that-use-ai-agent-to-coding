# 陈旧租户 URL 导致功能参数页「租户不存在」— 设计文档

- **日期**: 2026-08-11
- **状态**: proposed（待总体设计审批）
- **迭代**: stale-tenant-feature-params-404
- **作者**: claude
- **页面证据**: `https://www.daydaymoney.com/tenant/874599492341493760/settings/feature-params/`
  - 元素：`span.text-sm.text-red-600` 可见文本「租户不存在」
  - 选择器指向 `WorkspaceSettingsFeatureParams.vue` 第 5 个白底卡片内的 `saveMessage` 错误展示

---

## 1. 问题陈述

用户打开租户「环境变量设置 / 功能参数」页时，页面仅显示红色文案「租户不存在」，无跳转或创建公司引导，无法继续配置。

## 2. 运行时证据（2026-08-11）

| 检查项 | 结果 |
|--------|------|
| `task_tenant.tenant_company` 行数 | **0** |
| `GET /api/internal/tenant/companies/creator?company_id=874599492341493760` | `{"found":false}` |
| `task_auth.auth_user` | 2 行（含 `bootstrap-admin` + 微信用户 `874694567671132160`） |
| `auth_customtoken` | ≥1（用户可登录） |
| 页面文案来源 | `WorkspaceSettingsFeatureParams.loadForm` → `GET /api/cloud/feature-params/tenant_id/{id}` → **404** `message=租户不存在` → `saveMessage` |
| Loki `\|="租户不存在"` / `\|="874599492341493760"`（24h） | 0 条（本次无用户侧 traceId；公开探测曾得 `c8a0f25e…` 为未登录 401，与本缺陷无关） |

结论：后端判定正确（公司 SSOT 为空 / 该 id 不存在）。缺陷在**前端对陈旧租户 URL 无恢复路径**；与清库后书签/`lastActiveTenantId` 残留同一族问题（参见 `2026-08-11-db-reset-stale-login-ui-design.md`）。

## 3. 架构理解（基线）

根据当前架构设计稿：

- **current**: `enterprise-landscape` **v13**；`application-integration` 积压 target 至 **v70**（vendor-review-toggle 等未交付）
- **业务层**: Identity / Cloud Resource / Tenant 等能力
- **应用层**: `taskFE` → APISIX → `taskCloudService`（feature-params）→ `taskTenantService`（`tenant_company` SSOT）
- **技术层**: MySQL `task_tenant` / `task_cloud`；无新组件需求

本次需求在此基础上做**前端恢复路径接线**，不改变服务边界。

📋 架构版本历史（近期）：
- v13 ✅ current — enterprise-landscape 基线
- v17/v69 🎯 target — 移除本机容器模拟启动
- v70 🎯 target — 厂商申请审核开关 + 镜像市场 SSO

## 4. 根因分析

```
浏览器 URL / localStorage.lastActiveTenantId = 874599492341493760  （清库前残留）
        ↓
进入 WorkspaceSettingsFeatureParams
        ↓
GET /api/cloud/feature-params/tenant_id/874599492341493760
        ↓
taskCloudService.verifyTenantCompanyExistsFP → taskTenantService creator → found=false
        ↓
404 { message: "租户不存在" }
        ↓
❌ 仅 saveMessage = "租户不存在"；未清 lastActiveTenantId；未跳转 /onboarding/ 或其它公司
```

### 4.1 已有但未接线的工具

| 资产 | 状态 |
|------|------|
| `taskFE/app/src/utils/staleTenantRecovery.js` | ✅ 已实现 `clearStaleLastActiveTenantId` / `resolveMissingCompanyRedirectPath` |
| `staleTenantRecovery.test.js` | ✅ 单测覆盖 util |
| **生产页面 import** | ❌ **零引用**（仅测试导入） |
| `useProjectsListLoad` | 部分恢复（有 fallback tenant 才 replace；无公司只展示文案） |
| 路由 `beforeEach` | 有意不做统一租户 guard（`router.js` OPT-20260806-006） |

### 4.2 为何不是后端 bug

`feature_params_public_company.go` 在公司不存在时返回 404「租户不存在」符合 SSOT；不应为陈旧 id 伪造公司或改成 200 空配置。

## 5. 方案

### 5.1 推荐方案（前端接线 + 举一反三）

**A. 扩展 `staleTenantRecovery.js`**

新增异步编排（保持纯函数可测）：

```js
// 伪代码
async function recoverMissingTenantNavigation({ router, tenantId, listCompanies }) {
  clearStaleLastActiveTenantId(tenantId)
  const companies = await listCompanies() // GET /api/tenant/_/accounts/companies/
  const path = resolveMissingCompanyRedirectPath({ companies })
  await router.replace(path) // 无公司 → /onboarding/；有公司 → /tenant/{first}/work-panel/
  return path
}
```

判定「公司缺失」信号（任一）：

| 信号 | 来源 |
|------|------|
| HTTP 404 + `message`/`detail` 含「租户不存在」 | feature-params 等 |
| HTTP 404 from `/accounts/companies/current/` | Sidebar / 公司设置 |
| `found:false` 等价语义 | 内部探测（若前端复用） |

**B. 接线点（必做）**

1. **`WorkspaceSettingsFeatureParams.vue`**（本工单）：`loadForm` 在 404「租户不存在」时调用恢复，不再只写 `saveMessage`；恢复中可短暂展示「租户已失效，正在跳转…」。保存路径同样处理 404。
2. **`WorkspaceFeatureParamsSettings.vue`**（工作区级同构页）：相同 404 恢复。
3. **`Sidebar.vue`**：`companies/current` 非 ok（尤其 404）时 **停止把陈旧 id 写入 `lastActiveTenantId`**，并 `clearStaleLastActiveTenantId`；可选触发同一次 `recoverMissingTenantNavigation`（避免每个子页各自闪红字）。
4. **`TenantCompanySettings.vue`**：`companies/current` 404 → 走同一恢复。
5. **`useProjectsListLoad.js`**：无 fallback 公司时改为 `resolveMissingCompanyRedirectPath`（→ `/onboarding/`），与 util 对齐，消除「请从控制台重新进入」死胡同。

**C. 不引入全局 `beforeEach` 租户校验**

与 OPT-20260806-006 一致：不在路由层对所有 `/tenant/:tenant/*` 预检；以 API 404 为触发点 + Sidebar 入口收敛即可。

### 5.2 拒绝的方案

| 方案 | 原因 |
|------|------|
| 后端对缺失公司返回空默认配置 (200) | 掩盖脏 URL，可能写到错误租户上下文 |
| 按 URL id 自动重建同名公司 | 违背租户创建事件链 / 计费副作用，清库场景应走 onboarding |
| 仅改文案「请手动打开 onboarding」 | 已有 util 未接线；可自动恢复却要求用户手工操作 |
| 全局 beforeEach 每跳查 companies/current | 延迟 + 与现有路由哲学冲突 |

### 5.3 架构变更

**不需要**更新 `docs/architecture/`（无组件/数据流增删，纯前端恢复路径）。  
**No-ADR**: trivial bugfix / UX recovery, no architectural impact。

### 5.4 Python 新增接口

**不触发**（无新增 Python endpoint；复用既有 `GET /api/tenant/_/accounts/companies/`）。

## 6. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 发现 URL 租户不存在并导航到安全落点 | — | — | — | 纯查询失败后的前端导航，无系统事实变更，无 MQ 事件 |
| 用户在 onboarding 新建公司 | `COMPANY_CREATED`（既有） | taskTenantService 创建公司路径 | 既有 company-created 链 | 不在本迭代实现；仅保证落点可达 onboarding |

## 7. 🕸️ Code Review Graph 分析

- **CRG status**: 图存在（Nodes 103 / 更新于 2026-08-11），语言覆盖 JS/TS/Python/bash；MCP `code-review-graph` 未在本会话注册。
- **记录**: `CRG unavailable for symbol impact via MCP; proceeded with ripgrep + live DB + internal tenant API.`
- **手工爆炸半径**:
  - 修改：`staleTenantRecovery.js`（+编排）、`WorkspaceSettingsFeatureParams.vue`、`WorkspaceFeatureParamsSettings.vue`、`Sidebar.vue`、`TenantCompanySettings.vue`、`useProjectsListLoad.js`
  - 测试：扩展 `staleTenantRecovery.test.js`；新增 feature-params 视图级单测（404 → replace `/onboarding/` + 清 storage）
  - 不改：`taskCloudService` 404 语义、`taskTenantService` SSOT

## 8. 价值流影响

- **影响流**: `company-management`（租户上下文有效性）、`user-auth` 之后的首屏落点；feature-params 配置流在租户存在后方可继续
- **新流**: 不需要
- **字段**: 无 schema 变更
- **测试**: taskFE vitest；可选 Playwright：清库后打开陈旧 feature-params URL → 落在 `/onboarding/`
- **完整切片**: 交 `/4-value-stream`（若进入流水线）

## 9. Domain Concept Inventory（轻量）

| 概念 | 说明 |
|------|------|
| Bounded Context | Tenant（SSOT）+ FE Session presentation |
| Key Entities | Company / Tenant、User |
| Candidate Aggregates | Company membership |
| Domain Events | 无新增；新建公司沿用既有 `COMPANY_CREATED` |

## 10. 验收标准

1. 库中无公司、URL 仍为 `/tenant/874599492341493760/settings/feature-params/`：打开后自动进入 `/onboarding/`（或用户其它有效公司 work-panel），**不长期停留红字「租户不存在」**。
2. `localStorage.lastActiveTenantId` 等于该陈旧 id 时被清除。
3. 有效租户 + 成员权限：功能参数页加载/保存行为不变。
4. 单测：404「租户不存在」→ 调用 clear + `router.replace` 目标符合 `resolveMissingCompanyRedirectPath`。
5. 举一反三：Sidebar / 公司设置 / Projects 列表在同类 404 下不再只留死胡同文案。

## 11. 🏛️ 架构变更影响

- **结论**: 不创建 target 架构文件（Bug/UX fix）。
- **关联文档**:
  - `docs/superpowers/specs/2026-08-11-db-reset-stale-login-ui-design.md`（清库后会话 UI）
  - `docs/superpowers/specs/2026-05-31-dev-database-reset-design.md` §16（清库后浏览器残留）— 实施后补充：陈旧 `/tenant/:id/*` 在公司 404 时须自动恢复导航

---

## 审批记录

- `python_api_approval`: not_applicable
- `design_approval`: pending
