# 权限分析：项目详情内部仓磁盘占用

日期：2026-07-16  
设计：`2026-07-16-project-internal-repo-disk-size-design.md`

## 结论

无新 endpoint；沿用项目详情 **读权限** + 当前用户对内部 GitLab 的 OAuth token。不扩大授权面。

## 改动点权限矩阵

| 改动点 | 角色/主体 | 权限要求 | 拒绝行为 |
|--------|-----------|----------|----------|
| GET 项目详情（enrich disk size） | 已认证且可读该项目的租户用户 | 与既有 `handleGetProject` 一致 | 401/403/404 同现网 |
| GitLab `?statistics=true` | 当前用户 OAuth token（gitlab-local） | Reporter+ 通常可读 statistics | 省略 `disk_size_bytes`，不泄露他仓错误细节 |
| 外部仓 | 任意 | 不调用外部 statistics | 响应仅 `is_internal: false` |

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| 无 OAuth 时内部仓无大小 | 省略字段；用户可先完成 OAuth（与分支预览一致） |
| 用管理员 token 绕过用户权限 | **禁止**；仅用当前用户 token |
| 通过大小推断仓是否存在 | 仅对用户已关联且可读的项目仓 enrich |

## 审计

- 成功/失败打结构化日志：project_id、repo host（非完整 token）、是否 internal、是否拿到 size。
- 禁止日志打印 Bearer token。
