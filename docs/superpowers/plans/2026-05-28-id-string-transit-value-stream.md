# Value Stream: ID 字符串传输规范

## 增量 1 — taskBill 边界（必做）

- 入站：`parseIDField` 拒绝 float64 ID
- 出站：`formatID` 统一字符串
- 验证：`get-or-create-account` 大 ID 回归

## 增量 2 — billing_bridge 调用方（必做）

- `_json_safe_payload` 覆盖 `*_id` / `id` 字段
- `charge_*` / `credit_recharge` 一律 `str(tenant_id)`
