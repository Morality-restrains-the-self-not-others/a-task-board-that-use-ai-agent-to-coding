# v95 enterprise-landscape

```mermaid
flowchart LR
  Admin[系统超级管理员] --> Review[审批推荐资格]
  Review -->|approved| Revoke[取消分账资格]
  Revoke --> Audit[查阅操作审计]
  Revoke --> Bill[停止未来分成]
  Review --> FE[taskFE]
  Revoke --> REF[taskReferral]
  Bill --> TB[taskBill]
```
