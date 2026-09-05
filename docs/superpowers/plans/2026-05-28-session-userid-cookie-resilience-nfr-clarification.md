# NFR 澄清：Session userId Cookie 业务韧性

## 适用类别与等级

| 类别 | 等级 | 说明 |
|------|------|------|
| 安全 | L3 | auth 域；profile 为真源，cookie 仅提示 |
| 可用性 | L2 | session 有效时核心路径不可用视为缺陷 |
| 性能 | L2 | guard + Navbar 各一次 profile 可接受；resolver inflight 去重 |
| 可维护性 | L2 | 集中 sync/resolver，避免批量改 getCookie |

## 质量场景

| ID | 刺激 | 响应 |
|----|------|------|
| Q1 | 清除 userId cookie，session 有效，访问 task-detail | Git 身份 API 200，推送可操作 |
| Q2 | 同上，访问 git-site-oauth | 不显示「非本人」警告 |
| Q3 | profile 403 | 跳转登录，清除 stale cookie + authToken |
| Q4 | 同 tick 多次 resolveAuthenticatedUserId | 仅一次 profile 请求 |

## 领域模型影响

- `SessionUserIdResolver` 保持无状态工具函数 + inflight Promise
- `UserIdCookieSync` 幂等，不与 guard 清除冲突

## 边界

- 不保证 guard 完成前同步 computed 100% 正确（Increment 3 缓解 Navbar）
