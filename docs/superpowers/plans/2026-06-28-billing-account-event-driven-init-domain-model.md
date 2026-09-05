# Domain Model: BillingAccount 事件驱动初始化

> 输入:
> - 设计文档: `design.md`
> - 价值流: `docs/superpowers/plans/2026-06-28-billing-account-event-driven-init-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-28-billing-account-event-driven-init-nfr-clarification.md`
>
> NFR 关键约束:
> - 数据一致性 L3: BillingAccount 创建必须幂等 (tenant_id UNIQUE)
> - 安全性 L3: 审计追踪 (billing_outbox → BILLING_TRANSACTION_CREATED)
> - 可用性 L2: 降级策略 (账户不存在→503, 非500)
> - 容错 L2: 重试+死信 (5xx→retry, 4xx→DLQ)

## 限界上下文

```
┌─────────────────────────────────────────────────────────────┐
│                     task-collaboration                       │
│  (Django: Todo 创建时只读查询余额 → 扣费)                     │
│                                                              │
│  Entity:  Todo                                              │
│  Service: BillingService.get_account(tenant_id) → BillingAccount│
│  Event:   (consumer of) BillingAccountCreated               │
└──────────────────────┬──────────────────────────────────────┘
                       │ GET /accounts/{tenant_id} (只读)
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                        billing                               │
│  (taskBill Go: 计费账户管理 + 扣费)                           │
│                                                              │
│  Aggregate: BillingAccount (root)                            │
│    ├─ PricingPackage (ref by ID)                            │
│    └─ BillingUnit (ref by ID)                               │
│  Service:  AccountService.getOrCreate(tenant_id)             │
│  Service:  ChargeService.charge_task_post(...)               │
│  Event:    BillingAccountCreated (billing_outbox → Kafka)    │
│  Event:    TaskPostCharged (billing_outbox → Kafka)          │
└──────────────────────┬──────────────────────────────────────┘
                       │ POST /get-or-create-account/ (内部)
                       ▲
┌──────────────────────┴──────────────────────────────────────┐
│                      taskEvents                              │
│  (Go: 事件消费者，桥接 tenant → billing)                      │
│                                                              │
│  Handler: BillingInitHandler                                 │
│    consumes: COMPANY_CREATED                                 │
│    calls:    billing.AccountService.getOrCreate(company_id)  │
│  (无自有实体，纯编排服务)                                      │
└─────────────────────────────────────────────────────────────┘
```

## 聚合设计

### billing 上下文

#### 聚合根: BillingAccount

```
BillingAccount (聚合根)
├── id: int64 (Snowflake)
├── tenant_id: int64 (UNIQUE — 幂等键)
├── balance: int64 (余额，积分)
├── pricing_package_id: int64 (→ PricingPackage)
├── locked_post_creation_points: int64
├── locked_server_start_points: int64
├── excluded_pricing_package_ids: []string
├── created_at: time.Time
├── updated_at: time.Time
│
├── Entity: PricingPackage (独立聚合，ID 引用)
│   ├── id, package_number, valid_from, valid_to
│   ├── normal_task_points, programming_task_points
│   └── normal/programming_renewal_points_per_month
│
└── Entity: BillingUnit (独立聚合，ID 引用)
    ├── id, unit_type (post_creation|server_start)
    ├── name, price, unit, is_active
    └── created_at, updated_at
```

**一致性边界**: BillingAccount 内部字段强一致（SQLite 单行事务）。PricingPackage 和 BillingUnit 为外部引用，通过 ID 关联，各自独立演化。

**不变条件**:
- `balance >= 0`（不允许透支）
- `tenant_id` 唯一（一个租户一个账户）
- `locked_post_creation_points > 0`

### task-collaboration 上下文

Todo 创建时通过 `BillingService.get_account(tenant_id)` 获取只读 BillingAccount 快照。Todo 本身是独立聚合根（已有），BillingAccount 是其外部依赖。

## 领域服务

### BillingService (Python — task-collaboration 上下文)

```python
# billing_bridge/domain/services.py

class BillingService:
    """计费账户查询服务（只读）。创建由 taskEvents 消费者负责。"""

    def __init__(self, account_repo: BillingAccountRepository):
        self._repo = account_repo

    def get_account(self, tenant_id: str) -> BillingAccount:
        """只读查询计费账户。账户不存在时抛 AccountNotFoundError。"""
        ...

    def check_balance(self, account: BillingAccount, cost: int) -> None:
        """余额不足时抛 InsufficientBalanceError。"""
        ...
```

### AccountService (Go — billing 上下文，已有)

```go
// taskBill/src/tenant_pricing.go (现有，添加只读方法)

func GetBillingAccount(tenantID int64) (*BillingAccount, error)  // NEW: 只读
func GetOrCreateBillingAccount(tenantID int64, forUpdate bool) (*BillingAccount, bool, error)  // 现有
```

### BillingInitHandler (Go — taskEvents 上下文)

```go
// taskEvents/internal/handlers/billinginit/handler.go (NEW)

type Handler struct {
    TaskBillBaseURL string
    InternalSecret  string
    HTTP            *http.Client
}
func (h *Handler) Dispatch(ctx context.Context, cmd DomainCommand) (DispatchOutcome, error)
```

## 仓储接口

### BillingAccountRepository (Python)

```python
# billing_bridge/domain/repositories.py
from abc import ABC, abstractmethod

class BillingAccountRepository(ABC):
    """BillingAccount 只读仓储。实现通过 HTTP 调用 taskBill GET 端点。"""

    @abstractmethod
    def find_by_tenant_id(self, tenant_id: str) -> 'BillingAccount':
        """查询账户。不存在时抛 AccountNotFoundError。"""
        ...

    @abstractmethod
    def get_task_post_charge_points(self, tenant_id: str) -> int:
        """查询任务帖创建消耗积分。"""
        ...
```

## 领域事件

| 事件 | 发布方 | 消费方 | 触发时机 |
|------|--------|--------|---------|
| `COMPANY_CREATED` | taskEvents `0_create_company` | `4_init_billing_account` | 公司创建后 |
| `BillingAccountCreated` | taskBill (billing_outbox → Kafka) | 审计/通知 | getOrCreate 首次创建 |
| `BILLING_TRANSACTION_CREATED` | taskBill (billing_outbox → Kafka) | taskEvents `1_process_billing_transaction` | 扣费/充值后 |

### BillingAccountCreated 事件契约

```python
# billing_bridge/domain/events.py
from dataclasses import dataclass
from datetime import datetime

@dataclass(frozen=True)
class BillingAccountCreated:
    account_id: str
    tenant_id: str
    balance: int
    pricing_package_id: str
    locked_post_creation_points: int
    locked_server_start_points: int
    occurred_at: datetime
```

## Python 领域模型文件

### 实体

```python
# billing_bridge/domain/entities.py

class BillingAccount:
    """计费账户实体（只读快照）。真源在 taskBill Go 服务。"""

    def __init__(
        self,
        id: str,
        tenant_id: str,
        balance: int,
        pricing_package_id: str = "0",
        locked_post_creation_points: int = 3,
        locked_server_start_points: int = 30,
    ):
        self.id = id
        self.tenant_id = tenant_id
        self.balance = balance
        self.pricing_package_id = pricing_package_id
        self.locked_post_creation_points = locked_post_creation_points
        self.locked_server_start_points = locked_server_start_points

    def has_sufficient_balance(self, cost: int) -> bool:
        return self.balance >= cost

    def can_create_todo(self) -> bool:
        return self.balance >= self.locked_post_creation_points
```

## Go 领域类型（参考）

现有类型在 `taskBill/src/tenant_pricing.go`:
- `PricingPackage` struct — 已有
- `BillingAccount` struct — 已有
- `BillingUnit` — 隐式（`billing_unit` 表行）

新增类型:
- `ErrAccountNotFound` — 只读查询错误哨兵

---

## 自检

- [x] 限界上下文已识别: billing, taskEvents, task-collaboration
- [x] 聚合根已定义: BillingAccount
- [x] 聚合间通过 ID 引用 (PricingPackage, BillingUnit)
- [x] 领域服务已定义: BillingService, AccountService, BillingInitHandler
- [x] 仓储接口为 ABC 抽象 (Python)
- [x] 领域事件已识别: COMPANY_CREATED, BillingAccountCreated, BILLING_TRANSACTION_CREATED
- [x] NFR 决策已体现在模型中: 幂等(tenant_id UNIQUE), 审计(billing_outbox), 降级(AccountNotFoundError)
