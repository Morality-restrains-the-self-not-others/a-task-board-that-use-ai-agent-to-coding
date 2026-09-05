# 实施计划：工作空间机器节点闲置策略

- 日期：2026-07-13
- 设计：`docs/superpowers/specs/2026-07-13-workspace-machine-idle-policy-design.md`

## Task checklist

### T1 — Schema + policy 仓储（红→绿）

- [ ] 迁移：`workspace_machine_policies` + `cloud_server_configs.idle_since`
- [ ] 登记 `db/table_ownership.yaml`
- [ ] 单测 T1/T2：默认 GET、PUT/GET 回显
- [ ] 实现 GET/PUT handlers + 路由挂载 + openapi

### T2 — Summary API

- [ ] 单测 T3：busy+idle 计数
- [ ] 实现 `workspace-machine-summary/`
- [ ] 清空/设置 `server_url` 时维护 `idle_since`

### T3 — start-vm 门禁 + idle reuse

- [ ] 单测 T4：未启用 CPA → 403
- [ ] 单测 T5：prefer + idle → reuse
- [ ] 接入 `compute_start_vm.go`

### T4 — recycle intent

- [ ] 注册 port 18044 intent `workspace_machine_idle/1_recycle_idle_nodes`
- [ ] 定时扫描 + stop 编排单测 T6
- [ ] runAll/port 配置（若需要）

### T5 — 前端设置 + work-panel

- [ ] `WorkspaceMachinePolicyModal.vue`
- [ ] `WorkspaceSettingsTaskPanel` 入口按钮
- [ ] WorkPanel 摘要条 + fetch util
- [ ] Playwright 冒烟（或组件级测试）

### T6 — 文档同步

- [ ] `conf/value-stream.yaml` 激活 stream
- [ ] `intent_index.md` 登记
- [ ] `machine_container.md` 若需交叉引用策略（可选）

## 验证命令

```bash
cd taskCloudService/src && go test ./... -count=1 -timeout 120s
cd taskEvents && go test ./... -count=1 -timeout 120s
```
