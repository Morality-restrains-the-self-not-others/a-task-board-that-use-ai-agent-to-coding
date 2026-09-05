# value-stream.yaml 规则文件

> 本文件为 `conf/value-stream.yaml` 的 AI 协作规则。编排语义与校验实现以 [valueStream/README.md](../valueStream/README.md)、[docs/superpowers/specs/2026-05-19-value-stream-service-design.md](../docs/superpowers/specs/2026-05-19-value-stream-service-design.md) 为准。

## 基本信息

- 版本：1.1.0
- 创建日期：2026-05-28
- 最后修改：2026-06-25
- 维护者：Trae AI
- 关联实现：`valueStream/src/fields.go`（`ParseFieldName`）、`valueStream/src/config.go`（`validate`）
- 关联编排：`conf/runAll.yaml`（`runall_config` 服务名白名单）

## 文件定位

| 路径 | 用途 |
|------|------|
| `conf/value-stream.yaml`（本文件） | 项目配置目录完整价值流注册；`runner.working_dir` 相对**仓库根** |
| `docs/examples/value-streams.example.yaml` | 精简示例 |
| `conf/runAll.yaml` | 字段首段（服务名）白名单来源 |

## 启动方式

```bash
cd valueStream && ./build.sh
./bin/valueStream --config ../conf/value-stream.yaml --ui-port :9998
# Web UI: http://localhost:9998
```

**校验时机：** 进程启动时 `LoadConfig`；Web UI 拖拽改域顺序写回 YAML 前（`POST /api/domain-order`）同样校验。任一环节字段不合法即**拒绝启动/写盘**。

**缺失 test_file：** `status: active` 且文件不存在时启动仅 WARNING；长期缺失应改为 `status: planned`（不校验文件存在），或修正相对 `runner.working_dir` 的路径。`front_project/` / `playwright/` 相对 `task2app/Saas_project` 时用 `../../taskFE/...`、`../playwright/...`。


---

## 核心规则

### 字段名必须是恰好三段 `<service>.<table>.<column>`（一级）

**实现正则**（`valueStream/src/fields.go`）：

```text
^([a-zA-Z0-9-]+)\.([a-z0-9_]+)\.([a-z0-9_]+)$
```

| 段 | 允许字符 | 含义 | 示例 |
|----|----------|------|------|
| `<service>` | 字母（含 CamelCase）、数字、**连字符**（无下划线） | runAll 服务名 | `saas-backend`、`taskFE`、`task-events-accounts` |
| `<table>` | 小写字母、数字、**下划线** | 逻辑表/虚拟表名 | `accounts_user`、`runtime`、`config` |
| `<column>` | 小写字母、数字、**下划线** | 列/属性名 | `email`、`lifecycle_status` |

**典型启动报错：**

```text
Config error: stream "domain-events-consumer-split" step "increment1-accounts-thin-slice":
field name "port_config.domainEvents.transport" must be <service>.<table>.<column>
```

**优先级：** 高

#### 禁止写法（常见 AI 误用）

| 错误写法 | 失败原因 | 正确思路 |
|----------|----------|----------|
| `port_config.domainEvents.transport` | 首段 `port_config` 不是 runAll 服务；`domainEvents` 含大写（camelCase） | 映射到**读取该配置的 runAll 服务** + snake_case |
| `port_config.domainEvents.redis.streamKeyPrefix` | **四段**；camelCase | 拆成单条三段字段，列名 snake_case |
| `redis.sse.channelPrefix` | `redis` 不是 runAll 服务；`channelPrefix` camelCase | 用 `task-sse` 或 `docker-infra` + `config.*` |
| `saas-backend.accounts_user.email.extra` | 超过三段 | 只保留一段 service、一段 table、一段 column |
| `DomainEvents.transport.kafka` | table/column 含大写、段数/语义错误 | service 可与 runAll 名一致（含 `taskFE`）；table/column 须小写 snake_case + 恰好三段 |

| `providers[].budget_enabled` | `saas-backend.projects_tenant_feature_params.providers_budget_enabled` | JSON 嵌套键扁平进 column |
| `providers[].use_sub_token` | `saas-backend.projects_tenant_feature_params.providers_use_sub_token` | JSON 嵌套键扁平进 column |

#### 正确示例

```yaml
fields:
  # 数据库列（Django 默认表名）
  - name: saas-backend.accounts_user.id
    description: 用户主键
  # 虚拟运行时表（非 DB，已在多条流中使用）
  - name: task-git-oauth.runtime.lifecycle_status
    description: runAll 编排生命周期
  # HTTP 健康探针
  - name: task-events-accounts.health.status
    description: accounts 域消费者健康
  # port_config.json 配置项（见下节映射表）
  - name: saas-backend.config.domain_events_transport
    description: 对应 port_config.domainEvents.transport
  - name: task-sse.config.channel_prefix
    description: 对应 taskSSE / SSE channel 前缀配置
```

**优先级：** 高

---

### 首段 `<service>` 必须在 runAll 服务名白名单内（一级）

`runall_config: conf/runAll.yaml` 加载 `groups[].services[].name`；字段首段必须命中其一，否则：

```text
field "foo.bar.baz": provider "foo" not in runAll config
```

**当前合法服务名（与 conf/runAll.yaml 同步维护）：**

```text
以 conf/runAll.yaml 的 groups[].services[].name 为准（含 task-cloud-service、
task-credential-service、各 task-events-* 意图消费者等）。
另：valueStream 校验会合成白名单项 `runall`（编排器自身 runtime/api 字段，
见 valueStream/src/config.go），无需把 runAll 再登记为可启动服务。
```

**不是合法首段：** `port_config`、`redis`、`kafka`、`django`、`taskEvents`、
`taskCloudService`（camelCase 进程/包名 ≠ runAll `name`；应写 `task-cloud-service`）。

**优先级：** 高

---

### port_config.json 路径 → fields.name 映射（一级）

`port_config.json` 使用 **camelCase JSON 键**；`value-stream.yaml` 的 `<table>.<column>` 必须使用 **snake_case**，且首段为**实际消费该配置的服务**，而非 `port_config`。

| port_config.json 路径 | 建议 fields.name | 归属说明 |
|----------------------|------------------|----------|
| `domainEvents.transport` | `saas-backend.config.domain_events_transport` | Django 发布/读取传输选型 |
| `domainEvents.redis.streamKeyPrefix` | `saas-backend.config.domain_events_redis_stream_key_prefix` | Django 侧 Redis Streams 前缀 |
| `domainEvents.kafka.bootstrapServers` | `docker-infra.config.kafka_bootstrap_servers` | 基础设施 broker 地址 |
| `domainEvents.consumers.accounts.port` | `task-events-accounts.config.port` | 域消费者监听端口 |
| `taskSSE.port` | `task-sse.config.port` | SSE 服务端口 |
| `taskSSE.redis.channelPrefix` | `task-sse.config.channel_prefix` | SSE Redis channel 前缀 |
| `django.port` | `saas-backend.config.port` | Django 监听端口 |
| `relayToTrae.port` | `go-relay.config.port` | go-relay 端口 |

**规则：**

1. 禁止把 JSON 点路径原样粘贴进 `fields[].name`。
2. camelCase → snake_case：`domainEvents` → `domain_events`，`streamKeyPrefix` → `stream_key_prefix`。
3. 配置类字段统一用虚拟表 `config`（或既有惯例 `runtime`、`health`），列名描述配置项语义。
4. 在 `description` 中保留 JSON 路径便于人类对照，例如：`description: 对应 port_config.domainEvents.transport`。

**优先级：** 高

---

### 虚拟表命名惯例（二级）

非数据库字段时，沿用仓库已有模式，勿发明四段路径：

| 虚拟 `<table>` | 用途 | 示例 |
|----------------|------|------|
| `runtime` | runAll 生命周期、日志、级联启停 | `saas-backend.runtime.lifecycle_status` |
| `health` | HTTP 健康检查 | `task-events-projects.health.status` |
| `config` | port_config / 环境配置（推荐用于新条目） | `saas-backend.config.domain_events_transport` |
| `online_service_js` | go-relay 拉起的 JS 层 | `go-relay.online_service_js.repo_match_key` |

**优先级：** 中

---

### step 与 test_file 约束（一级）

| 字段 | 规则 |
|------|------|
| `version` | 必须为 `"1"` |
| `value_streams[].name` | 全局唯一 |
| `value_streams[].domain` | 必填，非空 |
| `steps[].name` | 流内唯一 |
| `steps[].status` | `active`（默认）或 `planned` |
| `steps[].test_file` | 必填；`active` 时相对 `runner.working_dir`，**启动时文件必须存在** |
| `steps[].fields[].name` | 同 step 内不重复 |

`runner.working_dir` 当前为 `../task2app/Saas_project`（相对 `conf/`，即 config 文件所在目录；Go 实现为 `filepath.Join(ConfigDir, WorkingDir)`，解析后绝对路径为 `<repo_root>/task2app/Saas_project`）。

**优先级：** 高

---

### test_file 路径解析规则（一级）

`test_file` 以 `runner.working_dir`（`<repo_root>/task2app/Saas_project`）为基准解析。**路径前缀决定文件从哪个目录开始查找：**

| 文件实际位置 | 应使用的前缀 | 示例 |
|-------------|-------------|------|
| `task2app/Saas_project/` 内部 | 直接相对路径（无 `../`） | `tests/test_login.py`、`accounts/view_test/UserViewSet_test.py`、`cloud/tests/test_cloud_compute.py`、`scripts/ci/check_no_user_objects.sh` |
| `taskFE/` | `../../taskFE/` | `../../taskFE/app/src/tests/domain/auth/auth_domain_model.test.js` |
| repo root 下其他顶层目录 | `../../<dir>/` | `../../gitService/scripts/fix_oidc_ssl.sh`、`../../runAll/src/runner_test.go`、`../../taskGitOauth/src/handlers_test.go`、`../../taskAuth/src/openapi_test.go` |

**规则：**

1. 文件在 `runner.working_dir` 子树内 → 使用相对于 `working_dir` 的路径（**不加** `../`）。
2. 文件在 `working_dir` 同级目录（如 `taskFE/`）→ 使用 `../<sibling_dir>/...`（上一级）。
3. 文件在 repo root 下其他顶层目录（如 `gitService/`、`runAll/`、`gitOauth/`、`taskAuth/`）→ **必须**使用 `../../<top_dir>/...`（向上两级到达 repo root）。

**常见错误：**

| 错误写法 | 失败原因 | 正确写法 |
|----------|----------|----------|
| `gitService/scripts/fix.sh` | 解析到 `task2app/Saas_project/gitService/...`（不存在） | `../../gitService/scripts/fix.sh` |
| `runAll/src/foo_test.go` | 解析到 `task2app/Saas_project/runAll/...`（不存在） | `../../runAll/src/foo_test.go` |
| `../../../taskFE/...` | `front_project` 在 `task2app/` 下，只需上一级 | `../../taskFE/...` |

**优先级：** 高

---

## 修改 value-stream.yaml 的检查清单

在新增或修改 `domain-events-consumer-split` 等流之前，对每条 `fields[].name` 逐项确认：

1. **段数：** 按 `.` 分割后是否**恰好 3 段**？
2. **字符：** 是否**全小写**？是否无 camelCase（如 `domainEvents`、`channelPrefix`）？
3. **首段：** 是否在上文 runAll 服务名列表中？
4. **配置项：** 若来自 `port_config.json`，是否已映射为 `<服务>.config.<snake_case>`，而非 `port_config.*`？
5. **本地校验：**

```bash
cd valueStream && ./build.sh
./bin/valueStream --config ../conf/value-stream.yaml --ui-port :9998
# 无 "Config error" 即通过；或：
go test ./src/... -run ParseFieldName
```

6. **test_file 路径前缀：** 对照上节「test_file 路径解析规则」的路径模式表，确认每个 active step 的 `test_file` 前缀符合其实际位置（`task2app/Saas_project` 内 → 直接路径；`taskFE/` → `../../taskFE/`；其他顶层目录 → `../../<dir>/`）。

**优先级：** 高

---

## 历史违规条目（已修复，勿回退）

以下写法曾导致 valueStream 启动失败，已于 2026-05-28 按映射表修正；AI **禁止**改回：

| 流 | step | 已废弃 name | 当前合法 name |
|----|------|-------------|---------------|
| `domain-events-consumer-split` | `increment1-accounts-thin-slice` | `port_config.domainEvents.transport` | `saas-backend.config.domain_events_transport` |
| `domain-events-consumer-split` | `increment2-transport-redis-memory` | `port_config.domainEvents.transport` | `saas-backend.config.domain_events_transport` |
| `domain-events-consumer-split` | `increment2-transport-redis-memory` | `port_config.domainEvents.redis.streamKeyPrefix` | `saas-backend.config.domain_events_redis_stream_key_prefix` |
| `domain-events-consumer-split` | `increment2-in-memory-sync` | `port_config.domainEvents.transport` | `saas-backend.config.domain_events_transport` |
| `domain-events-consumer-split` | `increment3-sse-realtime` | `redis.sse.channelPrefix` | `task-sse.config.channel_prefix` |
| `task-llm-budget-governance` | `tenant-llm-budget-feature-toggle` | `...providers.budget_enabled` | `...providers_budget_enabled` |
| `oidc-ssl-protocol-fix` | `swd-url-builder-http-fix` | `gitService/scripts/fix_oidc_ssl.sh` | `../../gitService/scripts/fix_oidc_ssl.sh` |
| `oidc-ssl-protocol-fix` | `oidc-playwright-diagnostic` | `gitService/playwright/tests/oidc-ssl-diagnostic.playwright.test.js` | `../../gitService/playwright/tests/oidc-ssl-diagnostic.playwright.test.js` |
| `oidc-ssl-protocol-fix` | `oidc-playwright-e2e-login` | `gitService/playwright/tests/oidc-sso-login.playwright.test.js` | `../../gitService/playwright/tests/oidc-sso-login.playwright.test.js` |
| `oidc-ssl-protocol-fix` | `oidc-ssl-fix-verify` | `gitService/playwright/tests/oidc-ssl-fix-verify.playwright.test.js` | `../../gitService/playwright/tests/oidc-ssl-fix-verify.playwright.test.js` |

---

## 最佳实践

### 新增价值流步骤（二级）

1. 先确定 pytest `test_file` 路径（相对 `task2app/Saas_project`）。
2. 列出环节涉及的**服务提供方**（runAll 名）与数据/配置触点。
3. 每个触点写一条三段 `fields[].name`；配置写 `config`，DB 写 Django 表名。
4. 启动 valueStream 验证配置加载。

### description 写法（二级）

- 数据库字段：简短业务语义。
- 配置字段：`description` 中注明 `port_config` JSON 路径，便于与 `task2app/conf/port_config.json` 对照。
- 避免在 `name` 里写 JSON 路径。

### 与 conf/runAll.yaml.ai.md 协同（二级）

- 新增 runAll 服务后，须同步更新本文件「合法服务名」列表，并在新价值流中使用该 `name` 作为字段首段。
- 端口/健康相关字段优先对照 `conf/runAll.yaml.ai.md` 端口表（`port_config.json` 已废弃）。

---

## 规则冲突处理

1. 核心规则（三段式、白名单、port_config 映射）> 最佳实践 > 风格指南  
2. 本文件（`value-stream.yaml.ai.md`）> 设计文档中的宽松描述（实现以 `fields.go` 正则为准）  
3. 用户显式指令 > 本规则文件  
4. `description` 可写 camelCase JSON 路径；**`name` 永远不允许**

---

## 变更日志

- **2026-06-25：** 新增「test_file 路径解析规则」章节（路径模式对照表 + 常见错误）；修正 `runner.working_dir` 语义文档（相对 `conf/` 非仓库根）；追加 `oidc-ssl-protocol-fix` 流 4 处 `gitService/` → `../../gitService/` 修复记录；检查清单新增第 6 项（test_file 路径前缀验证）
- **2026-05-28：** 修正 `domain-events-consumer-split` 流 5 处非法字段名；valueStream 启动校验通过
- **2026-05-28：** 版本 1.0.0 — 初始创建；记录 `port_config.domainEvents.*` 类启动报错根因、port_config 映射表与 domain-events-consumer-split 已知违规项
