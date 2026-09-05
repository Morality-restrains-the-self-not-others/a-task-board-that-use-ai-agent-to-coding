# 角色权限分析：容器拉取评论级 Git 提交者身份

- **日期**: 2026-08-20
- **前置**: `2026-08-20-comment-level-container-git-identity-design.md`
- **结论**: 无新角色、无新权限点；沿用现有容器 token 作用域。

## 角色

| 角色 | 本增量动作 | 权限 |
|------|-----------|------|
| 容器（持有 server-container-token） | GET/POST task-detail、repo-clone-credentials、layer-oauth | 仅本 token 绑定的 tenant/workspace/task/**comment** |
| 评论作者 | 发评时选定 repo_identities_json | 既有 UI / 发评 API（v83） |
| 平台运维 | 无 | — |

## 权限规则

1. 身份查询必须带 token 内 `CommentID`；URL path 的 `comment_id` 与 token 不一致时拒绝（403 SCOPE_MISMATCH）。
2. 不得因「作者还有别的 identity」而改写评论已选定的 git_identity_id。
3. 无 RBAC 新资源组。
