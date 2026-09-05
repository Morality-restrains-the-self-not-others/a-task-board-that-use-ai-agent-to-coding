# NFR 澄清：微信关联账号查单

支撑等级：L2（标准）。鉴权为既有员工门禁（L3 认证，本增量不改认证栈）。

## 路径分片键审视

| 路径 | 是否携带分片 ID | 该 ID 是否合适 | 可伸缩性 | 动作 |
|------|-----------------|----------------|----------|------|
| GET `/api/system-admin/orders/?wechat_account=` | 否 | n/a | L0 | 超管全局只读；补 `user_id`/`pay_openid`/`pay_unionid` 等值索引。升级：管理查单 p95>200ms 或 QPS>20 |
| GET `/api/internal/users/wechat-linked-account/?q=` | 否 | n/a | L0 | 等值 + LIMIT 50；nickname 前缀索引。升级：解析 p95>50ms |
| FE `/system-admin/order-records/?wechat_account=` | 否 | n/a | L0 | 页面级查询串，非分片键 |

Hard Gate：通过（L0 均有理由与升级触发）。

## 幂等性审视

| 路径 | 副作用 | 等级 | 理由 |
|------|--------|------|------|
| 上述 GET | 无 | L0 | 只读；重复触发不写库、不发事件 |
| 查询按钮 | 无（只读 GET） | L0 | 同步门闩防连点；不需要 Idempotency-Key |

Hard Gate：通过。

## 其他

- 安全：内部密钥、员工门禁、日志脱敏、响应不下发 openid。
- 性能：禁止 LIKE；LIMIT 50 user_ids。
- 可观测：trace_id 贯穿 taskBill→taskAuth；出站调用记 duration_ms。
- 无新 MQ。
