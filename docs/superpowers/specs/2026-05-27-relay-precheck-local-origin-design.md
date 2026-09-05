# 设计文档：relay 直启预检 502 与 OAuth 已授权矛盾

**日期：** 2026-05-27  
**状态：** 已实现  
**页面：** `http://localhost:4000/tenant/.../task-detail/.../?relayToTrae=true`

---

## 现象

用户在任务详情「直接启动」点击「启动」后，出现：

> 仓库凭证预检失败（HTTP 502）。请先在「关联项目」中完成 OAuth 绑定，并为每个仓库选择账号后点击"保存账号"。

但「关联项目」面板中对应仓库已显示 **OAuth 已授权**。

---

## 根因（已验证）

### 1. 预检请求打到了不可达的公网网关

本地 `port_config.json` 配置：

- Django：`127.0.0.1:8001`（`internalApiBase`）
- 前端：`vue.apiBaseUrl = http://api.daydaymoney.com`

`SettingsManager.get_relay_task_api_base_url()` 优先选用非回环的 `vue.apiBaseUrl`，因此 `env-prepare` 与 relay 预检使用的 `TASK_API_ENDPOINT_ORIGIN` 为 **`http://api.daydaymoney.com`**。

启动链路：

1. **token-init**：本地 Django 签发 `container_access_token` 并写入本地 DB ✅  
2. **repo-credentials-precheck**：Django 代理向  
   `http://api.daydaymoney.com/.../repo-clone-credentials/` POST 该 token ❌  
3. 公网网关返回 **502**（实测：`curl` 对该 URL 返回 502；本地 `127.0.0.1:8001` 同路径返回 401）

**结论：** 502 是 **TASK_API 端点不可达/网关故障**，与 OAuth 绑定状态无关。token 在本地库，预检却请求远端，即使网关正常也会 401。

### 2. 前端错误文案误导

`ServerConfig.logic.vue` 在**所有**预检失败时追加固定引导：

```text
请先在「关联项目」中完成 OAuth 绑定，并为每个仓库选择账号后点击"保存账号"。
```

该文案仅适用于 **409 + missing_repo_credentials**（凭证不完整）。对 502 会误导用户重复 OAuth。

### 3. 「OAuth 已授权」≠「克隆凭证已就绪」

| 阶段 | UI 表现 | 预检所需 |
|------|---------|----------|
| OAuth 绑定 | 「OAuth 已授权」 | gitOauth 侧有 active 凭据 |
| 保存账号 | 无明确「已保存」态 | `TaskGithubRepoOAuthBinding` + `TaskRepoIdentity` |
| 预检通过 | — | 上述全部完成且 token 可换票 |

当前 UI 只暴露第一阶段，用户在 409 场景也会困惑；本次 502 则完全无关。

---

## 方案

### A. 预检走内网 origin（核心修复）

**原则：** `repo-credentials-precheck` 是 **Django 进程内/同机** 校验，必须使用 **`get_internal_task_api_base_url()`**，不得使用面向容器/公网上报的 `get_relay_task_api_base_url()` 或用户填写的 `TASK_API_ENDPOINT_ORIGIN`。

**实现（`relay_to_trae_proxy.py`）：**

- 新增 `_resolve_precheck_task_api_origin()` → 固定返回 `settings_manager.get_internal_task_api_base_url()`
- `relay_to_trae_repo_credentials_precheck` 构建 URL 时使用该 origin
- **`TASK_API_ENDPOINT_ORIGIN` 仍按现逻辑注入 relay/onlineServiceJS**（容器可能需公网可达地址）

可选增强：若 internal 请求仍失败，响应增加 `error_code: RELAY_PRECHECK_TASK_API_UNREACHABLE` 与 `task_api_origin` 字段便于排障。

### B. 前端错误分流（UX）

**`ServerConfig.logic.vue`：**

| 条件 | 展示 |
|------|------|
| 409 且 `missing_repo_credentials` 非空 | 现有 OAuth + 保存账号引导 + 缺失仓库摘要 |
| 502 / 5xx / 连接失败 | 「任务 API 不可达，预检无法完成。本地开发请确认 Django 在 8001 运行；若面板 TASK_API_ENDPOINT_ORIGIN 指向公网网关，预检已改为走内网，无需手动修改。」 |
| 401/403 | 「容器令牌无效或过期，请重试启动」 |

仅在 409 场景设置 `relayToTraeRepoCredentialGuideVisible = true`。

### C. 关联项目：展示「账号已保存」（增量 UX）

在 GitHub 仓库行，当 `repo_bindings` 中已有该 slug 的 `github_user_id` 时显示 **「克隆账号已保存」** 徽章（绿色），与「OAuth 已授权」并列，减少两阶段混淆。

非 GitHub 仓库沿用现有 Git 身份选择 + 保存逻辑。

### D. env-prepare 说明（文档级，可选 UI hint）

在 relay 环境面板为 `TASK_API_ENDPOINT_ORIGIN` 增加 tooltip：**「供 onlineServiceJS 回调任务 API；启动预检由服务端走内网地址，与此项无关。」**

---

## 价值流影响

| 流 | 影响 |
|----|------|
| `task-detail-oauth-binding-guidance` | 预检失败分类与引导文案需分流 |
| `task-detail-repo-clone-credentials-contract` | 预检仍校验同一 endpoint，仅 origin 选择变更 |
| `relay-token-audit-observability` | 可选记录 precheck 使用的 internal origin |
| `task-detail-runtime-relay` | 直启链路 502 误报消除 |

不涉及 gitOauth schema 变更。

---

## 领域概念（轻量）

- **Bounded Context：** 任务协作 / 容器 runtime / relay 直启
- **实体：** `CloudServerConfig`（container_access_token）、`TaskRepoIdentity`、`TaskGithubRepoOAuthBinding`
- **值对象：** `TaskScope`、`RepoCloneCredentialCoverage`
- **领域服务：** `TaskRepoCloneCredentialsGuardService`、`RepoCloneCredentialsFetchService`
- **领域事件：** `RepoCloneCredentialsFetchFailed`

---

## 测试计划

### 后端

- `test_relay_to_trae_repo_credentials_precheck_uses_internal_origin`：mock `requests.post`，断言 URL host 为 `127.0.0.1:8001`（或 internalApiBase），而非 `api.daydaymoney.com`
- 回归现有 `test_relay_to_trae_repo_credentials_precheck_returns_ok` / `returns_incomplete_contract`

### 前端

- 预检 502 时不展示 OAuth 引导卡片
- 预检 409 时仍展示引导 + missing 摘要
- （可选）repo_bindings 存在时显示「克隆账号已保存」

### E2E

- 扩展 `TaskDetail.relay-to-trae-direct-start.playwright.test.js`：mock precheck 502 → 断言文案不含「OAuth 绑定」

---

## 非目标

- 不修改 `get_relay_task_api_base_url()` 全局策略（容器状态上报仍可能需要公网 gateway）
- 不改变 `repo-clone-credentials` 409 契约
- 不在此变更中重构预检为进程内直接调用（可后续优化）

---

## 风险

| 风险 | 缓解 |
|------|------|
| 生产多节点 Django，internal URL 不可自调用 | internalApiBase 指向负载均衡或本机 loopback；生产 config 应显式配置 |
| 用户依赖手动 override TASK_API_ENDPOINT_ORIGIN 做预检 | 文档说明预检忽略 override；override 仅影响容器 runtime |

---

## 验收标准

1. 本地 `localhost:4000` + Django `8001`，OAuth 已授权且账号已保存 → 直启预检 **200**，可继续 start  
2. OAuth 已授权但未保存账号 → 预检 **409**，展示缺失仓库与保存引导  
3. 任意场景网关 502 → **不再**提示「完成 OAuth 绑定」  
4. 关联项目已保存账号的仓库显示「克隆账号已保存」
