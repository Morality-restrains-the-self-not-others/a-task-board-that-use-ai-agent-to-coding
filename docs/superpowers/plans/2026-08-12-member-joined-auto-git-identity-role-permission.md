# Step 2 — Role-Permission：MEMBER_JOINED 自动 Git 身份

## 新/改路径权限

| 路径 | 操作 | 所需权限 | 说明 |
|------|------|----------|------|
| 成员创建（invite/join、建公司、internal upsert） | 发布 `MEMBER_JOINED` | 既有成员写路径 | 无新 perm；事件由服务端发出 |
| `POST /api/internal/git-identities/ensure-default/` | 确保默认身份 | 内部服务凭证 | 仅 taskEvents 消费者 |
| `GET/POST /api/git-identities/tenant/{tid}/member/{mid}/` | 列/建身份 | self **或** `member:manage` **或** `group-members:manage` | 目标成员所属公司校验 |
| `PATCH/DELETE /api/git-identities/tenant/{tid}/identity/{iid}/` | 改/删身份 | 同上 | 按 identity 归属公司校验 |
| PeopleManage「Git 身份」弹窗 | UI | 页内同 API 鉴权 | 无新页面码 |

## 数据访问

- 表 `task_git_identities` 仅 taskTaskService 直写；禁止跨服务直连。
- 组管理员以 `group-members:manage` 粗放行（不按单组边界细拆 → OPT）。

## 审计结论

鉴权为 L3：管理 API 必须服务端校验 self/perm，前端隐藏非安全边界。内部 ensure-default 不得对公网开放。
