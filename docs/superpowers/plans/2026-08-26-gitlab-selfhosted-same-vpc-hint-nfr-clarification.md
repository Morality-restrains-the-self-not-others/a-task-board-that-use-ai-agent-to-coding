# NFR 澄清：自建 GitLab 同专有网络提示

- **日期**: 2026-08-26
- **默认级别**: L2 Standard

## 路径分片键强制审视

| 路径 | 分片 ID | 适配？ | 可伸缩性 | 动作 |
|------|---------|--------|----------|------|
| `GET /api/cloud/server-config-default/tenant_id/{tid}/` | `tenant_id` | 是，与 `company_id` 一致 | L1：按租户列表，行数=授权数（个位数～数十） | 保持 tenant 过滤；禁止无 tid 全表 |
| `POST /api/cloud/create-vpc/tenant_id/{tid}/` | `tenant_id` + `authorization_id` | 是 | L2 云厂商限流，非本增量 | 复用既有 |
| 前端 `/tenant/:tenant/settings/gitlab-connection/` | `tenant` | 是 | L0 单页设置 | — |

升级触发：单租户默认配置 > 1000 条（远超当前授权模型）再考虑分页。

## 幂等性强制审视

| 路径 | 副作用 | 级别 | 重复边界 / 键 | 重放 |
|------|--------|------|---------------|------|
| GET default-config | 无 | L0 | 纯查询 | — |
| 打开创建弹窗 | 无 | L0 | 纯 UI | — |
| POST create-vpc（既有） | 有：云 VPC | L3（云资源） | 云厂商侧创建；本页不新造幂等键 | 双击由弹窗 `loading` 抑制；不在本增量改 SDK |
| POST create-vswitch（既有） | 有 | L3 | 同上 | 同上 |

资金路径不涉及。用户点击创建走既有 modal；本增量按钮仅 `Anti-Replay-OK: open modal`。

## 质量场景

- 保密性 L2：提示只展示 VPC/交换机 ID，不展示云密钥。
- 可用性 L2：GET 失败不阻断 OAuth 表单。
- 可观测 L2：失败带 `data-traceId`。
- 性能 L2：一次 GET，无轮询。

## 领域模型影响

无新聚合。只读投影「默认机器网络摘要」。创建 VPC 不自动写入 `cloud_server_config_defaults`（保持机器策略为唯一写入口）。
