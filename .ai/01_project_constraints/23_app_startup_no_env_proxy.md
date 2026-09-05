# 应用启动禁止使用环境变量 Proxy

- **版本**：1.2.0
- **日期**：2026-07-16
- **适用范围**：monorepo 内**所有**由 runAll / `*/run.sh` / systemd / 手工拉起的业务与基础设施进程（Go / Python / Node / sidecar）；以及这些进程内的出站 HTTP(S) 客户端

## 目标

区分两类用途，避免开发机「网络加速代理」泄漏进应用运行时：

| 用途 | 是否允许使用 `HTTP(S)_PROXY` / `ALL_PROXY` 等 |
|------|-----------------------------------------------|
| **开发者本机工具**（浏览器、`curl`、`go`/`npm` 拉依赖、Claude/Cursor 等） | ✅ 允许；仅为开发时网络加速 |
| **应用程序进程**（业务服务、编排拉起的子进程、其出站 Client） | ❌ 禁止继承或依赖环境代理 |

历史踩坑：进程继承 `https_proxy=socks5h://127.0.0.1:1234` 而本地代理未起 → OCI registry 拉取 `proxyconnect … connection refused`（见失败经验 24）。

## 强制要求（禁止忽略）

1. **默认无代理启动**：经 runAll、各服务 `run.sh`、生产/预发部署脚本启动的进程，**不得**把宿主机 shell 的 `http_proxy` / `https_proxy` / `HTTP_PROXY` / `HTTPS_PROXY` / `ALL_PROXY` / `all_proxy`（及对应 `NO_PROXY`/`no_proxy` 若仅服务代理场景）当作业务出站配置。
2. **runAll 全局默认**：`conf/runAll.yaml` 的 **`use_proxy` 必须为 `false`**（或省略且实现默认 false）。禁止为「方便翻墙」把全局 `use_proxy` 改为 `true` 并长期保留。
3. **启动脚本**：各服务 `run.sh` / 等价入口在 `exec` 业务二进制前应 `unset` 上述代理变量（与 `taskAiProvider/run.sh`、`mint_token.sh` 一致），防止手工 `./run.sh` 时继承 IDE/shell 代理。
4. **出站 HTTP 客户端纵深防御**：访问公有云 / 国内镜像站 / 支付网关等**生产依赖**的 `http.Client`（Go）、`requests`/`httpx`（Python）、`fetch`/`axios`（Node）等，**不得**默认信任环境代理；应显式直连（例：Go `Transport.Proxy = nil`），或仅对**明确文档化的、可选的**加速端点使用独立配置（不得静默读全局 env）。
5. **Agent 义务**：新增服务、改启动脚本、审查 PR、排障 `proxyconnect` / 出站超时失败时，若发现应用进程环境含死代理或 Client 走 `ProxyFromEnvironment` 且无豁免理由，**必须改掉**，不得合并、不得建议「生产再关代理」。

## 允许与禁止对照

| 场景 | 正确 | 错误 |
|------|------|------|
| 开发者 shell 加速 | 本机导出 `https_proxy=…` 给 curl/go | 指望业务服务继承同一代理访问 Aliyun registry |
| runAll 拉起服务 | `use_proxy: false`，子进程 env 被 strip | 全局 `use_proxy: true` 导致全站出站经 127.0.0.1:1234 |
| 服务 `run.sh` | start 前 `unset http_proxy https_proxy …` | 直接 `exec` 二进制并继承 Cursor/终端代理 |
| Go registry/HTTP Client | `Clone()` DefaultTransport 后 `Proxy = nil` | `&http.Client{}` 默认走环境代理 |
| 单服务临时调试翻墙 | 书面例外 + 短生命周期 + 文档说明；优先改工具侧 | 改全局 `use_proxy` 并提交 |

## 与 runAll 实现的关系

- 编排器已实现：`use_proxy: false` 时 `stripProxyEnvVars`（见 `runAll/src/lifecycle_exec.go`）。
- UI `GET/POST /api/proxy-config` 可临时切换；**默认与仓库 conf 必须保持 false**；会话内打开仅用于排障，不得写回 conf 为 true。
- 本元规则要求：**编排剥离 + 启动脚本 unset + Client 直连** 三层一致，任一缺口都可能复发。

## 验收自检

```bash
# 1) 编排配置
rg -n '^use_proxy:' conf/runAll.yaml   # 期望 false 或不存在（默认 false）

# 2) 运行中业务进程不得带死本地代理（将 <pid> 换成服务 pid）
tr '\0' '\n' < /proc/<pid>/environ | rg -i '^(https?|all)_proxy=' || echo 'ok: no proxy env'

# 3) 本地加速代理未起时，业务出站仍应成功（例：provider resolve）
# 不得出现 detail 含 proxyconnect / 127.0.0.1:1234

# 4) Aliyun Tea 禁止手写 RuntimeOptions()
python3 db/scripts/ci/check_aliyun_tea_direct_runtime.py

# 5) 其它云 SDK 须走 direct_network 助手
python3 db/scripts/ci/check_cloud_sdk_direct_network.py
```

## 历史踩坑（交叉引用）

- Provider 拉 Aliyun manifest 经失效 SOCKS：`.ai/09_failure_experience/02_runtime_errors/24_provider_registry_proxyconnect_1234.md`

## 落地实现（存量）

| 层 | 实现 |
|----|------|
| 启动脚本 | 各服务 `run.sh` / `start.sh` / `runall-*.sh` 在 exec 前 `unset` 代理（与 `taskAiProvider/run.sh` 同文案） |
| Go 进程 | `shareLib/tracelog`：`Init`/`InitConsumer` 调用 `disableDefaultEnvProxy()`；可用 `tracelog.DirectClient(timeout)` |
| Go Aliyun | `taskCloudService`/`taskEvents`：`stripOutboundProxyEnv` + Config 上 `HttpProxy`/`HttpsProxy`/`Socks5Proxy` 置空 |
| Python requests | `core/utils/http_client.py`（`trust_env=False`）；出站热点 Session 显式 `trust_env=False` |
| Python urllib | `core/utils/urlopen_direct.py`（`ProxyHandler({})`）；Django↔Go 同机 client 已改用 `urlopen_direct` |
| Python Aliyun Tea | `cloud/providers/aliyun/direct_network.py`：`apply_direct_network(Config)` + **`direct_runtime_options()`**（覆盖 env）；**禁止**手写 `RuntimeOptions()` |
| Python 腾讯云 SDK | `cloud/providers/tencent/direct_network.py`：`direct_http_profile(...)` + `apply_direct_client(client)`（`trust_env=False`，因 SDK 空 proxy 仍读 env） |
| Python 华为云 SDK | `cloud/providers/huawei/direct_network.py`：`direct_http_config()` / `apply_direct_http_config`（清空 `proxy_*`）；首次接入 SDK 时必须经此助手 |
| Python AWS boto3 | `cloud/providers/aws/direct_network.py`：`direct_botocore_config(...)`（强制 `proxies={}` 覆盖 env）；首次接入时必须经此助手 |
| Go 短信出站（taskAuth） | `tracelog.DirectClient` 调阿里云/腾讯云 HTTP API（不经 DefaultClient） |
| CI 门禁 | `check_aliyun_tea_direct_runtime.py`（禁止裸 `RuntimeOptions(`）；`check_cloud_sdk_direct_network.py`（boto3 / Huawei HttpConfig / 腾讯 HttpProfile 须引用直连助手） |
| runAll | 健康检查 Client 已 `Proxy: nil`；`use_proxy: false` 时 `stripProxyEnvVars` |

无 `tracelog` 的进程（如 `go_run_container`）在 `main`/`init` 内自行将 `DefaultTransport.Proxy = nil`，并依赖 `start.sh` unset。

### Tea / 云 SDK 约定（强制）

1. **Aliyun Tea**：新增或修改 `*_with_options` 调用时，runtime 参数**一律** `direct_runtime_options(...)`，不得 `RuntimeOptions()` / `util_models.RuntimeOptions()`。
2. **腾讯云 Python SDK**：新建 Client 时用 `direct_http_profile`，创建后立刻 `apply_direct_client`（SDK 对空 `proxy` 仍回退 `HTTPS_PROXY`，且 requests 默认 `trust_env=True`）。
3. **华为云 Python SDK**：新建 Client 时用 `direct_http_config()`（或 `apply_direct_http_config`）再 `.with_http_config`。
4. **AWS boto3/botocore**：`boto3.client` / `resource` 必须传 `config=direct_botocore_config(...)`（`proxies={}` 覆盖 env）。
5. **其它云 SDK**：若文档写明可读环境代理，按 `tencent`/`huawei`/`aws` 的 `direct_network.py` 模式补助手后再接入，禁止裸默认客户端出站。

## 与相关规则的关系

- **不禁止**开发者在交互式 shell 中设置代理做网络加速。
- **不改变**服务 listen 须 `0.0.0.0` 的约定（见 [22_service_listen_host_not_localhost_only.md](./22_service_listen_host_not_localhost_only.md)）。
- 根目录元规则摘要：仓库根 [`.ai.md`](../../.ai.md)「应用启动与环境 Proxy」一节。
