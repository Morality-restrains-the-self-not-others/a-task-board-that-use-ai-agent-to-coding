# [运行时] start-vm-auto 报 Go /v1/token/init returned 502

## 现象

- TraceID 例：`2a8dcf7f-6739-433a-8b17-e0ee31ba8476`
- 前端 `POST .../cloud/compute/start-vm-auto/` → 500
- 响应：`{"message":"Go /v1/token/init returned 502","status":"error"}`
- saas-backend 日志响应体为 nginx HTML：`502 Bad Gateway`（`nginx/1.24.0`）

## 根因

1. Django `taskCredentialServiceBase` 曾配置为公网 `${scheme}://${subdomains.credential}`（`credential.api.daydaymoney.com`）。
2. 边缘 nginx 将 credential 子域反代到 `主机:8015`，而 `taskCredentialService` **仅监听 `127.0.0.1:8015`**，公网/局域网 IP 连不上 → nginx 502。
3. nginx 示例已注明：`aiendpoint / credential / ... 若上游仅绑 127.0.0.1，边缘会 502`。

## 修复

1. Django / TCG 内网调用 CRED 改为 **loopback** `http://127.0.0.1:8015`（与 `internalApiBase` 同理）。
2. Token init 路径改为携带 scope：  
   `POST /v1/token/init/tenant/{tenantId}/workspace/{workspaceId}/task/{taskId}`

## 验证

```bash
curl -s -X POST \
  "http://127.0.0.1:8015/v1/token/init/tenant/t/workspace/w/task/task_x" \
  -H 'Content-Type: application/json' -d '{}'
# 期望 200 + access_token
```
