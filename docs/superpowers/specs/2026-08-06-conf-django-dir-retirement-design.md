# conf/core/django 目录退役：配置数据迁移至语义化键

**日期**: 2026-08-06
**状态**: 头脑风暴（设计评估）
**作者**: claude
**迭代**: 评估「conf/core/django/config.yaml 配置数据迁至语义化键（如 conf/core/sms.yaml），彻底移除 django 目录名」

## 背景与动机

Django saas-backend 已于 2026-07-30 退役（`docs/superpowers/specs/2026-07-05-task2app-api-go-split-brainstorm-design.md` 的 Go 迁移目标达成）。但 `conf/core/django/` 目录仍以「Django 服务配置」命名存在，其中：

1. **服务连接字段已死**：`host: 0.0.0.0 / port: 8001 / internalApiBase` 指向已不存在的 Django runserver
2. **配置数据仍被消费**：sms（taskAuth 经 sync 片段）、email（taskEvents 经 sync 片段）、paypal（taskBill 经语义化文件 + django fallback）、ssoJwtSecret/ssh_login_allowed_addresses（conf_loader 外提）
3. **大量历史 sync 片段已死**：目录内 ai-provider.yaml / task-auth.yaml / vue.yaml 等 10+ 片段**运行时 0 引用**

移除 `django` 目录名可消除「Django 服务配置」的误导语义，使配置结构反映真实归属。

## 现状全景（消费方盘点）

### A. Monorepo Root 定位锚点（最深的耦合 — 6 处）

| 位置 | 用途 | 现状 |
|------|------|------|
| `shareLib/confload/load.go` FindMonorepoRoot | 11+ Go 服务的 root 定位（markers[0]） | 依赖 `conf/core/django/config.yaml` |
| `runAll/scripts/conf_lib.py` repo_root | conf-read/conf-sync 脚本 | 依赖同文件 |
| `taskSSE/src/config.mjs` findRepoRoot | taskSSE root 定位 | 依赖同文件 |
| `taskEvents/run.sh` find_monorepo_root | taskEvents 编排 | 依赖同文件 |
| `taskAIEndPoint/src/main.go` | taskAIEndPoint root 定位 | 依赖同文件 |
| `conf/scripts/ci/check_conf_sync.sh` | CI 前置检查 | 依赖同文件 |

### B. 配置数据消费者（仍活跃）

| 数据 | 消费方 | 读取方式 | 迁移状态 |
|------|--------|----------|----------|
| sms | taskAuth | `conf/auth/task-auth/sms.yaml`（sync 片段，源=core/django config.yaml） | ⚠️ 源仍指向 django |
| email | taskEvents | `conf/events/domain-events/django.yaml`（sync 片段，源=core/django config.yaml） | ⚠️ 源仍指向 django + 片段名含 django |
| paypal | taskBill | `conf/billing/paypal/config.yaml`（优先）+ django config.yaml fallback | 🟡 fallback 仍指向 django |
| ssoJwtSecret / ssh_login_allowed_addresses | conf_loader → snapshot-json | 从 django 配置块外提 | ⚠️ 键仍从 django 块读取 |
| test_phone_numbers | BillingRecharge 测试 | `conf/core/django/config.test.yaml` | ⚠️ 直接读 django 路径 |
| internalApiBase | taskEvents（django.yaml 片段 pick） | sync 片段 | 🔴 死数据（Django 已退役） |

### C. 死引用（无消费者）

- `conf/core/django/` 内 10+ sync 片段（ai-provider.yaml / task-auth.yaml / task-bill.yaml / task-gateway.yaml / task-sse.yaml / vue.yaml / relay-to-trae.yaml / mock-run-container.yaml / domain-events.yaml / docker-infra.yaml / task-agent-support.yaml / task-ai-endpoint.yaml / task-container-gateway.yaml）——**运行时 0 引用**，纯历史 sync 遗留
- `config.yaml` 的 host/port/internalApiBase/messageQueue/containerRegister*SSO/taskCredentialServiceBase——服务连接字段，已死
- `git-oauth-providers/` 目录（sync.sh 拷贝产物）——无运行时消费（gitOauth 走 conf/auth/git-oauth/providers）

### D. sync 机制耦合

- `conf/core/django/sync.manifest.yaml` + `sync.sh`：把 13 个其他服务的配置 pick 同步进本目录（历史 Django 单文件时代遗留）
- `conf/events/domain-events/sync.manifest.yaml`：`from: ../../core/django/config.yaml → to: django.yaml`（email/sms 真源）
- `conf/auth/task-auth/sync.manifest.yaml`：`from: ../../core/django/config.yaml → to: sms.yaml`（SMS 真源）
- `conf-sync-all.sh`：遍历所有 `conf/*/*/sync.manifest.yaml` 应用同步
- `check_conf_sync.sh`：CI 校验整个 conf/ 树无漂移

### E. 其他

- `db/registry.yaml`：仅注释提及（无功能依赖）
- `migrate_port_config_to_conf.py`：历史迁移脚本（port_config.json → conf），django 键映射，一次性工具

## 方案设计

### 目标目录结构（语义化）

```
conf/
  core/
    sms.yaml              # 新真源：SMS 配置（从 django config.yaml sms 块迁出）
    email.yaml            # 新真源：SMTP 配置（从 django config.yaml email/email2 块迁出）
    sso.yaml              # 新真源：ssoJwtSecret / ssh_login_allowed_addresses
    paypal.yaml           # 新真源：paypal（或并入 conf/billing/paypal/，见下）
    config.test.yaml      # 迁至 conf/core/sms.test.yaml（test_phone_numbers）
  auth/task-auth/
    sms.yaml              # sync 片段：from ../../core/sms.yaml（不再引用 django）
  events/domain-events/
    email.yaml            # sync 片段：from ../../core/email.yaml（替换 django.yaml）
  billing/paypal/
    config.yaml           # 已是真源；删除 taskBill 的 django fallback
```

### 迁移步骤（4 个切片）

**Slice 1 — Root 锚点统一（6 处，低风险）**
- `confload.FindMonorepoRoot`：markers[0] 从 `conf/core/django/config.yaml` 改为 `conf/base.yaml`（已是 markers[1]，且必存在）
- `conf_lib.py` / `taskSSE config.mjs` / `taskEvents run.sh` / `taskAIEndPoint main.go` / `check_conf_sync.sh`：同步改为 `conf/base.yaml` 锚点
- 保持向后兼容：锚点检测支持旧路径（过渡期）或直接切换（仓库内同步改，无外部部署）
- 验证：所有 Go 服务 root 定位 + conf-read 脚本 + CI

**Slice 2 — 配置真源迁移（4 个数据块）**
- 新建 `conf/core/sms.yaml`（sms 块）、`conf/core/email.yaml`（email+email2）、`conf/core/sso.yaml`（ssoJwtSecret+ssh_login_allowed_addresses）
- 更新 `conf/auth/task-auth/sync.manifest.yaml`：`from: ../../core/sms.yaml`
- 更新 `conf/events/domain-events/sync.manifest.yaml`：`from: ../../core/email.yaml → to: email.yaml`（去掉 internalApiBase/host/port 死 pick）
- 更新 `conf_loader.py`：`out["task2appSsoJwtSecret"]` / `ssh_login_allowed_addresses` 从 `conf/core/sso.yaml` 读取（不再读 django 块）
- 删除 `conf/core/django/config.yaml` 的 host/port/internalApiBase/messageQueue 等死字段（或整文件退役）

**Slice 3 — 消费者解除 django 引用**
- `taskBill/paypal_pay.go`：删除 django config.yaml fallback（`conf/billing/paypal/config.yaml` 已是完整 SSOT，含 webhook_id）
- `BillingRecharge` 测试：`conf/core/django/config.test.yaml` → `conf/core/sms.test.yaml`
- 删除 `conf/core/django/` 内 10+ 死 sync 片段 + `sync.manifest.yaml` + `sync.sh` + `git-oauth-providers/`

**Slice 4 — 目录退役 + CI 固化**
- 删除 `conf/core/django/` 整个目录（config.yaml 中无存活数据后）
- `migrate_port_config_to_conf.py` 标记 deprecated（历史一次性工具）
- `check_conf_sync.sh` 验证新结构无漂移
- 全局 grep 确认 0 残留 `core/django` 引用

### 关键风险与缓解

| 风险 | 等级 | 缓解 |
|------|------|------|
| Root 锚点改动影响 11+ 服务启动 | 高 | markers 数组已含 conf/base.yaml（必然存在）；切到 base.yaml 后逐服务启动验证 |
| sync 片段源路径变更导致 email/sms 断供 | 高 | 先迁真源 + 更新 manifest → 跑 conf-sync-all.sh → 校验片段内容一致 → 再退役旧文件 |
| paypal fallback 删除后若 billing/paypal 配置不全 | 中 | 核对 billing/paypal/config.yaml 已含全部字段（client_id/secret/mode/currency/webhook_id×2），缺则先补齐 |
| conf_loader django 键被 snapshot 消费者隐式使用 | 中 | snapshot-json 的 django 键保留为空 dict 占位（过渡），taskSSE 无专门消费已验证 |
| check_conf_sync 全树校验失败 | 低 | 切片内同步执行，CI 幂等 |

### 不迁移项（明确排除）

- `conf/core/django/` 内的 git-oauth-providers：gitOauth 服务实际读 `conf/auth/git-oauth/providers`（已语义化），目录内副本纯冗余 → 直接删除
- messageQueue / containerRegisterSSO / taskCredentialServiceBase：死字段 → 随目录退役删除
- 若评估发现 email/sms 已有其他真源（如 taskAuth 直接内联），则连 sync 片段一并简化——**待 Slice 2 前置核对**

## 🕸️ Code Review Graph 分析

本任务为纯配置/文档迁移（无 Go/前端代码逻辑变更，仅 root 锚点与配置读取路径修改）。涉及符号：`confload.FindMonorepoRoot`（11+ 服务依赖的共享库函数）、`conf_lib.repo_root`、`taskSSE findRepoRoot`。CRG 图存在性检查：`skipped_non_code` — 配置迁移以静态 grep 覆盖面为准（已全量盘点 A-E 五类耦合）。

## Domain Concept Inventory

无新领域概念。配置迁移不改变业务边界、实体或事件。**业务意图 → 事件对照：无对应事件** — 纯配置结构重组，无服务端状态变更，无跨边界副作用（例外理由：配置迁移，非业务行为）。

## Value Stream Impact

本变更影响基础设施层（配置供应），不触碰业务价值流步骤。value-stream.yaml 中无 `<service>.<table>.<field>` 被修改（配置非 DB 字段）。受影响的是**配置供应横切关注点**：所有依赖 conf 的服务（taskAuth/taskEvents/taskBill 等）的启动配置加载路径。无新价值流，无字段变更，无测试文件业务变更（仅 BillingRecharge 配置路径更新）。

## 🏛️ 架构变更影响

- **迭代版本**: 评估阶段（实施时 v13+ target）
- **架构判断**: 本变更属「配置结构重组」，不增删服务组件、不改服务间调用关系、不改数据流所有权 → **不触发架构视图更新**（纯配置微调，按架构规则「纯配置变更不需要更新架构」）。若实施时新增/删除配置文件而非服务，维持此判断。
- **相关既有 spec**: `2026-07-05-task2app-api-go-split-brainstorm-design.md`（Django 退役的 Go 化主线，本变更是其配置收尾）

## 决策点（已确认 2026-08-06）

| 决策点 | 结论 |
|--------|------|
| 1. 执行范围 | ✅ **完整 4 切片执行**（root 锚点 → 真源迁移 → 消费者解除 → 目录退役 + CI 固化） |
| 2. email2 处理 | ✅ **保留迁移**（并入 conf/core/email.yaml，保留双 SMTP 结构） |
| 3. paypal 真源 | ✅ 保留 `conf/billing/paypal/config.yaml`（已语义化 SSOT），删除 taskBill 的 django fallback |
| 4. 执行方式 | ✅ **作为 OPT-20260806-054 落盘逐步执行**（切片粒度，每切片提交 + 验证） |
