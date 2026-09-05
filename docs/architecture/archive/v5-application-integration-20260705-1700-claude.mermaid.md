# Application Integration — v5 Target Architecture

> **状态**: 🎯 target | **版本**: v5 | **迭代**: relay lifecycle 公网绕 Django + 副作用事件消费者 | **基于**: v4  
> **设计日期**: 2026-07-05 17:00 | **交付**: 2026-07-05

## 架构变迁总览 (v4 → v5)

```mermaid
flowchart TB
  subgraph v4_target ["🎯 v4 Target — lifecycle 仍经 Django proxy"]
    V4_VUE["Vue Frontend"]
    V4_GW["task-gateway"]
    V4_CGW["taskContainerGateway"]
    V4_BE["saas-backend\n(relay_to_trae_proxy 公网)"]
    V4_RELAY["go-relay"]
    V4_VUE --> V4_GW
    V4_GW --> V4_BE
    V4_BE --> V4_RELAY
  end

  subgraph v5_target ["🎯 v5 Target — 同步 tcg + 异步事件副作用"]
    V5_VUE["Vue Frontend"]
    V5_GW["task-gateway"]
    V5_CGW["taskContainerGateway\nregister/start/stop 编排"]
    V5_RELAY["go-relay"]
    V5_KAFKA["Kafka relay-lifecycle"]
    V5_TE["taskEvents\n:18039-18042"]
    V5_BE["saas-backend\ninternal APIs 真源"]
    V5_CRED["taskCredentialService\naudit SSOT"]
    V5_SSE["taskSSE"]

    V5_VUE --> V5_GW
    V5_GW --> V5_CGW
    V5_CGW -->|"同步 forward"| V5_RELAY
    V5_CGW -->|"RELAY_* 发布"| V5_KAFKA
    V5_KAFKA --> V5_TE
    V5_TE -->|"open-session / clear-reachability / workflow"| V5_BE
    V5_TE -->|"append audit"| V5_CRED
    V5_CGW -->|"dispatch 失败 SSE"| V5_BE
    V5_BE --> V5_SSE
  end

  v4_target -.->|"Inc 1-4"| v5_target
```

## Relay lifecycle 数据流

```mermaid
sequenceDiagram
  participant Browser
  participant GW as task-gateway
  participant TCG as taskContainerGateway
  participant Relay as go-relay
  participant Kafka
  participant TE as taskEvents
  participant BE as saas-backend internal
  participant CRED as taskCredentialService

  Browser->>GW: POST relay-to-trae/start
  GW->>TCG: proxy register/start/stop
  TCG->>Relay: /v1/start (sync)
  Relay-->>TCG: 200/202
  TCG->>Kafka: RELAY_START_ACCEPTED
  par 异步副作用
    Kafka->>TE: audit consumer
    TE->>CRED: POST /v1/audit/append
    Kafka->>TE: open_session consumer
    TE->>BE: POST open-runtime-session
    Kafka->>TE: workflow consumer
    TE->>BE: POST relay-workflow/transition
  end
  Note over Browser,TCG: stop 路径同理 → RELAY_STOP_SUCCEEDED → clear-reachability
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
  subgraph ContainerStack
    CGW[task-container-gateway :8014]
    CRED[task-credential-service :8015]
    RELAY[go-relay :8797]
  end
  subgraph Events
    KFK[Kafka relay-lifecycle]
    TE[taskEvents :18039-18042]
  end
  subgraph Platform
    BE[saas-backend :8001 internal]
    SSE[task-sse :8007]
  end

  VUE --> GW
  GW --> CGW
  CGW --> RELAY
  CGW --> KFK
  KFK --> TE
  TE --> BE
  TE --> CRED
  CGW -->|"SSE_MESSAGE"| KFK
  BE --> SSE
```

## v5 变更明细

| 标记 | 组件/关系 | 说明 |
|------|-----------|------|
| 🟡 MOD | taskContainerGateway | relay register/start/stop 同步编排 + Kafka RELAY_* |
| 🟢 NEW | taskEvents relay-lifecycle | audit / open-session / clear-reachability / workflow |
| 🟢 NEW | Django internal APIs | open-runtime-session、clear-reachability、workflow、relay SSE |
| 🟡 MOD | taskCredentialService | POST /v1/audit/append（audit SSOT） |
| 🔴 DEP | relay_to_trae_proxy 公网 | 410 stub；`TASK_CONTAINER_GATEWAY_ENABLED=false` 回滚 |
