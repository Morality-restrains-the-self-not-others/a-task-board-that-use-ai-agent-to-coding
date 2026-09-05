# Value Stream: SSO 厂商门户原始 HTML 修复

> Derived from design: `.claude/plans/sso-raw-html-fix/design.md`

## Value Summary
修复 SaaS AI Provider (:8010) 全部 Marketplace API 端点因 `views.py` 拆分遗漏导入导致的 NameError，恢复厂商门户 SSO 登录及镜像管理功能。

## Related Value Streams
- **ai-provider-oidc-login**: modification — 本修复恢复该流中 `/api/vendor/auth/me/` 等端点的正常响应
- **SSO bridge Connection Refused fix**: sibling — 同为 ai-provider 稳定性修复

## End-to-End Flow
[主站镜像市场点击"厂商门户(SSO)"] → [SSO bridge 换票] → [Vue SPA 加载] → [调用 /api/vendor/auth/me/] → [返回 JSON 用户信息] → [厂商门户可用]

## Value Increments

### Increment 1: 补全 views 拆分遗漏导入（唯一增量）
**Value to user:** 厂商门户 SSO 登录恢复正常，所有镜像管理 API 可用
**Scope:** 6 个 view 文件各添加 `from .utils import <symbols>` 导入语句
**Depends on:** nothing
