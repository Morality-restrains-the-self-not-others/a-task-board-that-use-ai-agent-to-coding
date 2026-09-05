# Application Integration v35 — Create Task Field Settings

```mermaid
flowchart LR
  Settings["Vue WorkspaceSettings\nCreateTaskFieldSettingsModal"]
  Work["Vue WorkPanel\nCreateTaskModal"]
  GW[task-gateway]
  Project["taskProjectService\ncreate-task-field-settings"]
  Tbl["workspace_create_task_field_settings"]
  C["Constraint\n默认全 true"]

  Settings -->|GET/PUT| GW
  Work -->|GET| GW
  GW --> Project
  Project -->|R/W| Tbl
  C -.-> Project
```

## Plateau / Gap

- **Plateau v34 ✅ current**: 项目详情内部仓磁盘占用
- **Gap**: 创建任务可选字段始终展示，无法按工作区精简
- **Plateau v35 ✅ current**: 工作区可配置创建任务可选字段显隐；默认全开
