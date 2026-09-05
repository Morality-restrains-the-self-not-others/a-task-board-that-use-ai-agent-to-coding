# 测试意图：任务帖配额拆分赠送与购买剩余

## 测试目标

证明 quotas API 在混合赠送/购买、仅购买、过期赠送三种数据下返回正确的合计与拆分字段。

## 测试分层

- Go 单元：`taskBill/src/task_post_quota_source_test.go`
- OpenAPI：`taskBill/src/openapi.yaml`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 赠送 10 + 购买 5 | gifted=10，purchased=5，total=15 |
| T2 | 消耗优先扣赠送再扣购买 | 流水 related_order_id 指向对应订单 |
| T3 | 历史已支付订单回填 purchase lot | 幂等，remaining 不超过账户购买剩余 |

## 数据与环境

`setupMySQLTestDB`；独立 tenant_id。

## 通过标准

T1–T3 全绿；既有 `TestHandleResourceQuotasMultiRegion` 不回归。

## 业务意图 → 事件对照（测试）

纯查询，不断言 MQ 投递。
