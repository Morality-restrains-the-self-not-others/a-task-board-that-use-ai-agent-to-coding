# [运行时] start-vm-auto 后容器 exchange-refresh 返回 TOKEN_ACCESS_INVALID

## 失败现象

云主机 UserData 拉起 `task2app-container` 后，`onlineServiceJS` 日志：

```text
token-exchange: FAIL Error: HTTP 401 .../server-container-token/exchange-refresh/
{"detail":"无效的 access_token","error_code":"TOKEN_ACCESS_INVALID"}
```

`docker run` 中已注入非空 `ACCESS_TOKEN`，且 `BUSINESS_API_ENDPOINT` 可被 normalize 为带端口 URL。

## 环境与上下文

- 路径：`POST .../cloud/compute/start-vm-auto/` → persist/finalize → UserData 替换 `__TASK2APP_ACCESS_TOKEN__`
- 换票：容器 → APISIX → taskAgentSupport → **taskCredentialService**（Go SSOT）
- relayToTrae 直启已改走 Go `/v1/token/init`；云主机 bootstrap 曾仍用 Django 本地 `ContainerTokenLifecycleService`

## 排查过程

1. 确认 401 来自 Go `ExchangeRefresh` → `ValidateToken` → `FindByAccessToken` 未命中（非 scope mismatch / 非已换票 403）
2. 对照 `upsert_bootstrap_container_tokens`：`DjangoContainerTokenSessionRepository.save` 在 token 字段迁出后**只写** `business_api_endpoint`，本地生成的 access **不入** `container_tokens`
3. 对照 relay 路径：`_issue_token_via_go` 已正确写入 SSOT

## 根因

`start-vm` / `start-vm-auto` 的 `build_userdata_runtime_payload` → `upsert_bootstrap_container_tokens` 在 Token 迁 Go 后仍本地造票并注入 UserData，**未调用** `POST /v1/token/init`，导致容器持有的 token 在 SSOT 中不存在。

## 解决方案

1. `upsert_bootstrap_container_tokens` 改为调用 `_issue_token_via_go`（Go `/v1/token/init`）
2. CloudServerConfig 仅更新元数据 / SSH；token 生命周期以 Go 为准
3. 部署后重启 `saas-backend`，对新启动的 VM 生效（已写入旧 UserData 的实例需重新 start-vm-auto）

## 预防措施

- 凡注入 `ACCESS_TOKEN` / `userdata_access_token` 的路径必须经 Go token init
- 单测断言 bootstrap 调用 `_issue_token_via_go`（见 `tests/test_container_runtime_tokens.py`）
- Token 字段迁表后禁止再依赖 Django repository「内存返回、不落库」冒充已持久化
