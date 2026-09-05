<!-- markdownlint-disable MD013 MD060 -->
# 功能意图：GitLab 连接页已用流量

## 背景与目标

租户设置「GitLab」页 `data-testid="gitlab-traffic-used-gb"` 原先只汇总
`billing_usage` 里「GitLab 流量费」扣费流水。克隆/拉取从未调用
`charge-gitlab-traffic`，且余额为 0 时扣费会失败，因此即使用户已经
用系统 GitLab 克隆，页面仍显示「0 GB / 0 GB」。

已用流量应反映**出站计量水位与扣费流水**的较大值。各区域 GitLab
workhorse 在 `git-upload-pack` 完成后把 `written_bytes` 经 sidecar POST
`/api/internal/taskbill/charge-gitlab-traffic/`（按实际字节、不按 1GB 取整；ADR-0042）。
磁盘占用不再作为流量下限持续抬高（push 会误增「已用流量」）。预购为 0 时公网 clone 由 GitAccess 闸门阻断（ADR-0036）。
容器是否上报 `received_bytes` **不影响**计量。

## 范围与边界

- 范围内：`taskBill` GET 单区 `gitlab-resources` 的 `traffic_used_gb`；
  设置页 composable 在无 `?region=` 时从 `resources[]` 发现区域再拉详情；
  GitLab workhorse `written_bytes` → `charge-gitlab-traffic`（ADR-0042）
- 范围外：SSH gitlab-shell 无 `written_bytes` 时的漏计；容器进度上报字段本身

## 约束与风险

- 无 workhorse sidecar 时，已用流量为扣费流水或历史计量；与当前磁盘换算相等的历史下限会被丢弃
- SSH 出站字节见后续 OPT；公网 clone 由流量闸门阻断
- `data-testid="gitlab-traffic-used-gb"` 保持不变
- 预购为 0 时分母展示「未预购」，避免「0 GB / 0 GB」被当成计量故障

## 验收标准

1. 已开通区域且仅有磁盘占用、无流量计量时，GET `traffic_used_gb` 不因磁盘抬高（含历史磁盘下限存量）
2. 设置页无预选区域时先 GET `resources[]`，再带 `?region=` 拉详情，
   映射 `traffic_used_gb`
3. 预购 0 时 `gitlab-traffic-used-gb` 含「未预购」而非 `/ 0 GB`

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|----------|--------|--------|--------|---------|
| GET 已用流量 | — | — | — | 纯查询；磁盘占用不计入已用流量 |

## 实施计划

1. `gitlabTrafficUsedGB`：计量水位与扣费流水较大值（磁盘不参与）
2. `reportGitlabTrafficUsageFloor` 仅记录真实出站计量，磁盘刷新不再调用
3. 前端 `load()` 发现区域后拉单区详情；未预购由 GitAccess 闸门阻断
