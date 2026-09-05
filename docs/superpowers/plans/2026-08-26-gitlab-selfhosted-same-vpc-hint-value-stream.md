# 价值流：自建 GitLab 同专有网络提示

- **日期**: 2026-08-26
- **Related**: 扩展既有「租户设置 · GitLab 资源购买与自建连接」（`docs/flows/value-stream-test-integration.wsd` TGR），非全新价值流。

## Related Value Streams

- `2026-07-15-tenant-gitlab-oauth-connection-*`：自建 OAuth 登记。本增量在同一页面区块增加网络提示，不改 OAuth 契约。
- `2026-07-18-tenant-gitlab-settings-resource-purchase-*`：内建 GitLab 配额。范围外。

## Increments

### I1 — 识别默认机器并展示提示（MVP）

管理员打开 GitLab 设置 → 拉取默认机器配置 → 有则展示同 VPC 文案与目标 ID。

验收：T1–T4。

### I2 — 创建专有网络 / 交换机入口

提示内按钮打开既有创建弹窗；无 VPC 时交换机按钮禁用；VPC 创建成功后可创建交换机。

验收：T5；浏览器打开创建弹窗。

### I3 — 失败可追踪

GET 失败展示 `data-traceId`。验收：T6。

不单独建 `conf/value-stream.yaml` 条目：测试为 vitest，不走 valueStream pytest runner。
