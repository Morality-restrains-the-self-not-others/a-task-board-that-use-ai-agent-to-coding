# 测试意图：导航栏多账号切换

## 对应功能意图

`docs/intents/frontend/navbar_multi_account_switcher.intent.md`

## 测试点

| ID | 场景 | 期望 | 层级 |
|----|------|------|------|
| T1 | savedAccounts upsert / 上限 5 | 第 6 个被拒绝 | Vitest |
| T2 | 激活账号与 cookie/token 一致 | getActive 匹配 | Vitest |
| T3 | activate-session 合法 token | 200 + user | Go |
| T4 | activate-session 非法 token | 401 | Go |
| T5 | activate-session user_id 不匹配 | 400 | Go |
| T6 | 下拉单击开 / 双击关 | 状态正确 | Vitest |
| T7 | 切换成功后 userId/authToken 更新 | 指向目标账号 | Vitest |
| T8 | 登录成功写入槽 | 列表含新账号 | Vitest |
| T9 | add_account=1 不清除已有槽 | 槽数量不减 | Vitest |
| T10 | resolveSwitchHref 在 `/tenant/...` | 落到 `/user/{B}/profile/`，去掉 workspace_id | Vitest |
| T12 | activate 成功后清 JS 可读旧 sessionid | cookie 无旧值 | Vitest |
| T13 | Session(A)+Token(B) 同时存在 | DRF request.user == B | Django |
| T14 | me() path user_id ≠ 认证用户 | 403 | Django |
| T15 | activate-session Set-Cookie sessionid，body 无 session_key | HttpOnly cookie | Go |
| T16 | enrich-login 返回 session_key | 非空 | Django |
| T17 | A→B 切换后 me 为 B，path(A)+Token(B)→403 | 身份隔离 | Playwright |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-14 | 初版 |
| 2026-07-15 | 增加 T10/T12/T13/T14 隔离用例 |
| 2026-07-15 | 增加 T15/T16/T17 sessionid 回写与跨账号 E2E |
