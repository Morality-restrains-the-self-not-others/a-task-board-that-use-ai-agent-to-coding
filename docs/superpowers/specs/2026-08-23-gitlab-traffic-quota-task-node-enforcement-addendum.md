# 补遗：任务贴公网节点不再享受 RFC1918 误豁免

- **Date**: 2026-08-23
- **Parent**: `2026-08-23-gitlab-traffic-quota-enforcement-design.md` / ADR-0036
- **Status**: accepted

## Context

GitLab 跑在 Docker bridge 端口映射后，`REMOTE_ADDR` 常为 `172.17.0.1` / `172.18.0.1`。
旧闸门把 RFC1918 一律当「同区域内网」skip，导致阿里云任务机经公网 Host 克隆也被当成内网。
Workhorse 把客户端做成 `127.0.0.1` 时，公网 HTTP git 同样被误跳过。

系统**从不查 VPC ID**。同区域内网以 **请求 Host**（内网域名 / GitLab 私网 IP）以及 **10/8、192.168/16 源 IP** 判定。

## Decision

| 调用方 | 预购耗尽时 |
|--------|------------|
| 公网 GitLab Host（含任务贴节点、Docker NAT 源 IP） | 拒绝 |
| 同区域内网 Host / GitLab 私网 IP / VPC 10/8·192.168 | 放行 |
| GitLab CI（`request_from_ci_build?` / `from_ci`） | 放行 |
| 闸门不可达 | 仍 fail-open |

GitLab 仅在 Host/VPC 规则命中时向 taskBill 传 `is_intranet: true`。taskBill 对 `is_intranet` 与 `from_ci` 放行。

## 验收

1. 公网 Host + `172.17.0.1` 预购 0 → deny。
2. `from_ci=true` → allow / `CI_SKIP`。
3. `is_intranet=true`（Host/VPC）预购 0 → allow / `INTRANET_SKIP`。
4. 设置页写明任务节点走公网一并阻断、同区域内网与 CI 仍可拉。

## 补记（2026-08-23 修复完成）

### request_host 快照（RequestContextHostPatch）

`Gitlab::RequestContext#start_request_context` 只保留 `client_ip`、丢弃 Rack request，
导致请求内 `request_host` 恒为空 → 私网 Host 克隆（`Host: 10.2.150.68:8012`）被误按公网拦截。
修复：`start_request_context` 处将 `Rack::Request.new(request.env).host` 快照进
`Gitlab::SafeRequestStore[:trae_http_request_host]`，`request_host` 优先读取该快照。

验证：公网 Host → 403；私网 IP Host（10.0.1.8）→ 200；日志 `host=gitlab.daydaymoney.com` 正常捕获。
loopback/127.0.0.1 仍按设计排除（防 Workhorse 误判），同区域 HTTP 走公网域名需 real_ip（OPT-20260823-052）。

### 区域实例（tencent-sh-1）部署要点

- 远程 → taskBill 依赖 8004 反向隧道（`ensure-edge-tunnels.sh` DOCKER_BIND_PORTS，
  bind docker0 172.17.0.1，GatewayPorts clientspecified；容器经 `host.docker.internal`）。
  直连 `${INFRA_HOST}:8004` 从 sh 不可达（VPC 隔离）。
- 远程实例租户组缺失（`tenant-*` 组为零）→ 全部项目「未映射 fail-open」，见 OPT-20260823-060。
