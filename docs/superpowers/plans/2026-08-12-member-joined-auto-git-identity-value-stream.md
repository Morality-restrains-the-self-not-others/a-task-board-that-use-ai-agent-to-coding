# Step 4 — Value Stream：成员加入 → 默认 Git 身份

```
管理员/系统创建公司成员
  → taskTenantService 落库成员行
  → 发布 MEMBER_JOINED (Kafka member-joined)
      payload: member_id, user_id, company_id, member_name, …
  → taskEvents consumer 1_create_default_git_identity
      → POST taskTaskService /api/internal/git-identities/ensure-default/
      → 写入 task_git_identities（默认邮箱哈希 + member_name，幂等）
  → 成员/管理员打开 PeopleManage
      → 「Git 身份」弹窗 CRUD / 设默认
      → 后续任务提交使用所选身份
```

测试点：见 `docs/intents/backend/member_joined_auto_git_identity*.intent.md`。
