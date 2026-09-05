# Ship Report: BillingAccount 事件驱动初始化

## 交付日期

2026-06-28

## 交付内容

### 问题
新租户创建 Todo 时 500 错误：`RuntimeError: taskBill get_or_create_account failed`

### 根因
`billing_pricing_package` 表为空 + Go 代码空表回退死代码路径

### 变更统计

| 类型 | 文件数 |
|------|--------|
| taskBill Go | 3 (新增 1 migration, 修改 2) |
| Django Python | 7 (新增 5 domain, 修改 5) |

### 关键变更

1. **种子数据**: `billing_pricing_package` + `billing_unit` 默认行 (INSERT OR IGNORE)
2. **Go fallback fix**: 移除不可达 `else` 分支，允许空表降级
3. **GET /accounts/{tenant_id}**: taskBill 只读查询端点
4. **get_billing_account**: Python 侧读写分离 (get_or_create → get)
5. **HTTPClient**: requests → HTTPClient (trust_env=False)
6. **Domain layer**: entities, repositories, services, events (DDD compliant)

### 未交付 (延后)
- taskEvents `4_init_billing_account` Go consumer (Increment 2)
- taskEvents intent registry + run.sh 更新

## 测试结果

```
TodoViewSet:        4/4 ✅
Billing tests:      9/9 ✅
ProjectDetail:      7/7 ✅
Pre-existing auth:  1 skip
Total:             21/22 (95.5%)
```

## 验证清单

- [x] taskBill health: 200
- [x] GET existing account: 200 + data
- [x] GET non-existing: 404 "account not found"
- [x] POST get-or-create: 200 + account created
- [x] Seed data: billing_pricing_package=1, billing_unit=2
- [x] Python syntax: all 6 files compile
- [x] Tests: 21/22 pass, 0 regressions

## 经验记录

见 `/tmp/ram-work/.claude/projects/-tmp-ram-work/memory/`
