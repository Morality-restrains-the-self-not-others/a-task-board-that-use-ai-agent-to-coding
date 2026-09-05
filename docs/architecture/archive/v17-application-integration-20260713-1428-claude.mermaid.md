# Application Integration v17 — 任务终态自动释放服务器

```mermaid
flowchart LR
  Vue[Vue TaskDetail / WorkPanel] -->|PATCH todos| GW[task-gateway]
  GW --> TTS[taskTaskService]
  TTS -->|TASK_STATUS_CHANGED| Kafka[Kafka task-status-changed]
  Kafka --> TEV[taskEvents release_servers]
  TEV -->|load config| TCS[taskCloudService]
  TEV -->|CLOUD_SERVER_STOPPED| KafkaStop[Kafka cloud-server-stopped]
  KafkaStop --> CSS[cloudserverstopped handler]
  TEV -->|HTTP stop| CGW[taskContainerGateway]
```

## 变更摘要

- 🟢 Kafka topic `task-status-changed` + 事件 `TASK_STATUS_CHANGED`
- 🟢 taskEvents intent：终态时释放 ECS / relay / mock
- 🟡 taskTaskService：状态字段变化后发事件
- 复用既有 `CLOUD_SERVER_STOPPED` 与 container stop 路径
