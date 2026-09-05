# 测试意图：任务详情展示子仓库克隆状态

- **对应功能意图**: `task_detail_nested_repos_clone_status.intent.md`

## 覆盖点

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 无进度 + 容器未就绪 | `waiting_container` / 「等待容器」 |
| T2 | 无进度 + 容器就绪 | `idle` / 「未开始」 |
| T3 | 进度 100 | `done` / 「已完成」 |
| T4 | 进度中 | `running` + progress |
| T5 | 失败文案 | `error` / 「克隆失败」 |
| T6 | bootstrap 完成无进度行 | `done` |
| T7 | Map/Object 查找进度 | lookup 命中 |
| T8 | 汇总统计 | done/running/error/idle |
| T9 | UI 有子仓 | `task-nested-repos-clone-status` 可见；徽章与汇总正确 |
| T10 | UI 无子仓 | 不渲染区块 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-17 | 初版 |
