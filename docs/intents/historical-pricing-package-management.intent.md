# 意图：历史套餐管理

## 背景与目标

超管在 `/system-admin/price-management` 管理全部历史价格套餐：查看列表（含 GitLab 磁盘价目）、结束在售、查看订户。

## 范围与边界

- 含：列表、结束在售（`valid_to`）、订户查看、GitLab 磁盘字段透传。
- 不含：改已创建套餐单价、删除套餐、租户换绑。

## 约束与风险

- 结束在售不影响已开户锁价。
- 仅 `is_superuser`。

## 验收标准

见设计文档 `docs/superpowers/specs/2026-07-16-historical-pricing-package-management-design.md` S1–S7。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | MQ类型/契约 |
|---------|--------|--------|--------|------------|
| — | — | — | — | — |

**无对应事件**：价目窗口配置写操作，与既有「创建套餐」一致，无新聚合状态跃迁；已开户锁价不变，扣费仍走既有用量事件。

## 实施计划

1. taskBill `POST .../pricing-packages/{id}/end/`
2. Django bridge + stub + 超管 API
3. 前端「历史套餐管理」区块
4. 测试与 collectstatic
