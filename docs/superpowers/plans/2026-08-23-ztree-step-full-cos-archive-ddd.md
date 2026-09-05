# DDD：ztree step_full COS 归档

- **Date:** 2026-08-23
- **BC:** Cloud（taskCloudService）

## Aggregates

### LayerGraphSnapshot（既有）

根：`(workspace_id, task_id, comment_id)`。不改边界。

### JobStepFullArchive

根：`(workspace_id, task_id, comment_id, job_id)`。

- VO `StepFullObjectKey`：pathRule 渲染，禁止 `..` `/` 逃逸
- Entity 指针：object_key, bytes, etag, source(cos|local)

## Port

```
StepFullObjectStore
  Put(ctx, key, jsonBytes) (etag, error)
  Get(ctx, key) (jsonBytes, found, error)
```

适配器：TencentCOS（DirectClient）、Memory（测）、LocalTable（backend=local 把 bytes 放 payload_json）。

## Domain services

- `RenderStepFullObjectKey(rule, ids)`
- `MergeStepFullBundle(existing, jobID, steps)` — 按 job_id 覆盖

## Events

- `JobStepFullArchived` {workspace_id, task_id, comment_id, job_id, object_key, bytes}
- `StepFullCOSConfigUpdated` {backend, bucket, region, key_prefix} 无密钥

领域层禁止 import database/sql 与 cos SDK。
