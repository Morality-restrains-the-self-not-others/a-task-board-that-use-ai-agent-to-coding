# 平台管理员导航栏公司切换 — 设计文档

- **日期**: 2026-08-11
- **状态**: implemented（/goal 已落地，2026-08-11；待公网精准编译重启验证）
- **迭代**: platform-staff-company-switcher
- **作者**: cursor-grok
- **页面证据**: `https://www.daydaymoney.com/user/bootstrap-admin/profile/` 导航仅「系统管理」纯链接，无公司下拉

---

## 1. 问题陈述

平台管理员（`super_admin` / `employee` / 遗留 `is_superuser`）可被邀请加入一个或多个公司。`/api/accounts/users/me/` 已返回 `companies`，`Navbar.logic` 也已填入 `userCompanies`。但导航栏对平台角色只渲染「系统管理」纯链接，**公司切换下拉被 `v-else-if` 绑在非平台角色分支**，导致管理员看不到其它公司。

## 2. 根因

```
isPlatformStaffUser === true
        ↓
<template v-if="isPlatformStaffUser"> → 仅 <a>系统管理</a>
<template v-else-if="hasTenantContext"> → 工作面板 / 公司下拉  ← 永不进入
```

数据层无缺；纯前端互斥分支设计缺陷。

## 3. 方案决策（采用 A）

| 方案 | 描述 | 结论 |
|------|------|------|
| **A. 并列渲染** | 平台角色始终显示「系统管理」；若有租户上下文，**额外**显示工作面板链接或公司下拉 | ✅ 采用：最小改动、语义清晰、复用既有下拉 |
| B. 「系统管理」变下拉 | 下拉含系统管理 + 各公司 | 拒绝：混淆平台控制台与租户上下文 |
| C. 后端改角色 | 邀请后降级平台角色 | 拒绝：与「管理员也可加入公司」产品意图冲突 |

### 目标行为

| 用户态 | 导航右侧主项 |
|--------|----------------|
| 平台角色 + 无公司 | 「系统管理」 |
| 平台角色 + 单公司 | 「系统管理」+「工作面板」 |
| 平台角色 + 多公司 | 「系统管理」+ 公司切换下拉（现有 `nav-company-switcher`） |
| 普通用户 | 不变（工作面板 / 下拉 / 开始使用） |

约束保持：平台角色**永不**显示「开始使用」。

## 4. 范围

- **改**：`taskFE/app/src/components/Navbar.ui.vue` 模板分支
- **测**：`Navbar.ui.test.js` 增补平台角色+多公司用例；必要时更新既有「超管不渲染工作面板」断言
- **不改**：后端 `/me/`、RBAC、路由守卫、侧栏
- **架构**：无服务边界/协议变更 → **无 ADR / 无 ArchiMate 四件套**

## 5. 成功标准

1. 平台角色 + `userCompanies.length > 1` → 同时存在 `a[href="/system-admin/"]` 与 `[data-testid="nav-company-switcher"]`
2. 平台角色 + 单公司 → 「系统管理」+ `nav-work-panel` 链接
3. 平台角色 + 无公司 → 仅「系统管理」，无「开始使用」
4. 普通多公司用户行为回归不变
5. 相关单元测试全绿

## 6. NFR / 权限 / 领域

- **NFR L2**：纯前端展示，无新增 API；无性能敏感路径
- **权限**：仅展示已返回的 membership；切换仍走既有 `company-switched` → `switchCompany`；不扩大后端授权面
- **角色影响**：`super_admin` / `employee` / 遗留超管 — 导航可见性；租户内权限仍由既有 membership/RBAC 决定
- **DDD 例外**：纯查询/UI 展示，无新业务意图、无 MQ 事件（书面例外）
- **Step 3 worktrees**：SKIP（当前工作区直接改）
- **价值流测试点**：`Navbar.ui.test.js` — 平台角色+多公司并列「系统管理」与 `nav-company-switcher`

## 7. 实施计划（勾选）

- [x] Red：更新/新增 `Navbar.ui.test.js` 平台角色+公司用例
- [x] Green：`Navbar.ui.vue` 将租户导航从 `v-else-if` 改为与平台角色并列的 `v-if="hasTenantContext"`
- [x] 回归：普通用户多公司下拉 + 无公司平台角色不显示「开始使用」
- [x] 验证：vitest 相关用例 + `wc -l` ≤500 + 语法检查
