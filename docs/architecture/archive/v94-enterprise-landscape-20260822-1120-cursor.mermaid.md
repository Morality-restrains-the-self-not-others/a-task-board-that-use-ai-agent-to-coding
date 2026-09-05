# v94 enterprise-landscape

```mermaid
flowchart LR
  Member[租户成员] --> Push[推送并创建PR]
  Push -->|html_url| Reply[PR 链接回复]
  Reply --> Merge[一键合并PR]
  Push --> FE[taskFE]
  Reply --> TS[taskTaskService]
  Merge --> GO[taskGitOauth]
```
