# v15 Enterprise Landscape — 任务帖定价模型重构 (Target, 2026-08-07 19:20)

```mermaid
graph TD;
  gateway["API Gateway (APISIX :18081)"];
  billSvc["taskBill (Go :8004) [创建帖/续存计费]"];
  taskTaskSvc["taskTaskService (Go :8017) [帖子存续期 + 续存接口]"];
  taskCloudSvc["taskCloudService (Go :8018) [chargeServerStart 退役]"];
  eventsSvc["taskEvents (Kafka intents)"];
  taskDB["taskTask DB (+post_expires_at)"];
  billDB["taskBill DB"];
  cloudDB["taskCloud DB"];
  kafka["Kafka"];
  plateauV14["Plateau v14 — 多会话互知"];
  plateauV15["Plateau v15 — 任务帖存续期"];
  gapPost["Gap: 帖子无存续期、按次扣费、无续存"];
  wpPost["WP-post-lifecycle"];
  gateway --> taskTaskSvc;
  gateway --> billSvc;
  taskTaskSvc ..> taskDB;
  billSvc ..> billDB;
  taskCloudSvc ..> cloudDB;
  eventsSvc --> kafka;
  plateauV14 --> gapPost;
  wpPost --|> gapPost;
  wpPost --|> plateauV15;
```
