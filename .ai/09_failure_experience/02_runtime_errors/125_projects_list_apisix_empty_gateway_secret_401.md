# [运行时] 云端开发「加载项目失败」401 请先登录（APISIX 空网关密钥）

## 基本信息

- 案例编号：RT-20260901-0125
- 录入日期：2026-09-01
- 最后更新：2026-09-01
- 关联服务：taskGateway（APISIX）、task-project-service、taskFE Projects
- 关联意图：`docs/intents/platform/projects_list_gateway_secret_conf_local.intent.md`
- traceId：`7af49e7f-8e9e-4686-9470-dd7620ef4e6d`

## 失败现象

- 页面：https://www.daydaymoney.com/tenant/882297276515512320/projects
- 文案：「加载项目失败」（`p.text-red-600`，`data-traceId` 如上）
- 部署根：`~/bin/daydaymoney-deploy`

## 环境与上下文

- Loki `{job=~".+"} |= "<traceId>"`：task-auth `forward-auth` 200；task-project-service `GET /api/projects/tenant_id/882297276515512320` 401、duration_ms=0；网关 content-length 与「请先登录」JSON 一致。
- 用户已登录：`X-User-Id`、`X-Gateway-Auth-Verified: 1`、租户权限含 `project:view` / `page:nav.projects`。
- 直连 `:8016`：verified + **conf-local** 密钥 → 200 `[]`；空密钥头 → 401「请先登录」。

## 根因

1. `ApplyGatewayUser` 要求 `X-TaskGateway-Internal-Secret` 等于服务从 `conf-local` 读取的 `gatewayInternalSecret`。
2. `routes-to-apisix.py` 只读已跟踪 `conf/gateway/task-gateway/config.yaml`（`gatewayInternalSecret: ""`），且 `REPO=envs/current`，看不到部署根 `conf-local`。clone-run `routes-apply` 写出 `X-TaskGateway-Internal-Secret: ''`。
3. 源码仓 `apisix.yaml` 仍残留本地-dev 占位符，与 conf-local 碰巧一致；部署树重新生成后密钥被掏空。
4. 前端把非 403/404 的失败统一成「加载项目失败」，掩盖了 401 正文。

## 修复

1. `load_gateway_conf` 经 `merge_conf_local` 合并 `conf-local`，并从 `envs/current` 向上走到部署根。
2. 部署树 `routes-apply` 后 APISIX 注入与 conf-local 相同的密钥。
3. 项目列表 401 展示「请先登录」（或接口 message）并保留 `data-traceId`。
4. `requireTenantMember` 在 verified=1 但用户未提升时打 warn 日志。

## 验收

```bash
python3 taskGateway/scripts/test_routes_to_apisix_merges_conf_local.py
# 部署树 apisix.yaml 不得再出现：
#   X-TaskGateway-Internal-Secret: ''
```

## 预防

- 凡生成 APISIX/运行时 YAML 的脚本，读 `conf/<app>` 必须叠加 `conf-local`（与 `conf_loader` 一致）。
- 排障「加载项目失败」时先看 401 正文与网关密钥头是否为空，不要只改前端文案。
