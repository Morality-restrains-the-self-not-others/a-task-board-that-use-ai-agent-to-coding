# DDD 领域模型: 跨公司数据隔离 — TenantMembershipResolver

> 输入:
> - 价值流文档: `docs/superpowers/plans/2026-06-28-cross-company-data-isolation-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-28-cross-company-data-isolation-nfr-clarification.md`
>
> 输出使用者: `/6-plans-实施计划`, `/7-build-构建`

## 领域建模增量

### 类型: Bug Fix — 现有领域约束的强制执行

此修复不引入新的限界上下文或实体。它增强现有 `accounts` 限界上下文的领域服务层，
将已有的 `CompanyMemberRepository.find_by_user_and_company(user_id, company_id)` 契约
强制执行到所有视图层调用点。

### 领域模型文件

#### 新增: `accounts/domain/services/tenant_membership_resolver.py`

```python
class TenantMembershipResolver:
    """根据 URL tenant_id 解析用户在该租户下的 CompanyMember"""
    def resolve(self, user_id: str, company_id: int) -> CompanyMember | None
```

**职责:** 从 URL 的 `tenant_id` 映射到 `company_id`，通过 `CompanyMemberRepository.find_by_user_and_company()` 确定性解析成员身份。

**与现有 `CompanyAdminPolicy` 的分工:**
- `TenantMembershipResolver`: "该用户属于这个租户吗？"（成员身份解析）
- `CompanyAdminPolicy`: "该用户能管理这个公司吗？"（权限判定）

### 上下文映射

```
URL tenant_id (string)
    │
    ▼
views 层 — workspace_access_views / group_views / member_views
    │
    │ tenant_id 原样传入 (之前被忽略)
    ▼
TenantMembershipResolver.resolve(user_id, company_id=int(tenant_id))
    │
    │ 调用领域仓储 (强制 company_id 过滤)
    ▼
CompanyMemberRepository.find_by_user_and_company(user_id, company_id)
    │
    │ 基础设施层实现 → CompanyMember.objects.filter(user_id=..., company_id=...).first()
    ▼
确定性 CompanyMember (属于 URL 指定的公司)
```

### 聚合边界

```
Company (聚合根)
├── CompanyMember.user_id, company_id, is_admin, role
│   └── TenantMembershipResolver: 通过 (user_id, company_id) 联合键解析
├── CompanyGroup.company_id
│   └── 分组查询必须经 company_id 过滤
└── Workspace.company_id (跨聚合引用，通过 company_id 校验)
    └── WorkspaceAccess.workspace_id, user_id
```

**关键约束:** 所有涉及 `company_id` 的查询必须显式过滤；禁止裸 `user_id` 单条件 `.first()`。

## 自检清单

- [x] 所有实体/服务文件位于 `domain/` 目录下 (`accounts/domain/services/`)
- [x] 领域层无任何 ORM 导入 (`TenantMembershipResolver` 仅导入 Repository ABC)
- [x] 领域层无任何外部服务导入
- [x] 仓储接口使用 ABC 抽象 (已有 `CompanyMemberRepository`)
- [x] 实体 ID 使用雪花算法 (现有实体不变)
- [x] 领域事件无需新增 (已有合规覆盖)
- [x] 每个聚合根有对应的仓储接口 (已有)
- [x] 领域服务通过构造函数注入依赖 (`member_repository: CompanyMemberRepository`)
- [x] 无数据库字段声明

## 领域模型影响汇总

| NFR 决策 | 模型实现 |
|----------|---------|
| 安全性 L4: 多租户隔离 | `TenantMembershipResolver.resolve()` 强制 `company_id` 参数；不接受无 tenant 上下文的查询 |
| 安全性 L4: URL tenant_id 为真源 | 调用方负责提取 URL tenant_id → int → 传入 resolver |
| 数据一致性 L2: 确定性查询 | `find_by_user_and_company(user_id, company_id)` 联合键查询，非 `.first()` 无排序 |
| 可维护性 L2: 统一工具函数 | 单一切入点 `TenantMembershipResolver`，禁止视图层直接 ORM |

## 基础设施层适配

视图层 (`workspace_context.py`) 提供便捷函数接入领域服务:

```python
# accounts/workspace_context.py (增强)
from accounts.domain.services import TenantMembershipResolver
from accounts.infrastructure.repositories import DjangoCompanyMemberRepository

def resolve_company_member_for_tenant(user_id: str, tenant_id: str | None) -> CompanyMember | None:
    resolver = TenantMembershipResolver(DjangoCompanyMemberRepository())
    return resolver.resolve(user_id, int(tenant_id))
```
