# v6 Enterprise Landscape — Mermaid Diagram

> 架构版本: v6 🎯 target | 作者: claude | 日期: 2026-07-06 15:43
> 迭代: Django 项目/任务/阿里云接口 → 3 个 Go 服务拆分

```mermaid
graph TD;
  dev["Developer"];
  admin["Platform Admin"];
  ent["Enterprise Customer"];
  ideSvc["IDE Workspace Service"];
  taskMgmt["Task Management Service"];
  cloudRes["Cloud Resource Service"];
  billingSvc["Billing & License Service"];
  llmSvc["LLM AI Service"];
  projMgmt["Project Management Service"];
  devFlow["Development Workflow"];
  projOnboard["Project Onboarding"];
  project["Project"];
  workspace["Workspace"];
  gw["API Gateway (APISIX)"];
  django["Django SaaS (internal 真源)"];
  authSvc["taskAuth (Go :8003)"];
  gitOauth["Git OAuth Service"];
  sse["Task SSE Service"];
  aiEp["taskAIEndPoint (Go :8013)"];
  credSvc["taskCredentialService (Go :8015)"];
  agt["taskAgentSupport (Go :8011)"];
  cgw["taskContainerGateway (Go :8014)"];
  relay["go-relay (Go :8797)"];
  claude["claude-agent (Go CLI)"];
  tps["taskProjectService (Go :8016)"];
  tts["taskTaskService (Go :8017)"];
  tcs["taskCloudService (Go :8018)"];
  authAPI["Authentication API"];
  projectAPI["Project Management API"];
  taskAPI["Task Management API"];
  cloudAPI["Cloud Resource API"];
  billingAPI["Billing API"];
  llmAPI["LLM Proxy API"];
  containerAPI["Container Proxy API"];
  relayAPI["Relay Lifecycle API"];
  restAPI["REST API"];
  wsAPI["WebSocket / SSE"];
  appNode["Application Host"];
  dbNode["Database Host"];
  cacheNode["Cache Host"];
  docker["Docker Engine"];
  goRuntime["Go Runtime"];
  postgres["PostgreSQL"];
  redis["Redis"];
  kafka["Kafka"];
  llmProvider["Upstream LLM Provider"];
  osjs["onlineServiceJS Container API"];
  platV5["Plateau v5 — Shipped Baseline"];
  platV6["Plateau v6 — 3-Service Go Split Target"];
  gapProj["Gap: Project/Workspace in Django"];
  gapTask["Gap: Task/Comment in Django"];
  gapCloud["Gap: Aliyun API in Django"];
  wp0["WP Phase 0 — Infra"];
  wp1["WP Phase 1 — taskProjectService"];
  wp2["WP Phase 2 — taskTaskService"];
  wp3["WP Phase 3 — taskCloudService"];
  gw --> restAPI;
  tps --|> projectAPI;
  tts --|> taskAPI;
  tcs --|> cloudAPI;
  authSvc --|> authAPI;
  projectAPI --> projMgmt;
  taskAPI --> taskMgmt;
  cloudAPI --> cloudRes;
  appNode --> tps;
  appNode --> tts;
  appNode --> tcs;
  postgres --> tps;
  postgres --> tts;
  postgres --> tcs;
  tts --> tps;
  tcs --> tps;
  tcs --> tts;
  platV5 --> gapProj;
  platV5 --> gapTask;
  platV5 --> gapCloud;
  wp1 --|> gapProj;
  wp2 --|> gapTask;
  wp3 --|> gapCloud;
  platV6 --* tps;
  platV6 --* tts;
  platV6 --* tcs;
  tps --> authSvc;
  tts --> authSvc;
  tcs --> authSvc;
```

## 变更图例

| 标记 | 含义 |
|------|------|
| 🟢 tps/tts/tcs | [NEW v6] 新增 Go 服务 |
| 🟡 django | [MODIFIED v6] 瘦身为 internal 真源 |
| 🔴 gapProj/gapTask/gapCloud | [DEPRECATED v6] Django views 废弃 |

## 架构变迁

```
Plateau v5 (Shipped) ──→ Gap: Project/Workspace ──→ WP Phase 1 (taskProjectService)
                      ──→ Gap: Task/Comment     ──→ WP Phase 2 (taskTaskService)
                      ──→ Gap: Aliyun API       ──→ WP Phase 3 (taskCloudService)
                                                       ↓
                                              Plateau v6 (3-Service Go Split)
```
