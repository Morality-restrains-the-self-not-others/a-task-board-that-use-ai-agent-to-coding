# DDD 领域模型: OIDC SLO 登出同步

> 输入:
> - 价值流: `docs/superpowers/plans/2026-06-25-oidc-slo-logout-sync-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-25-oidc-slo-logout-sync-nfr-clarification.md`
>
> NFR 关键约束: L2 安全（服务端 session 销毁）、L2 容错（降级）、L1 一致性（best-effort）

## 限界上下文

| 上下文 | 职责 | 现有/新增 |
|--------|------|----------|
| **Auth Context** (taskAuth Go) | OIDC Provider：用户认证、令牌管理、**会话终止 (EndSession)** | 扩展 |
| **Main App Context** (task2app Django) | 业务会话管理：Django session 生命周期、**登出编排** | 扩展 |
| **Git Service Context** (GitLab) | 第三方 RP：GitLab 自身会话管理 | 外部，不修改 |

## 实体与值对象

### Auth Context (taskAuth)

| 类型 | 名称 | 说明 | 变更 |
|------|------|------|------|
| Entity | CustomToken | 现有 — `accounts_customtoken` 表，API 认证令牌（单例） | 无结构变更 |
| **VO** | **SessionTermination** | **新增** — 封装 EndSession 重定向链构造 | 见 `domain/session_termination.go` |

### Main App Context (task2app Django)

无新增实体/VO。现有 Django `Session` 和 `User` 模型不变。

## 聚合

| 上下文 | 聚合根 | 内部实体/VO | 不变条件 |
|--------|--------|------------|---------|
| Auth | CustomToken (现有) | — | 登出时 token 必须物理删除 |
| Main App | User (现有 Django User) | Django Session | 登出时 session 必须 flush |

**NFR 驱动的聚合决策:** NFR 要求 L1 一致性（best-effort），因此 Auth Context 的 token 删除和 Main App Context 的 session 销毁**不在同一事务中**——各自独立执行，不引入分布式协调。

## 仓储接口

### Auth Context — 新增

```go
// domain/customtoken_repository.go
type CustomTokenRepository interface {
    DeleteByKey(key string) error  // 新增：幂等删除
}
```

实现位于 `taskAuth/src/db.go`（现有 `deleteTokenByKey` 函数）。

### Main App Context

使用 Django 内置 `django.contrib.sessions` 和 `django.contrib.auth.logout()`，无需自定义仓储。

## 领域服务

无新增领域服务。登出编排在应用层实现：
- taskAuth: `handleOidcEndSession` (HTTP handler)
- Django: `logout()` view + `UserViewSet.logout()` API

## 领域事件

**本期不做领域事件。** NFR 要求 L1 best-effort 一致性，不需要事件驱动的跨上下文协调。登出通过浏览器重定向链实现同步传播。

## 文件清单

| 文件 | 类型 | 说明 |
|------|------|------|
| `taskAuth/domain/session_termination.go` | 值对象 | SessionTermination — EndSession 重定向 URL 构造 |
| `taskAuth/domain/session_termination_test.go` | 测试 | VO 单元测试（4 个用例，全部 PASS） |
| `taskAuth/domain/customtoken_repository.go` | 仓储接口 | CustomTokenRepository — DeleteByKey 抽象 |

## 自检

- [x] 领域文件位于 `taskAuth/domain/` 目录
- [x] 无 ORM 导入（session_termination.go 仅用 `net/url` + `fmt`）
- [x] 无外部服务导入（无 HTTP/Kafka/云 SDK）
- [x] 仓储接口使用 Go interface 抽象
- [x] SessionTermination 是不可变值对象（无 setter）
- [x] 值对象有 Equals 方法
- [x] 领域测试通过
