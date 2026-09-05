# 意图：自建 GitLab 平台可达性检测（内网豁免）

## 背景与目标

租户设置页「自建 GitLab 连接」只回显已保存的 OAuth 配置，不探测 `base_url` 是否可达。GitLab 关机后页面仍像正常连接。目标：平台控制面探测已保存地址并展示状态；**若管理员标记为内网服务，网络不可达属预期，不得报故障**。

## 范围与边界

- 范围内：`taskGitOauth` `GET /api/git-oauth/tenant-connection/tenant_id/{tid}/reachability/`；连接 CRUD 增加 `intranet`；`taskFE` 自建连接区块展示状态与内网勾选。
- 范围内：探测只打已保存的 `base_url`（忽略查询参数中的任意 URL，防 SSRF）。
- 范围外：不从任务机器/VPC 内探测；不改 OAuth 换票；不轮询（无 `setInterval`）。
- 伸缩要素：连接表一行一租户，非时间累积，不分区。

## 约束与风险

- 出站 HTTP：`Transport.Proxy = nil`，超时 ≤ 4s，不跟随重定向。
- 内网标记后跳过探测，避免控制面超时 4s 误报红字。
- 未配置连接不探测。
- 请求失败 UI 须带 `data-traceId`；「不可达」是探测成功的业务结果，不伪造 traceId。
- 租户成员可读探测；改 `intranet` 须管理员（走既有 PUT）。

## 验收标准

1. 已配置且未标内网：平台探测失败（超时/拒绝）时，`gitlab-self-hosted-connection` 内展示「无法连接」红字（`data-status=unreachable`）。
2. 已配置且已标内网：即使平台不可达，也展示「内网服务，平台无法访问属预期」（`data-status=skipped_intranet`），不得用危险色报故障。
3. 探测得到任意 HTTP 状态码（含 401/302）视为可达（`data-status=reachable`）。
4. 勾选「这是内网 GitLab」后保存，GET 连接返回 `intranet: true`；缺省 `false`。
5. `GET .../reachability/?url=` 不得改探测目标。
6. 禁止进程内轮询；仅页面加载/保存后/用户点「重新检测」各一次。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 探测自建 GitLab 可达性 | — | — | — | — | 纯查询，无状态变更 |
| 保存连接（含 intranet） | TENANT_GITLAB_OAUTH_CONNECTION_UPSERTED | Kafka 既有 | PUT handler | catalog 缓存失效 | 沿用既有 upsert 事件，增补 `intranet` 字段 |

## 路径分片键 / 幂等

- 路径含 `tenant_id`，分片键合适。
- GET 无副作用（L0）；PUT 幂等键为 `company_id`（一行一租户，既有 upsert）。

## 实施计划

1. DDL `intranet` + 领域分类（内网跳过探测）。
2. GET reachability + PUT 持久化 `intranet`。
3. 前端勾选 + 状态条 + 单测。
