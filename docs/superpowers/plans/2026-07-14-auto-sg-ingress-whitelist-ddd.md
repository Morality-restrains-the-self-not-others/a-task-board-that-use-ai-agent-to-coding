# DDD — 自动 SG 入网白名单

## 限界上下文

- **Cloud Network Provisioning**（`taskEvents` / aliyun）：安全组聚合行为变更
- **Cloud Compute Start**（`taskCloudService`）：启动命令携带 `ClientPublicIP` 值对象

## 值对象

- `ClientPublicIP` — 规范化 IPv4/IPv6，存为 CIDR `/32` 或 `/128`
- `ServerPublicIP` — 同上
- `SGIngressWhitelist` — 允许源 CIDR 集合；显式排除 `0.0.0.0/0`

## 领域服务

- `ProvisionAutoNetworkResources(clientIP)` — Phase A
- `TightenAutoSGIngress(sgID, serverIP)` — Phase B
- `RevokeFullOpenIngress(sgID)` — 反腐败：清理历史全开规则

## 领域事件字段增量

`CLOUD_SERVER_START_AUTO` / 链式 `CLOUD_SERVER_STARTED`：

- `client_public_ip: string`
- `auto_sg_whitelist: bool`

无新聚合根；无新表。
