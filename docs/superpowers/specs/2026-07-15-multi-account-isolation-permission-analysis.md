# 权限分析：多账号切换私有资源隔离

**日期**: 2026-07-15  
**设计**: `2026-07-15-multi-account-isolation-design.md`  
**作者**: claude

## 变更点与权限边界

| 改动点 | 主体 | 资源 | 规则 | 风险若缺失 |
|--------|------|------|------|------------|
| 清 `sessionid` | 已登录用户（切换中） | 浏览器 Session Cookie | 切换成功必须删除旧会话 cookie | Session 身份仍为前一用户 |
| Token 优先鉴权 | 任意带 Token 的 API 客户端 | 全部 DRF 默认鉴权视图 | Authorization Token 胜于 Session | 多账号切换串权 |
| `me()` path 校验 | 已认证用户 | `/api/user/{id}/.../me/` | path id 必须等于认证用户 | 换 cookie/userId 仍读到 Session 用户资料 |
| 离开租户 URL | 切换后的用户 B | 租户私有页面 | 不得保留 A 的 `/tenant/{id}` | B 在 A 租户上下文拉资源（403 或误展示） |

## 角色矩阵

| 角色 | 切换账号 | 见他人私有资源 |
|------|----------|----------------|
| 普通用户 A→B | 允许（本机槽内） | **禁止** |
| 未登录 | 不可切换 | n/a |
| 管理员 | 同普通用户（本功能无特权旁路） | **禁止** |

## 审计结论

- 无新 endpoint；权限增强为纵深防御。
- 须保证显式 `authentication_classes` 与 DEFAULT 同序，避免局部回退 Session 优先。
