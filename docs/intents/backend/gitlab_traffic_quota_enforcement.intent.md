<!-- markdownlint-disable MD013 MD060 -->
# 功能意图：GitLab 出站流量配额强制执行

## 意图

租户预购流量为 0 或已用 ≥ 预购时，系统内建 GitLab **拒绝公网 clone/fetch/pull**
（`git-upload-pack`），**含任务贴走公网 Host 的服务器节点**。同区域内网与 GitLab CI job token 仍允许。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|----------|--------|--------|--------|---------|
| 查询是否允许下载 | — | POST gitlab-traffic-gate | GitLab GitAccess | 纯判定，无状态变更 |
| 超额拒绝计量 | — | chargeGitlabTraffic 402 | — | 未写入成功，无聚合跃迁 |

## 验收

- `gitlabTrafficDownloadAllowed(prepaid=0)` 公网 → false / `TRAFFIC_NOT_PURCHASED`
- `used >= prepaid > 0` → false / `TRAFFIC_QUOTA_EXCEEDED`
- 内网 → true / `INTRANET_SKIP`
- 磁盘刷新不再调用 `reportGitlabTrafficUsageFloor`
- GET 视图含 `traffic_download_allowed`
