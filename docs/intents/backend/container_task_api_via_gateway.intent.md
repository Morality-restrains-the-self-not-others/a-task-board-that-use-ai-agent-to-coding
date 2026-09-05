# 容器 TASK_API 经网关换票（修复 Django 404）

## 意图

云实例容器启动后调用 `…/server-container-token/exchange-refresh/` 时，不得再直连 Django `:8001`（该路径已迁出），须经 **APISIX 网关 → taskAgentSupport → taskCredentialService**，换票成功。

## 背景

- 热路径已迁：`exchange-refresh` / `refresh-access` 等由 taskAgentSupport 转发。
- UserData 仍注入 `http://<host>:8001/.../cloud` → Django 返回 404「未找到请求的 API 路径」。
- taskAgentSupport 仅绑 `127.0.0.1`，APISIX 无法从宿主机 IP 访问。

## 验收标准

1. APISIX 存在 `container-inbound-token` 路由，upstream=`taskAgentSupport`，`auth_mode=none`。
2. APISIX 存在 `server-userdata-verify` 路由，upstream=`django`，`auth_mode=none`。
3. `get_userdata_verify_base_url()` / Cloud `userdataVerifyBaseURL()` 优先网关公网 origin。
4. `POST http://<gateway>:18081/.../exchange-refresh/` 不再因路径缺失返回 Django 404（无 token 时可为 4xx 业务错误，非 404）。
5. 存量直连 Django `:8001` 的请求经兼容代理转发 TAS，同样非路径 404。
6. 相关单测通过。

## 变更日期

2026-07-10


## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：换票路径改经 Gateway（HTTP），无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 容器 TASK_API 经网关换票（修复 Django 404） | — | — | — | — | 换票路径改经 Gateway（HTTP），无新增业务事件 |
