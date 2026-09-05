# Intent: 云端开发项目列表须用 conf-local 网关密钥注入 APISIX

## 背景与目标

https://www.daydaymoney.com/tenant/882297276515512320/projects 显示「加载项目失败」（`data-traceId=7af49e7f-8e9e-4686-9470-dd7620ef4e6d`）。

Loki 时间线：task-auth forward-auth **200** → task-project-service `GET /api/projects/tenant_id/{tid}` **401**（0ms）→ 网关回传 401。响应体为「请先登录」。直连 `:8016` 在注入 `X-Gateway-Auth-Verified=1` + **conf-local** `gatewayInternalSecret` 时返回 200；空密钥或密钥不一致时 401。

clone-run（`~/bin/daydaymoney-deploy`）的 `taskGateway/apisix/apisix.yaml` 为 `X-TaskGateway-Internal-Secret: ''`，因 `routes-to-apisix.py` 只读已跟踪的 `conf/gateway/task-gateway/config.yaml`（空骨架），未合并 `conf-local/`。服务进程经 `confload` 读到 48 字符密钥 → 身份无法提升。

目标：生成 APISIX 路由时合并 `conf-local` 的 `gatewayInternalSecret`，使网关注入头与上游校验一致；已登录用户项目列表不再 401。

## 范围与边界

- **范围内**：`taskGateway/scripts/routes-to-apisix.py` 加载网关 conf；项目列表前端对 401 展示接口文案；契约测例。
- **范围外**：改 forward-auth 协议、放宽 `ApplyGatewayUser` 免密钥校验、把真实密钥写回已跟踪 `conf/`。

## 约束与风险

- 机密只在 `conf-local/`（ADR-0054 / 元规则 63）。生成器必须走与 `conf_loader.merge_conf_local` 相同的合并。
- 生成后的 `apisix.yaml` 含运行时密钥：clone-run 树不回提交到源码仓；源码仓本地-dev 占位符可保留。
- 改完须在部署树 `routes-apply` 并 reload APISIX，否则现网 yaml 仍为空串。

## 验收标准

1. `load_gateway_conf`：`conf/` 空密钥 + `conf-local/` 非空 → 采用 conf-local 值。
2. `_token_route_transform` 用合并后的密钥设置 `X-TaskGateway-Internal-Secret`。
3. 项目列表 401 时页面展示「请先登录」（或接口 `message`），并带 `data-traceId`。
4. 部署树 `apisix.yaml` 的该头不再是 `''`；带网关校验头直连项目列表为 200。

## 实施计划

1. 测例：临时仓库 conf 空 + conf-local 有密钥 → `load_gateway_conf` / token rewrite 命中 local。
2. `_load()` 合并 `conf-local/gateway/task-gateway/config.yaml`。
3. 前端 401 展示接口错误文案。
4. 部署树 routes-apply + 复现 200。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 已登录用户加载租户项目列表 | — | — | — | — | 只读查询；鉴权头对齐属网关配置，不改变业务事实 |

## 变更记录

- 2026-09-01：初稿。traceId `7af49e7f-8e9e-4686-9470-dd7620ef4e6d`，clone-run APISIX 空密钥 vs conf-local。
