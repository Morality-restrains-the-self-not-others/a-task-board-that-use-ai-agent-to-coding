# 价值流：多账号切换私有资源隔离

**日期**: 2026-07-15  
**设计**: `2026-07-15-multi-account-isolation-design.md`

## 端到端价值流

```
用户 A 已登录（sessionid=A, authToken=A）
  → 打开账号下拉，选择账号 B
  → POST activate-session（Authorization: Token B）
  → 写入 userId=B, authToken=B
  → 清除 sessionid
  → resolveSwitchHref：若在 /tenant/... → /user/B/profile/；去掉 workspace_id
  → 整页刷新
  → 后续 API：Token B 优先，身份为 B
  → B 看不到 A 的私有项目/凭证/me 资料
```

## 最小可行增量

1. 前端清 sessionid + 离开租户 URL（立刻阻断主泄漏路径）
2. 后端 Token 优先 + me() 校验（根因与纵深）

## 测试点映射

| ID | 价值流节点 | 期望 |
|----|------------|------|
| T10 | resolveSwitchHref 租户路径 | 落到 `/user/{B}/profile/`，无 workspace_id |
| T12 | 切换后清 sessionid | cookie 无 sessionid |
| T13 | Session(A)+Token(B) | request.user == B |
| T14 | me() path 不匹配 | 403 |

同步更新：`docs/flows/用户与认证/导航栏多账号切换.wsd`、`value-stream-test-integration.wsd`。
