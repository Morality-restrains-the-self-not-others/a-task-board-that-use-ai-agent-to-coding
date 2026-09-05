# [运行时] resolve-target-architectures 经失效本地代理拉 registry → proxyconnect refused

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-16
- 最后修改：2026-07-16
- 维护者：Trae AI 团队

## 现象

- `POST https://provider.daydaymoney.com/api/vendor/container-images/resolve-target-architectures/` → **400**
- `detail` 形如：

```text
Get "https://registry.cn-qingdao.aliyuncs.com/v2/.../manifests/x86_64-latest":
proxyconnect tcp: dial tcp 127.0.0.1:1234: connect: connection refused
```

- 同时可能伴随 `ai-provider` 在 runAll 中为 **failed**（:8010 无监听），公网边缘返回 **502**。

## 根因

1. 开发机 shell / runAll 父进程常带 `https_proxy=socks5h://127.0.0.1:1234`（本地 Clash/V2Ray 等），而 **1234 未监听**。
2. `taskAiProvider` 拉 OCI manifest 使用 Go 默认 `http.Client`，其 Transport 走 `ProxyFromEnvironment`，出站被导向失效代理。
3. 错误被 handler 原样写入 `detail` → 客户端见 400（非网络层 502）。

## 解决方案

1. **代码**：`taskAiProvider/infrastructure/registry_manifest.go` 的 `defaultRegistryClient()` Clone `DefaultTransport` 后设 `Proxy = nil`，registry 直连，不继承 shell 代理。
2. **启动**：`taskAiProvider/run.sh` 在 start 前 `unset` `http(s)_proxy` / `ALL_PROXY`（与 `mint_token.sh` 一致）。
3. **运维**：`ai-provider` failed 时经 runAll `POST /api/restart`（`{"name":"ai-provider"}`）重建拉起；确认 runAll `use_proxy=false`（默认会 strip 子进程代理环境变量）。

## 预防

- 凡出站访问公有云 registry / 国内镜像站的 `http.Client`，默认 **显式禁用环境代理**（或仅对需要翻墙的目的地启用）。
- 回归：`TestDefaultRegistryClientIgnoresEnvProxy`（故意设置死代理 `127.0.0.1:1` 仍应成功）。
- 排障时先 `ss -ltnp | rg ':8010|:1234'` 与 `tr '\\0' '\\n' < /proc/<pid>/environ | grep -i proxy`。

## 验证

```bash
# 期望 200 + target_architectures
TOKEN=$(bash taskAiProvider/scripts/mint_token.sh vendor | tail -1)
curl -sS -w '\n%{http_code}\n' -X POST \
  'http://127.0.0.1:8010/api/vendor/container-images/resolve-target-architectures/' \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"image_url":"registry.cn-qingdao.aliyuncs.com/ruandao/task2app-trae","version":"x86_64-latest"}'
```

## 关联

- **元规则**：[应用启动禁止使用环境变量 Proxy](../../01_project_constraints/23_app_startup_no_env_proxy.md)；根摘要 `.ai.md`；Cursor `.cursor/rules/app-startup-no-env-proxy.mdc`
- runAll 子进程代理剥离：`runAll/src/lifecycle_exec.go`（`stripProxyEnvVars` / `use_proxy`）
- SSO bridge sub 解析：`23_sso_bridge_missing_sub_string_claim.md`
