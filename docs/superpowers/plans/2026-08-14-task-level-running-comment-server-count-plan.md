# 实施计划：任务级运行中机器/容器计数

> 设计 / 权限 / 价值流 / NFR / DDD 已齐。Step 3 worktree SKIP。

## 任务清单

### T1 — 领域计数纯函数（已写骨架）

- [x] `taskCloudService/domain/running_counts.go` + 单测
- 命令：`cd taskCloudService && go test ./domain/`

### T2 — DDL 清空任务级运行态 + 两列

- [x] `dataMigrate/taskCloudService/019_task_running_counts.sql`
- [x] ADD `running_machine_count` / `running_container_count`
- [x] UPDATE 模板行清空 instance/status/ip/url
- [x] 按评论行回填两计数（不 heal）

### T3 — Red：计数重算与禁止任务级写

- [x] `task_running_counts_test.go`：T1–T7、T10–T11
- [x] persist 无 comment → 任务级 instance 仍空
- [x] setLastRuntimeStatus 必须 comment 或 instance
- [x] 删除 heal 测例，改为「不领养」

### T4 — Green：写路径

- [x] `recomputeTaskRunningCounts` 调 domain.Compute + UPDATE 模板行 + 日志
- [x] persist 无 csc/comment → no-op + warn
- [x] `setCloudServerLastRuntimeStatus` 收窄 WHERE
- [x] 删除 `healCommentCSCInstanceFromTaskLevel`
- [x] 评论 runtime/URL 变更后重算

### T5 — snapshot / indicators

- [x] 扫描跳过 `comment_id=''` 的 instance/status
- [x] JSON 增加两计数；布尔 = count>0
- [x] 更新既有 indicators 测例（种子须带 comment_id）

### T6 — 前端

- [x] `workPanelRuntimeIndicators.js` 归一 counts
- [x] 任务壳展示 N/M，不用任务级 Starting 盖评论

### T7 — 事件豁免已登记

- [x] `TaskRunningCountsRecomputed` → `_meta/publish_evidence_exempt.yaml`

## 验证

```bash
cd taskCloudService && go test ./domain/ ./src/ -count=1 -timeout 180s
cd taskFE/app && npx vitest run src/utils/workPanelRuntimeIndicators.test.js
```
