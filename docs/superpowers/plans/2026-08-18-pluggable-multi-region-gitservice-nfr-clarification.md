# NFR 澄清：可插拔多区域 gitService

- **日期**: 2026-08-18
- **默认档位**: L2（计费/开通路径抬升至 L3 一致性）

## 路径分片键强制审视

| 路径 | 分片 ID | 适配性 | 等级 | 动作 |
|------|---------|--------|------|------|
| GET/POST `/api/tenant/{tenant_id}/billing/gitlab-resources/?region=` | tenant_id + region | ✅ 合适（租户+区域） | L2 | 强制 region 查询参数 |
| GET `/api/tenant/{tenant_id}/billing/gitlab-regions/` | tenant_id | ✅ | L1 | 只读目录 |
| `/api/system-admin/gitlab-regions/` | 无（全局运营） | L0 合理 | L0 | 低频；升级触发：区域数>50 再分片 |
| GitLab Admin API `{gitlab_api_base}/api/v4/groups` | region→实例 | ✅ | L2 | 每实例独立 |
| 事件 `TenantGitlabResourcePurchased` key | tenant_id | ✅ | L2 | 建议 key=tenant_id |

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 幂等键 | 判定 | 等级 | 重放语义 / 动作 |
|------|--------|------------|--------|------|------|-----------------|
| POST `/api/tenant/{tenant_id}/billing/gitlab-resources/?region=` | 扣费+配额+开通 | 双击、超时重试 | `(tenant_id, region)` 唯一购区 + 请求 `Idempotency-Key` | 合适 | L3 | 已购则返回现有配额，禁止二次扣费 |
| GET `/api/tenant/{tenant_id}/billing/gitlab-regions/` | 无 | — | — | 只读 | L0 | 无副作用 |
| `/api/system-admin/gitlab-regions/` 写 | 登记区域配方 | 重试 | `region` slug 唯一 | 合适 | L2 | 同 slug 重入返回已有行 |
| GitLab Admin API 建 group | 远端建组 | 开通重试、pending 人工重试 | group path = 租户在该区域的稳定路径 | 合适 | L3 | 路径已存在视为成功；禁止换 path 重试 |
| 事件 `TenantGitlabResourcePurchased` 消费 | 开通/配额投影 | Kafka at-least-once | `event_id` 或 `tenant_id:region` | 合适 | L2 | **禁止** `user_id`/`company_id`；Seen 命中须可观测 |
| GET `/api/system-admin/gitlab-regions/` | 无 | — | — | 只读 | L0 | 无副作用 |

## 质量场景摘要

| 类别 | 档位 | 决策 |
|------|------|------|
| 可用性 | L2 | 单区域宕机不影响其他区域；边缘 502 告警 |
| 性能 | L1–L2 | SH 精简实例接受低吞吐；开通 API 超时→pending |
| 安全 | L3 | PAT 脱敏；OIDC 独立 client；DirectClient 无 env proxy |
| 一致性 | L3（计费） | 扣费与配额同事务；开通异步最终一致；购区幂等键 `(tenant_id, region)` |
| 可观测 | L2 | 结构化日志带 tenant_id/region/trace_id；禁 token |

## 对 DDD 影响

- Aggregate `TenantRegionQuota` 以 `(tenant_id, region)` 为一致性边界。
- 开通失败不回滚扣费（已购额度保留，状态 pending）——与设计 hybrid 一致。
- 购区命令与 `TenantGitlabResourcePurchased` 消费按 `(tenant_id, region)` / `event_id` 去重；实体用状态转移（`pending`→`active`），禁止对已购区域再次 `increment` 扣费。

### QS-幂等：同一区域重复购买

| 要素 | 内容 |
|------|------|
| 刺激 | 同一 `tenant_id` 对同一 `region` 连续两次 POST 购区（双击或超时重试） |
| 响应度量 | 扣费恰好 1 次；第二次返回已有配额（200/201 同体或明确已购），不新建第二笔订单 |
