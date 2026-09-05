# v97 enterprise-landscape

```mermaid
flowchart LR
  Author[评论作者] --> Clone[评论容器克隆]
  Clone -->|need credential| Exchange[按 Git site 换票]
  Exchange --> GO[taskGitOauth]
```
