# 测试意图: 云端开发项目列表须用 conf-local 网关密钥注入 APISIX

## 测试目标

证明 APISIX 路由生成读取 `conf-local` 的 `gatewayInternalSecret`，避免空串注入导致项目列表 401「请先登录」；前端 401 展示该文案并带 traceId。

## 测试分层

| 层 | 覆盖 |
|----|------|
| 生成器单元 | 临时 `conf/` 空密钥 + `conf-local/` 非空 → `load_gateway_conf` 返回 local 值 |
| 生成器单元 | `_token_route_transform` 把合并后的密钥写入 `X-TaskGateway-Internal-Secret` |
| 服务回归 | 已校验网关头但密钥为空 → 列表 401；密钥匹配 → 200 |
| 前端 | 401 响应展示「请先登录」且 `data-traceId` 绑定 |

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| GW-1 | conf `gatewayInternalSecret: ""`，conf-local 为 `from-conf-local` | `load_gateway_conf(repo)` | 返回 `from-conf-local` |
| GW-2 | 同上 | `_token_route_transform(conf)` | `proxy-rewrite` 头等于 `from-conf-local` |
| GW-3 | 无 conf-local | `load_gateway_conf` | 保持 conf 空串（不发明密钥） |
| API-1 | `GatewayInternalSecret` 已配置，请求 verified=1 但密钥头为空 | GET `/api/projects/tenant_id/{tid}` 经 `gatewayUserMiddleware` | 401 且 body 含「请先登录」 |
| API-2 | verified=1 且密钥头匹配 | 同上 | 200 |
| FE-1 | `apiFetch` 401 + `message: 请先登录` + traceId | 挂载 Projects | 红字为「请先登录」，`data-traceId` 为该 id |

## 数据与环境

- 生成器测例用 `tmp_path`，不读本机真实 `conf-local` 密钥值做断言。
- Go 测例用 `setupTestDB`；`TaskTenantURL` 保持空以跳过租户 HTTP。

## 通过标准

```bash
python3 taskGateway/scripts/test_routes_to_apisix_merges_conf_local.py
cd taskProjectService && go test -count=1 -run 'TestProjectsListGateway' ./src
cd taskFE/app && npx vitest run src/views/Projects.loadError.traceId.test.js
```

## 业务意图 → 事件对照（测试）

只读列表与网关配置对齐，不断言 Kafka。
