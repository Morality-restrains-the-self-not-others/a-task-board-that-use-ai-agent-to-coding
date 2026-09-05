# DDD 笔记：relay 启动所选镜像

## 限界上下文

- **任务运行时（Relay Local）**：编排本地启动/停止/状态。
- **镜像目录（Cloud）**：TenantInstalledImage 所有权与 resolve。

## 概念

| 概念 | 类型 | 说明 |
|------|------|------|
| InstalledImageId | VO | 租户已安装镜像 ID |
| ContainerImageRef | VO | 可 docker pull 的引用（url+version 合并） |
| RelayStartCommand | 命令 | tenant/workspace/task + env + InstalledImageId |
| RelayRuntime | 实体（侧车进程内） | Running、ContainerID、Image、Logs |

## 领域规则

1. 有 InstalledImageId 时必须先解析出非空 ContainerImageRef，否则拒绝启动。
2. 有 ContainerImageRef 时运行时必须为该镜像的容器，不得改跑 host run.sh。
3. 停止须释放对应容器资源。

## 架构变更影响

- Application Integration v15：Vue → CGW → Cloud(resolve) → go_relayToTrae(docker)。
- 交付：`docs/architecture/v15-application-integration-20260710-2009-claude.{puml,archimate,mermaid.md}`
