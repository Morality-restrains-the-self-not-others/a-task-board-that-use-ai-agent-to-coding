# Value Stream: DEPLOY_MODE 寻址模式切换

> Derived from design: `.claude/skills/1-brainstorming-设计文档/design.md`

## Value Summary

开发者通过单一环境变量 `DEPLOY_MODE` 在域名寻址（`https://daydaymoney.com`）和 IP:端口寻址（`http://183.250.1.132:18081`）之间全局切换所有服务的外部可见 URL。

## Related Value Streams

- **remove-remote-sync-unify-infra-host** (`2026-06-05`): **extension** — 本流建立在 `${INFRA_HOST}` 统一变量的基础之上，新增 `base.yaml` 模板变量解析管道
- **gateway-routes-codegen** (`conf/value-stream.yaml`): **modification** — routes-to-apisix.py 新增 base.yaml 读取，upstream host 从硬编码 IP 改为运行时解析
- **port-config-split-monorepo-conf** (`2026-06-02`): **extension** — conf_loader.py/conf-read.py 在现有 monorepo conf/ 加载管道上增加模板变量解析层
- **gateway-skeleton-tls** (planned): **extension** — domain 模式为 TLS 终结提供配置基础（publicBase 从 IP → `https://daydaymoney.com`）

## End-to-End Flow

[开发者设置 DEPLOY_MODE=domain] → [conf/base.yaml 解析 mode=domain] → [load_base_yaml() 展开 subdomains.*] → [resolve_template_vars() 替换各 config.yaml 中 ${subdomains.xxx}] → [routes-to-apisix.py 解析 upstream host] → [各服务以域名对外暴露] → [开发者通过 https://daydaymoney.com 访问系统]

## Value Increments

### Increment 1: Python 配置解析核心（Thin Slice）
**Value to user:** `conf_loader.py` 和 `conf-read.py` 能正确解析 `base.yaml` 并替换模板变量，`DEPLOY_MODE=local` 下行为与当前完全一致（向后兼容验证）
**Scope:**
- `conf_lib.py` 新增 `load_base_yaml()` + `resolve_template_vars()`
- `conf_loader.py` 的 `load_app_config()` 集成模板变量解析
- `conf-read.py` 的 `build_runtime_snapshot()` 注入 resolved addressing
- 单元测试：domain/local 解析、环境变量覆盖、模板替换
**Depends on:** nothing

### Increment 2: 网关路由生成适配
**Value to user:** `taskGateway/routes/routes.yaml` 的 upstream host 不再硬编码 IP，由 `routes-to-apisix.py` 根据 `DEPLOY_MODE` 运行时解析
**Scope:**
- `routes-to-apisix.py` 的 `_upstream_host()` 集成 base.yaml 解析
- `conf/base.yaml` 新增 `gateway` 寻址键
- `conf/gateway/task-gateway/config.yaml` 的 `publicBase` → 模板变量
- 验证：`DEPLOY_MODE=local` 生成的 apisix.yaml 与当前一致
**Depends on:** Increment 1（Python 配置解析可用）

### Increment 3: 服务配置迁移（对外字段）
**Value to user:** 所有浏览器可见的外部 URL（L1 层）随 `DEPLOY_MODE` 统一切换
**Scope:**
- `conf/auth/task-auth/config.yaml` — `oidc.issuer`、`oidc.bootstrapRedirectUri`、`oidc.gitServicePublicBase` → 模板变量
- `conf/frontend/vue/config.yaml` — `publicBaseUrl`、`apiBaseUrl` → 模板变量
- `conf/frontend/vue/git-service.yaml` — `publicUrl` → 模板变量
- `conf/infra/git-service/config.yaml` — `allowedHost`、`publicUrl` → 模板变量
- `conf/gateway/task-gateway/config.yaml` — `cors.allowedOrigins` → 模板变量
- 端到端验证：`DEPLOY_MODE=domain` 启动后浏览器通过域名访问
**Depends on:** Increment 2（网关路由已适配）
