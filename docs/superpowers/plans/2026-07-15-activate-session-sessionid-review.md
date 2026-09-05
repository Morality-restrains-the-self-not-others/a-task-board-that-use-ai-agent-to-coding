# Review：activate-session sessionid + 跨账号 E2E

**日期**: 2026-07-15  
**设计**: `2026-07-15-activate-session-sessionid-design.md`

## 结论

**通过**

| 项 | 状态 |
|----|------|
| enrich 返回 session_key | ✅ Django T16 |
| activate/login Set-Cookie HttpOnly | ✅ Go T15 |
| body 不泄漏 session_key | ✅ Go |
| Playwright T17 | ✅ 已落地（缺第二账号或 PRE_COMMIT 时 skip） |
| OpenAPI 更新 | ✅ |

## 建议

- 部署时确保网关透传 `Set-Cookie`；`SSO_COOKIE_DOMAIN` 与前端/Django 一致。
