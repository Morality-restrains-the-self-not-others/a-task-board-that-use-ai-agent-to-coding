# v100 enterprise-landscape

```mermaid
flowchart LR
  TA[租户管理员] --> INV[电子发票服务]
  STAFF[平台员工] --> TAB[发票审批 Tab]
  TAB --> FE[taskFE]
  INV --> FE
  FE --> BILL[taskBill]
```
