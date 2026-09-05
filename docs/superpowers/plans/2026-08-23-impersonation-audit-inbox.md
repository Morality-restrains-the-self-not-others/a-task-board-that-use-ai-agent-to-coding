# 实施计划：模拟登录审计标识与收信箱

- **日期**: 2026-08-23

## 切片（TDD）

1. domain.ValidateStart 增加 Reason；单测缺理由/过短/过长。
2. DDL 037：reason 列 + `auth_user_inbox_message`。
3. handler 读 JSON reason；写会话+信件；改现有 impersonation 测试。
4. tracelog 头→ctx→http_request；单测。
5. forward-auth 会话头 + APISIX upstream_headers + Python 断言。
6. 前端理由弹窗 + impersonateUser(id, reason)。
7. GET/PATCH inbox API + Vue 页 + 侧栏。
8. 元规则/ADR/意图/v103 架构（本计划同时交付）。
