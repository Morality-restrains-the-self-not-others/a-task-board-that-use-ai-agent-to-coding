# 实施计划：任务状态变更事件与终态释放服务器

- 日期：2026-07-13
- 设计：`docs/superpowers/specs/2026-07-13-task-status-changed-release-servers-design.md`

## 任务清单

### Task 1 — 注册事件与 Topic

- [ ] `taskEvents/config/config.go`：`TASK_STATUS_CHANGED` → `task-status-changed`
- [ ] `taskEvents/config/intent_registry.go` + `event_registry.go`：注册 intent 18040
- [ ] Django `core/kafka/config.py` + `DOMAIN_EVENTS.md` 同步
- [ ] `taskEvents/bin/README.md` 行更新

### Task 2 — taskTaskService 发布（TDD）

- [ ] 新增 `events.go`：`publishDomainEvent`（对齐 taskCloudService）
- [ ] `handleUpdateTask`：检测 column/completed 变化后异步/同步发布
- [ ] 配置：`KafkaBootstrapServers`（env / config）
- [ ] 单测：变化发事件；无变化不发；Kafka 空配置跳过不 panic

### Task 3 — 终态判定纯函数（TDD）

- [ ] `taskEvents/.../taskstatuschanged/terminal.go`：列名 + completed → TerminalKind
- [ ] 单测覆盖 已完成/已取消/英文别名/非终态

### Task 4 — Consumer 编排（TDD）

- [ ] Handler：非终态 no-op；无 config no-op；有 instance_id → publish CLOUD_SERVER_STOPPED
- [ ] 有 server_url → 调 ContainerGateway stop（mock HTTP）
- [ ] cmd main + port_config 若需要
- [ ] stop_reason：`task_status_completed` / `task_status_cancelled`

### Task 5 — 文档与价值流

- [ ] 更新 `DOMAIN_EVENTS.md`
- [ ] 可选：`conf/value-stream.yaml` 增加测点步骤

### Task 6 — 验证

- [ ] `go test` taskTaskService 相关包
- [ ] `go test` taskEvents taskstatuschanged 包
- [ ] Review + Ship（PR）

## 验证命令

```bash
cd /tmp/ram-work/taskTaskService && go test ./src/ -count=1 -run 'StatusChanged|TaskUpdate|Publish'
cd /tmp/ram-work/taskEvents && go test ./internal/handlers/taskstatuschanged/... -count=1
```
