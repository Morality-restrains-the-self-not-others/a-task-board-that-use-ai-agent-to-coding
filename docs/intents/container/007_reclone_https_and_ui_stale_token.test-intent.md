# 测试意图：reclone HTTPS + UI stale token

对应：`007_reclone_https_and_ui_stale_token.intent.md`

| 用例 | 位置 | 状态 |
|------|------|------|
| prepareOauthHttpsGitClone | `bootstrap.cloneCredentials.test.mjs` | ✅ |
| resolveUiPathAccessToken | `uiAccessToken.test.mjs` | ✅ |
| /ui stale 302 + session/ui-redirect | `e2e/ui-stale-token-redirect.api.spec.mjs` | ✅ |
| CDP 打开 /ui/{stale} | `e2e/ui-stale-token-redirect.cdp.spec.mjs` | ✅ |
