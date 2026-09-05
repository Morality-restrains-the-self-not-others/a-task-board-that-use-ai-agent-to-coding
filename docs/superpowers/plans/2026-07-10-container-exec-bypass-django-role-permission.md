# 角色权限分析：容器执行热路径零 Django

- **日期**: 2026-07-10
- **设计文档**: `docs/superpowers/specs/2026-07-10-container-exec-bypass-django-design.md`
- **范围**: Phase A（validate + resolve 迁出 Django）

## 权限影响表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST taskAuth `/api/internal/container-gateway/validate-session/` | tcg（internal secret）→ 代表浏览器用户 | Tenant | authz | Django 曾做 session+membership | 需在 Go 复现 | Internal secret + 身份解析 + tenant membership；拒绝打 WARN |
| GET taskCloudService `.../container-target/` | tcg（internal secret） | Task | read | 已有 InternalSecret | ✅ | 保持；补 override 时仍仅 internal |
| tcg handlers 热路径 | 已登录租户成员 | Task | execute | 原 Django validate+resolve | 迁移后不得弱化 | auth 401/403；无 target 409；跨租户 403 |
| Django validate/resolve（热路径） | — | — | — | — | 🔴 废弃 | 保留冷路径；热路径不再调用 |

## 角色

无新角色。沿用：已认证用户 + 公司（tenant）成员；relay-to-trae path 豁免 CloudServerConfig.exists。

## IDOR / 越权风险

| 风险 | 缓解 |
|------|------|
| 伪造 internal 调用 | X-Internal-Secret 必校验 |
| 用他人 Cookie 访问他租户 task | tenant membership 校验 +（非 relay）cloud lookup 存在性 |
| container_page_url 指向任意 upstream | 仅覆盖 base URL；仍要求 token 来自 Credential SSOT（按 scope） |

## 结论

Phase A 不引入新角色；权限边界从 Django internal 平移到 taskAuth + taskCloudService，语义不得弱于现网。
