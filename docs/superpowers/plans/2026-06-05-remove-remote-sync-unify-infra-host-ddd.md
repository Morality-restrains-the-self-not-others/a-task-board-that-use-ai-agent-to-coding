# DDD 领域建模: 移除远程同步，统一 INFRA_HOST

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-05-remove-remote-sync-unify-infra-host-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-05-remove-remote-sync-unify-infra-host-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-05-remove-remote-sync-unify-infra-host-nfr-clarification.md`
>
> 输出使用者: `/6-plans-实施计划`, `/7-build-构建`

## 限界上下文

不新增或修改限界上下文。runAll 的「基础设施编排」上下文保持不变，仅移除其中的远程概念。

## 领域模型变更

### 删除

| 文件 | 类型 | 原因 |
|------|------|------|
| `domain/infra_management_panel_url_value_object.go` | 值对象 | `BuildInfraManagementPanelURL` + `IsValidInfraManagementPanelURL` — Portainer URL 构建，仅被删除的 `remote_docker.go` 使用 |
| `domain/infra_management_panel_url_value_object_test.go` | 测试 | 对应测试 |
| `domain/tcp_probe_endpoint_value_object.go` | 值对象 | `ResolveServiceManagementURL` — TCP→Portainer 链接映射，仅被 `ui.go` 的 `management_url` 字段使用（该字段同步移除） |
| `domain/tcp_probe_endpoint_value_object_test.go` | 测试 | 对应测试 |
| `../remote_docker.go` | 基础设施 | RemoteDocker 结构体 + 远程同步函数，不属于 domain/ 但也在此列出以确保完整 |

### 不变

所有其他 domain/ 文件保持不变，包括：
- `managed_service_entity.go` — 服务实体不区分本地/远程
- `service_lifecycle_domain_events.go` — 无 RemoteSync 相关事件
- `service_cascade_orchestration_service.go` — 级联编排逻辑不变
- `port_resolver_service.go` — 端口解析不变
- 其余所有实体、值对象、仓储、事件

## 领域事件

无新增事件。无移除事件（`RemoteSyncCompleted` 不存在于代码库中）。

## 仓储接口

无变更。`ServiceOwnershipRepository`、`ServiceTopologyRepository` 等保持原样。

## 领域服务

无新增。移除影响：
- `BuildInfraManagementPanelURL` 调用（`remote_docker.go:54`）随文件删除
- `ResolveServiceManagementURL` 调用（`ui.go:357`）随 management_url 移除
- `resolveConfApps`（`config.go`）重写为统一 `${INFRA_HOST}` 解析——这是基础设施层变更，不影响领域接口

## NFR 决策对应的建模动作

| NFR 决策 | 建模动作 | 状态 |
|----------|---------|------|
| 可维护性 L2 — 每服务独立配置 | 移除 `RemoteDocker` 值对象，`Service` 实体通过 `conf_app` 持有配置引用 | `conf_app` 字段已存在，无需新增 |
| 可维护性 L2 — 变量统一 | 模板解析合并到单一 `resolveConfApps` 函数 | `config.go` 基础设施层变更 |
| 容错 L1 — 移除远程故障点 | 删除 infra 管理面板 URL 值对象 | 本次删除 2 个 VO 文件 |

## 自检

- [x] 领域文件删除列表明确，无遗漏
- [x] 无新增领域文件，无基础设施导入问题
- [x] 远程概念从领域层完全移除
- [x] 无跨聚合直接调用变更
- [x] 事件流保持不变
- [x] 仓储接口保持不变
