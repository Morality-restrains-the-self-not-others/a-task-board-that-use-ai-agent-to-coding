# 实施计划：厂商版本行管理

## Task 1：领域测试（红）

- [ ] `TestContainerImageCanDeleteAndVendorWithdraw`：激活不可删；非激活 approved 可删；pending 不可删；withdraw approved→draft+!active；draft withdraw no-op。

## Task 2：领域实现（绿）

- [ ] `VendorWithdraw`、`DeleteGuard`、`Unpublish` 清 `IsActive`。
- [ ] 事件常量。

## Task 3：持久化 + HTTP

- [ ] `UpdateContainerImageStatus` 写入 `is_active`。
- [ ] withdraw / DELETE handler + SpyEventBus 测例。

## Task 4：前端操作矩阵 + 布局

- [ ] `vendorVersionRowActions.js` + unit test。
- [ ] CSS 5 列 + url 换行；Vue 接线；确认弹窗 + clickGuard。

## Task 5：文档

- [ ] intents、INDEX、skill.md、value-stream.yaml。
