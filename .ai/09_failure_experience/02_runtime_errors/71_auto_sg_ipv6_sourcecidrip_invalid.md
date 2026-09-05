# [运行时] 自动创建安全组：平台出口 IPv6 误写入 SourceCidrIp → InvalidParam.SourceCidrIp

## 现象

任务详情启动服务器（自动创建安全组）失败，SSE / 页面展示：

`自动创建网络资源失败: create security group: ensure whitelist ingress on sg-…: SDKError: StatusCode: 400 Code: InvalidParam.SourceCidrIp`

## 环境与上下文

- 路径：`start-vm-auto` → `CLOUD_SERVER_START_AUTO` → `taskEvents` `ProvisionAutoNetworkResources` → `ensureWhitelistIngress`
- 复现事件：`cloud_event_id=867414823409840128`，`task_id=task_13757169132576941867`
- `event_data.extra_ingress_cidrs` 含：`183.250.1.132/32`（合法 IPv4）与 `240e:…/128`（IPv6）

## 根因

1. `taskCloudService` / `taskEvents` 的 `detectPlatformEgressCIDR` 在双栈出口上可能拿到 **IPv6**，并写入 `extra_ingress_cidrs`
2. `authorizeIngress` 一律填阿里云 **`SourceCidrIp`**；该字段**仅接受 IPv4**，IPv6 须用 **`Ipv6SourceCidrIp`**
3. 复用已有 SG 时 `ensureWhitelistIngress` 授权失败 → 整次自动创建网络失败

## 修复

- `ingressSourceFields`：IPv4 → `SourceCidrIp`，IPv6 → `Ipv6SourceCidrIp`
- 平台出口探测优先 IPv4 端点（`ipv4.icanhazip.com` / `api.ipify.org`），IPv6 仅作回退
- IPv6 授权若仍不被 VPC 支持：记日志并跳过，不阻断自动 SG（IPv4 规则照常失败则仍报错）
- 错误包装带上失败 `cidr=`，便于下次排障

## 预防

- 向阿里云 SG 写规则前必须按地址族分流字段；禁止把 IPv6 放进 `SourceCidrIp`
- 平台探测 CIDR 默认偏好 IPv4；单测覆盖 `ingressSourceFields` / `isIPv6CIDR`
- 自动 SG 失败日志应包含具体 CIDR（勿只回传 SDK Code）
