# 工作面板从 Profile 回弹修复 + 禁止链接点击拦截

- **日期**: 2026-08-12
- **状态**: approved（用户确认 A 弹窗确认制 + B 元规则）
- **迭代**: work-panel-no-silent-profile-bounce
- **作者**: cursor

## 问题

在 `https://www.daydaymoney.com/user/:id/profile/` 点击「工作面板」：地址栏短暂变为 `/tenant/.../work-panel/`，随即回到 profile。

线上 WorkPanel chunk 证实：`/me/` 返回 403/404 时 `router.replace(profileHref)` 静默回弹，与来源页形成循环。

## 方案（已批准）

### A. 修回弹

- 删除静默 `replace(profile)`
- `/me/` 403/404 → `modalService.confirm`：「前往个人资料」/「留在本页」
- 确认 → `window.location.href` 整页跳转；取消 → 留在工作面板并降级初始化
- 共享工具：`taskFE/app/src/utils/tenantAccessDeniedPrompt.js`
- 同步修正 `useTenantAccessGuard.js`

### B. 元规则

- `.ai/01_project_constraints/49_no_link_click_interception.md`
- `.cursor/rules/no-link-click-interception.mdc`
- 索引第 44 条
- Navbar「价格 / 工作面板 / 开始使用 / 代码仓库」改为真实 `<a href>`；AccountSwitcher 个人资料同理

## 🕸️ Code Review Graph 分析

`CRG unavailable for structured query`（CLI 子命令与技能示例不一致）。已用静态检索 + 线上 chunk 交叉验证：`router.replace(profile)` 仅 WorkPanel / useTenantAccessGuard / workPanelTenantAccess（未接线）路径。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 例外理由 |
|---------|--------|---------|
| 用户确认离开工作面板 | — | 纯前端导航确认，无服务端状态变更 |
| 用户取消留在本页 | — | 只读/降级 UI，无 MQ 事件 |

## 🏛️ 架构变更影响

- **不更新** `docs/architecture/` target（纯前端行为 + 编码规范，无服务/数据流增删）
- `No-ADR: covered by existing rule` → 新元规则文件即决策载体

## 价值流影响

- 触及 `user-auth` / 工作面板进入路径的前端体验；无 `value-stream.yaml` 字段变更。完整切片交 `/4-value-stream`（可选）。

## 验证

- 单测：`tenantAccessDeniedPrompt.test.js`、`Navbar.ui.test.js`（VIP href）
- Playwright：`WorkPanel.access-denied-confirm-no-bounce.playwright.test.js`
