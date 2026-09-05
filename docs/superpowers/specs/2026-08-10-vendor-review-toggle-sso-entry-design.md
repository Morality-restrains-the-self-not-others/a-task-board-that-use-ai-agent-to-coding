# 容器镜像页：镜像市场 SSO 入口 + 厂商申请审核开关 — 设计文档

- **日期**: 2026-08-10
- **作者**: claude
- **迭代**: vendor-review-toggle-sso-entry
- **状态**: accepted（goal-mode 自动采纳）
- **前置**: vendor-application-entry（2026-08-06）、SystemAdminContainerImages（OPT-046）

## 1. 问题背景

1. 「镜像市场管理（SSO）」链在系统管理侧栏，与「容器镜像列表」割裂，运营难发现。
2. 厂商申请审核流（申请表单 → pending → 运营审核）强制开启；部分环境希望租户绑定邮箱后**直接**进入厂商门户 SSO，无需审核。

## 2. 目标

| # | 目标 | 验收 |
|---|------|------|
| G1 | SSO 入口从侧栏移到 `/system-admin/container-images` 页内 | 侧栏无该链；页面有「打开镜像市场管理（SSO）」 |
| G2 | 页内可开关「厂商申请审核」 | 开关可读写并持久化 |
| G3 | 关闭审核时，已绑邮箱租户可直接打开厂商门户 SSO | ImageMarket 显示 SSO；bridge 自动建档 `is_active=1` |
| G4 | 开启审核时保持现有四态申请流 | 回归 vendor-status / application / bridge 用例 |

## 3. 方案决策（已采纳）

**选定：taskAiProvider 单表设置 + 双读路径（vendor-status 嵌入 + 独立 GET/PATCH）**

| 候选 | 决策 | 理由 |
|------|------|------|
| A. `auth_system_feature_policy` 扩字段 | 拒绝 | 厂商域不属 taskAuth 数据所有权 |
| B. taskCloudService 存设置 | 拒绝 | 容器镜像 CRUD 与厂商审核分属不同限界上下文 |
| C. taskAiProvider `ai_provider_marketplace_settings` | **采纳** | 审核/SSO 桥接均在 ai-provider；单库单服务 |

## 4. 数据模型

`dataMigrate/taskAiProvider/005_marketplace_settings.sql`：

```sql
CREATE TABLE IF NOT EXISTS ai_provider_marketplace_settings (
  id TINYINT NOT NULL PRIMARY KEY,
  vendor_application_review_enabled TINYINT(1) NOT NULL DEFAULT 1,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO ai_provider_marketplace_settings (id, vendor_application_review_enabled)
VALUES (1, 1);
```

- 冷热：单行配置表，年增量 0，无需分区。
- 默认 **开启**审核，兼容现网行为。

## 5. API

### 5.1 `GET /api/ai-provider/marketplace-settings/`（公开读）

```json
{ "vendor_application_review_enabled": true }
```

### 5.2 `GET|PATCH /api/ai-provider/admin-marketplace-settings/`

- **鉴权**：`requireStaff` **或** 网关 `X-User-Roles` 含 `super_admin`/`employee`（`authz.IsPlatformStaff`）
- PATCH body: `{ "vendor_application_review_enabled": false }`
- 响应同 GET；结构化日志 `event=MarketplaceSettingsUpdated`

### 5.3 `GET /api/ai-provider/vendor-status/` 扩展

新增字段（向后兼容）：

```json
{ "vendor_application_review_enabled": true, "is_vendor": false, "status": "none", ... }
```

## 6. 行为变更

### 6.1 `UpsertVendorFromBridge`（审核关闭时）

当 `vendor_application_review_enabled=0`：

1. 已有绑定/邮箱匹配 → 若 `is_active=0` 则置 `is_active=1` 并清空驳回字段后签发 vendor JWT
2. 均无记录 → **自动 INSERT** `is_active=1`（恢复审核关闭下的直达建档）

审核开启时逻辑不变（无档案报错；inactive 区分 pending/rejected）。

### 6.2 ImageMarket（taskFE）

| 条件 | 按钮 |
|------|------|
| `review_enabled=false` 且 `has_email` | 「厂商门户（SSO）」 |
| `review_enabled=false` 且无邮箱 | 引导绑定邮箱（同现申请入口） |
| `review_enabled=true` | 现有四态（申请 / 审核中 / 驳回 / SSO） |

### 6.3 SystemAdminContainerImages + Sidebar

- 页顶卡片：SSO 外链 + 「开启厂商申请审核」开关（读/写 admin-marketplace-settings）
- `SystemAdminSidebar` 删除「镜像市场管理（SSO）」`<a>`

## 7. 事件

本变更为配置读写 + 同步桥接行为，**无新增领域事件**（纯查询/配置；SSO 成功路径沿用既有 exchange，不新增 MQ）。例外书面记录：配置变更不投递事件（低频、无下游订阅者）。

## 8. 测试计划

| 层 | 用例 |
|----|------|
| Go | 设置默认 true；PATCH false；review off 时 bridge 自动建档 active；review on 无档案仍失败；vendor-status 含开关字段 |
| FE unit | ImageMarket：review off + hasEmail → SSO；review on + none → 申请按钮 |
| FE unit | SystemAdminContainerImages 含 SSO 链与开关；Sidebar 无 SSO 链 |

## 9. 架构影响

- application-integration **v70**：taskAiProvider 新增 marketplace-settings 读写与 bridge 条件建档；taskFE SystemAdmin 页承载入口。
- enterprise-landscape：无新服务/节点，不升版。
- ADR：配置归属既有服务内表，无新技术栈选型 → `No-ADR: covered by existing single-service data ownership`。

## 10. 部署

1. 9999「初始化全部数据库」应用 `005_marketplace_settings.sql`
2. 精准编译重启：`taskAiProvider`、`taskFE`
3. 冒烟：系统管理 → 容器镜像列表 → 关闭审核 → 租户镜像市场直达 SSO
