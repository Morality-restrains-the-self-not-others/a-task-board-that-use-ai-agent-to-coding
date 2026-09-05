# Application Integration v26 — Container Image @ Mention

```mermaid
flowchart LR
  Vue["🟡 Vue TaskDetail\n+ TaskPanel Settings"]
  GW[task-gateway]
  Proj["🟡 taskProjectService\nworkspaces flag"]
  Task["🟡 taskTaskService\ncomments+mentions"]
  AIC["🟡 taskAIComment\ncontainer_agent_comments"]
  Cloud["taskCloudService\nstart-vm + idle reuse"]
  OSJS["🟡 onlineServiceJS\nContextPack"]
  Kafka[Kafka]
  SSE[Task SSE]
  ECS[Aliyun ECS / idle pool]
  Flag["🟢 container_image_at_mode_enabled"]
  CAC["🟢 container_agent_comments"]
  C["🟢 Constraint\n@模式默认关闭"]

  Vue -->|PATCH/POST| GW
  GW --> Proj
  Proj -->|W| Flag
  GW --> Task
  Task --> Kafka
  Kafka --> AIC
  AIC --> Cloud
  Cloud --> ECS
  AIC --> OSJS
  OSJS -->|stream reply| AIC
  AIC --> SSE
  SSE --> Vue
  AIC -->|W| CAC
  C -.->|硬闸| Task
  C -.->|无 picker| Vue
```

## Plateau / Gap

- **Plateau v25**: 自动 SG 入网白名单已交付；无评论 `@` 镜像能力
- **Gap**: 无工作空间 `@` 开关；无镜像 mention 开机器；无 ContainerAgentComment SSE 回写；评论区与 Agent 路径未统一
- **Plateau v26**: 开关 + `@已安装镜像` → idle reuse 开跑 + ContextPack + 容器 Agent 流式回写统一 Feed
