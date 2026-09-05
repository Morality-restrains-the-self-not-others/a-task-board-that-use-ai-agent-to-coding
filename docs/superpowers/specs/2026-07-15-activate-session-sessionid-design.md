# 设计：activate-session 回写 sessionid + 跨账号隔离 E2E

**日期**: 2026-07-15  
**类型**: 认证增强  
**状态**: 已批准（goal-mode 自动采用）  
**迭代**: `activate-session-sessionid-cookie`  
**基于**: `2026-07-15-multi-account-isolation-design.md`

## 目标

1. `POST activate-session`（及登录）成功后，浏览器获得目标用户的 HttpOnly `sessionid`
2. Playwright E2E 验证 A→B 切换后身份隔离

## 方案（选定 D）

| 步骤 | 落点 |
|------|------|
| enrich-login 返回 `session_key` | Django internal |
| taskAuth 对浏览器 `Set-Cookie: sessionid`（HttpOnly, SameSite=Lax, Domain/Secure 对齐 auth_views） | taskAuth |
| JSON 对外响应**剥离** `session_key` | taskAuth |
| 前端保留 `clearCookie` 兜底清非 HttpOnly 旧值 | Vue |
| Playwright 双账号切换断言 | task2app/playwright |

## 成功标准

| # | 标准 | 验证 |
|---|------|------|
| S1 | enrich 200 含 `session_key` | Django 测试 |
| S2 | activate-session 响应含 `Set-Cookie: sessionid=` 且 body 无 `session_key` | Go 测试 |
| S3 | 登录响应同样写入 sessionid | Go 测试（复用 helper） |
| S4 | Playwright：A→B 后 `/me/` 为 B，且 cookie 有 sessionid | E2E（无第二账号则 skip） |

## python_api_approval

n/a — 仅改 internal enrich-login 响应字段；无新公网 Python 接口。
