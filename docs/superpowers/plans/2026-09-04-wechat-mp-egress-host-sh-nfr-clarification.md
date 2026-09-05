# NFR — wechat-mp-egress-host-sh

## 路径分片键审视

| 路径 | 分片 ID | 判定 |
|------|---------|------|
| POST follow-qr | user_id（会话） | L0 — 用户级票，非租户膨胀 |
| egress forward | n/a（internal） | L0 — 无租户数据 |

## 幂等性审视

| 路径 | 副作用 | 幂等 |
|------|--------|------|
| follow-qr | 写票 + 微信创码 | 既有 pending 复用 + Idempotency-Key |
| egress forward | 无本地状态 | L0 透传 |

## 其他 NFR

- Availability: SH egress 单点 — systemd restart；taskAuth 可临时清空 baseUrl 回退直连（仍受 INFRA IP 白名单限制）
- Security: 禁止 env proxy；显式 secret；host allowlist
