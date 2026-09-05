# AI Dev Platform — Application Architecture (Mermaid)
# Generated: 2026-07-01

```mermaid
graph TD;
  platform-root["AI Dev Platform"];
  group-frontend["Frontend Layer"];
  app-vue-frontend["Vue Frontend (:4000)"];
  group-gateway["Gateway Layer"];
  app-gateway["task-gateway APISIX (:18081)"];
  group-platform["Platform Services"];
  app-saas-backend["saas-backend Django (:8001)"];
  app-ai-provider["ai-provider Django (:8010)"];
  app-task-auth["task-auth Go (:8003)"];
  app-git-oauth["git-oauth Django (:8002)"];
  app-task-bill["task-bill Go (:8009)"];
  app-task-sse["task-sse Node.js (:8007)"];
  app-agent-support["task-agent-support Go (:8011)"];
  app-ai-endpoint["task-ai-endpoint Go (:8013)"];
  group-container["Container Stack"];
  app-container-gw["task-container-gateway Go (:8014)"];
  app-credential-svc["task-credential-service Go (:8015)"];
  app-go-relay["go-relay Go (:8797)"];
  app-go-run-container["go-run-container Go"];
  group-events["Domain Events (18 Consumers)"];
  app-events-reg["USER_CREATED→COMPANY→WORKSPACE"];
  app-events-billing["BILLING+SSE (:18020-21)"];
  app-events-notif["EMAIL+INVITATION+ACTIVATION (:18022-26)"];
  app-events-company["COMPANY_FANOUT (:18027-29)"];
  app-events-cloud["CLOUD_SERVER+AI_REPLY (:18030-37)"];
  group-tools["Dev Tools"];
  app-value-stream["valueStream Go (:9998)"];
  platform-root --* group-frontend;
  platform-root --* group-gateway;
  platform-root --* group-platform;
  platform-root --* group-container;
  platform-root --* group-events;
  platform-root --* group-tools;
  group-frontend --* app-vue-frontend;
  group-gateway --* app-gateway;
  group-platform --* app-saas-backend;
  group-platform --* app-ai-provider;
  group-platform --* app-task-auth;
  group-platform --* app-git-oauth;
  group-platform --* app-task-bill;
  group-platform --* app-task-sse;
  group-platform --* app-agent-support;
  group-platform --* app-ai-endpoint;
  group-container --* app-container-gw;
  group-container --* app-credential-svc;
  group-container --* app-go-relay;
  group-container --* app-go-run-container;
  group-events --* app-events-reg;
  group-events --* app-events-billing;
  group-events --* app-events-notif;
  group-events --* app-events-company;
  group-events --* app-events-cloud;
  group-tools --* app-value-stream;
```

## Inter-service Data Flow (supplementary, not in ArchiMate strict model)

```
Vue (:4000) ──HTTP──▶ APISIX (:18081) ──proxy──▶ saas-backend (:8001)
                                                    │
APISIX ──forward-auth──▶ task-auth (:8003)           │
                           auth.sqlite3              │
                                                    ├──▶ task-auth (:8003) [auth checks]
                                                    ├──▶ git-oauth (:8002) [OAuth]
APISIX ──route──▶ task-bill (:8009)                 ├──▶ task-credential (:8015) [tokens]
                   billing.sqlite3                   ├──▶ task-sse (:8007) [SSE push]
                                                    │
saas-backend ──publish──▶ Redis Streams / Kafka     │
                            ▲                        │
                            │                        │
               task-events (18 Go consumers)         │
               :18020-18037                          │
                            │                        │
                            └──HTTP callback─────────┘
```
