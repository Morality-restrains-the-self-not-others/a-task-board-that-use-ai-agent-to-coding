# Application Integration v27 — Terminal Hard Release + Container Migrate

```mermaid
flowchart LR
  Vue["Vue TaskDetail\nprogress PATCH"]
  GW[task-gateway]
  Task["taskTaskService"]
  Kafka[Kafka]
  Events["taskEvents\nmigrate then release"]
  Cloud["taskCloudService\nbindings/migrate/reuse"]
  CGW["taskContainerGateway"]
  ECS[Aliyun ECS]
  Docker["镜像容器"]
  CSC["cloud_server_configs\nterminal_released"]
  C["Constraint\n终态硬释放禁 reuse"]

  Vue -->|PATCH todos| GW
  GW --> Task
  Task -->|TASK_STATUS_CHANGED| Kafka
  Kafka --> Events
  Events -->|bindings+migrate+mark| Cloud
  Cloud -->|W| CSC
  Events -->|CLOUD_SERVER_STOPPED| Kafka
  Cloud -->|stop or start-vm-auto| ECS
  Events --> CGW
  CGW -->|stop+recreate| Docker
  C -.->|排除终态| Cloud
  C -.->|先迁后释| Events
```

## Plateau / Gap

- **Plateau v25**: archived — 自动 SG 白名单
- **Gap closed**: 终态节点仍可 reuse；共享实例误杀外任务容器；无迁回所属任务
- **Plateau v27 ✅ current**: 硬释放 + `terminal_released`；真实 ECS `start-vm-auto`；mock/reuse 后按原镜像 gateway start
