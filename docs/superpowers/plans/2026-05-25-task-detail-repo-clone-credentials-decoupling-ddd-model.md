# DDD Model: Task Detail 与 Repo Clone Credentials 解耦

## 1) 需求边界

- 输入来源：
  - `docs/superpowers/specs/2026-05-24-task-detail-repo-clone-credentials-design.md`
  - `docs/superpowers/plans/2026-05-25-task-detail-repo-clone-credentials-decoupling-value-stream.md`
  - `docs/superpowers/plans/2026-05-25-task-detail-repo-clone-credentials-decoupling-nfr-clarification.md`
- 当前增量目标：将 `task-detail`（仓库列表读取）与 `repo-clone-credentials`（凭证获取与完整性判定）解耦为两阶段流程。
- 非目标：不改 OAuth 发证主链路，不改数据库结构。

## 2) 限界上下文 (Bounded Context)

- **TaskDetailReadContext**：只负责以 `TaskScope` 为边界读取 task-detail 元数据与 `project_repo_urls`。
- **RepoCloneCredentialsContext**：负责凭证拉取结果判定、缺失仓库识别与失败事件化。
- **ContainerTokenAuthContext**：负责“令牌 + 路径作用域”授权不变量（由值对象表达，不下沉基础设施细节）。

## 3) 实体与值对象

### 实体 (Entity)

- `TaskDetailReadModel`（新增）
  - 文件：`task2app/Saas_project/cloud/domain/entities/task_detail_read_model.py`
  - 角色：两阶段调用中的只读任务模型，显式不包含凭证字段。
  - 标识：`scope.to_identity_key()` + `occurred_at` 快照语义。
- `TaskRepoCloneCredentialsContract`（复用）
  - 文件：`task2app/Saas_project/cloud/domain/entities/task_repo_clone_credentials_contract.py`
  - 角色：凭证覆盖契约聚合根，负责从覆盖快照生成“不完整”领域事件。

### 值对象 (Value Object)

- `ContainerTokenContext`（新增）
  - 文件：`task2app/Saas_project/cloud/domain/value_objects/container_token_context.py`
  - 角色：封装 `TaskScope + AccessToken`，统一新旧接口授权边界输入。
- `RepoCloneCredentialCoverage`（复用）
  - 文件：`task2app/Saas_project/cloud/domain/value_objects/repo_clone_credential_coverage.py`
  - 角色：标准化期望仓库与已下发凭证仓库，计算差集。
- `TaskScope`（复用）
  - 文件：`task2app/Saas_project/cloud/domain/value_objects/task_scope.py`
  - 角色：租户/工作空间/任务作用域主键。

## 4) 聚合与聚合根

- 聚合 A：**TaskDetailRead 聚合**
  - 聚合根：`TaskDetailReadModel`
  - 一致性边界：`project_repo_urls` 必须归一化且仅表达可克隆仓库集合。
- 聚合 B：**RepoCloneCredentialsContract 聚合**
  - 聚合根：`TaskRepoCloneCredentialsContract`
  - 一致性边界：`expected_repo_urls` 与 `credential_repo_urls` 差集为空才视为成功。

> NFR 引用：数据一致性 L2，要求差集非空时必须稳定映射为 `REPO_CLONE_CREDENTIALS_INCOMPLETE`。

## 5) 领域服务

- `RepoCloneCredentialsFetchService`（新增）
  - 文件：`task2app/Saas_project/cloud/domain/services/repo_clone_credentials_fetch_service.py`
  - 角色：两阶段凭证拉取编排服务。
  - 关键方法：
    - `mark_attempted(...)`：发出 attempted 事件
    - `evaluate_result(...)`：复用 `TaskRepoCloneCredentialsGuardService` 做差集判定，并返回 succeeded/failed 事件
- `TaskRepoCloneCredentialsGuardService`（复用）
  - 文件：`task2app/Saas_project/cloud/domain/services/task_repo_clone_credentials_guard_service.py`
  - 角色：覆盖完整性守卫（构造契约 + 生成不完整事件）。

## 6) 仓储接口

- `TaskDetailReadModelRepository`（新增）
  - 文件：`task2app/Saas_project/cloud/domain/repositories/task_detail_read_model_repository.py`
  - 契约：`fetch_by_token_context(token_context)`。
- `TaskRepoCloneCredentialsRepository`（复用，语义更新）
  - 文件：`task2app/Saas_project/cloud/domain/repositories/task_repo_clone_credentials_repository.py`
  - 契约：
    - `list_expected_repo_urls(scope)`
    - `list_credential_repo_urls(scope)`（语义更新为 repo-clone-credentials 来源）

## 7) 领域事件

- `RepoCloneCredentialsFetchAttempted`（新增）
  - 文件：`task2app/Saas_project/cloud/domain/events/repo_clone_credentials_fetch_attempted.py`
- `RepoCloneCredentialsFetchSucceeded`（新增）
  - 文件：`task2app/Saas_project/cloud/domain/events/repo_clone_credentials_fetch_succeeded.py`
- `RepoCloneCredentialsFetchFailed`（新增）
  - 文件：`task2app/Saas_project/cloud/domain/events/repo_clone_credentials_fetch_failed.py`
- `RepoCloneCredentialsIncompleteDetected`（复用）
  - 文件：`task2app/Saas_project/cloud/domain/events/repo_clone_credentials_incomplete_detected.py`

> NFR 引用：可观测性 L2，要求两阶段调用必须可事件化追踪。

## 8) 本次领域文件变更清单

### 新增

- `cloud/domain/value_objects/container_token_context.py`
- `cloud/domain/entities/task_detail_read_model.py`
- `cloud/domain/repositories/task_detail_read_model_repository.py`
- `cloud/domain/events/repo_clone_credentials_fetch_attempted.py`
- `cloud/domain/events/repo_clone_credentials_fetch_succeeded.py`
- `cloud/domain/events/repo_clone_credentials_fetch_failed.py`
- `cloud/domain/services/repo_clone_credentials_fetch_service.py`

### 更新

- `cloud/domain/value_objects/__init__.py`
- `cloud/domain/entities/__init__.py`
- `cloud/domain/repositories/__init__.py`
- `cloud/domain/events/__init__.py`
- `cloud/domain/services/__init__.py`
- `cloud/domain/repositories/task_repo_clone_credentials_repository.py`

## 9) Hard Gate 自检

- [x] 所有新增实体/值对象/仓储/服务/事件位于 `domain/` 目录
- [x] 领域层无 ORM 导入
- [x] 领域层无外部服务导入（HTTP/Kafka/云 SDK）
- [x] 仓储接口使用 ABC 抽象
- [x] 每个新增聚合根有仓储接口（`TaskDetailReadModel` -> `TaskDetailReadModelRepository`）
- [x] 领域事件采用过去式命名（Attempted/Succeeded/Failed/Detected）
- [x] 领域服务通过构造函数注入依赖（`RepoCloneCredentialsFetchService`）
- [x] 无持久化细节（`null=True`、`max_length` 等）渗入领域层

---

领域模型已生成到 `task2app/Saas_project/cloud/domain/`。仓储接口与领域服务契约已就绪，可用于 `/6-plans-实施计划` 生成可验证任务清单。
