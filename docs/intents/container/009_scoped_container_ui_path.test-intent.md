# 测试意图：容器页面 UI scoped path

| 场景 | 位置 | 期望 |
|---|---|---|
| `buildScopedUiPath` 拼接 | `onlineServiceJS/src/scopedUiPath.test.mjs` | 含 tenant/workspace/task/token |
| 无 scope 回退 | 同上 | `/ui/{token}` |
| stale token 重定向目标 | `uiAccessToken.test.mjs` | scoped path（有 env） |
| `GET /ui/tenant/…/task/…/{tok}` | e2e / 单测 | 200 HTML |
| 旧 `/ui/{tok}` | e2e | 有 scope 时 302 到 scoped |
| `ui-redirect.ui_path` | e2e | scoped |
| `build_container_page_url` | Django / Go 单测 | scoped URL |
| `_normalize_url_origin` / `extractAccessTokenFromUIURL` | 解析单测 | 新旧 path 均可取 token |
