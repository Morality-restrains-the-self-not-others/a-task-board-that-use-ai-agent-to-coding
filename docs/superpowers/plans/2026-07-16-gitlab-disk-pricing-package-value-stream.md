# GitLab 磁盘价格套餐 — 价值流

## 主价值流（超管配置）

```
超管打开价格管理 → 填写 GitLab 磁盘(积分/GB/月) → 创建套餐
  → taskBill 落库套餐列 + 同步 billing_unit.gitlab_disk 价
  → 列表展示新列 → 新开户/换套餐锁价
  → 公开定价页与租户账单展示单价
```

## 最小可行增量（本迭代）

1. 价目可配置、可展示、可锁价（完成）。
2. 用量计量与月结扣费（后续）。

## 测试点（映射用例）

| ID | 测试点 | 用例落点 |
|----|--------|----------|
| TP-GD-01 | 创建套餐可带 gitlab_disk 字段并回传 | Django `test_product_pricing` / Go admin_pricing |
| TP-GD-02 | 缺省时默认 1 | Go createPricingPackage |
| TP-GD-03 | 换套餐更新 locked_gitlab_disk | Go switch 测试 |
| TP-GD-04 | 非超管 403 | 既有权限测 |
| TP-GD-05 | 用户描述含 GitLab 磁盘行 | description 测试 |
