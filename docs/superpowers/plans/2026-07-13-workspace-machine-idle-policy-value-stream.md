# 价值流：工作空间机器节点闲置策略

- 日期：2026-07-13
- 域：cloud-integration / project-workspace

## 增量切片（按价值）

| 序 | 增量 | 用户价值 | 依赖 |
|----|------|----------|------|
| I1 | 策略持久化 GET/PUT + 设置 UI | 可配置启用节点与回收间隙 | — |
| I2 | summary API + work-panel 展示 | 一眼看到启动/闲置/间隙 | I1（读 recycle_minutes） |
| I3 | start-vm CPA 门禁 + prefer idle reuse | 控成本 + 加速冷启动 | I1 |
| I4 | idle recycle intent | 自动回收闲置机器 | I1 + idle_since |

## YAML 草案（写入 conf/value-stream.yaml）

```yaml
- name: workspace-machine-idle-policy
  domain: cloud-integration
  status: planned
  steps:
    - name: configure-workspace-machine-policy
      status: planned
      test_file: ../../taskCloudService/src/workspace_machine_policy_test.go
      fields:
        - name: task-cloud-service.workspace_machine_policies.idle_recycle_minutes
          description: 闲置回收分钟；0=关闭
        - name: task-cloud-service.workspace_machine_policies.enabled_authorization_ids
          description: JSON 数组；空=不限制 CPA
    - name: show-workspace-machine-summary
      status: planned
      test_file: ../../taskCloudService/src/workspace_machine_summary_test.go
      fields:
        - name: task-cloud-service.cloud_server_configs.idle_since
          description: 进入闲置时写入；复用时清空
    - name: prefer-idle-reuse-on-start
      status: planned
      test_file: ../../taskCloudService/src/compute_start_vm_idle_reuse_test.go
    - name: recycle-idle-machine-nodes
      status: planned
      test_file: ../../taskEvents/internal/handlers/workspacemachineidle/handler_test.go
```

## 测试点（wsd）

1. PUT policy → GET 回显
2. summary 计数正确
3. 未启用 CPA → start-vm 403
4. idle + prefer → reuse
5. 超时 → recycle stop
