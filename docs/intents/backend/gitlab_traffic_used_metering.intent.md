<!-- markdownlint-disable MD013 MD060 -->
# 功能意图：GitLab 已用流量计量下限

## 意图

GET 租户 GitLab 资源视图的 `traffic_used_gb` 取计量计数与扣费流水的较大值。
磁盘占用不再写入/抬高流量水位（避免 push 让「已用流量」持续增长）。
- 闸门：GitAccess deny when prepaid exhausted（ADR-0036）
- 计量：各区域 GitLab workhorse access 日志 `git-upload-pack` 的 `written_bytes` → taskBill `charge-gitlab-traffic`（ADR-0042）。**不**依赖容器上报 `received_bytes`。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|----------|--------|--------|--------|---------|
| GET 已用流量 | — | — | — | 纯查询 |
| GitLab 出站计量 | — | gitService traffic-shipper → taskBill `charge-gitlab-traffic` | — | 同步计量；无独立领域事件 |

## 验收

- `traffic_used_gb = max(计量计数, 扣费流水 GB)`（磁盘占用不作流量下限）
- 存量 `traffic_used_gb` 若等于当前磁盘换算 GB 且 < 1，视为历史磁盘下限，读取时丢弃
- 预购行不计入已用
- `reportGitlabTrafficUsageFloor` 只抬高、不降低；新行 `not_purchased`；供 upload-pack 采集调用
- 磁盘刷新不再抬高流量下限
- 容器克隆 `received_bytes` 不再作为生产计量（第三方镜像可不报）
- workhorse `written_bytes` 按 6 位小数 GB 增量写入；同 `wh:{correlation_id}` 重放不双计；github.com / 非平台 host 跳过
