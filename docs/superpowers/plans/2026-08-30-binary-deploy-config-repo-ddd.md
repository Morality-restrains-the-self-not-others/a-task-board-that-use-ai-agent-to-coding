# DDD：交付上下文 — 二进制部署与配置根

- **Date:** 2026-08-30
- **NFR:** `docs/superpowers/plans/2026-08-30-binary-deploy-config-repo-nfr-clarification.md`

## Bounded Context

**Delivery（平台交付）** — 与 Auth/Bill/Task 正交。不持有业务表。

## 战术模型（I1–I2）

| 类型 | 名称 | 职责 |
|------|------|------|
| VO | `ConfigRoot` | 含 `conf/base.yaml` 的部署/仓根路径 |
| VO | `ArtifactPin` | service → sha + package 坐标 |
| Entity | `PinnedBinary` | 磁盘 ELF + sidecar sha；失败不覆盖 |
| Domain service | `ResolveConfigRoot` | env 优先于向上查找 |
| Domain service | `SyncPinnedArtifacts` | 按钉拉齐；幂等键 service+sha |
| Port | `ArtifactFetcher` | 下载包；测试 fake，生产 GitHub Packages |

## 聚合 / 一致性

`PinnedBinary` 一致性边界 = 单文件 rename。不跨服务事务。

## 领域事件

无。运维交付，意图表已例外。禁止为 sync 成功发业务 Kafka。

## 端口

```
ArtifactFetcher.Fetch(ctx, pin) (tmpPath, error)
ConfigRootResolver.Resolve() (ConfigRoot, error)
```

基础设施：OS env + filesystem；GitHub Packages HTTP（本会话用 fake 即可让测试绿）。
