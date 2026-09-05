# v6 Application Integration — Mermaid Diagram

> 架构版本: v6 🎯 target | 作者: claude | 日期: 2026-07-06 15:43
> 迭代: Django 项目/任务/阿里云接口 → 3 个 Go 服务拆分

```mermaid
graph TD;
  VUE["Vue Frontend (:4000)"];
  GW["APISIX Gateway (:18081)"];
  BE["saas-backend Django (:8001) [internal 真源]"];
  AUTH["taskAuth Go (:8003)"];
  GOAUTH["git-oauth Django (:8002)"];
  BILL["task-bill Go (:8009)"];
  SSE["task-sse Node.js (:8007)"];
  AGT["task-agent-support Go (:8011)"];
  AIEP["task-ai-endpoint Go (:8013)"];
  TPS["🟢 task-project-service Go (:8016)"];
  TTS["🟢 task-task-service Go (:8017)"];
  TCS["🟢 task-cloud-service Go (:8018)"];
  CGW["task-container-gateway Go (:8014)"];
  CRED["task-credential-service Go (:8015)"];
  RELAY["go-relay Go (:8797)"];
  RUN["go-run-container Go"];
  OSJS["onlineServiceJS (:8765)"];
  LLM["Upstream LLM (DeepSeek/OpenAI)"];
  ALI["Aliyun ECS API"];
  VUE --> GW;
  GW --> TPS;
  GW --> TTS;
  GW --> TCS;
  TPS --> AUTH;
  TTS --> AUTH;
  TCS --> AUTH;
  TPS --> BE;
  TTS --> BE;
  TCS --> BE;
  TTS --> TPS;
  TCS --> TPS;
  TCS --> TTS;
  TCS --> ALI;
  CGW --> OSJS;
  CGW --> BE;
  OSJS --> AGT;
  OSJS --> CRED;
  AGT --> BE;
  AGT --> SSE;
  AIEP --> LLM;
  AIEP --> CRED;
```

## v6 新增数据流

| 数据流 | 说明 |
|--------|------|
| GW → TPS | 项目/工作空间 API 路由 |
| GW → TTS | 任务/评论 API 路由 |
| GW → TCS | 云平台 API 路由 |
| TPS/TTS/TCS → AUTH | JWT 验证 |
| TPS/TTS/TCS → BE | company 存在性校验 (Django internal) |
| TTS → TPS | workspace 存在性校验 |
| TCS → TPS | workspace 默认配置 |
| TCS → TTS | 实例状态回调 |
| TCS → ALI | ECS/VPC API (aliyun-sdk-go) |

## 变更图例

| 标记 | 含义 |
|------|------|
| 🟢 TPS/TTS/TCS | [NEW v6] 新增 3 个 Go 服务 |
| 🟡 BE | [MODIFIED v6] Django 瘦身 |
| 🔴 (已移除) | [DEPRECATED v6] Django views/provider |
