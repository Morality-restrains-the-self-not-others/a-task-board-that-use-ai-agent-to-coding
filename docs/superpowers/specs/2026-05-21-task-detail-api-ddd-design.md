# Task Detail API DDD 设计（task-detail 主链路）

## 背景与目标
- 目标链路：`task-detail` 主链路（详情读写）+ `container-task-ui-context` + `relay-to-trae status-push`。
- 本轮优先收敛 `task2app/Saas_project/cloud/` 的应用层与领域层边界，不做全量跨模块迁移。
- 约束：领域层不依赖 ORM/HTTP SDK；跨上下文通过应用服务与领域事件协同。

## 边界与职责
- `TaskDetailReadModel`：承载任务详情展示所需读模型契约，关注 task detail 的查询/补丁语义（本轮以契约定义为主）。
- `TaskContainerRuntime`：负责容器运行态上下文（是否可达、容器页面 URL、VSCode URL）投影与事件化。
- `RelayStartupOrchestration`：负责 relay 两阶段启动（token-init -> start-dispatch）与 status 收敛。

## Domain Model

### Bounded Contexts
- `TaskDetailReadModel`
- `TaskContainerRuntime`
- `RelayStartupOrchestration`

### Aggregates
- `TaskContainerRuntimeSession`（新增抽象聚合能力，当前以 `ContainerRuntimeContextSnapshot` + 领域服务实现最小切片）
- `RelayStartupSession`（既有聚合，继续作为 relay 启动收敛核心）

### Domain Events
- `TaskDetailPatched`（任务详情 patch 语义事件，字段归一化）
- `ContainerUiContextRefreshed`（容器 UI 上下文刷新）
- `RelayStatusConverged`（relay 状态收敛，既有）

### Repository Interfaces
- `ContainerRuntimeContextRepository.find_latest_by_scope(scope)`：读取任务 runtime 上下文快照。
- `RelayStartupSessionRepository.find_by_workflow_id/find_latest_by_scope/save`：管理启动会话。
- `ContainerTokenSessionRepository` 与 `ContainerTokenAuditEventRepository` 保持既有契约。

## 应用层编排
- `TaskContainerRuntimeContextAppService`：
  - 通过 `ContainerRuntimeContextRepository` 读取快照；
  - 调用 `TaskContainerRuntimeService` 触发 `ContainerUiContextRefreshed`；
  - 返回稳定 API payload，避免视图层直连 ORM。
- `relay_startup_app_service`：
  - 托管 `RelayTwoStepStartupService` 单例与进程内仓储；
  - 解除 `relay_to_trae_status` 对 `relay_to_trae_proxy` 内部实现的反向依赖。

## API 影响与兼容性
- `GET .../cloud/compute/container-task-ui-context` 响应字段保持兼容：
  - `status` / `has_server_config` / `container_endpoint_registered` / `container_page_url` / `container_vscode_url`。
- `relay-to-trae/status-push` 外部协议保持不变，仅内部收敛路径改为应用服务共享。

## 不变量与约束
- 容器 UI 上下文无 server config 时必须返回空快照（不抛 500）。
- `RelayStartupSession` 仅允许在 token-init 完成后受理 start；status 序列号不可为负。
- 领域对象仅持有值对象/实体，不导入 Django Model。

## 测试策略（TDD）
- 领域测试：
  - `ContainerRuntimeContextSnapshot.empty()` 默认值；
  - `TaskContainerRuntimeService.refresh_context()` 产出 `ContainerUiContextRefreshed`；
  - `TaskDetailPatched` patched fields 归一化。
- 应用层测试：
  - `TaskContainerRuntimeContextAppService` 输出 payload 与快照一致。
- 回归测试：
  - 既有 `container-task-ui-context` 与 `relay status-push` 相关用例继续通过。
