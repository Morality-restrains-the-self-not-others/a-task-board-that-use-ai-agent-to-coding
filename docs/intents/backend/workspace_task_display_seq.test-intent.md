# 测试意图：工作空间任务帖人读序号

## 测试目标

验证同一工作空间内任务帖人读序号从 1 递增、不可变、不回收；展示与搜索用 `#N`；技术主键仍为 `task_<digits>`。

## 测试分层

| 层 | 覆盖 |
|----|------|
| 单元（Go） | 事务发号；并发不撞号；JSON 含 `workspace_seq`；数字查询按序号且必须带工作空间 |
| 单元（FE） | `formatTaskIdTitleLabel` / Badge / 搜索 / 父任务过滤 / 任务辅助信息 `taskDisplayNo` 用 `workspace_seq` |
| 意图→事件 | 创建成功后 `TASK_CREATED` payload 含 `workspace_seq`（既有 publish 路径） |

## 用例矩阵

| # | 场景 | 期望 |
|---|------|------|
| T1 | 空工作空间连续创建 3 帖 | seq=1,2,3 |
| T2 | 两请求并发创建同一工作空间 | 两个不同 seq，无唯一键冲突 |
| T3 | 创建失败回滚（如关联项目非法） | `next_val` 不前进或与未提交行一致 |
| T4 | 删除 seq=2 后再创建 | 新帖 seq=4（若已发到 3），不复用 2 |
| T5 | 列表/详情 JSON | 同时有 `id` 与 `workspace_seq` |
| T6 | 搜索 `2` / `#2`（当前工作空间） | 命中 seq=2 |
| T7 | 搜索完整 `task_…` | 仍按主键命中 |
| T8 | 另一工作空间也有 seq=2 | 当前工作空间搜索不串号 |
| T9 | 卡片展示 | `#12` 而非技术 ID 后 6 位 |
| T10 | 单击 / 双击复制 | `#12` / 完整 `task_…` |
| T11 | 父任务 / 派生自 | `#N 标题`，无后 6 位 |
| T12 | `TASK_CREATED` | payload 含 `workspace_seq` |
| T13 | 任务辅助信息面板 | `data-testid=task-aux-info-display-no` 为 `#N`；无序号为 `—`；技术 ID 仍可见 |

## 数据与环境

- `taskTaskService` 测试库；两工作空间夹具。
- 不依赖云主机 / 计费（可用 `testModeSkipDjango`）。

## 通过标准

上表全绿；与设计 S1–S8 对齐。
