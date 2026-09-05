# 系统内建 GitLab：磁盘已用量上报 + namespace/项目硬限额

- 日期：2026-07-18
- 状态：已实现并验收（goal-mode）
- 关联 OPT：`OPT-20260718-043`（completed）、`OPT-20260718-040`（completed）；后续迁仓见 `OPT-20260718-046`

## 1. 目标

| # | 标准 | 验收 |
|---|------|------|
| S1 | 租户设置页「已用磁盘」反映 gitlab-local 关联仓 `repository_size` 之和 | GET `disk_used_gb` > 0（有仓时） |
| S2 | 购买的 `disk_gb` 下发为 GitLab 硬限额 | 组 `tenant-{id}` + 各关联项目 `repository_size_limit` |
| S3 | 周期性同步（cron）+ 购买后可立即触发 | sync 脚本 / 内部 API |
| S4 | Swagger 可见新内部接口 | openapi |

## 2. 约束与决策

- **无既有 tenant→GitLab namespace 映射**；仓多在用户路径下（如 `user/repo`）。
- **计量**：按 SaaS `project_repos` 中属于该 `company_id` 且匹配 `gitlab:gitlab-local` 的仓汇总 `statistics.repository_size`。
- **执法**：
  1. 确保 Group `tenant-{tenant_id}`，`repository_size_limit = disk_gb GiB`（未来仓归组）；
  2. 对每个已关联内部仓设 **项目级** `repository_size_limit = disk_gb GiB`（覆盖现网用户命名空间仓）。
- `disk_gb=0`：项目/组 limit 设为 `1` 字节（GitLab 将 0 视为不限制）。
- Admin PAT：脚本经 `gitlab-rails` 幂等创建，写入 `gitService/gitlab_home/.taskbill_admin_pat`（gitignore）。

## 3. 组件

| 组件 | 职责 |
|------|------|
| taskProjectService | `GET /api/internal/tenants/{tid}/gitlab-local-disk-usage/` |
| taskBill | 列表配额、同步上报、GET 时 best-effort 刷新已用；购买后触发 sync 钩子 |
| gitService/scripts/sync_tenant_gitlab_disk_quota.sh | 编排：用量 + 硬限；供 cron |

## 4. 非目标

- 不强制迁移既有仓到 `tenant-*` 组。
- 不做跨仓聚合硬拒（CE 无 EE 命名空间聚合时，以单仓 limit≈配额为近似执法）。

## 5. 验收记录（2026-07-18）

| 标准 | 结果 |
|------|------|
| S1 已用磁盘 | 租户 `850256677331562496`：`disk_used_bytes=58966`，`disk_used_gb≈5.5e-05` |
| S2 硬限额 | rails：`tenant-850256677331562496` 与 `example-user/valueStream` → `5368709120`（5 GiB） |
| S3 周期同步 | crontab `*/15` → `gitService/scripts/sync_tenant_gitlab_disk_quota.sh` |
| S4 OpenAPI | `taskBill/src/openapi-internal.yaml`：`report-gitlab-disk-usage`、`sync-gitlab-disk-quotas` |
