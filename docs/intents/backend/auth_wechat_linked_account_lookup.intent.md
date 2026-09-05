# 功能意图：按微信关联账号解析平台用户

## 意图

内部调用方（taskBill 管理端查单）传入一条精确查询串，taskAuth 返回与该微信身份关联的 `user_id` 列表。

## 角色

- 内部服务（持 `X-TaskAuth-Internal-Secret`）
- 平台员工不直连本接口（经 taskBill 编排）

## 行为

1. `GET /api/internal/users/wechat-linked-account/?q=`（1–128 字符）。
2. 精确匹配（禁止 LIKE）：
   - `wechat_identity.nickname` / `openid` / `unionid`
   - 已有 `wechat_identity` 的用户之 `auth_login_method`（phone / email / username）`identifier`
   - 已有 `wechat_identity` 的 `auth_user_profile.username`
3. 200：`{ "user_ids": ["..."] }`；无命中空数组。
4. 缺 q 或超长：400。无内部密钥：403。
5. 日志只记 `query_len` 与命中人数，不记查询原文（可能为手机号）。

## 非目标

- 公开用户搜索。
- 模糊昵称。
- 返回 openid/unionid 给浏览器。

## 业务意图 → 事件对照

**无对应新事件（书面例外）**：只读解析，不改变绑定状态。
