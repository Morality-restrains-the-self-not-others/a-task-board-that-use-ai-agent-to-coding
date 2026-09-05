# 测试意图：闲置复用启动保护 + 孤儿删除交叉校验

- **日期**: 2026-07-22
- **功能意图**: `idle_reuse_boot_guard_orphan_cross_check.intent.md`
- **复用 T1–T3 / T6:** superseded（ADR-0013）
- **孤儿 T4–T5:** 仍有效

## 测试目标

验证孤儿交叉校验（C）。闲置复用启动保护（A）已随复用拆除，不再作为产品验收。

## 测试分层

| 层 | 范围 |
|----|------|
| 单元 | `taskCloudService/src` orphan reconcile |
| 回归 | start-vm-auto 对闲置机 **不** reuse |

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | （历史）CSC Running、空 server_url、`idle_since` NULL；他任务 tryReuse | 已不适用；现为不 reuse |
| T2 | （历史）可 reuse | 已不适用 |
| T3 | （历史）Starting 不可 reuse | 已不适用 |
| T4 | 源 CSC instance 已空；他评论 CSC 持有同 instance；orphan reconcile | 不调用 DeleteInstance；日志 skip_owned |
| T5 | 真孤儿：云上 InstanceName=评论级名且无任何 CSC 持有 | 仍 DeleteInstance |
| T6 | （历史）terminal_released / forbid_instance_id | 迁机冷启动，不再选闲置机 |

## 数据与环境

- SQLite 测试 DB + mock `describeInstancesByName` / `deleteCloudInstanceForOrphan`
- 不依赖真实阿里云

## 通过标准

- T4–T5 单测绿
- `taskCloudService/src/compute_start_vm_idle_reuse_test.go` 断言闲置机不 reuse
