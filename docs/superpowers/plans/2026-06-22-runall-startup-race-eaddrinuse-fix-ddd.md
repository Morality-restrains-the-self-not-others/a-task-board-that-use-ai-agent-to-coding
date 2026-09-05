# DDD 领域模型: runAll 启动竞态 EADDRINUSE 修复

> 输入:
> - 设计文档: `.claude/plans/01-brainstorming-设计文档.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-22-runall-startup-race-eaddrinuse-fix-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-22-runall-startup-race-eaddrinuse-fix-nfr-clarification.md`
>
> 输出使用者: `/6-plans-实施计划`, `/7-build-构建`

## 判定: 基础设施修复 — 无新增领域概念

本次变更为纯粹的 **基础设施层/orchestration 层 bug 修复**，不引入新的领域概念。
现有领域模型完全支撑修复所需。

### 现有领域模型覆盖验证

| 修复项 | 涉及的现有领域概念 | 所在文件 | 状态 |
|--------|------------------|---------|------|
| startAndCheck CAS 守卫 | `ManagedService.IsStartableServiceStatus()` — 已正确定义可启动状态 | `domain/managed_service_entity.go:51` | ✅ 无需变更 |
| startAndCheck CAS 守卫 | `transitionServiceToStarting()` — CAS 原子操作在 infrastructure 层 (`runner.go`) | `runner.go:1297` | ✅ 已存在，只需在 `startAndCheck` 中调用 |
| killPreviousRunAllProcess | `ServicePreflightDomainService.EnsurePortsReadyForLaunch()` — 已支持端口冲突检测和清理 | `domain/service_preflight_domain_service.go:151` | ✅ 无需变更 |
| killPreviousRunAllProcess | `PortListenerProbeRepository` — 已定义端口检测接口 | `domain/port_listener_probe_repository.go` | ✅ 无需变更 |
| SO_REUSEADDR | 基础设施层修复（Python WSGI 配置），无领域概念涉及 | `gitOauth/run.sh` | ✅ 与领域层无关 |
| listenerPIDs 错误处理 | `PortListenerProbeRepository` — 接口已定义，修复在调用方的错误处理 | `domain/port_listener_probe_repository.go` | ✅ 无需变更 |

### 分层归属

```
┌─────────────────────────────────────────┐
│  Infrastructure / Orchestration 层       │  ← 本次修复所在层
│  runner.go: startAndCheck (CAS 守卫)     │
│  main.go: killPreviousRunAllProcess     │
│  runner.go: listenerPIDs 错误处理        │
│  gitOauth/run.sh: SO_REUSEADDR          │
├─────────────────────────────────────────┤
│  Domain 层 (不变)                        │
│  ManagedService, ServiceStatus,         │
│  PortListenerProbeRepository,           │
│  ServicePreflightDomainService          │
└─────────────────────────────────────────┘
```

## 关键领域不变量（现有，本次修复保护）

| 不变量 | 说明 | 修复如何保护 |
|--------|------|------------|
| 单服务单启动实例 | 同一服务同时只有一个 launch 在执行 | CAS 守卫确保 `pending→starting` 原子转换 |
| 端口独占 | 一个端口不能被两个进程同时监听 | preflight + listenerPIDs + SO_REUSEADDR 三重保护 |
| 所有者会话一致 | 每个运行中的服务属于唯一的 runAll 会话 | `ServiceOwnershipGuardService` 不变 |

## 限界上下文（现有，无需变更）

```
┌──────────────────────┐     ┌──────────────────────┐
│  Service Orchestration│     │  OAuth Authentication │
│  (runAll)             │     │  (gitOauth)            │
│                       │     │                        │
│  ManagedService       │     │  OAuthCredentialBinding│
│  ServiceStatus        │     │  OAuthProviderRouteRule│
│  ServiceOwnership     │     │  (基础设施修复:          │
│  PortConflictDetector │     │   SO_REUSEADDR)        │
└──────────────────────┘     └──────────────────────┘
```

## 结论

DDD 步骤确认：现有领域模型无需变更。所有修复均在 infrastructure/orchestration 层完成。
进入步骤 6（实施计划）。
