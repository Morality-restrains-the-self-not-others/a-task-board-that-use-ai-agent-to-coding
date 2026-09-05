# NFR 澄清 — 超管用户页租户 Tab

- **日期**: 2026-08-25
- **价值流**: `docs/superpowers/plans/2026-08-25-system-admin-users-tenants-tab-value-stream.md`
- **默认等级**: L2；安全 L3

## 路径分片键审视

| 路径 | 分片 ID | 判定 | 可伸缩性等级 | 结论 / 动作 |
|------|---------|------|-------------|-------------|
| GET `/api/system-admin/accounts/admin/tenants/` | 无 | 缺键 | **L0** | 平台超管全局租户目录；分页 ≤100；操作人数极少。升级触发：`tenant_company` > 1 万且空搜索 P95 > 500ms 时加覆盖索引 `(name, id)`，仍不按 tenant 切（本路径本就是跨租户运营） |
| FE `/system-admin/users/?tab=tenants` | 无 | 缺键 | **L0** | 管理后台单页。升级触发：同 API |
| 内部 `batch/details`、`/api/internal/users/?q=` | `user_ids` / `q` | 非租户分片键 | L2 | 已有上限；fail-open |

无分片 ID 标 L0 的理由：超管低频跨租户只读目录 + 已分页 + 写明升级触发。

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| GET tenants 列表 | 无 | 刷新/翻页/搜索 | — | — | **L0** 同一快照 |
| 出站 Auth 查询 | 无（只读） | 列表重试 | — | — | L0 |
| FE 点 Tab / 搜索 / 刷新 | 无 | 连点 | — | — | Anti-Replay-OK: 只读 GET，不生成 Idempotency-Key |

无 HTTP 写、无 Kafka、无 Webhook、无 timer。禁止用 `tenant_id`/`user_id` 作消费键 — 本增量无消费者。

## 类别定级

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L0 | 见上 |
| 数据一致性 | L1 | 联系方式短窗口最终一致可接受 |
| 安全 | L3 | 仅平台员工 |
| 可用性 | L2 | Auth 失败不阻断公司列表 |
| 性能 | L2 | 默认 50 条；搜索合并上限 500 |
| 可观测性 | L2 | info 列出 count/limit/offset；禁 PII |

## 质量场景

1. 刺激：超管打开租户 Tab。响应：看到公司表格或空态。
2. 刺激：非平台员工 GET。响应：403。
3. 刺激：taskAuth 超时。响应：200，email/phone 空串。
4. 刺激：limit=50 且 total>50。响应：可翻到下一页。

## 领域模型影响

只读列表 DTO，不改变 Company 聚合写边界。无需纠正分片键。
