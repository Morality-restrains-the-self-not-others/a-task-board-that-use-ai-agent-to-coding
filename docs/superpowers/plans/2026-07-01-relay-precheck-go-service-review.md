# Code Review: 容器令牌基础设施 Go 服务迁移

> Reviewed against: `docs/superpowers/plans/2026-07-01-relay-precheck-go-service-plan.md`
> Date: 2026-07-01

## Summary

| Category | Count |
|----------|-------|
| 🔴 Critical | 1 |
| 🟡 Warning | 3 |
| 🟢 OK | 5 |

## Findings

### 🟡 Log Gap: sqlite_tokens.go — 缺少 DB 操作日志
**File:** `taskCredentialService/infrastructure/sqlite_tokens.go`  
**Issue:** Save / FindByAccessToken / FindByTaskID / UpdateAccessToken / UpdateRefreshToken — 共 5 个 DB 操作，均无 duration 日志。  
**Impact:** 性能问题无法快速定位；token 查询慢时无排查线索。  
**Fix:** 每个操作添加 `log.Printf("[task-credential-service] sqlite op=%s task=%s duration=%dms", op, taskID, ...)`
→ 下一步: `/8-build-构建` 补齐日志

### 🟡 Log Gap: handlers.go — 缺少请求级完成日志
**File:** `taskCredentialService/interfaces/handlers.go`  
**Issue:** repo-clone-credentials / task-detail handler 无请求完成日志（status + duration）。token-init 仅有错误日志。  
**Impact:** 线上无法追踪每个请求的延迟和成功率。  
**Fix:** 每个 handler 结尾添加 `log.Printf("[task-credential-service] request done: action=%s status=%d duration=%dms", ...)`
→ 下一步: `/8-build-构建` 补齐日志

### 🟡 Log Gap: services.go — IssueToken 缺少结构化上下文
**File:** `taskCredentialService/application/services.go:58`  
**Issue:** `recordAudit` 函数已定义但未在 `IssueToken` 成功后调用；token init 成功时无 INFO 日志。  
**Fix:** `IssueToken` 成功后添加 `log.Printf("[task-credential-service] token issued: task=%s", token.TaskID)`
→ 下一步: `/8-build-构建` 补齐日志

### 🔴 Build Blocked: go.sum 未生成
**Issue:** `go mod tidy` 因网络不可达失败。Go 服务无法编译部署。  
**Root cause:** 开发环境无外网，无法下载 modernc.org/sqlite 依赖。  
**Fix options:**
- A) 从 `taskEvents/go.sum` 复制完整依赖链到 `taskCredentialService/go.sum`（两服务依赖相同库）
- B) 在有网络的机器上 `cd taskCredentialService && go mod tidy && git add go.sum`

→ 下一步: 在有网络的机器上执行 `go mod tidy`，或从 taskEvents 复制 go.sum

### 🟢 Security: 无 token 泄露
- 日志只记录 token SHA256 哈希值（`auditRepo.Save`），不记录原始 token
- `generateToken()` 使用雪花 ID 生成，gitOauth token 不在日志中
- 配置文件中 `taskCredentialServiceBase` 不含凭据

### 🟢 DDD Compliance: 领域层纯净
- `domain/entities.go` — 无 `database/sql`、`net/http` 导入
- `domain/events.go` — 纯数据，无外部依赖
- `ports/repositories.go` — 仅依赖 `domain` 包
- `application/services.go` — 仅依赖 `domain` + `ports`
- `infrastructure/*` — 实现 `ports` 接口，正确依赖反转

### 🟢 Plan Alignment: Increment 1 基本完成
- ✅ 1.1 Go project skeleton (go.mod, build.sh, migrations)
- ✅ 1.2 Infrastructure adapters (sqlite_tokens, sqlite_business, gitoauth_client, composition)
- ✅ 1.3 Application services (TokenService + IssueToken/ValidateToken, CredentialService)
- ✅ 1.4 HTTP handlers (token-init, repo-clone-credentials, task-detail, layer-oauth-tokens stub)
- ✅ 1.5 runAll + registry config
- ⚠️ 1.5 Tests not yet written (blocked by build)

### 🟢 Django Changes: 最小化正确
- `settings_manager.py` — 新方法遵循现有模式，无副作用
- `relay_to_trae_proxy.py` — 仅变更 origin 函数的一行调用，风险最小
- `config.yaml` — 仅新增一个字段

### 🟢 DB Schema: 与设计对齐
- `container_tokens` 表字段与 Django `CloudServerConfig` 语义等价
- `token_audit_events` 表覆盖审计需求字段
- WAL 模式 + Go 只读 saas.sqlite3，并发安全

---

## Action Items

| Priority | Action | Skill |
|----------|--------|-------|
| 1 | Resolve go.sum (copy from taskEvents or network) | `/8-build-构建` |
| 2 | Add DB operation duration logs to sqlite_tokens.go | `/8-build-构建` |
| 3 | Add request completion logs to handlers.go | `/8-build-构建` |
| 4 | Add IssueToken success log to services.go | `/8-build-构建` |
| 5 | Write tests (handlers_test.go, sqlite_test.go, credentials_test.go) | `/8-build-构建` |

## Verdict

**CONDITIONAL PASS** — 架构和代码质量通过。日志需补齐（3 处），构建因环境限制未通过。修复后即可交付 Increment 1。

**Recommendation:** 在有网络的机器上 `go mod tidy` → 补齐日志 → 写测试 → E2E 验证 → ship。
