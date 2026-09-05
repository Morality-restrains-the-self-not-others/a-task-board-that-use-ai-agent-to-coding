# 价值流：可插拔多区域 gitService

- **日期**: 2026-08-18
- **设计**: `docs/superpowers/specs/2026-08-18-pluggable-multi-region-gitservice-design.md`

## 相关既有流

- 租户 GitLab 资源购买、SystemAdmin 区域 CRUD、GitLab OIDC SSO（`conf/value-stream.yaml`）

## 增量切片（按交付价值排序）

| # | 增量 | 用户价值 | 依赖 |
|---|------|----------|------|
| I1 | Conf + SH 精简实例配方 + 边缘 nginx 片段 | 第二实例可部署 | — |
| I2 | taskBill 按 region 路由开通 + 取消静默默认 | 开通打到正确实例 | I1 可并行 |
| I3 | Seed/登记 `tencent-sh-1` + hybrid pending | 商品可售 | I2 |
| I4 | FE 租户选购必选 region + 空态 | 租户能买对区域 | I3 |
| I5 | SystemAdmin 字段/pending 重试 + OIDC 第二 client | 运营可插拔 | I1–I3 |
| I6 | SH 实际起容器 + DNS/nginx 切流验收 | 公网可达 | I1 |

## Fields（三段式）

- `taskBill.billing_gitlab_region.gitlab_web_url`
- `taskBill.billing_gitlab_region.gitlab_api_base`
- `taskBill.billing_gitlab_region.admin_private_token`
- `taskBill.billing_tenant_gitlab_resource.region`
- `taskBill.billing_tenant_gitlab_resource.provisioning_status`

## YAML 追加意图（实现时写入 conf/value-stream.yaml）

新增 stream 名：`pluggable-multi-region-gitservice`（步骤对应该仓库 Go/FE 单测路径）。
