# Ship Reflection: AI Provider OIDC 登录

## 交付日期
2026-06-29

## 交付摘要

为 AI Provider (saas-ai-provider, port 8010) 新增标准 OIDC Authorization Code Flow + PKCE 登录支持，
Vendor 和 Admin 用户可通过 taskAuth OIDC Provider 独立登录，无需再强制依赖主站 SSO bridge。

## 交付文件

### 新建 (17 files)
| 文件 | 层 | 说明 |
|------|-----|------|
| `domain/value_objects/oidc_auth_session.py` | Domain | PKCE session 值对象 |
| `domain/value_objects/oidc_id_token_claims.py` | Domain | id_token claims 值对象 |
| `domain/events/oidc_events.py` | Domain | 3 种领域事件 |
| `domain/repositories/vendor_repository.py` | Domain | VendorRepository ABC |
| `domain/repositories/platform_staff_repository.py` | Domain | PlatformStaffRepository ABC |
| `domain/services/oidc_authentication_service.py` | Domain | OIDC 认证领域服务 |
| `oidc_rp.py` | Infrastructure | OIDC RP HTTP 客户端 |
| `repositories.py` | Infrastructure | Django ORM 仓储实现 |
| `views_oidc.py` | Interface | OIDC authorize/callback 端点 |
| `tests/test_oidc_domain.py` | Test | 11 domain tests |
| `tests/test_oidc_authentication_service.py` | Test | 16 service tests |
| `docs/specs/ai-provider-oidc-login/design.md` | Spec | 设计文档 |
| `docs/specs/ai-provider-oidc-login/permission-analysis.md` | Spec | 权限分析 |
| `docs/specs/ai-provider-oidc-login/value-stream.md` | Spec | 价值流 |
| `docs/specs/ai-provider-oidc-login/nfr-clarification.md` | Spec | NFR 澄清 |
| `docs/specs/ai-provider-oidc-login/plan.md` | Spec | 实施计划 |
| `docs/specs/ai-provider-oidc-login/review.md` | Spec | 代码审查 |

### 修改 (5 files)
| 文件 | 变更 |
|------|------|
| `urls.py` | +2 OIDC 路由 |
| `settings.py` | +6 OIDC RP 配置 |
| `VendorPortal.vue` | +31 OIDC 登录按钮 + callback |
| `AdminPortal.vue` | +30 OIDC 登录按钮 + callback |
| `conf/value-stream.yaml` | +28 新价值流条目 |

## 验证结果

- ✅ 27/27 unit tests pass
- ✅ DDD compliance check passes
- ✅ Frontend build successful
- ✅ Django settings/views/urls import verification

## 未完成项 (后续工作)

| 项目 | 状态 | 说明 |
|------|------|------|
| taskAuth 多 client 支持 | ⚠️ 待办 | 需要 Go 代码支持 bootstrapClients 列表格式 |
| E2E Playwright tests | ⚠️ 待办 | 需要 taskAuth client 注册完成后才可端到端测试 |
| OIDC view tests | ⚠️ 待办 | 需要 mock OIDC provider，计划在 Phase 5.3 |

## 经验记录

1. **taskAuth bootstrap client 限制**: taskAuth 当前仅支持单个 bootstrap client (gitlab-git-service)。新增 ai-provider client 需要 taskAuth Go 代码适配多 client 配置格式。此项在 YAML 中已预留占位。
2. **DDD 轻量实施**: 即使 OIDC 登录是相对简单的功能，DDD 分层（domain/infrastructure/interface）仍带来了清晰的职责分离和可测试性（27 个纯领域测试无需数据库）。
3. **权限分析的价值**: 权限分析提前发现了 2 个中等风险（saas_user_id 冲突、is_active 过滤），均在实施阶段修复，避免了上线后的安全漏洞。
