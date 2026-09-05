# 意图：打开容器页面前补齐安全组白名单

- **日期**: 2026-07-23
- **状态**: 已实现
- **关联页面**: 任务详情「打开容器页面」`a#open-container-page-btn`

## 背景

自动安全组仅放行用户公网 IP / 平台出口等白名单。浏览器直连 `http://{public_ip}:8765/ui/...` 时，若当前出口 IP 未在白名单（IP 漂移、代理/Surge DIRECT 与边缘 client-ip 不一致），会出现 Connection timeout，表现为「链接打不开页面」。

**不做** SaaS 反向代理容器控制台（会把容器 UI/SSE 流量压到 SaaS）。

## 验收标准

- [x] `POST …/cloud/compute/ensure-client-ingress/?task_id=`：将 body/`XFF` 解析的客户端公网 IP 写入任务 VM 安全组（Aliyun，幂等）
- [x] 点击「打开容器页面」先 ensure，再 `window.open(container_page_url)` 直连容器
- [x] mock / relay-local / 无 SG 时 skip，不阻断打开
- [x] 前端尽力收集边缘 client-ip + STUN srflx IP 一并授权

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 例外理由 |
|---------|--------|---------|
| 打开容器页面前补齐 SG | — | 同步云厂商 API，无 MQ 事件 |

## 变更记录

- 2026-07-23：初版 — 相对仅启动时写一次 client_public_ip，增加打开时补授权。
