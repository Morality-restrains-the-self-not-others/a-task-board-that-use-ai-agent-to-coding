# 实施计划 — 共有目标分支下拉

## 价值流（简）

编辑分支策略 → 拉全仓分支 → 求交 → 选目标分支 → 保存 `merge_target_branch_name`

## NFR

- L2：分支拉取失败按仓记录错误，不阻塞整页；交集仅含成功返回的仓集合中仍共有的名（若任一仓失败且无分支，该仓不参与交集 / 交集为空时走空态）。
- **采用**：仅对「已成功拿到非空分支列表」的仓库求交；若应拉的仓数为 0 → 空；若有仓失败导致该仓列表为空 → 不参与交集（避免把失败当成「无分支」误杀全部）；若所有仓都失败 → 空态 + 错误汇总。

## 任务

- [x] `intersectBranchNameLists` + 单元测试（T1–T3）
- [x] `useTaskDetail`：编辑态批量拉分支 + `commonMergeTargetBranches` 计算
- [x] `TaskDetailBranchStrategyPanel`：目标分支改为共有分支 select
- [x] `CreateTaskModal`：目标分支 datalist/选项并入共有分支
- [x] 跑相关 vitest
