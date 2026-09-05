# 价值流：ztree step_full COS 归档

- **Date:** 2026-08-23
- **Design:** `docs/superpowers/specs/2026-08-23-ztree-step-full-cos-archive-design.md`

## 影响流

任务协作（任务详情执行日志复查）+ 平台运维（COS 参数）。

## 增量

1. **Archive** — job 终态收集 step_full → Cloud → COS/local + 指针表 + JobStepFullArchived
2. **Hydrate** — GET 执行日志优先 COS
3. **Admin** — 系统管理配置 backend/bucket/region/pathRule/密钥

## 字段（三段式）

- `task-cloud-service.cloud_job_step_full_object.object_key`
- `task-cloud-service.step_full_cos.path_rule`
- `task-cloud-service.cloud_layer_graph_snapshot.graph_json`（既有）

## 测试点

见 `docs/intents/backend/cloud/ztree_step_full_cos_archive.test-intent.md` T1–T10。
