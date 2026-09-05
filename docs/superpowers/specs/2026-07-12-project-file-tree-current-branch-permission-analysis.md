# 权限分析：项目文件树提交日志旁显示当前工作分支

- 日期：2026-07-12
- 设计文档：`docs/superpowers/specs/2026-07-12-project-file-tree-current-branch-design.md`
- 结论：**无新增端点；沿用既有 git/log 鉴权与租户隔离**

## 改动点权限矩阵

| 改动点 | 角色 | 权限边界 | 变更 |
|--------|------|----------|------|
| `GET /api/layers/{id}/git/log` 响应扩展 | 容器内进程 | 仅本层 `layer_id` 工作区；无跨层 | 字段追加，鉴权不变 |
| Django `container-layer-git-log` | 已登录租户成员 | 既有 workspace/task 校验 + 转发 | 透传 JSON，无改 |
| Gateway `container-layer-git-log` | 持有效容器会话 | scopedAPI 按任务容器 | 补齐 query 转发，无扩权 |
| Vue 文件树预览 | 任务详情可访问者 | 与现文件树相同 | 展示字段，无新写操作 |

## 敏感信息

- `current_branch` 为分支名，非密钥；可展示。
- 不在日志中打印完整仓库绝对路径以外的凭证。

## 风险

- 无 privilege escalation；无新写路径。
