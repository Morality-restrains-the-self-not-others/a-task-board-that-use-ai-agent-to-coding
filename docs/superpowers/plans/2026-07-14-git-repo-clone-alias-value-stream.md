# 价值流：Git 仓库克隆别名

日期：2026-07-14

## 端到端流

```
用户填写 URL+别名 → Create/Edit Project API
  → taskProjectService 写入 project_repos(repo_url, clone_alias)
  → 创建任务关联项目
  → container-snapshot 带 git_repos + git_repo_entries
  → taskCredentialService task-detail
  → trae-agent bootstrap 按 clone_alias 或 URL 推导目录名 git clone
```

## 最小可行增量

1. 持久化 + API 兼容读写
2. 前端输入
3. 容器链路透传 + trae-agent 使用

## 测试点映射

见 `docs/intents/frontend/git_repo_clone_alias.test-intent.md` T1–T8。
