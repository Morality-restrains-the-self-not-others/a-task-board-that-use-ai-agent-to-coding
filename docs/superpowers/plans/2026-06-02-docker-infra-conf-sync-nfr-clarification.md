# NFR 澄清: docker-infra conf 碎片同步

## NFR 概览

| 类别 | 等级 | 量化 |
|------|------|------|
| 可维护性 | L3 | conf-sync CI 零 diff；infra IP 单点修改 |
| 数据一致性 | L2 | 碎片与 SSOT 强一致（CI gate） |
| 可用性 | L2 | 远程 Redis 不可达时 health 失败可观测 |
| 性能 | L0 | 不适用 |
| 安全性 | L1 | Redis 不对公网暴露；仅 LAN dev 机 |

## 质量场景

### QS-01: infra IP 变更
- **刺激:** 修改 `conf/docker-infra/config.yaml` host
- **响应:** `conf-sync-all.sh` 后三处 `docker-infra.yaml` 同步更新
- **度量:** `check_conf_sync.sh` 通过

## 领域模型影响

- 小聚合 `DockerInfraBundle` + 值对象 `InfraEndpoint`
- loader 层 deep-merge 碎片，不改业务聚合
