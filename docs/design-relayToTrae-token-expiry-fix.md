# 设计文档：relayToTrae 容器启动 Token 过期修复

- **日期**: 2026-07-02
- **作者**: claude
- **状态**: 🎯 待审批
- **关联页面**: `http://183.250.1.132:4000/tenant/{tid}/workspace/{wid}/task-detail/{taskId}/?relayToTrae=true`

---

## 1. 问题概述

用户点击任务详情页的「启动」按钮后，onlineServiceJS 容器启动失败，日志显示 `HTTP 401: access_token 已过期，请调用 refresh-access 续期`。

## 2. 日志分析

### 2.1 完整时序

```
[relayToTrae] token-exchange: skipped (skip_token_exchange=true, using Django-issued token directly)
[relayToTrae] start: bash /tmp/ram-work/trae-agent/onlineServiceJS/run.sh
[onlineServiceJS] token-exchange: skipped (TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE), using initial ACCESS_TOKEN as-is
[onlineServiceJS] server listening on http://0.0.0.0:8765
[onlineServiceJS] 已向 SaaS 注册可达地址...
[onlineServiceJS] 容器已启动，开始拉取任务详情…
[relayToTrae] status push HTTP 401: {"detail": "无效的 access_token"}
[onlineServiceJS] bootstrap (post-listen) error: HTTP 401 .../task-detail/: {"detail":"access_token 已过期，请调用 refresh-access 续期"}
[relayToTrae] status push HTTP 401: {"detail": "无效的 access_token"}
```

### 2.2 关键发现

| 时间点 | 事件 | 状态 |
|--------|------|------|
| T0 | Django 调用 Go `/v1/token/init` 签发 token | Go **复用了已存在的即将过期 token** |
| T1 | Django → Go relay `/v1/start` | `skip_token_exchange=True` 硬编码，不刷新 token |
| T2 | Go relay → 启动 onlineServiceJS | `TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE=1`，再次跳过 |
| T3 | onlineServiceJS → `task-detail/` | **401: token 已过期** |
| T4+ | Go relay status push 持续 401 | 同样 token，Django GoTokenValidator 拒绝 |

## 3. 根因分析

### 3.1 直接原因

**Token 在传输链路中从未被刷新。** 从 Django 签发（或复用）到容器实际使用，中间经历了 Go relay 转发和 Node.js 容器启动，但两个 token exchange 环节都被跳过。

### 3.2 根本原因：三个环节的连锁问题

#### 环节 1：Go IssueToken 复用策略过于激进

`taskCredentialService/application/services.go:36`:
```go
if existing != nil && existing.ContainerAccessToken != "" && 
   existing.ContainerRefreshToken == "" && !s.isExpired(existing) {
    return existing, nil  // ← 即使只剩 1 秒也复用
}
```

- Token TTL = 1 小时
- 用户打开页面时 Token 可能已存在 55 分钟（还剩 5 分钟有效）
- Go 判断「未过期」→ 复用旧 token
- 容器启动后（几十秒后）token 过期 → 所有 API 调用失败

#### 环节 2：skip_token_exchange 硬编码为 True

`task2app/Saas_project/cloud/services/relay_to_trae_proxy.py:993-994`:
```python
# 代理模式下由 Django 预先签发 token，避免 /start 同步链路内再回调 task2app 换票导致阻塞。
"skip_token_exchange": True,
```

设计意图：Django 应预先完成 exchange-refresh + refresh-access，然后告诉 Go relay「不用再换了」。但实际代码中 **Django 从未执行 exchange 操作** — 它仅调用 `_issue_token_via_go()` 获取初始 access_token，然后直接标记 `skip_token_exchange=True`。

#### 环节 3：双端都跳过，无处刷新

```
Django (_issue_token_via_go) → 初始 access_token (TTL ≤ 1h)
    ↓ skip_token_exchange=True
Go relay (performTokenExchange) → 跳过
    ↓ TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE=1
onlineServiceJS (runBootstrapTokenExchangeOnly) → 跳过
    ↓ 使用原始 access_token
POST /task-detail/ → 401 token 已过期
```

### 3.3 涉及的服务和代码路径

| 服务 | 文件 | 关键行 | 角色 |
|------|------|--------|------|
| **task2app (Django)** | `cloud/services/relay_to_trae_proxy.py` | L939-1038 | 接收启动请求，调用 Go 签发 token，转发到 relay |
| **taskCredentialService (Go)** | `application/services.go` | L30-60 | Token SSOT：签发/验证/交换/刷新 |
| **go_relayToTrae (Go)** | `src/process.go` | L254-385 | 接收 /v1/start，管理 onlineServiceJS 子进程 |
| **go_relayToTrae (Go)** | `src/token.go` | L155-246 | Token exchange 两阶段握手 |
| **onlineServiceJS (Node)** | `src/bootstrap.mjs` | L791-884 | 容器内 token exchange |
| **onlineServiceJS (Node)** | `src/saasTaskCloud.mjs` | L104-192 | postJson → task-detail/ 调用 |
| **onlineServiceJS (Node)** | `src/server.mjs` | L1805 | runBootstrapAfterListen 入口 |

## 4. `skip_token_exchange` 去除分析

### 4.1 这个变量是做什么的？

`skip_token_exchange` 在三个地方出现：

| 位置 | 变量名 | 语义 |
|------|--------|------|
| Django → Go relay (`relay_to_trae_proxy.py:994`) | `skip_token_exchange: True` (硬编码) | "我已经换过了，你不用再换" |
| Go relay → onlineServiceJS (`process.go:241`) | `TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE=1` | "relay 已经换过了，容器不用再换" |
| onlineServiceJS (`bootstrap.mjs:833`) | 读取 `TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE` | 决定是否跳过容器内 exchange |

两个 flag 是**不同层次**的：
- `skip_token_exchange`：Django 告诉 Go relay 是否做 token exchange
- `TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE`：Go relay 告诉容器是否做 token exchange（**这个应该保留**）

### 4.2 最初的设计意图 vs 实际行为

注释说（`relay_to_trae_proxy.py:993`）：
> "代理模式下由 Django 预先签发 token，避免 /start 同步链路内再回调 task2app 换票导致阻塞。"

意图：Django 先完成 exchange-refresh + refresh-access，得到一个新鲜 token，然后告诉 Go relay「不用换了」。

实际行为：**Django 从未执行 exchange 操作。** 它只调用 `_issue_token_via_go()` 获取初始 access_token（可能是复用的旧 token），然后直接标记 `skip_token_exchange=True`，导致 Go relay 也跳过 exchange。最终容器收到的就是原始 access_token，没有任何刷新。

### 4.3 去除 `skip_token_exchange` 可行吗？

**可以。** 而且是最简洁的修复方案。

#### 去除后的完整链路

```
Django _issue_token_via_go()
  → Go /v1/token/init
  → 返回 access_token (可能是复用的旧 token，TTL 可能 < 1h)
    │
Django → Go relay /v1/start  (不再传 skip_token_exchange)
    │
Go relay performTokenExchange()
  → Step 1: exchange-refresh  → 获取 refresh_token，废弃旧 access_token
  → Step 2: refresh-access   → 获取 **全新** access_token (TTL=1h)
    │
Go relay → onlineServiceJS  (TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE=1 保留)
    │
onlineServiceJS → POST task-detail/ → ✅ 200 (新鲜 token)
```

#### 关键前提验证

| 前提 | 验证结果 |
|------|---------|
| Go relay 的 exchange 调用不会形成循环依赖 | ✅ exchange-refresh 路径: APISIX → taskAgentSupport (:8018) → taskCredentialService (:8015)，不回调用 Django |
| exchange-refresh 是一致性操作，可以安全重复调用吗？ | ✅ Go `IssueToken` 保证返回的 token 的 `ContainerRefreshToken==""`（未交换过），所以 exchange-refresh 总能成功 |
| 延迟可接受吗？ | ✅ exchange 约 150-200ms（两次 HTTP + 150ms stagger），对于「启动容器」操作（通常 3-10 秒）可忽略 |
| 与原有注释"避免阻塞"冲突吗？ | ✅ 不冲突 — exchange 走 Go taskCredentialService 直连，不经过 Django，不会造成同步链路阻塞 |

#### 为什么以前要加 `skip_token_exchange`？

推测原因：
1. 早期 exchange-refresh 可能回调用到了 Django（形成同步环路）→ 现在由 taskAgentSupport 直接路由到 Go，没有这个问题
2. 担心 /v1/start 同步等待太长 → 实际延迟可忽略（200ms vs 数秒的容器启动时间）
3. 计划让 Django 自己做完 exchange 再传 → 但从未实现

### 4.4 推荐方案：去除 `skip_token_exchange` + 防御性修复

#### 第一层（核心修复）：去除 `skip_token_exchange`，让 Go relay 始终 exchange

**变更 1** — Django 端：删除 `skip_token_exchange` 字段

`task2app/Saas_project/cloud/services/relay_to_trae_proxy.py`:
```python
# 删除 payload 中的 skip_token_exchange 字段
payload = {
    "tenant_id": tenant,
    "workspace_id": wid,
    "task_id": tid,
    "env": runtime_env,
    # 删除: "skip_token_exchange": True,
}
```

**变更 2** — Go relay 端：始终执行 exchange

`go_relayToTrae/src/process.go`:
```go
func startOnlineService(envPayload map[string]string, tenantID, workspaceID, taskID string) (map[string]interface{}, error) {
    // ... 
    // 删除 skipTokenExchange 参数
    // 始终执行 token exchange
    if taskAPIOrigin != "" && businessAPIEndpoint != "" && accessToken != "" {
        newToken, err := performTokenExchange(...)
        // ...
    }
}
```

`go_relayToTrae/src/handlers.go`:
```go
// 删除 body 结构体中的 SkipTokenExchange 字段
```

**变更 3（已由 2026-07-10 云对齐迭代废止）** — ~~保留 `TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE`~~

> **2026-07-10 修订**：为使 `?relayToTrae=true` 模拟启动与云主机 UserData 路径一致，**默认不再**由 go_relay 父进程预换票，也**不再**注入 `TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE=1`。首次 exchange 由 onlineServiceJS 完成；侧车通过 `container_refresh_token.json` 同步 access/refresh。详见 `docs/superpowers/specs/2026-07-10-relay-token-cloud-parity-design.md`。

原「父进程 exchange + 子进程 SKIP」仅作历史说明，生产默认路径已废弃。

`taskCredentialService/application/services.go`:
```go
const tokenReuseMinTTL = 5 * time.Minute

func (s *TokenService) IssueToken(...) (*domain.ContainerToken, error) {
    existing, _ := s.tokenRepo.FindByScope(...)
    if existing != nil && existing.ContainerAccessToken != "" && 
       existing.ContainerRefreshToken == "" && 
       !s.isExpired(existing) && 
       s.remainingTTL(existing) >= tokenReuseMinTTL {  // 新增
        return existing, nil
    }
    // 签发新 token
}
```

这个修复很重要：即使 Go relay 会 exchange，但如果 IssueToken 返回的旧 token 在 exchange 的那一刻刚好过期，exchange-refresh 就会失败。5 分钟的缓冲确保 exchange 有足够时间窗口。

#### 不修改的部分

**`TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE` 保留不动：**
- 位置：`go_relayToTrae/src/process.go:241`（`buildChildEnv`）
- 语义：Go relay 已完成 exchange，容器无需再 exchange
- 独立模式（`TRAE_ONLINE_JS_DOCKER=0`，无 Go relay）不受影响

**onlineServiceJS 容器内 exchange 逻辑保留不动：**
- 位置：`trae-agent/onlineServiceJS/src/bootstrap.mjs:791-884`
- 独立模式仍然需要这个能力

### 4.5 为什么不选「Django 预刷新」（原方案 A）

| 维度 | 去除 skip_token_exchange | Django 预刷新 |
|------|--------------------------|--------------|
| 代码量 | 删除代码（Django 少一行，Go relay 简化） | 新增函数（exchange + refresh + 错误处理） |
| 复杂度 | 降低（少了一个 flag + 一个分支） | 增加（Django 也要理解 token 生命周期） |
| 职责划分 | Go relay 管 exchange（token 操作靠近使用者） | Django 管 exchange（跨层操作） |
| exchange 幂等 | 无需处理（Go IssueToken 保证返回未交换的 token） | 需要处理 ALREADY_EXCHANGED 等边界情况 |
| 延迟 | ~200ms（和原方案一样） | ~200ms（和原方案一样） |

## 5. 影响范围

### 5.1 变更文件

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `task2app/.../relay_to_trae_proxy.py` | **删除** | 删除 payload 中的 `"skip_token_exchange": True` |
| `go_relayToTrae/src/handlers.go` | **删除** | 删除 body 结构体中的 `SkipTokenExchange` 字段 |
| `go_relayToTrae/src/process.go` | **简化** | 删除 `skipTokenExchange` 参数和条件判断，始终执行 exchange |
| `taskCredentialService/application/services.go` | 修改 | `IssueToken()` 新增 `remainingTTL` 检查（不复用 < 5min TTL 的 token） |
| `go_relayToTrae/src/process.go:241` | **不动** | `TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE=1` 保留 |
| `trae-agent/onlineServiceJS/src/bootstrap.mjs` | **不动** | 容器内 exchange 逻辑保留（独立模式需要） |

### 5.2 测试影响

| 测试文件 | 变更 |
|---------|------|
| `task2app/.../test_relay_to_trae_proxy.py` | 修改：删除 `assert captured["payload"]["skip_token_exchange"] is True` |
| `taskCredentialService/.../services_test.go` | 新增：TTL 阈值测试（刚好 5min、小于 5min、已过期） |

### 5.3 架构影响

本次变更为**行为修复 + 简化**：删除一个冗余 flag，不涉及新增/删除服务组件，不需要更新架构设计稿。

## 6. 领域概念清单

| 概念 | 类型 | 边界上下文 | 说明 |
|------|------|-----------|------|
| ContainerToken | 实体 | Token 管理 | access_token + refresh_token 生命周期 |
| TokenExchange | 领域事件 | Token 管理 | access_token → refresh_token 的一次性交换 |
| TokenRefresh | 领域事件 | Token 管理 | refresh_token → 新 access_token |
| TaskScope | 值对象 | Token 管理 | tenant+workspace+task 三元组 |
| RelayStartupSession | 聚合 | 容器启动 | 两阶段启动工作流 |

## 7. 验证方法

1. **单元测试**：Go `IssueToken` TTL 阈值测试
2. **集成测试**：Django `relay_to_trae_start` 完整链路（mock Go relay）
3. **手动验证**：
   - 创建一个任务，等待 55 分钟后点击「启动」
   - 确认容器启动成功，task-detail 返回 200
   - 确认 status push 不再返回 401
4. **日志验证**：
   - 确认 `_ensure_fresh_access_token` 日志出现
   - 确认 Go relay 日志中不再出现 `status push HTTP 401`

---

## 📋 总结清单

- **`skip_token_exchange` 可以去除**: 删除 Django→Go relay 的 `skip_token_exchange` flag，让 Go relay 始终执行 token exchange。两个关键前提已验证：(1) exchange 不形成循环依赖（taskAgentSupport 直接路由到 Go taskCredentialService）；(2) 延迟可忽略（~200ms vs 数秒容器启动）
- **防御性修复**: Go `IssueToken` 不复用剩余 TTL < 5 分钟的旧 token，确保 exchange 有时间窗口执行
- **保留不动**: `TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE`（防止容器二次 exchange）和容器内 exchange 逻辑（独立模式需要）
- **架构变更**: 无需更新架构设计稿（行为修复 + 简化，无组件增删）
