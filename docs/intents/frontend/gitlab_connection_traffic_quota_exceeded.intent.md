<!-- markdownlint-disable MD013 MD060 -->
# 功能意图：GitLab 连接页流量超额阻断提示

## 意图

当 `traffic_download_allowed=false`（未预购或已用 ≥ 预购）时，设置页说明公网（含任务贴节点）已阻断、同区域内网与 CI 仍可拉取。
`gitlab-traffic-used-gb` 旁展示阻断说明，并提供资源订单购买入口。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 例外理由 |
|----------|--------|---------|
| 展示阻断 | — | 纯 UI |

## 验收

- `data-testid="gitlab-traffic-quota-blocked"` 在未预购且已开通区域时可见
- 含购买链接 `/tenant/{tid}/billing/orders/create/`
- `gitlab-traffic-used-gb` 超额时带警示样式
- 真实 `<a href>`（Anti-Replay-OK: navigation link）
