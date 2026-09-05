# 实施计划：登录历史

- **日期**: 2026-08-25
- **设计**: `docs/superpowers/specs/2026-08-25-login-history-design.md`

## 任务

- [ ] 1. `taskAuth/domain/login_history.go`：entry/method 校验与中文标签
- [ ] 2. `dataMigrate/taskAuth/044_login_history.sql`：分区表
- [ ] 3. store INSERT/LIST + `recordSuccessfulLoginFromRequest`（含 `USER_LOGGED_IN` 增补字段）
- [ ] 4. 接入 finalizeLogin（customer/admin）、wechat、access_token、register、activate-session
- [ ] 5. GET 自己 / 超管列表 handlers + 网关 token 路由 + openapi
- [ ] 6. PIPL export `login_history` section
- [ ] 7. taskFE 侧边栏 + 用户页 + 超管页/行链接
- [ ] 8. 价值流 YAML、INDEX、VERSION_HISTORY、wsd
