# OAuth 回调失败 Toast — 薄切片验收

## 自动化

在 `taskFE/app` 下执行：

```bash
npm test -- --run utils/gitSiteOAuthCallbackUtils.test.js tests/domain/oauth_callback/oauth_callback_domain_model.test.js
```

**签收（auto-flow）：** 2026-05-27 — Vitest 14 passed（utils 8 + domain 6）

## 手工 E2E

1. 打开项目详情页，对 GitLab 仓库点击「OAuth 授权」。
2. 制造或等待回调失败（如 `profile_failed`）。
3. 回跳 URL 曾含 `?gitlab=profile_failed` 时：
   - 出现红色 Toast：「授权失败：无法读取 GitLab 用户资料」
   - URL 已清除 `gitlab` query
4. 成功 `gitlab=ok` 时不出现 Toast。

## 关联实现

- `app/src/utils/gitSiteOAuthCallbackUtils.js`
- `app/src/main.js` → `setupOAuthCallbackToastGuard`
