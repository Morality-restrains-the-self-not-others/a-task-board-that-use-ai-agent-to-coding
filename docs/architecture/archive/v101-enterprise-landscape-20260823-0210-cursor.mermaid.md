# v101 enterprise-landscape

```mermaid
flowchart LR
  OPS[本机操作员] --> UI[Status UI 监督]
  UI --> RA[runAll]
  STACK[托管服务栈] --> BIZ[业务能力]
  UI -->|"adopt / stop-all"| STACK
```
