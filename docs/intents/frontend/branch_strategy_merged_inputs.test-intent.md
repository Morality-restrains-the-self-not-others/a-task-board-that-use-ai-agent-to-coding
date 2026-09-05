# 分支策略双输入合并 — 测试意图

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 编辑态渲染 | 无 `edit-work-branch-preset` / 无独立「目标分支模板」select |
| T2 | 工作分支 datalist | 含 feature/bugfix 等模版展开值 |
| T3 | 目标分支 datalist | 选项等于共有分支列表 |
| T4 | 手改工作分支为非模版值 | `workBranchPreset === 'custom'` |
