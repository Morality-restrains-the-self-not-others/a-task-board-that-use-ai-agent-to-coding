# v94 application-integration

```mermaid
flowchart LR
  FE[taskFE] -->|POST git_pr comment| TS[taskTaskService]
  FE -->|status / merge| GO[taskGitOauth]
  TS --> CMT[task_comments]
  GO -->|GET/PUT MR| GIT[GitHub/GitLab]
  GO --> AUD[git_oauth_taskcredentialaudit]
```
