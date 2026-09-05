# 测试意图：历史套餐管理

## 测试目标

验证超管可管理历史套餐（列表、结束在售、权限）。

## 测试分层

- Go：`TestEndPricingPackageSetsValidToAndRejectsRepeat`
- Django：`test_system_admin_ends_pricing_package` / `test_system_admin_end_pricing_package_requires_superuser`

## 用例矩阵

| # | 场景 | 期望 | 事件断言 |
|---|------|------|----------|
| T1 | 超管打开价格管理页 | 见「历史套餐管理」 | 无对应事件 |
| T2 | 列表含 GitLab 磁盘列 | 字段有值 | 无对应事件 |
| T3 | 结束在效期内套餐 | `valid_to` 已设 | 无对应事件 |
| T4 | 重复结束 | 400 | 无对应事件 |
| T5 | 非超管 end | 403 | 无对应事件 |

## 通过标准

上述 Go/Django 测例通过；公网 SPA 可见「历史套餐管理」与「结束在售」。
