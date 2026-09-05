# [运行时] GitHub OAuth 换票失败「无法与 GitHub 交换令牌」（provider 配置死代理 + 超时过长 + 错误被覆盖）

## 基本信息

- 案例编号：OPT-20260807-023-EXCHANGE-FAILED（2026-08-07 修复）
- 关联服务：taskGitOauth（:8002）、provider 配置（conf/auth/git-oauth/）、GitHub OAuth App
- 影响链路：项目页「OAuth 授权」→ GitHub 授权 → 回调换票 → 前端提示「授权失败：无法与 GitHub 交换令牌」，授权中断

## 失败现象

- 项目页再次点击「OAuth 授权」，GitHub 授权页正常，授权后回调跳回，前端提示
  「授权失败：无法与 GitHub 交换令牌」（旧文案）。
- 服务日志（trace_id=64004f02 定位）：换票请求**直连先耗时 46.8s**（http2 ResponseHeaderTimeout），
  随后回退 socks5://127.0.0.1:1080 又失败（`dial tcp 127.0.0.1:1080: connection refused`）。

## 根因链（三个独立缺陷叠加）

1. **死代理配置（主根因）**：provider 配置
   `conf/auth/git-oauth/providers/http-github-com--app-daydaymoney.yaml` 硬编码了
   `outbound_proxy: socks5://127.0.0.1:1080`——这是本机 `ssh -D` 开发代理，**生产未运行**。
   代码层 `http.Transport{Proxy: nil}` 正确（且 runAll 进程 environ 干净，无代理变量泄漏），
   但配置层注入的代理回退必然失败（connection refused）。
2. **超时过长放大失败**：`ResponseHeaderTimeout: 45s`。github.com 官方 API 正常 <1s，
   抖动/不可达时每个直连尝试挂满 45s，用户感知 ≈46s 的「无响应」。
3. **错误被覆盖（可观测性）**：`ExchangeGitHubCode` 循环中 `lastErr = err` 无条件覆盖，
   直连 46s 超时错误被回退代理的 connection refused 覆盖 → 日志只能看到代理错误，
   无法定位直连挂起是主因。

## 诊断线索

- 前端失败提示带 `data-traceId`（query trace_id）→ 服务日志按 trace_id 检索到完整换票时序
- 日志显示 46.8s 直连 http2 超时 → socks 1080 refused：两条线索都指向**出站网络路径**
- 检查代码层 Transport（Proxy: nil ✓）、进程 environ（无代理变量 ✓）后锁定 provider 配置的 outbound_proxy

## 修复

- 移除 provider 配置中硬编码的 `outbound_proxy`（死代理契约），注释说明生产必须直连
- `ResponseHeaderTimeout` 45s → 15s：直连抖动快速失败，失败路径上限 46s → ≤15s
- `ExchangeGitHubCode`：`lastErr` 只保留**首个（直连）错误**，不被回退代理错误覆盖（保真可观测性）
- 新增 `GitHubExchangeRejectedError` 错误类型：HTTP 4xx / 200+OAuth error body
  （bad_verification_code 等）→ 分类为「GitHub 明确拒绝」（前端 `exchange_rejected`，
  提示重新授权）；网络类失败保持 `exchange_failed`（提示检查网络）
- 前端提示分级：exchange_failed →「授权失败：无法连接 GitHub 服务器，请检查网络后重试」；
  exchange_rejected →「授权失败：GitHub 拒绝了授权请求（授权码可能已过期），请重新发起授权」
- 回归测试：4xx 分类（roundTripFunc 注入 400）、网络错误首错保真（dial timeout →
  proxy fallback → 断言保留直连错误）、handler 分类（ExchangeGitHubCodeFn 注入）、
  删除固化死代理的配置断言（改为断言 outbound_proxy 为空）

## 防再发

- **出站代理配置必须是活契约**：任何 `outbound_proxy` 写入 provider 配置前，
  须确认目标地址在目标环境（生产）可达；配置测试应断言代理为空或可达，禁止把
  本机开发代理（ssh -D / 127.0.0.1:xxxx）固化为生产配置
- **错误覆盖 = 可观测性事故**：多路径重试循环中，首个（主路径）错误必须保留；
  回退路径错误可追加，不可覆盖。凡 `err = ...` 覆盖语义的循环，review 时检查保真性
- 超时值以正常 p95 为参照（github <1s → 15s 足够），过长超时把故障拖成「假死」
- 伴读：失败经验 40（`40_github_oauth_exchange_timeout_outbound_proxy.md`）——同类死代理
  历史案例；本项目约束 `23_app_startup_no_env_proxy.md`（进程绝不继承 shell 代理环境）
