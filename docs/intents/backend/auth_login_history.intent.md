# 功能意图：成功登录写入登录历史并可按身份查询

## 背景与目标

`auth_user.last_login` 只保留最后一次时间。用户需要在账号中心查看自己的登录 IP / 入口；管理员入口登录也必须留下可区分记录，供超管排查。

## 范围与边界

- 范围内：成功认证落库 `auth_login_history`；用户查自己；超管按 user_id 查；PIPL 导出 section。
- 范围外：失败登录列表、异地登录告警、踢下线、地理定位。

## 约束与风险

- 日志禁止输出完整 User-Agent 以外的 PII；identifier / token 不入库。
- 插入失败不得阻断登录（fail-open + `login_history_insert_failed`）。
- 模拟登录不得给目标用户写历史。
- 用户 API 禁止用 query `user_id` 越权读取他人。

## 验收标准

1. 客户入口与管理员入口密码登录成功各写一行，`entry` 分别为 `customer` / `admin`，`client_ip` 取 `resolveClientIP`（XFF 从右跳过 RFC1918/loopback，避免 Docker 网桥 172.26.0.1）。
2. `GET /api/auth/login-history/` 仅返回当前 `X-User-Id` 的记录，分页 `total/limit/offset`。
3. 用户 A 的 token 不能读到用户 B 的行。
4. 超管 `GET /api/system-admin/users/{id}/login-history/` 非超管 403；目标用户不存在 404。
5. 模拟登录 start 后目标用户历史行数不增加。

## 业务意图 → 事件对照

| 意图 | 事件名 | 发布点 | 消费者 | MQ类型/契约 |
|------|--------|--------|--------|-------------|
| 成功登录 | `USER_LOGGED_IN` | `recordSuccessfulLogin` → `publishUserLoggedIn` | 既有；无新消费者 | Kafka `user-logged-in`；payload 增补 `client_ip`/`entry`/`user_agent` |
| 查询登录历史 | 无 | GET handlers | — | 例外：纯查询 |
