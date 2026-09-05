# Feature-Params 表迁 Cloud + env-preview 迁 Go

- **日期**: 2026-07-20
- **状态**: 已采用（goal-mode）
- **对应 OPT**: OPT-20260720-001（表迁）、OPT-20260720-002（env-preview）；用户口中的 051/052 因编号冲突重登记

## 成功标准

1. 六表在 `task_cloud.db`：tenant/workspace/personal/access_audit/snapshot/api_key_usage
2. Cloud 公网/resolve/snapshot/usage **不再**调 Django `/api/internal/feature-params/*` store
3. Django 上述 internal store → **410**；ownership 改 `task-cloud`
4. `compute/feature-params-env-preview` 在 Cloud 实现；经网关 200；Django 同路径 410
5. 迁移脚本幂等拷贝 saas→task_cloud；Go/Django 测例通过

## 方案

- Schema：`ensureFeatureParamsSchema` in Cloud `db.go` 旁路文件
- Store：本地 SQL 替换 `feature_params_saas_http` / `public_store` HTTP
- 数据：`db/scripts/migrate_feature_params_to_task_cloud.sh`（ATTACH + INSERT OR IGNORE）
- env-preview：`handleFeatureParamsEnvPreview` + `serializeFeatureParamsEnv`
- 架构：路由切流已完成；本次为数据所有权与遗漏端点；不新画 ArchiMate
