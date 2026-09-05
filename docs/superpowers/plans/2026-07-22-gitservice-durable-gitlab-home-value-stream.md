# 价值流：gitservice-durable-gitlab-home

## 影响

不新增业务流步骤。加固依赖系统内建 GitLab 的既有流：

- 创建项目 / 仓库 push
- GitLab OAuth / OIDC SSO
- 租户磁盘配额同步（`sync_tenant_gitlab_disk_quota.sh`）

## Increment

1. **durable-home-resolve** — 路径解析 + tmpfs 门禁
2. **legacy-migrate** — rsync 迁移 + compose 变量化
3. **verify-recreate** — 容器重建后数据仍在

## YAML

无需改 `value-stream.yaml`（infra 加固，无新业务 step/fields）。
