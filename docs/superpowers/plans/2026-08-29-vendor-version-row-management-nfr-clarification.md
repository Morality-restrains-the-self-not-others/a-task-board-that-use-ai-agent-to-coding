# NFR：厂商版本行管理

- **Date:** 2026-08-29
- **Default level:** L2；删除/下架 L2（非资金，但是目录写路径）

## 质量场景

- 可用性：操作列在常见桌面宽度第一行可见。
- 安全：仅 owner；激活版本不可删。
- 可观测：失败 `data-traceId`；成功发领域事件（LogEventBus）。

## 路径分片键强制审视

| 路径 | 方法 | 是否有分片 ID | 该 ID 是否适合作分片键 | 可伸缩性 | 动作 |
|------|------|---------------|------------------------|----------|------|
| `/api/vendor/container-images/{id}/` | DELETE | 是 `image_id` | 是（镜像聚合根） | L1 | 保持按 id 路由；列表已按 vendor_id |
| `/api/vendor/container-images/{id}/withdraw/` | POST | 是 `image_id` | 是 | L1 | 同上 |
| `/api/vendor/container-images/` | GET | 隐式 vendor_id（JWT） | vendor_id 适合租户膨胀 | L1 | 不改列表契约 |
| 厂商门户 SPA 版本行 | UI | 行内 image id | n/a | L0 | 纯展示 |

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 | 级别 |
|------|--------|------------|--------------|--------|----------|------|
| DELETE `/{id}/` | 删行 | 双击、超时重试 | 同一 `image_id` | `image_id` + Idempotency-Key（前端） | 已删 → 404（owner 查询不到）；激活拒绝 400 | L2 |
| POST `/{id}/withdraw/` | status→draft, is_active=0 | 双击 | 同一 `image_id` 下架意图 | `image_id` | 已是 draft → 200 空操作 | L2 |
| GET 列表 | 无 | — | — | — | L0 只读 | L0 |

资金/云资源：否。前端 `createClickGuard` 防双击。
