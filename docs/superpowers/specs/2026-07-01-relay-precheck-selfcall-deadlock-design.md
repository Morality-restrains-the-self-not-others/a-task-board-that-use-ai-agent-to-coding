# 设计文档：relay 直启预检 self-call 死锁导致 502

**日期：** 2026-07-01  
**状态：** 设计中  
**页面：** `http://183.250.1.132:4000/tenant/.../task-detail/.../?relayToTrae=true`

---

## 现象

用户在任务详情「直接启动」点击「启动」后，出现：

> 仓库克隆凭证预检失败。任务 API 不可达或网关异常；请确认 Django 服务（如 8001）已运行。

但 Django 服务（8001）确实在运行——`token-init` 接口先一步成功返回了 `access_token`。

---

## 根因分析

### 完整调用链

```
浏览器 (183.250.1.132:4000)
  ↓ POST /relay-to-trae/repo-credentials-precheck/
网关 (APISIX :4000)
  ↓ proxy_pass → Django
Django (:8001 --noreload, 单线程)
  ↓ relay_to_trae_repo_credentials_precheck()
  ↓   第 522-529 行: requests.post("http://127.0.0.1:8001/.../repo-clone-credentials/")
  ↓   ↑ ↑ ↑ 自调用 — 阻塞等待同一进程的同一线程 ↑ ↑ ↑
  ↓   → 死锁！8 秒超时 → requests.RequestException
  ↓ 返回 502 {"status":"error","message":"仓库凭证预检请求失败: ..."}
前端
  ↓ 502 + error_code 为空 ≠ REPO_CLONE_TOKEN_REFRESH_FAILED
  → 命中 >=500 分支: "任务 API 不可达或网关异常；请确认 Django 服务（如 8001）已运行。"
```

### 根因三要素

#### 1. Django runAll 模式是单线程的

**文件:** `task2app/scripts/runall-saas-backend.sh` 第 13 行

```bash
exec python3 -u manage.py runserver 0.0.0.0:8001 --noreload
```

`--noreload` 使 Django 使用 `wsgiref.simple_server.WSGIServer`（而非 `WSGIThreadedServer`），**同一时刻只能处理一个请求**。开发模式 `runDjango.sh` 不加 `--noreload`，使用多线程 `WSGIThreadedServer`，死锁概率低（除非所有线程都被占满）。

#### 2. 预检对自身发起同步 HTTP 调用

**文件:** `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py` 第 517-529 行

```python
precheck_origin = _resolve_relay_precheck_task_api_origin()
# → "http://127.0.0.1:8001"
precheck_url = f"{precheck_origin}/api/tenant/{tenant}/workspace/{wid}/task/{tid}/cloud/server-container-token/repo-clone-credentials/"
resp = session.post(precheck_url, json={"access_token": access_token}, timeout=8)
```

预检是 Django view 处理函数内发起的 **同步 HTTP 调用**，目标是同一个 Django 实例（`127.0.0.1:8001`）。该调用需要服务端接受新连接并分配线程处理——但唯一的线程正在执行预检本身。

#### 3. 2026-05-27 的修复不完整

上次修复（`relay-precheck-internal-origin`）解决的痛点是：预检误用了公网网关地址（`http://api.daydaymoney.com`），导致 502。修复将 origin 固定为 `internalApiBase`（`http://127.0.0.1:8001`）。

但该修复 **明确推迟了** 进程内直接调用的重构：

> 不在此变更中重构预检为进程内直接调用（可后续优化）

—— `docs/superpowers/specs/2026-05-27-relay-precheck-local-origin-design.md` 第 145 行

**现在正是实施这个优化的时机**。自调用不仅在 `--noreload` 模式下死锁，在多线程 dev 模式下也有隐患（线程池耗尽、连接池排队——项目自身在 `container_forward_requests.py` 第 8-12 行已记录了相关问题）。

---

## 方案

### A. 预检改为进程内直接调用（核心修复，推荐）

**原则：** `repo-credentials-precheck` 是 **Django 进程内** 校验——它已有 token、tenant/workspace/task ID、ORM 访问权限，不需要通过 HTTP 再调用自己。

**实现步骤：**

1. **提取凭证构建编排函数** — 将 `fetch_container_repo_clone_credentials()` 中从 Todo 查询到 `RepoCloneCredentialsBuildResult` 的步骤提取为独立函数 `_build_credentials_for_precheck(cfg, tenant_id, workspace_id, task_id)`，返回结构与 HTTP endpoint 保持语义等价。

2. **预检直接调用编排函数** — `relay_to_trae_repo_credentials_precheck()` 不再通过 `requests.post`，改为：
   ```python
   # 旧（自调用 HTTP）:
   resp = session.post(precheck_url, ...)
   
   # 新（进程内调用）:
   build_result = _build_credentials_for_precheck(cfg, tenant_id, workspace_id, task_id)
   ```
   然后根据 `build_result` 直接构造 Response（200 / 409 / 502）。

3. **HTTP endpoint 保持不变** — `fetch_container_repo_clone_credentials` 仍通过 HTTP 暴露给容器侧回调，内部调用同一个编排函数，契约不变。

4. **移除自调用相关代码** — 删除 `_resolve_relay_precheck_task_api_origin()`（不再需要），删除 `requests.post` 调用及 `requests.RequestException` 异常处理。

**变更文件清单：**

| 文件 | 变更 |
|------|------|
| `cloud/services/relay_to_trae_proxy.py` | 重构 `relay_to_trae_repo_credentials_precheck()`，移除自调用；删除 `_resolve_relay_precheck_task_api_origin()` |
| `cloud/views/container_task_detail_views.py` | 提取 `_build_credentials_for_precheck()` 编排函数；`_build_repo_clone_credentials_with_diagnostics` 不变 |
| `tests/test_relay_to_trae_proxy.py` | 更新预检测试：不再 mock `requests.post`，改为 mock 编排函数；删除 `test_relay_to_trae_repo_credentials_precheck_uses_internal_task_api_origin` |
| `docs/superpowers/specs/2026-05-27-relay-precheck-local-origin-design.md` | 标注"非目标"第 3 条已实现 |

**风险缓解：**
- 编排函数复用现有 `_build_repo_clone_credentials_with_diagnostics`（已充分测试的生产代码），不引入新逻辑
- 预检与容器回调 endpoint 共享同一代码路径，行为一致性由已有测试保证
- 去除 HTTP 调用还消除了 8 秒超时延迟——预检从 ~8s（死锁超时）降为 ~100ms

### B. 错误消息精准化（UX 增强）

当前 5xx 通用错误消息为：
> 任务 API 不可达或网关异常；请确认 Django 服务（如 8001）已运行。

方案 A 消除了死锁，但预检仍可能因其他原因返回 502（如 gitOauth 不可达导致 token 换发失败——该场景已有专用 `REPO_CLONE_TOKEN_REFRESH_FAILED` error_code）。

增强：当预检返回 502 且无 `REPO_CLONE_TOKEN_REFRESH_FAILED` 时，区分两种情况并给出针对性提示：

| 条件 | 消息 |
|------|------|
| `error_code == "REPO_CLONE_TOKEN_REFRESH_FAILED"` | Token 换发失败详情（已有，不变） |
| `error_code == "REPO_CLONE_CREDENTIALS_INCOMPLETE"` (409) | 缺凭证仓库引导（已有，不变） |
| 连接失败 / 5xx 且 precheck 为进程内调用 | 不再可能连接失败，移除误导性"请确认 Django 已运行"提示，改为"预检异常，请查看日志" |
| 其他 5xx | 「预检服务异常，请稍后重试」|

### C. runAll Django 启动方式评估（长期优化，本次不做）

`--noreload` 改为多线程 `runserver` 或 gunicorn。但此变更影响面大（部署模型、SQLite 并发、WSGI server 兼容），作为独立的后续议题。

---

## 价值流影响

| 流 | 影响 |
|----|------|
| `relay-precheck-internal-origin` | 本设计是该流的**延续**——origin 修复的第二步"进程内直接调用"，原定"非目标"条件已成熟 |
| `task-detail-repo-clone-credentials-contract` | 凭证构建逻辑不变量，仅调用方式从 HTTP → 函数调用；契约不变 |
| `task-detail-oauth-binding-guidance` | 前端消息分流逻辑可以简化（不再有"连接失败"引导） |
| `task-detail-runtime-relay` | 直启链路自调用死锁彻底消除 |

不涉及 gitOauth schema 变更，不涉及容器 runtime 行为变更。

---

## 领域概念（轻量）

- **Bounded Context：** 任务协作 / 容器 runtime / relay 直启
- **实体：** `CloudServerConfig`（container_access_token）、`Todo`、`TaskRepoIdentity`
- **值对象：** `TaskScope`、`ContainerTokenContext`、`AccessToken`
- **领域服务：** `RepoCloneCredentialsFetchService`、`TaskRepoCloneCredentialsGuardService`
- **领域事件：** `RepoCloneCredentialsFetchFailed`、`RepoCloneCredentialsFetchSucceeded`
- **应用服务 (新增):** `relay_to_trae_repo_credentials_precheck`（重构后为应用编排）

---

## 测试计划

### 后端

- **更新** `test_relay_to_trae_repo_credentials_precheck_returns_ok`：mock 编排函数返回成功 → 断言 200
- **更新** `test_relay_to_trae_repo_credentials_precheck_returns_incomplete_contract`：mock 编排函数返回缺失 → 断言 409
- **更新** `test_relay_to_trae_repo_credentials_precheck_returns_token_refresh_failed_contract`：mock 编排函数 token 换发失败 → 断言 502
- **删除** `test_relay_to_trae_repo_credentials_precheck_uses_internal_task_api_origin`（不再有 self-call origin 选择）
- **新增** `test_precheck_handles_token_lookup_failure_gracefully`：Todo 不存在 → 断言 409
- **新增** `test_precheck_handles_no_project_repos`：任务无仓库 → 断言 200 + repo_count=0

### 前端

- 预检成功（200）→ 进入 start 流程 ✓（已有测试覆盖）
- 预检缺少凭证（409）→ OAuth 引导正常显示 ✓（已有测试覆盖）
- 预检 token 换发失败（502 + error_code）→ 精准提示 ✓（已有测试覆盖）
- 预检 5xx 其他 → 提示"预检服务异常"而非"请确认 Django 已运行" ✓（新增断言）

### E2E

- `TaskDetail.relay-to-trae-direct-start.playwright.test.js`：回归直启全链路（token-init → precheck → start），确保预检不超时

---

## 验收标准

1. `runall-saas-backend.sh start`（`--noreload` 单线程模式）→ 直启预检 200（OAuth 已授权 + 账号已保存），不再 502 死锁
2. 预检响应时间从 ~8s（超时）降为 <1s
3. OAuth 已授权但未保存账号 → 预检 409，前端显示缺失仓库引导
4. Token 换发失败（gitOauth 不可达）→ 预检 502 + `REPO_CLONE_TOKEN_REFRESH_FAILED`，前端显示精准错误
5. 容器回调 endpoint（`/server-container-token/repo-clone-credentials/`）行为不变，已有测试全部通过
6. 预检测试不再需要 mock `requests.post`——改为 mock 进程内编排函数

---

## 非目标

- 不修改 `--noreload` 配置（独立议题）
- 不改变 `repo-clone-credentials` HTTP 契约
- 不改变前端 relayToTrae 启动流程（token-init → precheck → start 三步不变）

---

## 风险

| 风险 | 缓解 |
|------|------|
| 编排函数提取改变语义 | 复用 `_build_repo_clone_credentials_with_diagnostics` 生产代码，提取为薄封装 |
| 缺少 HTTP 层的自然隔离 | 进程内调用更快，但需显式处理异常；编排函数已有完善的错误分类 |
| Token 验证路径可能不同 | precheck 已有 token（`_issue_relay_access_token`），与 endpoint 的 token 验证路径等价但参数来源不同；需要统一验证逻辑 |
