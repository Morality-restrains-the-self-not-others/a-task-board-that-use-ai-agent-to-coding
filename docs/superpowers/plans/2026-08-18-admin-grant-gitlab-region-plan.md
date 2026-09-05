# 实施计划 — 管理端赠送 GitLab 须选区域

- **日期**: 2026-08-18
- **设计 / NFR / DDD**: 见同前缀 specs/plans

不新增 Kafka（意图例外已书面）。事件任务：N/A — 核对意图表即可。

## Task 1 — 后端：缺 region 必须失败（Red→Green）

- 测：`taskBill/src/admin_grant_region_test.go`
- 改：`ResourceGrantInput.Region`；`adminGrantResources` 对 `gitlab_disk`/`gitlab_traffic` 校验 `getGitlabRegionBySlug`；handler 解析 `region`
- 验证：`go test ./src -count=1 -run 'AdminGrantGitlab|AdminGrantTaskPost'`

## Task 2 — 后端：配额与订单行写入所选 region

- 测：指定 slug 后查询 `billing_tenant_gitlab_resource` 与 `billing_resource_order_item.region`
- 改：INSERT 带 `region`；订单行不再写 `''`；成功后 `recalcRegionAllocated`
- 验证：两区域独立测例

## Task 3 — 前端：区域下拉 + POST body

- 测：`taskFE/app/src/views/SystemAdminGrantPoints.region.unit.test.js`
- 改：`SystemAdminGrantPoints.vue` 加载 `/api/system-admin/gitlab-regions/`，GitLab 类型显示 select
- 验证：`npx vitest run src/views/SystemAdminGrantPoints.region.unit.test.js`

## Task 4 — 意图与文档对照

- 更新 `docs/intents/INDEX.md`、B-049 验收一条
- 行数：赠送页 ≤ 500

## 验证命令

```bash
cd /tmp/ram-work/taskBill && go test ./src -count=1 -run 'AdminGrant'
cd /tmp/ram-work/taskFE/app && npx vitest run src/views/SystemAdminGrantPoints.region.unit.test.js
```
