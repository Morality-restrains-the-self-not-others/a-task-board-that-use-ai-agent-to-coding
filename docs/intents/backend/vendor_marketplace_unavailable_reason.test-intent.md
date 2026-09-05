# 测试意图：厂商门户版本「不可用」须展示具体原因

## 测试目标

验证厂商容器镜像列表返回市场可用性字段，且前端徽章文案包含具体原因。

## 测试分层

| 层 | 覆盖 |
|----|------|
| Go 接口 | `GET /api/vendor/container-images/` 含 `is_marketplace_available` / `unavailable_reason` |
| 前端 unit | `formatUnavailableBadge` / `isMarketplaceUnavailable` |

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 新建草稿镜像、无区域关联 | API：`is_marketplace_available=false`，`unavailable_reason` 含「未设置任何区域运行环境」 |
| T2 | 原因非空 | `formatUnavailableBadge` → `不可用：{原因}` |
| T3 | 原因为空 | `formatUnavailableBadge` → `不可用` |
| T4 | `is_marketplace_available` 缺失/true | 不视为不可用（`isMarketplaceUnavailable` 为 false） |
| T5 | `is_marketplace_available === false` | `isMarketplaceUnavailable` 为 true |

## 数据与环境

- Go：`handlers_test.go` 内 sqlite 测试库（与既有 vendor list 测同模式）。
- 前端：`node:test`，`taskAiProvider/frontend/tests/`。

## 通过标准

- 上述测例全部通过。
- 厂商门户展开版本后，不可用徽章可见文本含具体原因（非仅 title）。

## 意图 → 事件投递

本意图无对应业务事件（只读）；无需断言 MQ 投递。
