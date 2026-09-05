# v99 application-integration

```mermaid
flowchart LR
  FE[taskFE 待分账 Tab] -->|GET /api/system-admin/profit-sharing/| GW[APISIX]
  GW -->|forward-auth staff| BILL[taskBill]
  BILL --> PS[billing_profit_sharing]
```
