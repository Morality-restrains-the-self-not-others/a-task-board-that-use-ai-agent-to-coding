# 多入口同源 API（域名 + IP）设计

- **日期**: 2026-07-11
- **状态**: approved
- **迭代**: multi-entry-same-origin-api
- **作者**: claude
- **架构版本**: v16 🎯 target
- **触发问题**: `https://www.daydaymoney.com/auth/login/` 打开后接口报错（Mixed Content）

## 问题分析

### 现象

用户经 HK HTTPS 入口访问登录页时，策略/协议等公共接口失败。

### 根因

| 环节 | 实际 |
|------|------|
| 页面 Origin | `https://www.daydaymoney.com`（HK nginx → Vue `:4000`） |
| 前端 `VITE_API_BASE_URL` | 绝对地址 `http://183.250.1.132:18081` |
| 浏览器 | HTTPS 页请求 HTTP API → **Mixed Content 拦截** |
| 边缘路由 | `location /` 全量打到 Vite；`/api` 未分流到 gateway（即便改相对路径也会拿到 HTML） |

### 最终目标（已确认）

- 可挂**多个域名**与 **IP** 入口
- 全部入口采用 **A：同源反代**（浏览器不直连 `:18081`）

## 目标架构

```text
任意公网入口（域名 / IP）
  Edge nginx (TLS 可选)
    ├─ /        → Vue (:4000)   [或生产静态]
    └─ /api/    → task-gateway (:18081) → Django / Go
```

浏览器始终请求**当前入口 Host** 的 `/api/...`（`API_BASE_URL = ''`）。

本地直连 Vite `:4000`：Vite `server.proxy['/api']` → gateway，与边缘行为一致。

## 方案明细

### 1. 边缘 nginx（每个入口一份，模式相同）

以 HK `example.com` 为第一落地：

- `location /api/` → `http://183.250.1.132:18081`
- `location /` → `http://183.250.1.132:4000`
- 必须转发：`Host`（公网 Host）、`X-Forwarded-Proto`、`X-Forwarded-For`、`X-Real-IP`
- WebSocket/HMR：保留 Upgrade（dev）

### 2. Vue API 基址

- 默认 `API_BASE_URL = ''`（同源相对路径）
- 配置层：`conf/frontend/vue/config.yaml` 的 `apiBaseUrl` 改为空或显式 same-origin 语义；停止把 local gateway 绝对 URL 注入为浏览器默认基址
- 保留 `window.__TASK2APP_API_BASE_URL__` 作为例外覆盖（一般不需要）

### 3. Vite 开发代理

- 恢复 `server.proxy['/api']` → task-gateway origin（来自 conf）
- 保证 `http://IP:4000` 直连时 `/api` 仍可用

### 4. 公网入口白名单 `publicEntryOrigins`

新增配置（建议落在 `conf/base.yaml` 或独立 `conf/frontend/public-entries.yaml`），列出：

- `https://www.daydaymoney.com`
- `https://example.com`
- 既有 `http://183.250.1.132:4000` 等 IP 入口
- 日后其它域名按行追加

驱动：

- Django `CSRF_TRUSTED_ORIGINS` / `ALLOWED_HOSTS`（及 Host 归一化）
- Vite `allowedHosts`
- gateway CORS（仅 **直连 Vite→gateway** 兜底仍需要；同源反代路径下浏览器不跨源）

### 5. Cookie / CSRF / Host

- 同源后 `Set-Cookie` 落在入口 Host（正确）
- `X-Forwarded-Proto=https` 避免 Secure cookie / 绝对跳转误判
- Django 须接受公网 `Host: www.daydaymoney.com`（ALLOWED_HOSTS / allowedExtendHosts）

### 6. OAuth / OIDC（分期）

- **本迭代**：登录页公共 GET API 与密码登录 CSRF 通路
- **后置**：各入口 OAuth redirect_uri 注册；OIDC issuer 与回调与多入口对齐

## 本迭代范围

### 必做

1. HK nginx：`/api/` 分流到 `:18081`
2. Vue：浏览器默认同源 `API_BASE_URL`
3. Vite：`/api` 代理到 gateway
4. 白名单加入 daydaymoney（及现有 IP 入口）
5. 验收登录页公共接口

### 不做

- 不为每个域名单独打前端包
- 不强制浏览器直连 `:18081`
- 不新增 Python HTTP 接口
- 不在本迭代完成全部 OAuth 多回调注册

## 验收标准

1. 打开 `https://www.daydaymoney.com/auth/login/`：无 Mixed Content
2. `GET /api/public/system-feature-policy/`、`/api/privacy-policy/public/current/`、`/api/license-agreement/public/current/` 经入口返回 JSON（非 HTML），HTTP 2xx
3. 本地 `http://127.0.0.1:4000`（或配置的 IP:4000）经 Vite 代理访问同一组 API 仍可用
4. 新增域名的操作文档化：DNS → 边缘 TLS + 同构 nginx location + 白名单一行

## Domain Concept Inventory（轻量）

| 概念 | 说明 |
|------|------|
| **Bounded Context** | 平台接入 / 边缘入口（非新业务域） |
| **Key Entity** | PublicEntry（逻辑）：origin + TLS + 反代目标 |
| **Aggregate** | 配置侧入口白名单集合 |
| **Domain Events** | 无新增业务事件 |

## 价值流影响

- 影响流：用户与认证 → 登录页可达与会话建立
- 字段：主要为配置（origins / Host），无新业务表字段
- 测试：补充 E2E 或 curl 验收入口同源 `/api`；Playwright `BASE_URL` 可用 daydaymoney HTTPS

完整切片交 `/3-value-stream`。

## 🏛️ 架构变更影响

- **迭代版本**: v16 🎯 target
- **迭代名称**: 多入口同源 API
- **作者**: claude
- **设计日期**: 2026-07-11 17:26
- **新增文件**（每个视图三类伴生格式）:
  - 🆕 `docs/architecture/v16-application-integration-20260711-1726-claude.puml`
  - 🆕 `docs/architecture/v16-application-integration-20260711-1726-claude.archimate`（含 Plateau/Gap/WP 架构变迁视图）
  - 🆕 `docs/architecture/v16-application-integration-20260711-1726-claude.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] Edge nginx（多入口同源反代）
  - 🟡 [MODIFIED] Vue — `API_BASE_URL` 默认同源
  - 🟡 [MODIFIED] Vite — `/api` 代理恢复
  - 🟡 [MODIFIED] conf — `publicEntryOrigins` 白名单
  - 🔴 [DEPRECATED] 浏览器默认绝对 `http://IP:18081` API 基址（Mixed Content 路径）

### .archimate 架构变迁要点

| 元素类型 | 内容 |
|----------|------|
| **Plateau v15** | relay 所选镜像启动（既有 target 基线之一） |
| **Plateau v16** | 多入口同源 API |
| **Gap** | HTTPS 入口仍用 HTTP 绝对 gateway → Mixed Content |
| **WorkPackage** | WP-v16-multi-entry-same-origin-api |
| **视图** | 架构变迁 v15→v16；目标拓扑 Browser→Edge→Vue/Gateway |

## 审批记录

- **总体设计**: approved（用户 2026-07-11）
- **入口模式**: A — 全部同源反代
- **Python 新接口**: 未触发
