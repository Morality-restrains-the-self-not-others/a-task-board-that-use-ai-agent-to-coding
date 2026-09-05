# 修复：自动 SG InvalidParam.SourceCidrIp（IPv6）

- **日期**: 2026-07-21
- **状态**: implemented（goal-mode 直接执行）
- **关联**: `docs/superpowers/specs/2026-07-14-auto-sg-ingress-whitelist-design.md`
- **失败经验**: `.ai/09_failure_experience/02_runtime_errors/71_auto_sg_ipv6_sourcecidrip_invalid.md`

## 根因

平台出口探测在双栈环境可能返回 IPv6，写入 `extra_ingress_cidrs` 后经 `AuthorizeSecurityGroup.SourceCidrIp` 提交；该字段仅接受 IPv4 → `InvalidParam.SourceCidrIp`。

复现证据（`cloud_event_id=867414823409840128`）：

```json
"extra_ingress_cidrs": ["183.250.1.132/32", "240e:37a:24d0:8100:48de:c783:d660:34b1/128"]
```

## 方案（已落地）

1. `ingressSourceFields`：IPv4→`SourceCidrIp`，IPv6→`Ipv6SourceCidrIp`
2. 平台出口探测优先 IPv4 端点；IPv6 仅回退
3. IPv6 规则若 VPC 不支持则跳过并记日志，不阻断 IPv4 白名单建组
4. 授权失败错误包装带 `cidr=`，便于排障

## 🕸️ CRG

`code-review-graph` MCP unavailable；无独立 CLI → soft-skip，已用事件库 + 源码定位。
