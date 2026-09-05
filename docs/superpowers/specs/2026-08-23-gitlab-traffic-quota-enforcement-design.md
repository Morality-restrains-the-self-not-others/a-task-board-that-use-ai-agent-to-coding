# GitLab 租户流量配额强制执行 — 设计

- **Date**: 2026-08-23
- **Status**: accepted（/goal 自动采用）
- **Page**: `/tenant/{tid}/settings/gitlab-connection/`
- **ADR**: ADR-0036

## Context

页面 `data-testid="gitlab-traffic-used-gb"` 出现 `0.01049 GB / 未预购`。租户未买流量，已用却在涨。根因：

1. GitLab CE 无租户出站流量硬限（与磁盘 `repository_size_limit` 不对称）。
2. `charge-gitlab-traffic` 无 git 路径调用方，且扣费不检查预购。
3. 磁盘同步把 `disk_used_bytes` 幂等抬成 `traffic_used_gb`，push 也会让「已用流量」变大。
4. 区域 `total_bandwidth_mbps` 仅为展示，不是限速。

## Decision

| 层 | 行为 |
|---|---|
| 判定 | `prepaid<=0` 或 `used>=prepaid` → 公网 **与任务节点走公网 Host** download 拒绝；**GitLab CI 与同区域内网** 放行 |
| 执法 | GitLab `GitAccess#check_download_access!` → taskBill `gitlab-traffic-gate` |
| 计量 | 停止从磁盘刷新抬高流量水位；显示 `max(计量, 扣费流水)` |
| UI | 超额/未预购时警告条 + 购买链接；`traffic_download_allowed` |

## Role / NFR / DDD（压缩）

- **权限**：闸门为内部密钥；租户 GET 配额沿用既有 billing 鉴权。
- **路径分片**：闸门 body 含 `tenant_id`（从 `tenant-{id}` 解析）+ `region`（实例 conf `regionSlug`）。L2。
- **幂等**：闸门只读；重放同一判定。charge 超额不再累加。
- **领域**：`TenantGitlabResource` 增加下载是否允许；无新 Kafka（纯拒绝，无聚合跃迁）。GET 例外。
- **架构**：无新服务。GitLab→taskBill 为既有 initializer 注入 + 内部 API。不升 ArchiMate（同区域带宽展示先例）。

## 验收

1. 预购 0：公网 `git-upload-pack` 被 GitLab Forbidden；设置页有阻断提示。
2. 预购 > 已用：clone 允许。
3. 磁盘再增大：`traffic_used_gb` 不因磁盘刷新继续抬高。
4. CI 与同区域内网：闸门 `allowed=true`。任务节点走公网 Host 时与公网一样受配额约束（Docker NAT RFC1918 不豁免）。
