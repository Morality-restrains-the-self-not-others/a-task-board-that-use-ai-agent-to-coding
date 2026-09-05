# 实施计划: OAuth Session Cookie 冲突修复

> 输入:
> - 设计文档: `docs/specs/oauth-redirect-loop-port-4000/design.md`
> - 价值流: `docs/superpowers/plans/2026-06-27-oauth-session-cookie-collision-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-27-oauth-session-cookie-collision-nfr-clarification.md`

## 修复范围

一行配置变更：`gitOauth/config/settings.py` 添加 `SESSION_COOKIE_NAME = 'gitoauth_sessionid'`

## 任务清单

### Task 1: 添加 SESSION_COOKIE_NAME 配置 ✅

- [ ] **文件**: `gitOauth/config/settings.py`
- [ ] **位置**: 在现有 Django session 相关配置附近（`SESSION_COOKIE_NAME` 默认值被覆盖的位置）
- [ ] **变更**: 添加 `SESSION_COOKIE_NAME = 'gitoauth_sessionid'`
- [ ] **验证**: `grep SESSION_COOKIE_NAME gitOauth/config/settings.py` 确认输出 `gitoauth_sessionid`

### Task 2: 运行现有测试确认无回归

- [ ] **命令**: `cd gitOauth && python manage.py test --verbosity=2`
- [ ] **预期**: 所有测试通过（session cookie 命名变更不影响功能，测试使用 `override_settings` 或默认行为）

### Task 3: E2E 手动验证 (Playwright 可选)

- [ ] 清除 `183.250.1.132` 的 cookies
- [ ] 在 port 4000 登录
- [ ] 进入项目详情页 → 点击 OAuth 授权 → 完成 GitLab 授权
- [ ] 预期: 回到项目页，未跳转到登录页
- [ ] DevTools → Application → Cookies: 确认 `sessionid` 和 `gitoauth_sessionid` 并存

## 依赖关系

```
Task 1 (代码) → Task 2 (单测) → Task 3 (E2E 手动)
```

## 回滚方案

删除 `gitOauth/config/settings.py` 中的 `SESSION_COOKIE_NAME = 'gitoauth_sessionid'` 行即可恢复。
