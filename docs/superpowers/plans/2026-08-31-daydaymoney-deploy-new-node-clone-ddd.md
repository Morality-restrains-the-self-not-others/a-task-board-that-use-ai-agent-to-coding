# DDD: 新节点 clone-run

- **日期**: 2026-08-31
- **边界**: 平台交付（daydaymoney-deploy），无业务聚合。

## 概念

| 类型 | 名称 | 说明 |
|------|------|------|
| Bounded Context | DeployAssembly | clone 根 + conf-local + Release 钉 |
| Entity | DeployRoot | 含 `conf/` 符号链接的目录 |
| Value Object | ConfLocalRelPath | 与 `conf/` 同相对路径 |
| Value Object | ReleasePin | `github://owner/repo/asset@tag` + content sha |
| Aggregate | CloneRun | layout + overlay + deploy-sync 一次装配 |

## 业务意图 → 事件

运维控制面，无 MQ。例外见意图文档。

旧路径 `db/task-auth/oidc_signing_key.pem`、`secrets/taskGateway` overlay：**删除为 SSOT**（一次性 migrate 拷入 conf-local 后不再双写）。
