# 测试意图：导航栏当前公司菜单项须能进入工作面板

## 对应用户意图

`navbar_current_company_work_panel_nav.intent.md`

## 测试点

| ID | 场景 | 期望 | 自动化 |
|----|------|------|--------|
| TP-1 | 多公司菜单项 | 均为真实 `a[href]` 指向 `/tenant/{id}/work-panel/` | `Navbar.ui.test.js` |
| TP-2 | 任务详情 + accessCode | 当前公司 href 含 work-panel 与 accessCode | `Navbar.ui.test.js` |
| TP-3 | 点击当前公司 | emit `company-switched`，菜单关闭 | `Navbar.ui.test.js` |
| TP-4 | 任务详情 E2E | 点击「我的公司」进入工作面板 | `Navbar.companySwitcher.playwright.test.js` |

## 价值流锚点

`docs/flows/value-stream-test-integration.wsd` → `PSCS` / `NCC`
