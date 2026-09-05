# 实施计划 — ztree 层图快照服务端持久化

> **For agentic workers:** 按任务顺序 TDD。每任务红→绿→提交相关文件。Goal 模式跳过 AskQuestion。

**Goal:** 关容器后 ztree 层图与 job 步骤仍可从 Cloud 读取；不存克隆日志。

**Architecture:** v89 target。Owner `taskCloudService`。DDL `031_cloud_layer_graph_snapshot.sql`。

---

### Task 1: 领域身份校验（纯函数）

**Files:**
- Create: `taskCloudService/domain/layer_graph_snapshot.go`
- Create: `taskCloudService/domain/layer_graph_snapshot_test.go`

- [ ] 空 workspace/task/comment → invalid
- [ ] 合法身份 + 缺 layers 数组 → invalid
- [ ] `EmptyGraphJSON()` 返回 `layers:[]`,`jobs:[]`

**Verify:** `go test ./domain/ -count=1`

---

### Task 2: DDL + UPSERT/GET store

**Files:**
- Create: `dataMigrate/taskCloudService/031_cloud_layer_graph_snapshot.sql`
- Create: `taskCloudService/src/layer_graph_snapshot_store.go`
- Create: `taskCloudService/src/layer_graph_snapshot_store_test.go`
- Modify: `taskCloudService/src/main_test.go` cleanup 表名

- [ ] utf8mb4、`cloud_` 前缀、无 AUTO_INCREMENT、UNIQUE(workspace,task,comment)
- [ ] UPSERT 同键覆盖；GET 三键；错 workspace 找不到

**Verify:** `go test ./src/ -count=1 -run 'LayerGraphSnapshot'`

---

### Task 3: PUSH 落库 + 事件 + GET hydrate 路由

**Files:**
- Modify: `taskCloudService/src/container_inbound_actions.go` `handleLayerGraphPush`
- Create: `taskCloudService/src/layer_graph_snapshot_handlers.go`
- Modify: `taskCloudService/src/compute_handlers.go` 在 Gateway 代理前拦截 GET
- Modify: `taskCloudService/src/events.go` eventTopicMap
- Create: `taskCloudService/src/layer_graph_snapshot_handlers_test.go`

- [ ] T1 缺数组 400
- [ ] T2/T3/T8 UPSERT + spy 事件
- [ ] T4 无容器 GET 200
- [ ] T5 缺键 400
- [ ] 落库失败 ERROR 日志仍 SSE

**Verify:** `go test ./src/ -count=1 -run 'LayerGraph|InsertAndGetJobExecution'`

---

### Task 4: FE 无 endpoint 仍拉层图

**Files:**
- Modify: `taskFE/app/src/composables/taskDetail/taskDetailContainerFns.js`
- Modify: `taskFE/app/src/composables/taskDetail/taskDetailContainerFns.commentId.test.js`

- [ ] 去掉「必须有 endpoint」门
- [ ] 无 live endpoint 时跳过 runtime not-serving gate
- [ ] `fetchContainerTaskUiContext` 无 endpoint 仍 refresh 层图
- [ ] clone-log 函数仍要求 endpoint（不改存储）

**Verify:** vitest 该文件

---

### Task 5: 价值流 YAML + 流程图测试点

**Files:**
- Modify: `conf/value-stream.yaml`（`task-container-gateway` 流）
- Modify: `docs/flows/value-stream-test-integration.wsd`
- Modify: `docs/intents/INDEX.md` 可执行测试列

**Verify:** `python3 -c "import yaml; yaml.safe_load(open('conf/value-stream.yaml'))"`
