# 切换镜像 × 硬件硬拦截 — 实施计划

## Task 1: inferInstanceArchitecture 收紧

- 文件：`taskCloudService/src/compute_image_resolve.go` + `compute_image_resolve_arch_test.go`
- 测：`ecs.r6.xlarge`→x86_64；`ecs.r6r.xlarge`/`g8y`→arm64
- 命令：`cd taskCloudService && go test ./src -count=1 -run InferInstanceArchitecture`

## Task 2: Project 聚合校验（TDD）

- 文件：`taskProjectService/src/instance_architecture.go`、`image_template_compat.go`、`project_handlers.go`
- 测：跨架构只改镜像 400 且 DB 不变；同单匹配 200；同架构只改镜像 200；清空镜像 200；lookup 无架构 400
- 命令：`cd taskProjectService && go test ./src -count=1 -run 'ImageTemplate|UpdateProjectContainer'`

## Task 3: 前端架构推导

- 翻转 `containerImageArchitecture.test.js`；兼容性纯函数
- 命令：`cd taskFE/app && npx vitest run src/utils/containerImageArchitecture.test.js`

## Task 4: 项目 UI 同单提交

- `buildImagePatchBody` 可附带 `server_run_template`
- ProjectDetail 草稿镜像驱动面板；保存带 live payload
- 暴露 `scheduleFetchAvailableInstances`（不改 2580 行 composable 本体）
- vitest：`projectDetailInlineEditUtils.test.js`

## Task 5: OpenAPI 400

- `taskProjectService/src/openapi.yaml` PATCH 400 说明

## Task 6: Intent→Event

- 已书面例外；不新增 publish 任务

## 验证

```
cd /tmp/ram-work/taskCloudService && go test ./src -count=1 -run InferInstance
cd /tmp/ram-work/taskProjectService && go test ./src -count=1 -run 'ImageTemplate|UpdateProjectContainer|UpdateProjectResolves'
cd /tmp/ram-work/taskFE/app && npx vitest run src/utils/containerImageArchitecture.test.js src/utils/projectDetailInlineEditUtils.test.js src/utils/projectRunTemplateUtils.test.js
```
