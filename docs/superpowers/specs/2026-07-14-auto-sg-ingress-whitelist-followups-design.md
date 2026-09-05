# 增量设计：自动 SG 三项优化（2026-07-14）

基于 `2026-07-14-auto-sg-ingress-whitelist-design.md`。

## 变更目标

1. **创建时自动查询用户公网 IP**：启动（创建 SG）时强制向 taskAuth `client-ip` 查询最新公网 IP，写入请求体 `client_public_ip`（不用导航栏旧缓存）。
2. **可配置平台探测 CIDR**：环境变量 `TASK2APP_SG_EXTRA_INGRESS_CIDRS`（逗号分隔），写入自动 SG 入站；事件可带 `extra_ingress_cidrs`。
3. **拆分硬件面板启动逻辑**：将启动请求构建抽到 util，面板显式传 `client_public_ip`，并净减少 `ServerConfigHardwarePanel.vue` 行数以通过门禁。

## 实现要点

| 层 | 改动 |
|----|------|
| 前端 `publicClientIp.js` | `fetchPublicClientIp({ force: true })` 强制刷新 |
| 前端 `hardwarePanelStartVm.js`（新） | 组装启动请求 + 自动 SG 时附带 client_public_ip |
| 前端硬件面板 | 调用 util；模板启动路径同样附带 |
| taskCloudService | 透传 `extra_ingress_cidrs`（env）；client_ip 仍优先 body |
| taskEvents | Phase A/B 合并 host IP + extra CIDRs；禁止 `0.0.0.0/0` |

## 配置示例

```bash
# SaaS 出口 / 管控网段（按实际填写）
export TASK2APP_SG_EXTRA_INGRESS_CIDRS="203.0.113.0/24,100.104.0.0/16"
```

## 后续完成（2026-07-14）：硬件面板区域拆分

`ServerConfigHardwarePanel.vue` 已收束为组合壳（≤100 行），业务逻辑迁至：

- `front_project/app/src/composables/hardwarePanel/useServerConfigHardwarePanel.js`
- `front_project/app/src/components/hardware-panel/*`（Header / 模版摘要 / 临时配置工具条 / 网络选择 / 实例过滤与列表 / 价格 / 自动释放 / 启停控制）

各 `.vue` 子组件均 ≤500 行门禁；启动路径仍经 `hardwarePanelStartVm.js` 显式附带 `client_public_ip`。

