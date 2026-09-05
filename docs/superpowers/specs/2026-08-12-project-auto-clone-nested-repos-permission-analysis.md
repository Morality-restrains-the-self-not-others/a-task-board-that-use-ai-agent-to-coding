# Role-Permission：创建项目自动克隆子仓库开关

- **日期**: 2026-08-12
- **设计**: `docs/superpowers/specs/2026-08-12-project-auto-clone-nested-repos-design.md`
- **结论**: 无新权限码；复用项目 create/update/read。

## 端点与权限

| 操作 | 端点 | 角色 | 资源 | 效果 |
|------|------|------|------|------|
| 创建含开关 | `POST /api/projects/tenant_id/{tid}` | 租户成员（既有项目创建权） | Project | 写入 `auto_clone_nested_repos` |
| 更新开关 | `PATCH/PUT` 既有项目更新 | 项目编辑权（与 name/git_repos 同） | Project | 更新字段 |
| 读取 | `GET` 项目详情 / container-snapshot | 项目读权 / 内部 secret | Project | 返回字段 |
| 内部 enrich | credential 服务间 | 容器 token scope | Task | 按字段门控 merge；无新 PDP |

## 威胁模型（简）

- 字段为 bool，无注入面；禁止日志输出 token。
- 关闭开关不得绕过父仓 OAuth / 仓库访问校验。

## 决策

无需新增 RBAC resource group；SKIP 扩展 PDP。
