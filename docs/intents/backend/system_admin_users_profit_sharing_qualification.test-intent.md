# 测试意图：超管用户列表分账资格字段

## 测试目标

验证列表 DTO 正确投影当前分账资格，且下游失败不阻断列表。

## 测试分层

- taskReferral：批量 IN 查询与 HTTP 契约
- taskAuth：列表回填与 best-effort

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 两名用户：一人 approved 未过期，一人无码 | batch 返回 true/false |
| T2 | 已取消 / 已过期 | false |
| T3 | 空 user_ids | 200 空 map |
| T4 | 缺内部密钥 | 403 |
| T5 | 列表页用户有资格 | `has_profit_sharing_qualification=true` |
| T6 | 资格服务宕机 | 列表 200，字段 JSON null |

## 数据与环境

SQLite/MySQL 测试库 + httptest 下游。

## 通过标准

上述用例全绿。意图发生时无新事件（只读例外，见功能意图对照表）。
