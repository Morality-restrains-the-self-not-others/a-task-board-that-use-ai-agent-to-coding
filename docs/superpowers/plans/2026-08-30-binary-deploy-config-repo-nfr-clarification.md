# NFR 澄清：二进制部署与独立配置仓

- **Date:** 2026-08-30
- **Value stream:** `docs/superpowers/plans/2026-08-30-binary-deploy-config-repo-value-stream.md`
- **Increments in scope:** I1 FindConfigRoot、I2 releases 钉 + deploy-sync

## 路径分片键审视

| 路径 | 分片 ID | 判定 | 可伸缩性等级 | 结论 / 动作 |
|------|---------|------|-------------|-------------|
| 进程启动读 `$CONF_ROOT/<area>/<app>/config.yaml` | 无 | 缺键 | L0 | 单机部署根；升级触发：多区域多 DEPLOY_ROOT 时按 env 目录分，不按租户 |
| `releases.yaml` artifacts.<service> | 服务名（非租户） | 不适合租户分片 | L0 | 整栈钉版本；热修单服务仍一主机 |
| GitHub Packages GET 产物 | 无 tenant | 缺键 | L0 | 低频发布；升级触发：多区域副本时按 region 钉 URL（非本增量） |
| runAll start_command exec bin | 无 | 缺键 | L0 | 单主机进程监督；不引入租户维 |

L0 理由：交付拓扑是平台运维、单部署根；可预见窗口无租户级水平扩展。升级触发：第二个独立部署区域需要 `envs/<region>/` 与区域产物坐标。

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| FindConfigRoot | 无 | n/a | n/a | — | L0 只读定位 |
| deploy-sync 拉产物 | 写 `$DEPLOY_ROOT/bin` | 重复跑 sync / 网络重试 | 同一 service+sha | `service + sha` | 已存在且 sha 文件匹配 → skip；下载失败不覆盖 last-good（L3） |
| runAll start | exec 已有 ELF | 重复点启动 | 端口上已有进程 | 既有 adopt | 与 ADR-0027 一致 |
| 业务 Kafka | 无本增量 | — | — | — | 例外：运维交付无领域事件 |

禁止用 `tenant_id` 作本增量幂等键。

## 类别支撑程度（I1–I2）

| 类别 | 等级 | 说明 |
|------|------|------|
| 可伸缩性 | L0 | 见路径表 |
| 数据一致性 | L2 | last-good 磁盘与钉 sha 一致才切换 |
| 容错 | L3（拉产物） | 失败不覆盖；不在部署机编译 |
| 安全 | L2 | 私有仓、密钥不进 Git、部署 token 只读 |
| 可观测性 | L2 | sync/定位打结构化日志，不含 token |

## 质量场景

1. **刺激**：CONF_ROOT 指向仅有 `base.yaml` 的目录，无 `.gitmodules`。**响应**：FindConfigRoot 成功，ReadAppConfig 读该树。
2. **刺激**：fetcher 失败且磁盘已有 bin。**响应**：旧文件字节不变，sync 非零或记 error 但不删除 last-good。
3. **刺激**：start 时无 bin。**响应**：失败且日志指向 deploy-sync，不调用 `go build`。

## 领域模型影响

- `ArtifactPin`（service, sha, package URL）为值对象；一致性边界是「单服务磁盘文件 + sidecar sha」。
- 无新聚合持久化到业务 DB。
