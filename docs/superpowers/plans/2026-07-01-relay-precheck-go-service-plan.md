# 实施计划: 容器令牌基础设施 Go 服务迁移

> 输入:
> - 设计: `docs/superpowers/specs/2026-07-01-relay-precheck-go-service-design.md`
> - 价值流: `docs/superpowers/plans/2026-07-01-relay-precheck-go-service-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-07-01-relay-precheck-go-service-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-07-01-relay-precheck-go-service-ddd-model.md`

---

## Increment 1: Go 服务骨架 + token init（Thin Slice）

### 1.1 Go 项目基础

- [ ] **1.1.1** 创建 Go module + build.sh  
  `taskCredentialService/go.mod`, `taskCredentialService/build.sh`

- [ ] **1.1.2** 创建 DDL migration  
  `taskCredentialService/migrations/001_create_tables.sql`

- [ ] **1.1.3** 创建领域层实体与值对象  
  `taskCredentialService/domain/entities.go`

- [ ] **1.1.4** 创建领域事件  
  `taskCredentialService/domain/events.go`

- [ ] **1.1.5** 创建端口接口  
  `taskCredentialService/ports/repositories.go`

### 1.2 Go 基础设施适配器

- [ ] **1.2.1** 实现 SQLite Token 仓储 + Audit 仓储  
  `taskCredentialService/infrastructure/sqlite_tokens.go`

- [ ] **1.2.2** 实现 SQLite 业务数据仓储（saas.sqlite3 只读）  
  `taskCredentialService/infrastructure/sqlite_business.go`

- [ ] **1.2.3** 实现 gitOauth HTTP 客户端  
  `taskCredentialService/infrastructure/gitoauth_client.go`

- [ ] **1.2.4** 实现 DI 组装（Composition Root）  
  `taskCredentialService/infrastructure/composition.go`

### 1.3 Go 应用服务

- [ ] **1.3.1** 实现 TokenService（IssueToken / ValidateToken）  
  `taskCredentialService/application/services.go`

### 1.4 Go HTTP 接口

- [ ] **1.4.1** 实现 HTTP handlers + 路由注册  
  `taskCredentialService/interfaces/handlers.go`

- [ ] **1.4.2** 实现 main.go 入口  
  `taskCredentialService/cmd/main.go`

### 1.5 Go 单元测试

- [ ] **1.5.1** Token 签发+校验测试  
  `taskCredentialService/src/handlers_test.go` → `TestHandleTokenInit`, `TestValidateToken`

- [ ] **1.5.2** SQLite 仓储测试（用 :memory: DB）  
  `taskCredentialService/src/sqlite_test.go`

### 1.6 runAll 集成

- [ ] **1.6.1** 创建 Go 服务配置  
  `conf/container/task-credential-service/config.yaml`

- [ ] **1.6.2** runAll.yaml 新增 `task-credential-service` service 条目  
  添加 `build_command`, `start_command`, `depends_on`, `health_check`

- [ ] **1.6.3** `db/registry.yaml` 新增 `container` DB 注册  
  `path: db/container/tokens.sqlite3`, `owner: task-credential-service`

---

## Increment 2: 组 A 查询端点迁移（死锁消除）

### 2.1 Go 凭证服务

- [ ] **2.1.1** 实现 CredentialService（BuildRepoCloneCredentials）  
  `taskCredentialService/application/services.go`（追加）

- [ ] **2.1.2** 实现 TaskDetailService（FetchTaskDetail）  
  `taskCredentialService/application/services.go`（追加）

- [ ] **2.1.3** 实现 repo-clone-credentials / task-detail / layer-oauth-tokens handler  
  `taskCredentialService/interfaces/handlers.go`（追加）

- [ ] **2.1.4** 实现 `_build_credentials_for_precheck()` 的 Go 等效逻辑  
  复用 `infrastructure/sqlite_business.go` + `infrastructure/gitoauth_client.go`

### 2.2 Django 侧变更

- [ ] **2.2.1** `settings_manager.py` 新增 `get_task_credential_service_url()`  
  `task2app/Saas_project/core/config/settings_manager.py`

- [ ] **2.2.2** `conf/core/django/config.yaml` 新增 `taskCredentialServiceBase`  
  值为 `http://127.0.0.1:8015`

- [ ] **2.2.3** `relay_to_trae_proxy.py` 预检改为调用 Go 服务  
  `_resolve_relay_precheck_task_api_origin()` → 返回 `taskCredentialServiceBase`

- [ ] **2.2.4** `cloud_compute_views.py` env-prepare 注入 `TASK_CREDENTIAL_SERVICE_ORIGIN`

- [ ] **2.2.5** `taskAgentSupport` 内部路由切换（A1/A2/A3 → Go :8015）  
  `taskAgentSupport/src/` 或配置

### 2.3 测试

- [ ] **2.3.1** Go 凭证构建测试（正常/缺 identity/token 换发失败）  
  `taskCredentialService/src/credentials_test.go`

- [ ] **2.3.2** Go gitOauth client mock 测试  
  `taskCredentialService/src/gitoauth_test.go`

- [ ] **2.3.3** Django 预检测试更新（调 Go URL）  
  `task2app/Saas_project/tests/test_relay_to_trae_proxy.py`

- [ ] **2.3.4** E2E playwright 回归  
  `playwright/front_project/tests/TaskDetail.relay-to-trae-direct-start.playwright.test.js`

---

## Increment 3: 组 B token 端点迁移

- [ ] **3.1** Go 实现 exchange-refresh handler  
  `taskCredentialService/interfaces/handlers.go`（追加）

- [ ] **3.2** Go 实现 refresh-access handler  
  `taskCredentialService/interfaces/handlers.go`（追加）

- [ ] **3.3** go_relayToTrae 换票目标切换到 Go :8015  
  `go_relayToTrae/src/token.go`

- [ ] **3.4** onlineServiceJS token 端点调用 taskAgentSupport → Go  
  无需改代码（taskAgentSupport 路由已切换）

- [ ] **3.5** Go token 生命周期测试  
  `taskCredentialService/src/token_test.go`

- [ ] **3.6** Django 删除 `container_runtime_token_views.py`

---

## Increment 4: 组 C+D 状态上报 + 杂项端点（收尾）

- [ ] **4.1** Go 实现组 C 状态上报 handlers（6 个端点）  
  `taskCredentialService/interfaces/handlers.go`（追加）

- [ ] **4.2** Go 实现组 D handlers（4 个端点）  
  `taskCredentialService/interfaces/handlers.go`（追加）

- [ ] **4.3** Go 状态上报测试  
  `taskCredentialService/src/status_test.go`

- [ ] **4.4** Django 删除 9 个 container view 文件  
  见设计文档「Django 清理清单」

- [ ] **4.5** Django 删除 CloudServerConfig + ContainerTokenAuditEvent Model + migration  
  `cloud/models.py` + 新建 migration

- [ ] **4.6** Django 清空 `instance_callback_urls.py`

- [ ] **4.7** Django 清理 `cloud/domain/` 6 个文件

- [ ] **4.8** Django 清理相关测试文件（6 个 pytest）

---

## Increment 5: 前端错误提示精准化

- [ ] **5.1** 更新 `ServerConfig.logic.vue` 第 1735-1737 行  
  5xx 非 token-refresh 场景文案改为"预检服务异常"

- [ ] **5.2** 前端单元测试更新

---

## Increment 6: 回归保护

- [ ] **6.1** Go 全量测试套件通过（覆盖率 ≥ 80%）  
  `cd taskCredentialService && go test ./... -cover`

- [ ] **6.2** Django pytest 全量通过（排除已迁移删除的测试）  
  `cd task2app/Saas_project && pytest`

- [ ] **6.3** Playwright E2E 直启全链路通过  
  `TaskDetail.relay-to-trae-direct-start.playwright.test.js`

- [ ] **6.4** runAll 启动全服务验证  
  `./bin/runAll` → 所有服务 healthy（含 task-credential-service）

---

## 任务依赖图

```
1.1 (Go skeleton) → 1.2 (adapters) → 1.3 (app services) → 1.4 (HTTP) → 1.5 (tests) → 1.6 (runAll)
                                                                                        ↓
2.1 (Go credentials) ← 1.4 ─────────────────────────────────────────────────────────────┐
    ↓                                                                                    ↓
2.2 (Django changes) → 2.3 (integration tests) ───────────────────── 死锁消除 ←─────────┘
    ↓
3.x (token endpoints) → Django cleanup starts
    ↓
4.x (status + misc) → Django fully clean
    ↓
5.x (frontend) + 6.x (regression)
```
