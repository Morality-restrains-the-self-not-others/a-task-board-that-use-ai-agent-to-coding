# 权限分析：relay token-init / env-prepare / precheck 迁 TCG

**日期：** 2026-07-09  
**设计：** `docs/superpowers/specs/2026-07-09-relay-token-init-tcg-routing-design.md`

## 结论

| 端点 | 鉴权 | 授权边界 |
|------|------|----------|
| env-prepare / token-init / precheck | taskGateway Token + TCG `validate_session` | 租户成员即可；**不要求** CloudServerConfig（本地直启） |
| Django 直连同路径 | `@require_django_forward` → 410 | 强制走网关 |

无新增角色；与既有 register/start/stop 同族。
