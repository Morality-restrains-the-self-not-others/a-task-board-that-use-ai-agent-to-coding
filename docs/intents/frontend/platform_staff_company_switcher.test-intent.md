# 测试意图：平台管理员导航栏公司切换

## 对应用户意图

`platform_staff_company_switcher.intent.md`

## 测试点

| ID | 场景 | 期望 | 自动化 |
|----|------|------|--------|
| TP-1 | 平台角色 + 无公司 | 仅「系统管理」，无工作面板/开始使用 | `Navbar.ui.test.js` |
| TP-2 | 平台角色 + 单公司 | 「系统管理」+「工作面板」 | `Navbar.ui.test.js` |
| TP-3 | 平台角色 + 多公司 | 「系统管理」+ `nav-company-switcher`，菜单列全公司 | `Navbar.ui.test.js` |
| TP-4 | 普通用户多公司 | 仅公司下拉，无「系统管理」 | 既有 `Navbar.ui.test.js` 回归 |
| TP-5 | 平台角色永不「开始使用」 | 无公司仍无 onboarding 链接 | 既有 + TP-1 |

## 价值流锚点

`docs/flows/value-stream-test-integration.wsd` → `PSCS` / `PSD`
