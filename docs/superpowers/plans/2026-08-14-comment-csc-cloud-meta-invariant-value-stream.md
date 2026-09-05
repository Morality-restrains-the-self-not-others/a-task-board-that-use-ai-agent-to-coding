# Value Stream: 评论 CSC 云平台元数据不变量

- **日期**: 2026-08-14
- **设计**: `docs/superpowers/specs/2026-08-14-comment-csc-cloud-meta-invariant-design.md`
- **配置**: `conf/value-stream.yaml` → `comment-csc-cloud-meta-invariant`

## Related Value Streams

- `task-running-comment-server-count`（v80）：任务级只作模板+两计数；本流在评论行上收紧云账号三元组不变量，不改计数语义。
- 无冲突：不把任务级重新写成运行实例。

## Increments

| # | 步骤 | 价值 | 测试 |
|---|------|------|------|
| 1 | ensure-rejects-illegal-comment-csc | 不再落 mock 占位行 | `comment_csc_ensure_test.go` |
| 2 | resolve-fills-illegal-keeps-override | 脏行按字段补齐；用户覆盖保留 | `compute_scoped_csc_test.go` |
| 3 | workbench-real-ecs | 真实 ECS 打开阿里云控制台 | `compute_workbench_link_test.go` |

## Fields

三段名：`task-cloud-service.cloud_server_configs.{platform,region,authorization_id,instance_id}`
