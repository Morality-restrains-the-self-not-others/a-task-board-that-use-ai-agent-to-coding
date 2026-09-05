# 价值流：文件树目录 → 提交日志 + 当前分支

- 日期：2026-07-12

## 端到端流程

```
用户点击目录 → Vue handleSelectDir → SaaS/Gateway 转发
  → onlineServiceJS git/log → resolveLayerGitLogContext
  → [若仓库根] rev-parse HEAD → 响应 text + is_repo_root + current_branch
  → 预览面板「提交日志 · 分支名」
```

## 最小可行增量（本迭代）

1. 后端判定仓库根并附带分支。
2. Gateway 转发 path/limit。
3. 前端标题旁展示。
4. 单测 + Playwright 断言。

## 测试点

| ID | 场景 | 期望 |
|----|------|------|
| VS-1 | 点击多仓根 | 显示分支 |
| VS-2 | 点击仓内子目录 | 不显示分支 |
| VS-3 | 读分支失败 | 仍显示提交日志 |
