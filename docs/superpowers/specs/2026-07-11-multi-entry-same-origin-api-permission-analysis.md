# 多入口同源 API — 角色权限分析

- **日期**: 2026-07-11
- **设计**: `2026-07-11-multi-entry-same-origin-api-design.md`
- **结论**: 无新 endpoint；权限边界不变

## 变更面

| 改动 | 权限影响 |
|------|----------|
| Edge `/api/` 反代 | 浏览器仍走既有 API 鉴权（session/CSRF）；Host 变为入口域名 |
| `API_BASE_URL=''` | 不扩大能力面 |
| `publicEntryOrigins` | 仅放宽 CSRF/ALLOWED_HOSTS/CORS 至明确列出的入口 Origin |

## 风险与控制

- **Host 伪造**：边缘须覆盖 `Host`/`X-Forwarded-*`；不信任客户端伪造的转发头进入信任链时保持现有网关策略
- **白名单膨胀**：仅追加运营确认的入口；禁止 `*` CORS + credentials

## 裁决

无需新增角色或权限矩阵变更；登录页公共 GET 仍为匿名可读。
