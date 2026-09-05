# Enterprise Landscape — v4 Target Architecture

> **状态**: 🎯 target | **版本**: v4 | **迭代**: task2app 接口 Go 拆分 | **基于**: v3  
> **设计日期**: 2026-07-05 15:31

## 三层视图 + 架构变迁

```mermaid
flowchart TB
  subgraph Business ["Business Layer"]
    DEV[Developer]
    CLOUD[Cloud Resource Service]
    IDE[IDE Workspace Service]
  end

  subgraph Application ["Application Layer"]
    GW[API Gateway]
    DJANGO[Django SaaS CRUD+internal]
    CGW[taskContainerGateway]
    AGT[taskAgentSupport]
    RELAY[go-relay]
    PROXY_API[Container Proxy API]
    RELAY_API[Relay Lifecycle API]
  end

  subgraph Migration ["Implementation & Migration"]
    P1[Plateau v1 Current]
    P4[Plateau v4 Target]
    G1[Gap: outbound in Django]
    G2[Gap: relay self-call]
    WP1[WP Phase 1 Gateway]
  end

  DEV --> IDE
  CGW --> PROXY_API
  RELAY --> RELAY_API
  PROXY_API --> CLOUD
  RELAY_API --> IDE
  P1 --> G1
  P1 --> G2
  WP1 --> G1
  WP1 --> P4
```

## 架构变迁 Plateau 对照

| Plateau | 状态 | 容器出站 | relay 入口 | Django 职责 |
|---------|------|----------|------------|-------------|
| v1 Current | ✅ 已交付 | Django requests | Django proxy | 全量 API |
| v4 Target | 🎯 本次 | taskContainerGateway | go-relay 直路由 | CRUD + internal |
