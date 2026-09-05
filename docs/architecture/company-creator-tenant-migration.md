# accounts_company：全量迁入 taskTenantService

- 日期：2026-07-20
- 状态：**已落地**（saas 表已 DROP；Django 为 unmanaged 门面）

## 所有权

| 项 | Owner |
|----|--------|
| `accounts_company`（id/name/creator_id） | **taskTenantService**（`data/task_tenant.db`） |
| Django `accounts.models.Company` | unmanaged 门面 → `company_tenant_api` |
| saas `accounts_company` | **已删除**（子表保留 `company_id` 列；`db_constraint=False` + SQLite 无 `REFERENCES`） |

## Internal API（taskTenantService）

| Method | Path |
|--------|------|
| GET | `/api/internal/tenant/companies/by-id?company_id=` |
| GET | `/api/internal/tenant/companies/by-name?name=` |
| GET | `/api/internal/tenant/companies/by-creator?creator_id=` |
| GET | `/api/internal/tenant/companies/search?q=&limit=` |
| GET | `/api/internal/tenant/companies/name-taken?name=&exclude_id=` |
| GET | `/api/internal/tenant/companies/creator?company_id=` |
| POST | `/api/internal/tenant/companies/batch` `{"ids":[...]}` |
| POST | `/api/internal/tenant/companies/upsert` |
| POST | `/api/internal/tenant/companies/import` |

## 迁移脚本

```bash
bash db/scripts/migrate_companies_to_task_tenant.sh
bash db/scripts/drop_accounts_company_from_saas.sh   # 内含 strip FK REFERENCES
# 或单独：
python3 db/scripts/strip_saas_accounts_company_fk.py
```

Django 迁移：`accounts.0042` / `cloud.0053` / `projects.0062` / `subscriptions.0002`（state：`db_constraint=False`；DB：strip 脚本）。

## 调用约定

- 业务代码可继续 `Company.objects.get/filter/create/save`（门面转发 tenant）
- 新代码优先 `accounts.company_tenant_api`
- 禁止再向 saas 写 `accounts_company`
- 子表 `company_id` 为逻辑引用，不在 saas 建 FK
