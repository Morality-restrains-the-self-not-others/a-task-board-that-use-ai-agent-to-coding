# 实施计划：导航栏多账号切换

**日期**: 2026-07-14  
**设计**: `docs/superpowers/specs/2026-07-14-navbar-multi-account-switcher-design.md`

## Tasks

### Task 1: savedAccounts store（TDD）

- [ ] Red: `saved_accounts_store.test.js` — upsert / max5 / remove / getActive
- [ ] Green: `domain/auth/services/saved_accounts_store.js`
- [ ] 日志：无（纯本地）；禁止 console 打印 token

### Task 2: activate-session Go API（TDD）

- [ ] Red: `auth_activate_session_test.go`
- [ ] Green: `handleActivateSession` + 路由注册
- [ ] OpenAPI schema 同步
- [ ] 日志：userId + status（脱敏）

### Task 3: activate_session_service.js

- [ ] 调 API、写 cookie/token、upsert 槽
- [ ] Vitest mock fetch

### Task 4: Navbar AccountSwitcher UI

- [ ] 下拉组件（单击开/双击关）
- [ ] 接入 Navbar.logic / Navbar.ui
- [ ] 切换成功刷新；失败提示

### Task 5: Login upsert + add_account

- [ ] 登录成功 upsert
- [ ] `?add_account=1` 保留槽
- [ ] 退出当前：logout + 切下一槽或登录页

### Task 6: 文档与价值流同步

- [ ] 更新 `conf/value-stream.yaml` planned→active（实现后）
- [ ] flows / ProjectFeature 如需补一条

### Task 7: 回归

- [ ] Go test activate-session
- [ ] Vitest saved accounts + navbar 交互
