# v129 enterprise-landscape — 金丝雀平滑重启

```mermaid
flowchart LR
  OPS[平台运维] --> RST[平滑重启托管栈]
  RST --> UI[runAll Status]
  UI --> ORCH[runAll Runner]
  ORCH --> SVC[业务 HTTP]
  HOST[SO_REUSEPORT] --> SVC
```
