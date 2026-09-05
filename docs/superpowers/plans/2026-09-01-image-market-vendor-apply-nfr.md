# NFR 澄清 — ImageMarket 恢复厂商申请入口

- **Date:** 2026-09-01
- **Default level:** L2；身份/申请写路径 L3（既有库约束）

## 路径分片键审视

| 路径 | 分片 ID | 适配？ | 可伸缩性 | 动作 |
|------|---------|--------|----------|------|
| `GET /api/ai-provider/vendor-status/` | 无 tenant；按 saas_user_id | 用户级查找，年增量远低于千万 | L0 | 理由：单用户一行 vendor；升级触发：vendor 表 >100 万行再按 saas_user_id 哈希 |
| `POST /api/ai-provider/vendor-application/` 及 upload/sms | saas_user_id | 是（申请主体） | L1 | 已按用户建档；不引入 tenant 键 |
| `/tenant/:tenantId/image-market` | tenantId | 仅页面壳；申请数据不按租户 | L0 | 申请跨租户同一账号；升级不把 vendor 按 tenant 拆 |

## 幂等性审视

| 路径 | 副作用 | 重复触发 | 业务边界 | 键 | 重放 | 级别 |
|------|--------|----------|----------|-----|------|------|
| GET vendor-status | 无 | — | — | — | — | L0 |
| POST vendor-application | 写 vendor 行 | 双击、刷新重提 | 同一 saas_user_id 一份申请 | saas_user_id（pending/qualified → 409） | 冲突返回已有行 | L3 |
| send-sms / verify-phone | 短信/绑定 | 连点 | 用户+手机 | 既有短信门禁 | 限流 | L2 |
| 前端提交按钮 | 触发 POST | 连点 | 同一次意图 | `createClickGuard` + `Idempotency-Key` | 同键重试 | L2（服务端 L3 不依赖该头） |

## 其它类别

| 类别 | 级别 | 场景 |
|------|------|------|
| 安全 | L3 | 合成邮箱拒绝；证照不经业务进程；无 pending SSO |
| 可用性 | L2 | 失败节点 `data-traceId`；禁止 30s 轮询 |
| 一致性 | L2 | 提交后本页重载 vendor-status |

## 领域模型影响

不新开聚合。Vendor 仍按 `saas_user_id` / email 一致性边界。NFR 不要求改 schema。
