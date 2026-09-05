# 镜像市场「申请成为厂商门户」— 申请表单 + 运营审核流 — 设计文档

> **superseded (2026-08-17)**: 认证申请入口已迁至厂商门户 `provider.*`（[2026-08-17-vendor-apply-on-provider-portal-design.md](./2026-08-17-vendor-apply-on-provider-portal-design.md)），镜像市场不再展示申请/审核中/驳回 UI；qualified 保留「厂商门户（SSO）」。
> 本设计仅作历史记录；后续实现**禁止**再把「申请成为厂商门户」入口加回镜像市场。

- **日期**: 2026-08-06
- **作者**: claude
- **迭代**: vendor-application-entry
- **前置**: OPT-20260806-065（需绑邮箱）/ vendor-status 资格接口（已交付）
- **状态**: superseded（入口迁厂商门户后仅作历史记录）

## 1. 问题背景

未获准用户过去在镜像市场页无任何「申请成为厂商」入口。上一轮审批确认产品机制为**显式申请表单 + 运营审核流**——申请不再等同于自动建档。2026-08-17 起该入口迁至厂商门户 `provider.*` 的「申请认证」卡片，镜像市场不再承载申请 UI。

## 2. 目标流程

```
未获准(无邮箱) → 点击申请 → 引导绑定邮箱（复用既有 /profile/?sso_error=email_required）
未获准(有邮箱) → 点击申请 → 申请表单(公司名/联系人) → 提交 → 待审核(pending)
运营(admin)   → 厂商审核面板 → 通过(is_active=1) / 驳回(review_note)
用户侧        → 审核中(禁用按钮) / 被驳回(提示+可重提) / 获准(SSO 主按钮)
```

## 3. 状态模型（vendor-status 扩展）

`ai_provider_vendor` 行即申请单（复用现有表 + 审核字段），状态由 `is_active` + `review_note` 推导：

| status | 判定 | 用户按钮 |
|--------|------|---------|
| `none` | 无记录 | 「申请认证」（厂商门户未登录卡片，非镜像市场） |
| `pending` | is_active=0 且 review_note 为空 | 「审核中」禁用 |
| `rejected` | is_active=0 且 review_note 非空 | 「申请被驳回：{note}」+ 重新申请 |
| `qualified` | is_active=1 | 「厂商门户（SSO）」主按钮 |

**响应契约**（`GET /api/ai-provider/vendor-status/` 扩展，向后兼容）:

```json
{
  "is_vendor": false,          // 兼容字段 = (status == "qualified")
  "status": "none" | "pending" | "rejected" | "qualified",
  "has_email": true,           // X-User-Email 非合成（前端决定申请入口行为）
  "vendor": null | { "id", "email", "company_name", "contact_name", "is_active", "saas_user_id", "review_note" }
}
```

## 4. 接口设计（全部落 taskAiProvider，Go）

### 4.1 `POST /api/ai-provider/vendor-application/`（convention）/ `/api/vendor/application/`（legacy）— 提交/重提申请

- **身份**: 网关 forward-auth（`X-User-Id` 必填，`X-User-Email` 必填且非合成——绑定邮箱为申请前提，OPT-065）
- **Body**: `{"company_name": "...", "contact_name": "..."}`（均必填，长度 ≤255）
- **逻辑**:
  - 无记录 → `INSERT`（is_active=0, saas_user_id, email, company_name, contact_name, password_hash=随机）→ 201
  - `rejected` 记录 → `UPDATE` 覆盖申请单（清空 review_note/reviewed_*）→ 200
  - `pending` → 409 `{"detail": "申请审核中，请耐心等待"}`（幂等防重复）
  - `qualified` → 409 `{"detail": "已是厂商门户成员"}`
- **安全**: 邮箱取可信头（不可客户端传）；company_name/contact_name 为展示字段（HTML 转义由前端）

### 4.2 `PATCH /api/ai-provider/admin-vendors/{id}/`（convention）/ `/api/admin/vendors/{id}/`（legacy）— 运营审核

- **身份**: `requireStaff`（ai-provider staff JWT，同 handleAdminVendors）
- **Body**: `{"action": "approve" | "reject", "note": "..."}`
  - approve → `is_active=1, review_note=note, reviewed_at=now, reviewed_by=staff_id`
  - reject → `is_active=0, review_note=note, reviewed_at=now, reviewed_by=staff_id`
- **响应**: 200 + 更新后厂商对象；404 不存在；409 已审核过（reviewed_at 非空且 action 与之矛盾）——简化：允许重复审核覆盖（运营二次修正），不做 409
- **补充**: `GET /api/ai-provider/admin-vendors/` 列表已存在（ListVendors），响应增加 `review_note/reviewed_at` 字段

### 4.3 行为变更：ExchangeBridge 移除自动建号（收紧）

`UpsertVendorFromBridge` 不再 INSERT 自动建档：

- saas_user_id 匹配 → 更新 email（email changed 分支保留）→ 返回
- email 匹配未绑定 → 绑定（保留）→ 返回
- 均无 → **返回错误「未找到厂商档案，请先在厂商门户申请认证」**（原自动建档删除）
- `IsActive=false` 拒绝文案细分：pending → 「厂商申请审核中」；rejected → 「厂商申请已被驳回」

**影响**: ① 申请成为唯一建档路径（配合审核流）；② **堵住 OPT-066 后门**（手动构造 bridge URL 不再能自注册）；③ 既有已获准用户不受影响

### 4.4 数据迁移（dataMigrate/taskAiProvider/ 新 sql）

```sql
ALTER TABLE ai_provider_vendor
  ADD COLUMN review_note TEXT NULL,
  ADD COLUMN reviewed_at DATETIME NULL,
  ADD COLUMN reviewed_by BIGINT NULL;
```

## 5. 前端变更

### 5.1 taskFE `ImageMarket.vue` — 四态按钮 + 申请表单

> **superseded (2026-08-17)**: 申请入口已迁厂商门户「申请认证」，`ImageMarket.vue` 已删除申请/审核中/驳回按钮与 30s 轮询，仅保留 qualified 的「厂商门户（SSO）」。下表仅作历史记录。

| 状态 | 按钮 | 点击 |
|------|------|------|
| none + 有邮箱 | 次级按钮「申请认证」（厂商门户） | 弹申请表单 modal（公司名/联系人）→ POST vendor-application → 变「审核中」 |
| none + 无邮箱 | 次级按钮「申请认证」（厂商门户） | toast + 跳 `/profile/?sso_error=email_required`（复用既有引导） |
| pending | 禁用按钮「审核中」 | — |
| rejected | 次级按钮「申请被驳回」+ note 提示 | 弹表单预填旧值 → 重新提交 |
| qualified | 主按钮「厂商门户（SSO）」 | SSO href（现状） |

- `isVendorQualified` 逻辑不变（= is_vendor）；新增 `vendorStatus`/`hasEmail` ref
- 接口失败：默认 none 态（显示申请入口，宽松策略）

### 5.2 taskFE `AdminPortal.vue` — 厂商审核面板

- 新增厂商列表区（复用 `GET /api/ai-provider/admin-vendors/`）：展示 id/email/company/contact/is_active/申请时间/审核状态
- 每行操作：**通过** / **驳回**（prompt note）→ `PATCH admin-vendors/{id}/`
- 未审核（pending）行高亮「待审核」徽标

## 6. 测试计划

### taskAiProvider（TestSSO 命名门禁 + 常规）

| 用例 | 断言 |
|------|------|
| 申请提交成功（无记录） | 201，记录 is_active=0，状态 pending |
| 重复提交（pending） | 409 审核中 |
| 已获准再提交 | 409 已是厂商 |
| rejected 重提 | 200 覆盖申请单，review_note 清空 |
| 无 X-User-Id | 401 |
| X-User-Email 合成 | 400 需先绑定邮箱 |
| 审核 approve | is_active=1，reviewed_* 落库，status→qualified |
| 审核 reject | is_active=0 + note，status→rejected |
| 审核无 staff 权限 | 401/403 |
| vendor-status 四态 | none/pending/rejected/qualified + has_email |
| exchange 无档案 | 拒绝且**未建号**（自动建号移除回归） |
| exchange pending | 拒绝「审核中」 |

### taskFE

- ImageMarket：四态按钮渲染（含 modal 交互：提交成功→审核中态）
- AdminPortal：审核操作（approve/reject 调用断言）

## 7. 决策记录

| # | 决策 | 理由 |
|---|------|------|
| D1 | 申请单复用 `ai_provider_vendor` 表（+3 审核字段），不建新表 | 一行一申请，状态推导简单；避免双表同步 |
| D2 | 移除 ExchangeBridge 自动建号 | 申请成为唯一建档路径；顺带关闭 OPT-066 后门 |
| D3 | 驳回允许重提（覆盖申请单） | 用户可修正资料重新申请，无需运营删记录 |
| D4 | 审核重复操作允许覆盖 | 运营二次修正灵活，无需复杂状态机 |
| D5 | 无邮箱用户点击申请 → 前端直接跳绑定引导 | 复用既有链路，无需后端额外接口 |
| D6 | 事件不发布（见 §9 例外理由） | 无订阅方，状态闭环在 ai-provider 内部 |

## 8. 🕸️ Code Review Graph 分析

`CRG available: 12 files（小图，未覆盖 taskAiProvider/taskFE）`——设计基于直接源码阅读；影响面由各仓全量回归守护（taskAiProvider 含 SSO 门禁测试）。

## 🔍 Trace 日志分析

`skipped_no_traceid` — 无运行时错误路径；状态流转正确性由 §6 测试守护。

## 9. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 用户提交厂商申请 | VendorApplicationSubmitted | — | — | **无订阅方**：审核流状态闭环于 taskAiProvider 内部（vendor-status 直接读表）；taskAiProvider 无事件发布基建（domainevents 仅在 taskAuth），引入总线需新增依赖。审核结果通知用户（站内/邮件）待定订阅方后实施 → 记入 OPT |
| 运营审核通过/驳回 | VendorApplicationReviewed | — | — | 同上 |

## 10. Value Stream 影响

- 受影响 stream：镜像市场厂商资格（ai-provider.marketplace_vendor）
- **字段变更**: `ai_provider_vendor` 新增 `review_note/reviewed_at/reviewed_by`（三字段归属 `ai-provider.ai_provider_vendor.review_*`）；现有 `is_active` 语义扩展（0=待审核/驳回）
- 无新增 stream；测试影响见 §6

## 11. 🏛️ 架构变更影响

**不需要更新架构图**（`docs/architecture/`）：

- application-integration（current v63）为组件级连线，**ai-provider 不在图内**（独立子域服务走 APISIX 直连）；本次为 ai-provider 内部接口/字段变更，不增删组件/跨服务连线
- enterprise-landscape（current v13/v14 target）为组件拓扑，无组件增删
- 新接口（vendor-application / admin-vendors PATCH）已在网关既有路由 `auth_mode: token` 覆盖范围（`/api/ai-provider/*`），无需网关配置变更
- 既有 target 积压（application-integration v64-v66）与本次无关，建议交付时合并处理

## 12. 🐍 Python 新增接口清单与 Go 替代评估

**不触发**：全部接口落 Go（taskAiProvider），无 Python 服务改动。

## 13. 交付物

| 文件 | 变更 |
|------|------|
| `taskAiProvider/src/app.go` | +2 路由（双家族注册） |
| `taskAiProvider/src/vendor_application_handlers.go`（新） | 申请提交 + 审核 handler |
| `taskAiProvider/src/auth_handlers.go` | vendor-status 扩展（status/has_email/review_note） |
| `taskAiProvider/infrastructure/store_auth.go` | UpsertVendorFromBridge 移除自动建号；IsActive 文案细分 |
| `taskAiProvider/infrastructure/store_marketplace.go` | ListVendors 扩展字段 + 申请/审核存储函数 |
| `taskAiProvider/dataMigrate` 迁移 sql（随 dataMigrate 仓） | +3 审核字段 |
| `taskFE ImageMarket.vue` | 四态按钮 + 申请表单 modal |
| `taskFE AdminPortal.vue` | 厂商审核面板 |
| 测试 ×2 仓 | 见 §6 |

## 14. 风险与注意事项

- **行为变更风险**: 移除自动建号影响既有"点击即建档"用户流——但既有已建档用户（is_active=1）不受影响；仅未申请用户无法再自助建档（符合审核流语义）
- **审核通知缺失**: 审核通过后用户需刷新页面看到变化（无主动通知）→ 记入 OPT（后续补通知/刷新提示）
- **部署顺序**: 迁移 sql → taskAiProvider（含行为变更）→ taskFE；taskAuth 零改动
- **存量 pending 数据**: 迁移前若存在 is_active=0 记录（历史手动禁用），审核面板按 pending 展示——运营在面板统一处理即可
