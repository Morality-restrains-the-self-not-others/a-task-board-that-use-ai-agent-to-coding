# 测试意图：login_history_profile_sidebar

## 可执行测试

- `taskFE/app/src/components/UserCenterSidebar.login-history.test.js`
- `taskFE/app/src/views/UserLoginHistory.test.js`
- `taskFE/app/src/components/UserListRow.login-history.test.js`

## 用例

| ID | 断言 |
|----|------|
| F1 | 侧边栏链接指向 `/profile/login-history/`（有 userId cookie 时 `/user/{id}/profile/login-history/`） |
| F2 | 列表渲染 IP 与入口文案 |
| F3 | 空态「暂无登录记录」 |
| F4 | 失败时错误节点含 `data-traceId` |
| F5 | 超管行是 `<a href="/system-admin/users/{id}/login-history/">` |
