# v83 — Enterprise landscape comment run identity

```mermaid
flowchart LR
  Dev[Developer] -->|select identity when commenting| FE[taskFE]
  FE -->|display only| Link[Task Linked Projects]
  FE --> Task[taskTaskService]
  Task --> Ident[Comment Run Repo Identity NEW]
```

- **Version:** v83 target
- **Iteration:** comment-level-repo-identity-v83
- **Based on:** v81
