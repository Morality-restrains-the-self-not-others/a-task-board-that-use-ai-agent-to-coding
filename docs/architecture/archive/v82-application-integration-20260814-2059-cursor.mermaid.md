# v82 — Terminal release comment CSC

```mermaid
flowchart LR
  FE[taskFE] -->|indicators| Cloud[taskCloudService MOD]
  Cloud -->|counts| FE
  TaskSvc[taskTaskService] -->|TASK_STATUS_CHANGED| EvS[TASK_STATUS_CHANGED]
  EvS --> Rel[taskEvents release-servers MOD]
  Rel -->|list-by-task NEW| List[GET list-by-task NEW]
  List --> Cloud
  Rel -->|read releaseable| Cmt[(comment CSC MOD)]
  Rel -->|per comment if started| EvStop[CLOUD_SERVER_STOPPED MOD]
  Cloud --> Cmt
  Cloud --> Tpl[(task template)]
```

- **Version:** v82 target
- **Iteration:** terminal-release-comment-csc-v82
- **Based on:** v80
- **Decisions:** 终态按评论 CSC 逐台释放；空模板 / 无运行资源 success no-op，禁止 retry 至 DLT
