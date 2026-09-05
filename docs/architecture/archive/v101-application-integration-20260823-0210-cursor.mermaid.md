# v101 application-integration

```mermaid
flowchart LR
  RA[runAll :9999]
  SVC[托管业务服务]
  RA -->|"start / adopt / 显式 stop-all"| SVC
  RA -.->|"SIGTERM 默认不拆栈"| SVC
```
