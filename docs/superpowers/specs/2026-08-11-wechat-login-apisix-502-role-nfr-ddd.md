# wechat-login-apisix-502 — Role / NFR / DDD（精简）

## Role-Permission
- 无新 endpoint；预检走既有公开 `GET /api/auth/wechat/login/`。
- 无 RBAC 变更。

## NFR（L2，认证入口 L3 可用性）
- 可用性：清库窗口后自动恢复 auth；前端不裸 502。
- 可观测：APISIX access/error → Loki，按 X-Trace-Id 检索。
- 安全：预检同源；不新增密钥暴露。

## DDD
- 无新聚合/事件；微信登录成功路径既有事件不变。
- 运维编排与前端预检为支撑能力，非领域意图。
