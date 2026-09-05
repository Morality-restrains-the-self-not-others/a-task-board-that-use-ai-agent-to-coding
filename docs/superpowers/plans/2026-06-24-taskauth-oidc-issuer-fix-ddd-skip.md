# DDD 领域建模: taskAuth OIDC Issuer 修复 — 跳过声明

> 输入:
> - 设计文档: `docs/specs/taskauth-oidc-issuer-fix-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-24-taskauth-oidc-issuer-fix-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-24-taskauth-oidc-issuer-fix-nfr-clarification.md`

## 跳过理由

**本修复为配置变更 + 基础设施增强，不引入新的业务概念。**

依据 DDD 技能的跳过条件：_"配置变更、文档修改、或价值流增量不涉及新的业务概念"_

### 变更性质

| 变更 | 层级 | 新领域概念？ |
|------|------|:---:|
| `conf/auth/task-auth/config.yaml` — `oidc.issuer` 显式设置 | 配置 | ❌ |
| `gitService/docker-compose.yml` — `GITLAB_OIDC_ISSUER` 默认值修正 | 部署配置 | ❌ |
| `gitService/docker-compose.yml` — `redirect_uri` 参数化 | 部署配置 | ❌ |
| `taskAuth/src/oidc_handlers.go` — `issuerURL()` fallback 增强 | 基础设施层 | ❌ |
| `gitService/run.sh` — 导出 `GITLAB_OIDC_ISSUER` | 脚本 | ❌ |

### 现有领域模型不受影响

taskAuth 已有的领域模型 (`taskAuth/domain/`):
- `login_method.go` — LoginMethod 实体（身份验证方式）
- `repository.go` — 仓储接口（LoginMethodRepository, UserRepository）
- `domain_test.go` — 领域层测试

OIDC Provider 和 OIDC Client（`oidc_bootstrap.go`, `oidc_db.go`, `oidc_handlers.go`）属于基础设施层/应用层，
已通过 `Issuer` 值对象提供 issuer URL。本次修复仅修正该值对象的构造逻辑，不改变其语义或行为契约。

### NFR 确认

NFR 澄清文档已确认（`领域模型影响` 表）:
- "本修复不引入新领域实体或聚合"
- "issuer URL 作为值对象已存在于 OIDC Provider 上下文中"
- "代码变更集中在基础设施层 (config loading + env injection)"

## 决议

**跳过 DDD 建模。** 现有领域模型 (`taskAuth/domain/`) 无需修改。

下一步: `/6-plans-实施计划` — 将 4 项配置/代码变更拆解为可验证的任务清单。
