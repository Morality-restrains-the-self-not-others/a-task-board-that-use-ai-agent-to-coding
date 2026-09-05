# Application Integration v25 — Auto SG Ingress Whitelist

```mermaid
flowchart LR
  Vue[Vue TaskDetail]
  GW[task-gateway]
  Cloud["🟡 taskCloudService\nresolveClientIP"]
  Evt["🟢 client_public_ip\nauto_sg_whitelist"]
  Kafka[Kafka CLOUD_SERVER_*]
  TE["🟡 taskEvents\nPhase A/B whitelist"]
  SG[Aliyun ECS SG]
  C["🟢 Constraint\n默认拒绝未授权入站"]

  Vue -->|start-vm-auto| GW
  GW --> Cloud
  Cloud -->|W| Evt
  Cloud --> Kafka
  Kafka --> TE
  TE -->|Authorize/Revoke| SG
  C -.->|约束| TE
```

## Plateau / Gap

- **Plateau v21**: cloud events/IAM 已在 taskCloud；自动 SG 仍全开 `0.0.0.0/0`
- **Plateau v24**: 导航栏多账号切换（并行 target，不改 SG 路径）
- **Gap**: 自动创建安全组入站全开；start-vm-auto 未透传用户公网 IP
- **Plateau v25**: 两阶段入网白名单（用户 IP + 服务器 IP）；撤销全开入站；无新服务组件
