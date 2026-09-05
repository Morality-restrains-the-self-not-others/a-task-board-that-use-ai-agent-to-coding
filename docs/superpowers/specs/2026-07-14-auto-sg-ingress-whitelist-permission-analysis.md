# 权限分析 — 自动 SG 入网白名单

- **日期**: 2026-07-14
- **设计**: `docs/superpowers/specs/2026-07-14-auto-sg-ingress-whitelist-design.md`

## 结论

无新公网 API、无新角色。沿用既有 `start-vm-auto` 鉴权（租户成员 + 云授权）。

| 改动点 | 角色 | 权限 | 变更 |
|--------|------|------|------|
| POST start-vm-auto | 租户成员 | 启动云主机 | 仅多写 `client_public_ip` 到事件，不扩大权限 |
| CLOUD_SERVER_START_AUTO | 系统（taskEvents） | 用租户云 AK 调 ECS | 入站规则从全开改为白名单 |
| CLOUD_SERVER_STARTED Phase B | 系统 | 同上 | 补服务器 IP 规则 |
| 手动选择已有 SG | 租户成员 | 不变 | 无 |

## 敏感数据

- `client_public_ip` 写入事件/日志：属网络元数据，按现有 tracelog 脱敏惯例记录即可，禁止与 AK 同日志行明文拼接密钥。

## 审计

- 自动创建/收紧 SG 须打结构化日志：`sg_id`、`client_cidr`、`server_cidr`、`revoked_full_open`。
