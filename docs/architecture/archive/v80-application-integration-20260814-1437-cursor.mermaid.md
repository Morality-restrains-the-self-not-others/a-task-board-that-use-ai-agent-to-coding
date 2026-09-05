# v80 — Task-level running machine/container counts

```mermaid
flowchart LR
  FE[taskFE WorkPanel / TaskDetail MOD] -->|indicators + comment runtime-status| Cloud[taskCloudService MOD]
  Cloud -->|machines=N containers=M| FE
  Cloud --> Cmt[(comment CSC runtime MOD)]
  Cloud --> Tpl[(task template only MOD)]
  Cloud --> M[(running_machine_count NEW)]
  Cloud --> C[(running_container_count NEW)]
```

- **Version:** v80 target
- **Iteration:** task-level-running-comment-server-count-v80
- **Based on:** v79
- **Decisions:** 任务级只标记运行中机器数与容器数；运行态只存评论 CSC；Starting 不计入机器数
