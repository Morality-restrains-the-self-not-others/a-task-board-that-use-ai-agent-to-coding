# 测试意图：GitLab 磁盘价格套餐项

| ID | 场景 | 期望 |
|----|------|------|
| TI-GD-01 | 超管创建套餐带 gitlab_disk=5 | 200，回传 5 |
| TI-GD-02 | 创建时省略该字段 | 落库默认 1 |
| TI-GD-03 | 列表项含字段 | GET items 含键 |
| TI-GD-04 | 公开定价含字段 | public API |
| TI-GD-05 | 换套餐更新锁价 | account locked = 新套餐值 |
| TI-GD-06 | 非超管 POST | 403 |
