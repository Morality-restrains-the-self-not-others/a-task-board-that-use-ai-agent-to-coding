# v72 application-integration — Logical Resource Group RBAC

```mermaid
flowchart LR
  FE[taskFE hasRegion/hasPage] --> GW[APISIX]
  GW --> AUTH[taskAuth PDP]
  GW --> BIZ[业务 Go RequireRegion]
  GW --> TEN[taskTenant 多角色]
  AUTH --> SHARE[shareLib/authz]
  SHARE --> BIZ
  AUTH --> RG[(auth_resource_group page|ui_region)]
  AUTH --> RM[(auth_resource_member ui|api)]
  AUTH --> RRG[(auth_role_resource_group)]
  TEN --> AUTH
```

## 层级

```mermaid
flowchart TB
  PAGE[页面组 Page Group 载体] --> R1[UI 组件区域 Region = 资源组]
  PAGE --> R2[UI 组件区域]
  R1 --> UI1[UI 组件]
  R1 --> API1[API 端点]
  R2 --> UI2[UI 组件]
  R2 --> API2[API 端点]
  ROLE[角色] -.->|授予| R1
  ROLE -.->|整页勾选展开| PAGE
```

- **diff 要点**: Plateau v71 → Gap（无 Region 模型/仅 FE）→ WP → Plateau v72
- **full 要点**: FE→GW→Auth/Biz + 三张新表 Access
