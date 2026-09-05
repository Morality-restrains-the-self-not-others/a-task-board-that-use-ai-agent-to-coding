# Application Integration — v4 Target Architecture

> **状态**: 🎯 target | **版本**: v4 | **迭代**: task2app 接口 Go 拆分 | **基于**: v3  
> **设计日期**: 2026-07-05 15:31

## 架构变迁总览 (v1 → v4)

```mermaid
flowchart TB
  subgraph v1_current ["✅ v1 Current — 拥塞风险"]
    V1_BE["saas-backend Django\n(全部 API 单进程)"]
    V1_OS["onlineServiceJS"]
    V1_BE -->|"requests.Session 出站"| V1_OS
    V1_OS -->|"inbound 直打 Django"| V1_BE
  end

  subgraph v4_target ["🎯 v4 Target — 流量隔离"]
    V4_GW["task-gateway APISIX"]
    V4_CGW["taskContainerGateway Go"]
    V4_RELAY["go-relay Go"]
    V4_AGT["taskAgentSupport Go"]
    V4_CRED["taskCredentialService Go"]
    V4_BE["saas-backend Django\n(CRUD + internal 真源)"]
    V4_SSE["taskSSE Node"]
    V4_OS["onlineServiceJS"]
    V4_VUE["Vue Frontend"]

    V4_VUE --> V4_GW
    V4_VUE --> V4_CGW
    V4_GW --> V4_RELAY
    V4_GW --> V4_BE
    V4_CGW -->|"validate-session"| V4_BE
    V4_CGW -->|"L0/L2 outbound"| V4_OS
    V4_OS --> V4_AGT
    V4_OS --> V4_CRED
    V4_AGT --> V4_BE
    V4_AGT --> V4_SSE
  end

  v1_current -.->|"Phase 0-2 迁移"| v4_target
```

## 组件拓扑

```mermaid
flowchart LR
  subgraph Frontend
    VUE[Vue :4000]
  end
  subgraph Gateway
    GW[task-gateway :18081]
  end
  subgraph Platform
    BE[saas-backend :8001]
    AUTH[task-auth :8003]
    BILL[task-bill :8009]
    SSE[task-sse :8007]
    AIEP[task-ai-endpoint :8013]
  end
  subgraph ContainerStack
    CGW[task-container-gateway :8014]
    AGT[task-agent-support :8011]
    CRED[task-credential-service :8015]
    RELAY[go-relay :8797]
  end
  subgraph External
    OSJS[onlineServiceJS :8765]
    LLM[Upstream LLM]
  end

  VUE --> GW
  VUE --> CGW
  GW --> BE
  GW --> RELAY
  CGW --> BE
  CGW --> OSJS
  OSJS --> AGT
  OSJS --> CRED
  AGT --> BE
  AGT --> SSE
  AIEP --> BE
  AIEP --> CRED
  AIEP --> LLM
  RELAY --> OSJS
```

## v4 变更明细

| 标记 | 组件/关系 | 说明 |
|------|-----------|------|
| 🟢 NEW | onlineServiceJS 显式标注 | 容器 HTTP API 真源 |
| 🟡 MOD | taskContainerGateway | 完整 compute outbound + job-stream |
| 🟡 MOD | go-relay | 浏览器 relay 公网入口 |
| 🟡 MOD | taskAgentSupport | inbound 业务逻辑下沉 |
| 🟡 MOD | saas-backend | 瘦身为 CRUD + internal |
| 🔴 DEP | Django inline SSE | → taskSSE |
| 🔴 DEP | relay_to_trae_proxy | → go-relay 直路由 |
