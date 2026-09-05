# 实施计划：任务终态硬释放 + 镜像容器迁移

- 日期：2026-07-15
- 设计：`docs/superpowers/specs/2026-07-15-terminal-hard-release-container-migrate-design.md`

## 任务清单

### Task 1 — CSC schema + reuse 解绑 + 排除终态

- [ ] `db.go`：`terminal_released` 列 migrate
- [ ] `bindSharedMachineToTask`：成功后清空 source 绑定
- [ ] `findIdleMachineForReuse`：`terminal_released=0`
- [ ] 单测：reuse 源清空；终态标记不可 reuse

### Task 2 — Internal APIs

- [ ] `GET .../instance-bindings`
- [ ] `POST .../mark-terminal-released`
- [ ] `POST .../migrate-container-off-instance`（start-vm 禁复用 from_instance；更新 CSC）
- [ ] OpenAPI + 单测
- [ ] `main.go` 注册路由

### Task 3 — taskEvents Handler 扩展

- [ ] Dispatch：bindings → migrate foreign → stop self → STOPPED → mark
- [ ] migrate 失败 → Retryable
- [ ] 单测：顺序与失败路径

### Task 4 — 文档与价值流

- [ ] 更新 `docs/flows/value-stream-test-integration.wsd` 测试点（若文件可编辑）
- [ ] table ownership 若需登记列说明（列属既有表，通常不必新表）

### Task 5 — 验证

- [ ] `go test` taskCloudService 相关包
- [ ] `go test` taskEvents/taskstatuschanged
