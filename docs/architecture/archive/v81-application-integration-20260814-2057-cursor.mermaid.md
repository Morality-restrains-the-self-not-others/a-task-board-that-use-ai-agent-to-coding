# v81 — Workspace task display sequence

```mermaid
flowchart LR
  FE[taskFE WorkPanel / TaskDetail MOD] -->|create / list / search| Task[taskTaskService MOD]
  Task -->|id + workspace_seq to #N| FE
  Task --> Alloc[(task_workspace_seq NEW)]
  Task --> Seq[(task_tasks.workspace_seq NEW)]
  Task --> PK[(task_tasks.id technical PK)]
  Task -->|TASK_CREATED + seq| Evt[TASK_CREATED MOD]
```

- **Version:** v81 target
- **Iteration:** workspace-task-display-seq-v81
- **Based on:** v80
- **Decisions:** 技术主键不变；工作空间内 `workspace_seq` 展示为 `#N`；存量清空不回填
