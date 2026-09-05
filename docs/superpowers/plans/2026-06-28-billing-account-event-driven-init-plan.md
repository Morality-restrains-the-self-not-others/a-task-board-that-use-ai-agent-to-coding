# Implementation Plan: BillingAccount 事件驱动初始化

> 输入:
> - 设计文档: `design.md`
> - 价值流: `2026-06-28-billing-account-event-driven-init-value-stream.md`
> - NFR: `2026-06-28-billing-account-event-driven-init-nfr-clarification.md`
> - 领域模型: `2026-06-28-billing-account-event-driven-init-domain-model.md`

## 计划总览

| Increment | 名称 | 任务数 | 预估 |
|-----------|------|--------|------|
| 1 | 修复懒初始化 500 (Thin Slice) | 4 | taskBill Go |
| 2 | 事件驱动 BillingAccount 初始化 | 8 | taskEvents Go + taskBill Go |
| 3 | Python 读写分离 + 诊断 | 7 | Django Python |

---

## Increment 1: 修复懒初始化 500 错误

### 1.1 taskBill 种子数据迁移

- [ ] **T1.1.1** 创建 `taskBill/migrations/002_seed_default_pricing.sql`
  - `INSERT OR IGNORE` 默认定价套餐: package_number=0, 3/30 积分, 永久有效
  - `INSERT OR IGNORE` billing_unit: post_creation(3分), server_start(30分)
  - **验证**: `sqlite3 billing.sqlite3 "SELECT COUNT(*) FROM billing_pricing_package"` → 1

- [ ] **T1.1.2** 重启 taskBill 触发迁移
  - `cd taskBill && bash run.sh build && bash run.sh start`
  - **验证**: `curl http://127.0.0.1:8004/api/health/` → 200

### 1.2 taskBill 空表回退逻辑修复

- [ ] **T1.2.1** 修改 `taskBill/src/tenant_pricing.go:202-208`
  - 移除 `else { return nil, false, errors.New("no pricing package available") }`
  - 保留 `if pkg, _ := resolvePackageForListPrice(...); pkg != nil { pkgID = pkg.ID }`
  - **验证**: `go build ./...` 编译通过

- [ ] **T1.2.2** 验证修复: 用新 tenant_id 调 taskBill
  - `curl -X POST .../get-or-create-account/ -d '{"tenant_id":"999999999999999999"}'` → 200
  - 删除种子数据后再次测试 → 仍返回 200（降级到硬编码默认值）

---

## Increment 2: 事件驱动 BillingAccount 初始化

### 2.1 taskBill 只读查询端点

- [ ] **T2.1.1** 新增 `getBillingAccount(tenantID int64)` 函数
  - 文件: `taskBill/src/tenant_pricing.go`
  - 纯查询，`sql.ErrNoRows` → `ErrAccountNotFound`
  - **验证**: 已有 tenant → 返回账户; 不存在 tenant → 返回 error

- [ ] **T2.1.2** 新增 `GET /api/internal/taskbill/accounts/{tenant_id}` handler
  - 文件: `taskBill/src/handlers.go`
  - 校验 `X-TaskBill-Internal-Secret` (NFR 安全性 L3)
  - 返回 200 + account JSON / 404 + error
  - **验证**: `curl http://127.0.0.1:8004/api/internal/taskbill/accounts/{existing_id}/` → 200

### 2.2 taskEvents BillingInitHandler

- [ ] **T2.2.1** 创建 `taskEvents/internal/handlers/billinginit/handler.go`
  - 消费 `COMPANY_CREATED` 事件
  - 解析 `company_id` → 作为 `tenant_id`
  - `POST` 到 taskBill `/api/internal/taskbill/get-or-create-account/`
  - 携带 `X-TaskBill-Internal-Secret` 头
  - 返回规则: 2xx→Success, 5xx→Retryable, 4xx→Permanent
  - **验证**: 单元测试 mock taskBill HTTP

- [ ] **T2.2.2** 创建 `taskEvents/cmd/company_created/4_init_billing_account/main.go`
  - 加载配置 → 创建 handler → `eventbin.RunIntent(...)`
  - **验证**: `go build` 产出 binary

### 2.3 taskEvents 注册与配置

- [ ] **T2.3.1** 更新 `taskEvents/config/intent_registry.go`
  - 新增: `{EventSlug: "company_created", IntentSlug: "4_init_billing_account", ..., Port: 18039, DefaultEnabled: true}`
  - **验证**: `go test ./config/...`

- [ ] **T2.3.2** 更新 `taskEvents/config/config.go`
  - 添加 `TaskBillBaseURL`、`TaskBillInternalSecret` 字段
  - 从 YAML 配置或环境变量加载
  - **验证**: config 结构体包含新字段

- [ ] **T2.3.3** 更新 `taskEvents/run.sh`
  - `INTENT_PATHS` 新增 `company_created/4_init_billing_account`
  - `INTENT_PORTS` 新增 `18039`
  - **验证**: `bash run.sh build user_created` (验证脚本语法)

- [ ] **T2.3.4** 端到端验证
  - 启动 `4_init_billing_account` consumer
  - 发布 COMPANY_CREATED 事件 → consumer 消费 → taskBill 创建账户
  - **验证**: `sqlite3 billing.sqlite3 "SELECT * FROM billing_account"` → 新行

---

## Increment 3: Python 读写分离 + 诊断改进

### 3.1 client.py 改造

- [ ] **T3.1.1** 新增 `get_account(tenant_id)` 函数
  - 文件: `Saas_project/billing_bridge/client.py`
  - `GET /api/internal/taskbill/accounts/{tenant_id}/` + `X-TaskBill-Internal-Secret`
  - 返回 `(status_code, data)`
  - **验证**: 单测 mock HTTP → 200 返回 account dict

- [ ] **T3.1.2** 新增 `get_account_points(tenant_id)` 函数
  - 返回 `(balance, locked_post_creation_points)`
  - 4xx/5xx → `RuntimeError` 含 status/tenant_id/error
  - **验证**: 404 → 错误信息包含诊断三段信息

- [ ] **T3.1.3** 替换 `requests` → `HTTPClient`
  - `forward_to_taskbill` 改用 `core.http_client.HTTPClient`
  - `trust_env=False` 防止代理劫持 (NFR 安全性 L3)
  - **验证**: 现有测试通过 (`conftest.py` 已 monkeypatch `forward_to_taskbill`)

### 3.2 task_post_billing.py 重命名

- [ ] **T3.2.1** `get_or_create_billing_account` → `get_billing_account`
  - 移除 `select_for_update` 参数
  - 改为调用 `get_account()` (只读 GET)
  - 错误信息包含 status/tenant_id/error_detail 三段
  - 返回值从 `(account, created)` 改为 `account`
  - **验证**: 调用方不再传递 `select_for_update`

### 3.3 调用方更新

- [ ] **T3.3.1** 更新 `projects/views/todo_views.py`
  - `get_or_create_billing_account(workspace.company_id, select_for_update=True)` → `get_billing_account(workspace.company_id)`
  - 捕获 `AccountNotFoundError` → 返回 503 "账户正在初始化，请稍后重试"
  - **验证**: `TodoViewSet_test.py` 全部通过

- [ ] **T3.3.2** 更新 `core/grafana_errors/service.py`
  - 同步改为 `get_billing_account(tenant_id)`
  - **验证**: 相关测试通过

### 3.4 回归测试

- [ ] **T3.4.1** 运行全量测试套件
  - `cd Saas_project && python -m pytest projects/view_test/TodoViewSet_test.py -x`
  - `python -m pytest tests/ -k "billing" -x`
  - **验证**: 全部通过，无 regression

---

## 测试契约（来自 NFR 质量场景）

| QS | 测试方式 | 验收标准 |
|----|---------|---------|
| QS-01 (GET P95≤50ms) | taskBill 单测 benchmark | 主键查询 < 10ms |
| QS-02 (幂等创建) | taskEvents integration test | 同一 tenant_id 重复消费仅一条记录 |
| QS-03 (taskBill 不可达重试) | mock HTTP 返回 connection refused | handler 返回 DispatchRetryable |
| QS-04 (账户未就绪→503) | TodoViewSet test | 404 → 503 + 友好消息 |
| QS-05 (内部密钥校验) | curl 不带/带错误 secret | 403; 正确 secret → 200 |

---

## 依赖关系

```
T1.1 (种子迁移)
  ├── T1.2 (Go fallback fix) ─────────────────────┐
  │                                                 ├── T3.1 (client.py GET account)
  └── T2.1 (taskBill GET endpoint) ────────────────┤       │
       └── T2.2 (BillingInitHandler)               │       ├── T3.2 (rename)
            └── T2.3 (intent registry/run.sh)      │       │       │
                 └── T2.4 (E2E verify)             │       │       ├── T3.3 (views update)
                                                    │       │       │       │
                                                    └───────┴───────┴───────┴── T3.4 (regression)
```

Increment 1 和 Increment 2 可并行（不同服务），Increment 3 依赖两者完成。
