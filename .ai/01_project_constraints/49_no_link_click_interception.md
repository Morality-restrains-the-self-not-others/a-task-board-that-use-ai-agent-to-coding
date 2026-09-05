# 禁止链接点击拦截（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-12
- 维护者：Trae AI 团队
- 约束索引：`00_project_constraints.md` 第 44 条
- Cursor：`.cursor/rules/no-link-click-interception.mdc`（alwaysApply）

## 背景与动机

用 `@click.prevent` + `router.push`、或 `router-link` 默认左键拦截，会破坏：

- 中键 / Ctrl+点击新标签打开
- 复制链接地址、拖拽链接
- 浏览器前进后退与地址栏可预期性

更严重的是：SPA 半截导航 + 页面内再 `router.replace`（如工作面板 `/me/` 403 静默回 profile）会形成**地址栏闪一下又弹回**的循环跳转，用户无法稳定到达目标页。

本规则要求：**可导航地址一律使用真实 `<a href="...">`（或整页 `location.href`），禁止为「站内跳转」拦截默认点击。**

## 核心规则

### 1. 导航链接必须是真实 href

- 凡用户可理解为「打开某个地址」的控件，**必须**渲染为带真实 `href` 的 `<a>`（或等价可复制的绝对/相对路径）。
- **禁止**仅用 `<button @click="router.push(...)">` / `<div @click>` 冒充链接做路由跳转。
- **禁止**对导航用途的 `<a>` 使用 `@click.prevent` 后再手写 `router.push` / `router.replace`（除非下方例外）。

### 2. 优先原生导航，慎用 router-link

- `router-link` 默认拦截同域左键点击。跨布局壳层、跨认证边界、或易与守卫/回弹逻辑打架的入口（如 Navbar「工作面板」「价格」「个人资料」），**优先**原生 `<a :href="...">`（可触发整页加载或由浏览器默认处理）。
- 同页细粒度 SPA 切换（页内 Tab、已挂载同壳子路由）若继续用 `router-link`，不得再叠加 `preventDefault` 二次拦截。

### 3. 程序化导航不得静默「踢回」来源页

- 页面加载后若鉴权/租户校验失败，**禁止**无确认地 `router.replace` 回用户刚离开的来源页（典型：profile → work-panel → 自动回 profile）。
- 须弹窗（`modalService.confirm`，禁止浏览器原生 `confirm`）让用户选择「前往某页 / 留在本页」；确认后可用 `window.location.href` 整页跳转。

## 例外（须在代码旁注释理由）

| 例外 | 说明 |
|------|------|
| 下拉展开 / 关闭 | 触发器是 button，不是导航链接 |
| 表单提交 / 危险操作确认 | 非「打开 URL」语义 |
| 外链 + 业务门禁改写目标 | 可改 `href` 指向门禁落地页（如非 VIP → `/pricing/`），**仍禁止** `preventDefault` 后再 push |
| 第三方嵌入控件 | 无法控制 DOM 时在调用处注明 |

## 验收

```bash
# Navbar 工作面板应为真实 a[href*="work-panel"]，而非仅依赖 router-link 行为
rg -n 'data-testid="nav-work-panel"' taskFE/app/src/components/Navbar.ui.vue
# 工作面板禁止静默 replace(profile)
rg -n 'router\.replace\(.*profile' taskFE/app/src/views/WorkPanel.vue && echo FAIL || echo PASS
```

## 与前端规范关系

与 [`.ai/04_frontend_development/00_frontend_development.md`](../04_frontend_development/00_frontend_development.md)「弹窗须用 modalService」互补：本条管**链接点击与导航回弹**；弹窗规范管**确认 UI 实现**。
