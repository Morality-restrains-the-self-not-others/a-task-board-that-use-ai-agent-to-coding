# Value Stream: BillingAccount 事件驱动初始化

> Derived from design: `design.md` (2026-06-28 — Todo 创建时 taskBill get_or_create_account 失败修复)

## Value Summary

新租户注册后 BillingAccount 由事件驱动自动初始化，创建 Todo 时只需只读查询余额，消除懒加载的 500 错误。

## Related Value Streams

- **[domain-events-consumer-split](2026-05-28-domain-events-consumer-split-value-stream.md)**: extension — 在 `company_created` intent 组中新增 `4_init_billing_account`（port 18039），复用 v4 intent 消费者框架
- **[task-management](conf/value-stream.yaml#task-management)**: modification — Todo 创建步骤中 `get_or_create_billing_account` 改为 `get_billing_account`（只读），移除懒初始化

## End-to-End Flow

```
用户注册 → USER_CREATED → 0_create_company (已有) → COMPANY_CREATED
  → [NEW] 4_init_billing_account → POST taskBill → BillingAccount 就绪
  → 用户创建 Todo → GET billing account (只读) → 余额检查 → 扣费 ✅
```

## Value Increments

### Increment 1: 修复懒初始化 500 错误（Thin Slice）

**Value to user:** 新租户可以成功创建第一个 Todo，不再报 500 错误。

**Scope:**
- taskBill 种子数据迁移（`billing_pricing_package` + `billing_unit` 默认行）
- taskBill Go 空表回退逻辑修复（移除不可达 `else` 分支）

**Depends on:** nothing

**Test:** `curl` 直接调用 taskBill `get-or-create-account` 验证返回 200；前端创建 Todo E2E 验证。

### Increment 2: 事件驱动 BillingAccount 初始化

**Value to user:** BillingAccount 在注册时自动就绪，创建 Todo 时无需等待懒初始化，计费链路更可靠。

**Scope:**
- `taskEvents/internal/handlers/billinginit/handler.go` — COMPANY_CREATED 消费者
- `taskEvents/cmd/company_created/4_init_billing_account/main.go` — intent 入口
- `taskEvents/config/intent_registry.go` — 注册 port 18039
- `taskEvents/config/config.go` — TaskBillBaseURL/TaskBillInternalSecret 配置
- `taskEvents/run.sh` — INTENT_PATHS/INTENT_PORTS 更新
- taskBill `GET /api/internal/taskbill/accounts/{tenant_id}` 只读端点
- taskBill `getBillingAccount(tenantID)` 只读函数

**Depends on:** Increment 1（种子数据确保 taskBill 能正常创建账户）

**Test:** taskEvents integration test（发布 COMPANY_CREATED → 验证 consumer 调用 taskBill → billing_account 表有记录）

### Increment 3: Python 侧读写分离 + 诊断改进

**Value to user:** 错误信息清晰可诊断，架构职责分离（Create 归事件消费者，Read 归 Django）。

**Scope:**
- `client.py`: `get_or_create_account` → 新增 `get_account`（GET）、`get_account_points`
- `task_post_billing.py`: `get_or_create_billing_account` → `get_billing_account`（只读）
- `todo_views.py`: 调用改为 `get_billing_account`，移除 `select_for_update`
- `grafana_errors/service.py`: 同步更新调用
- `client.py`: `requests` → `HTTPClient`（合规）
- 错误信息包含 status/tenant_id/error_detail

**Depends on:** Increment 2（taskBill GET 端点已就绪）

**Test:** 现有 `TodoViewSet_test.py` 回归通过；新增 billing_bridge 单测覆盖只读路径和错误信息格式。
