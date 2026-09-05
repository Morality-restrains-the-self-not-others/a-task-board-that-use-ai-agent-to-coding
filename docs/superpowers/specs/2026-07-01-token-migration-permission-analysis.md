# 权限影响分析 — Token 存储迁移至 Go taskCredentialService

**日期**: 2026-07-01
**来源设计**: Token 存储从 Django CloudServerConfig 完整迁移至 Go taskCredentialService
**风险评级**: 🟢 低风险

---

## 1. 权限影响矩阵

本次迁移不涉及用户角色变更。所有容器回调端点使用 token-based authentication（`AllowAny`），
不依赖 Django session/user 体系。

| # | 改动点 | 类型 | 主体 | 资源 | 现有检查 | 分析 | 结论 |
|---|--------|------|------|------|----------|------|------|
| 1 | Go `POST /api/tenant/.../exchange-refresh/` | 路由转移 | 容器进程 (bearer token) | Task scope | `AllowAny` + access_token 校验 | Django→Go，token 校验逻辑不变，无权限影响 | ✅ |
| 2 | Go `POST /api/tenant/.../refresh-access/` | 路由转移 | 容器进程 (bearer token) | Task scope | `AllowAny` + refresh_token 校验 | Django→Go，token 校验逻辑不变，无权限影响 | ✅ |
| 3 | Go `POST /v1/token/validate` | 新增内部端点 | Django (服务间调用) | Token | ⚠️ 需新增保护 | 内部端点，不应暴露到外部。需要 internal secret 或仅监听 127.0.0.1 | ⚠️ 见下 |
| 4 | taskAgentSupport 路由分叉 | 路由逻辑变更 | 外部请求 | N/A | `X-TaskAgentSupport-Internal-Secret` | 新增 Go 转发需同样携带 internal secret | ⚠️ 见下 |
| 5 | Django heartbeat 调用 Go validate | 查询方式变更 | 容器进程 | Task scope | `AllowAny` + access_token 校验 | DB 查询 → RPC 调用，校验结果等价 | ✅ |
| 6 | Django register-reachability 调用 Go validate | 查询方式变更 | 容器进程 | Task scope | `AllowAny` + access_token 校验 | 同上 | ✅ |
| 7 | Django feature-params 调用 Go validate | 查询方式变更 | 容器进程 | Task scope | `AllowAny` + access_token 校验 | 同上 | ✅ |
| 8 | Django task-detail 调用 Go validate | 查询方式变更 | 容器进程 | Task scope | `AllowAny` + access_token 校验 | 同上 | ✅ |
| 9 | Django git-clone-progress 调用 Go validate | 查询方式变更 | 容器进程 | Task scope | `AllowAny` + access_token 校验 | 同上（3处） | ✅ |
| 10 | Django layer-github-oauth 调用 Go validate | 查询方式变更 | 容器进程 | Task scope | `AllowAny` + access_token 校验 | 同上 | ✅ |
| 11 | Django budget-usage 调用 Go validate | 查询方式变更 | 容器进程 | Task scope | `AllowAny` + access_token 校验 | 同上 | ✅ |

### 需要新增权限保护的改动点

#### 3. Go `POST /v1/token/validate`

- **风险**: 如果暴露到外部网络，任何人可调用此端点探测有效 token
- **方案**: 两种保护方式择一
  - **A (推荐)**: `/v1/token/validate` 仅监听 127.0.0.1（不绑定外网），配置 `TASK_CREDENTIAL_SERVICE_HOST=127.0.0.1`
  - **B**: 增加 internal secret header 校验，与 taskAgentSupport→Django 模式一致
- **推荐**: **方案 A+B 组合** — 绑定 127.0.0.1 + secret 校验双重保护

#### 4. taskAgentSupport → Go 转发

- **风险**: taskAgentSupport 新增对 `taskCredentialService` 的 HTTP 调用，响应内容直接透传给外部调用方
- **现有保护**: `taskAgentSupport` → Django 已有 `X-TaskAgentSupport-Internal-Secret`。Go 侧的新增转发需同样带有 secret
- **Go 侧**: `taskCredentialService` 的 `exchange-refresh`/`refresh-access` handler 本身不检查用户 auth（设计如此，容器 token 即认证凭证）— 但应检查 internal secret 以防止绕过 taskAgentSupport 直接调用
- **方案**: 
  - Go `exchange-refresh`/`refresh-access` handler 检查请求头 `X-TaskAgentSupport-Internal-Secret`
  - 仅当 secret 匹配时才处理（与 Django internal dispatch 一致）
  - 如果 Go 同时支持 Django 直接调用（repo-clone-credentials 等），需要同时接受两种认证模式

---

## 2. 角色与权限建模

**无需新增角色或权限**。本次迁移的端点均属于「容器运行时回调」类别：

| 属性 | 值 |
|------|-----|
| 认证方式 | Token-based（access_token / refresh_token 作为 bearer credential） |
| 权限模型 | 无用户身份 — 拥有有效 token 即拥有对该 task scope 的操作权限 |
| 角色依赖 | 无 |

### 容器 Token 权限模型（不变）

```
token = 临时凭证（1h TTL）
  ├── exchange-refresh: access_token → 一次性换取 refresh_token
  ├── refresh-access:   refresh_token → 获取新 access_token
  ├── heartbeat:         access_token → 上报容器存活
  ├── register-reachability: access_token → 回写网络可达性
  └── ... 其他容器回调
```

---

## 3. 安全审查结论

| 检查项 | 结论 | 说明 |
|--------|------|------|
| IDOR 风险 | ✅ 无新增风险 | URL scope 与 token scope 校验不变 |
| 权限提升 | ✅ 无新增风险 | 无用户角色变更 |
| 跨租户泄露 | ✅ 无新增风险 | Go ValidateToken 校验 `CompanyID == scope.TenantID` |
| 403 vs 404 | ✅ 无影响 | 容器回调端点无资源探测问题 |
| user_id 注入 | ✅ 不适用 | 无 user_id 参数 |
| 敏感操作 | ✅ 已有审计 | Go 已有 audit event 记录 |
| 内部端点暴露 | ⚠️ 需处理 | 新增 `/v1/token/validate` 需绑定 127.0.0.1 + secret 校验 |
| 服务间认证 | ⚠️ 需处理 | taskAgentSupport→Go 需新增 internal secret 传递 |

---

## 4. 测试用例补充

| # | 测试场景 | 预期 |
|---|----------|------|
| 1 | 外部直接调用 `/v1/token/validate`（无 secret） | 403 Forbidden |
| 2 | Django 调用 `/v1/token/validate`（带 secret） | 200 + scope info |
| 3 | 有效 access_token 调用 exchange-refresh（Go） | 200 + refresh_token |
| 4 | 无效 access_token 调用 exchange-refresh（Go） | 401 TOKEN_ACCESS_INVALID |
| 5 | 已交换过的 access_token 重复调用 exchange-refresh（Go） | 403 TOKEN_EXCHANGE_ALREADY_DONE |
| 6 | 有效 refresh_token 调用 refresh-access（Go） | 200 + new access_token |
| 7 | taskAgentSupport→Go 转发携带 internal secret | 200 |
| 8 | taskAgentSupport→Go 转发不带 internal secret | 403 |
| 9 | Django heartbeat 通过 Go validate 校验 token | scope 正确返回，CloudServerConfig 查出 |
| 10 | 过期 token 调用 Go validate | 401 TOKEN_EXPIRED |

---

## 5. 实施建议

在实施计划中追加以下权限相关任务：

1. **Go `taskCredentialService` 配置**: 新增 `TASK_CREDENTIAL_SERVICE_INTERNAL_SECRET` 环境变量
2. **Go handler 中间件**: 新增 internal secret check middleware（`/v1/token/validate` + `exchange-refresh`/`refresh-access` 经过 taskAgentSupport 转发的路径）
3. **taskAgentSupport 配置**: 新增 `taskCredentialServiceURL` 和 secret 转发
4. **Go 端口绑定**: `taskCredentialService` host 绑定 127.0.0.1（仅内部可达）

---

## 总结清单

- 内部端点 `/v1/token/validate`: 绑定127.0.0.1 + secret校验（双重保护）、或仅secret校验一种
- taskAgentSupport→Go 转发: 携带 internal secret、Go 侧校验
- 容器回调端点: 不变（AllowAny + token校验）
- 用户角色: 不变（无新增角色/权限）
