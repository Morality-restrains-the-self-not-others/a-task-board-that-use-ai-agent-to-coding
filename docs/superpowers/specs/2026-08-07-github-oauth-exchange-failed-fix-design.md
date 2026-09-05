# GitHub OAuth 授权「无法与 GitHub 交换令牌」根因分析与修复设计

- **日期**: 2026-08-07
- **作者**: claude
- **状态**: 🎯 target（待批准）
- **关联服务**: taskGitOauth（:8002，Go）、taskFE（前端提示）、conf（provider 配置）
- **前置事故**: 同会话已修复 callback 404（8c47b8a 路由前缀回归，失败经验 81）

## 1. 问题现象

项目页点击「OAuth 授权」→ GitHub 授权页 → 回调 `/api/accounts/github/oauth/callback/`
（已不再 404）→ 等待约 **46 秒**后前端提示：

> 授权失败：无法与 GitHub 交换令牌

用户要求：① 优化该提示；② 分析原因；③ 制定解决方案并修复。

## 2. 🔍 根因分析（服务日志实证，trace_id=64004f02…）

### 时间线（每次失败两条日志）

```
1) GitHub 换票直连失败，回退 outbound_proxy: Post ".../oauth/access_token":
   http2: timeout awaiting response headers          ← 直连 HTTP/2 45s 响应头超时
2) GitHub 回调 authorization_code 换票失败: Post ".../oauth/access_token":
   socks connect tcp 127.0.0.1:1080->github.com:443:
   dial tcp 127.0.0.1:1080: connect: connection refused  ← 兜底代理未运行
```

### 根因链（三层）

| 层 | 事实 | 证据 |
|----|------|------|
| **R1 兜底代理失效（直接原因）** | `conf/auth/git-oauth/providers/http-github-com--app-daydaymoney.yaml` 声明 `outbound_proxy: socks5://127.0.0.1:1080`（2026-07-18 为 ssh -D 开发加速添加），**本机无该代理运行** → 兜底路径必失败 | 日志 `dial tcp 127.0.0.1:1080: connection refused`；curl 直连 github.com 1s 可达 |
| **R2 直连路径慢且偶发超时（诱因）** | Go HTTP/2 直连 github.com 偶发 `http2: timeout awaiting response headers`（`ResponseHeaderTimeout=45s`），整次回调耗时 46.8s | `duration_ms=46863`；`outbound_http.go` 45s 头超时 |
| **R3 可观测性缺陷（放大）** | `ExchangeGitHubCode` 中 `lastErr = err` 被代理错误**覆盖**直连错误 → 日志只见 socks 错误，直连 `http2 timeout` 原因被吞 | `oauth_clients.go:69` |
| **R4 测试固化（防再发缺口）** | `config_outbound_proxy_test.go` 断言 `GithubOutboundProxy == "socks5://127.0.0.1:1080"` —— 把本机开发代理固化为配置契约，阻碍移除 | 测试断言 |

### 为什么 curl 通而服务直连超时？

本机直连 github.com **不稳定**（HTTP/2 建连后响应头偶发 >45s 才到；curl HTTP/1.1 走不同路径 1s 返回）。
46s 超时 + 兜底死代理 = 每次换票 46s 后必然失败。

## 3. 方案设计

### 3.1 配置层（治本）— 移除死代理

删除 `http-github-com--app-daydaymoney.yaml` 的 `outbound_proxy: socks5://127.0.0.1:1080`
（生产不引用本机开发代理；github.com 直连可达，历史上 07-18 前无代理亦正常）。
同步**删除**固化了该值的 `config_outbound_proxy_test.go`（改为断言默认直连、无 outbound_proxy）。

### 3.2 代码层（健壮性 + 可观测性）— taskGitOauth

| 改动 | 文件 | 说明 |
|------|------|------|
| 错误保真 | `infrastructure/oauth_clients.go` | `lastErr` 只保留**首个**错误（直连优先），代理错误不覆盖；失败日志输出两个路径的错误 |
| 快速失败 | `infrastructure/outbound_http.go` | `ResponseHeaderTimeout: 45s → 15s`（github 官方 API 正常 <1s；直连抖动快速返回，不再拖 46s） |
| 错误分类 | `oauth_clients.go` + `src/browser_handlers.go` | 区分两类：**网络类**（直连+代理均失败 → 现有码 `exchange_failed`）与 **GitHub 拒绝类**（HTTP 4xx，如 code 过期/redirect_uri 不匹配 → 新码 `exchange_rejected`） |

### 3.3 前端层（提示优化 — 用户明确要求）— taskFE

`oauth_callback_hint_catalog_entity.js`：

| 码 | 现文案 | 新文案 |
|----|--------|--------|
| `exchange_failed` | 授权失败：无法与 GitHub 交换令牌 | **授权失败：无法连接 GitHub 服务器，请检查网络后重试**（网络类语义） |
| `exchange_rejected`（新增） | — | **授权失败：GitHub 拒绝了授权请求（授权码可能已过期），请重新发起授权** |

GitLab 同步保留 `exchange_failed` 语义（本次不改 GitLab 服务端分类，仅文案对齐「无法连接 GitLab 服务器」）。
现有 `data-traceId` 机制不变（失败回跳已带 trace_id）。

### 3.4 测试计划

| 层 | 测试 | 断言 |
|----|------|------|
| Go | 新增 `TestExchangeGitHubCodeNetworkErrorClassification`（mock transport 网络错误） | 返回网络类错误（走 `exchange_failed`） |
| Go | 新增 `TestExchangeGitHubCodeRejectedClassification`（mock 4xx） | 返回 `exchange_rejected` 语义 |
| Go | 删除 `TestLoadConfigGithubOutboundProxy` | 配置不再含 outbound_proxy（或断言为空） |
| FE | 更新 `UserGitSiteOAuthSettings.test.js` / hint catalog 测试 | 新文案 + 新码映射 |

## 4. 🕸️ Code Review Graph 分析

- `CRG partial`：图（93 nodes / 12 files）覆盖不完整且与 HEAD 不同步（built 9922ebd vs head 6a9d172，taskGitOauth 未入图）。
- 手动影响面核查：`ExchangeGitHubCode` 调用方 = `handleGithubCallbackWithSP`（唯一）→ `clearAnd` 码值变化 → 前端 `OAuthCallbackHintCatalog` 按码查文案（新码自动生效，无需改前端路由逻辑）。

## 5. 业务意图 → 事件对照

| 业务意图 | 事件 | 例外理由 |
|---------|------|---------|
| GitHub 授权换票（失败） | 无 | 纯运行时错误路径，无状态变更、无跨边界副作用；失败结果经 HTTP 302 回跳传递（trace_id 可观测） |

## 6. 价值流影响

- `value-stream.yaml` 无 git-oauth/github 相关流 → **无现有流受影响**，不新增流。

## 7. 🏛️ 架构变更影响

- **不需要更新架构**（v66 current 保持）：纯 bug 修复 + 文案调整 + 配置微调（移除一项死配置），无组件/数据流/基础设施增删改。
- 关联失败经验：81（callback 404，同链路前一环）。

## 8. 验收标准

1. `grep outbound_proxy conf/` 为空；服务重启后配置无 outbound_proxy
2. `go test ./...` 全绿（新增分类测试 + 删除死测试）
3. 前端 hint catalog 新文案生效（单测断言）
4. 用户重试授权：直连成功即换票成功跳回原页面；若 GitHub 侧拒绝（如 code 过期）显示「GitHub 拒绝了授权请求」而非笼统网络文案
5. 失败路径耗时上限从 ~46s 降至 ≤15s
