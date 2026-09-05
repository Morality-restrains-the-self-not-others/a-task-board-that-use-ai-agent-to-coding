# 实施计划：ztree step_full COS 归档

- **Date:** 2026-08-23
- **Architecture:** v104 current。Owner `taskCloudService`。DDL `040_cloud_job_step_full_object.sql`。

## Tasks

- [x] Domain key + merge + tests
  - Create: `taskCloudService/domain/step_full_key.go`
  - Create: `taskCloudService/domain/step_full_key_test.go`
  - Create: `taskCloudService/domain/step_full_bundle.go`
  - Create: `taskCloudService/domain/step_full_bundle_test.go`
- [x] Event 契约
  - Modify: `taskEvents/config/config.go` EventTopic `JobStepFullArchived` / `StepFullCOSConfigUpdated`
  - Modify: kafka topic 注册（若有 kafka_recreate）
- [x] DDL + store
  - Create: `dataMigrate/taskCloudService/040_cloud_job_step_full_object.sql`
  - Create: `taskCloudService/src/step_full_store.go` + `_test.go`
- [x] COS adapter + conf
  - Modify: `conf/taskCloudService/config.yaml`
  - Create: `conf/taskCloudService/step-full-cos.yaml`
  - Create: `taskCloudService/src/step_full_cos.go` Fake + Real DirectClient
- [x] Inbound push + hydrate
  - Modify: `container_inbound.go` action `job-step-full-push`
  - Modify: `job_execution_log_handlers.go` COS-first
  - Create: handlers tests
- [x] Admin API + FE
  - Create: admin handler + tests
  - Modify: APISIX routes, openapi, api_route_ownership
  - Create: `SystemAdminStepFullCOS.vue` + route + sidebar
- [x] Container collect + POST
  - Create: `saasStepFullArchive.mjs` + test
  - Modify: `jobsRuntimeRunJob.mjs` close 调用
  - Modify: `taskAgentSupport` forwardsToCloudService
- [x] Intent INDEX + value-stream.yaml 增量
