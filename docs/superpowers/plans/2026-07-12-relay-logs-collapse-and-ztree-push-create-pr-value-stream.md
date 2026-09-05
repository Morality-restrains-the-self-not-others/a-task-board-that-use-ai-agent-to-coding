# 价值流：启动日志默认折叠 + 推送并创建PR

- **日期**: 2026-07-12
- **状态**: approved (auto)

## 端到端价值流

```
用户打开 relayToTrae 任务详情
  → 启动日志默认折叠（可展开排障）
  → 在 zTree 看到未推送提交
  → 点击「推送并创建PR」
  → SaaS 推送层提交到远端
  → SaaS 创建/复用 GitHub PR（wait_for_pr）
  → 浏览器打开 PR 页
```

## 最小可行增量

1. **MVI-1**：默认折叠启动日志（纯前端 + 意图）
2. **MVI-2**：按钮文案 + `wait_for_pr` + 响应含 PR 结果 + 打开 PR 页

## 测试点（对照意图）

| ID | 测试点 | 用例落点 |
|----|--------|----------|
| T1 | 默认折叠，按钮为「展开日志」 | vitest RelayDirectPanel |
| T2 | 切换后正文 v-show | vitest |
| T3 | canPush 时按钮文案「推送并创建PR」 | vitest LayerGraphZtreeNode / 文案 |
| T4 | wait_for_pr 响应含 html_url 时 open | vitest layerActions |
| T5 | wait_for_pr 时主路径等待 job 终态 | Django 单测 |
| T6 | 无 wait_for_pr 仍异步、响应无强制 github_pull_request | 既有 async 单测保留 |
