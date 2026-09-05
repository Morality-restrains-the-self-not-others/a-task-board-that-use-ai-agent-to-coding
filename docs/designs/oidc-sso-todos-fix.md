# OIDC SSO 待办事项解决方案

> 日期: 2026-06-24 | 关联: `taskauth-gitservice-sso-design.md`

## Todo 1: GitLab 容器重启自动化

**问题**: 修改 `docker-compose.yml` 添加 OmniAuth OIDC 后，GitLab 不会自动加载新配置。首次部署或配置变更后需要手动操作。

**方案**: 增强 `sync_omniauth_oidc.sh`，新增 `--reconfigure` 模式：

1. 检测 GitLab OmniAuth openid_connect 提供者是否已注册
2. 若未注册 → 执行 `docker exec gitlab gitlab-ctl reconfigure` 重载配置
3. 等待 GitLab 重新就绪（轮询 gitlab-rails runner）
4. 验证 OmniAuth 提供者已加载
5. 从 `conf/auth/task-auth/config.yaml` 读取最新的 client_id/secret，更新 GitLab Doorkeeper Application

**调用链**: `gitService/run.sh start` → 容器启动后自动调用 `sync_omniauth_oidc.sh --reconfigure`

**影响文件**: `gitService/scripts/sync_omniauth_oidc.sh`

---

## Todo 2: 持久化 RSA 密钥

**问题**: 每次 taskAuth 重启生成新 RSA 密钥对 → JWKS 变化 → GitLab 缓存的 JWKS 失效 → SSO 登录失败（需等 GitLab 重新 fetch JWKS，且 id_token 签名验证可能失败）。

**方案**:

1. **默认存储路径**: `$DB_DIR/oidc_signing_key.pem`（与 auth.db 同目录，如 `db/auth/task-auth/oidc_signing_key.pem`）
2. **首次启动**: 生成 RSA-2048 密钥对 → 写入 PEM 文件（0600）
3. **后续启动**: 从 PEM 文件加载，JWKS 稳定不变
4. **无需轮换（MVP）**: 先做持久化，密钥轮换后续需要时再加

**影响文件**:
- `taskAuth/src/jwt.go` — `initOidcSigningKey()` 改为：先尝试读 PEM 文件，不存在则生成并写回
- `taskAuth/src/config.go` — `signingKeyPath` 默认为 data dir 下

---

## Todo 3: Gateway 路由暴露 OIDC 端点

**问题**: OIDC 端点仅在 taskAuth 直连端口 (:8003) 可访问，不走 Gateway。GitLab 需要能通过统一入口访问这些端点，且 issuer URL 需要与 Gateway publicBase 一致。

**方案**: 在 `taskGateway/routes/routes.yaml` 新增路由：

| 路由 ID | URI | methods | upstream | auth_mode | 说明 |
|---------|-----|---------|----------|-----------|------|
| oidc-discovery | `/.well-known/openid-configuration` | GET | taskAuth | none | 公开发现文档 |
| oidc-jwks | `/api/oidc/jwks` | GET | taskAuth | none | JWKS 公钥 |
| oidc-authorize | `/api/oidc/authorize` | GET | taskAuth | none | 授权端点 |
| oidc-token | `/api/oidc/token` | POST | taskAuth | none | Token 端点 |
| oidc-userinfo | `/api/oidc/userinfo` | GET,POST | taskAuth | none | UserInfo 端点 |

**关键点**: 这些端点通过 Gateway 暴露后，`issuer` URL 自动使用 Gateway 的 publicBase（如 `https://127.0.0.1:8443`），与 GitLab 回调 URL 在同一域名下，避免跨域问题。

**影响文件**: `taskGateway/routes/routes.yaml`

---

## 价值流影响

无新增价值流。修改属于现有 `user-auth` 价值流的 `oidc-provider-core` step 增强。

## 实施顺序

1. **持久化 RSA 密钥**（taskAuth 内部变更，独立）
2. **Gateway 路由**（配置变更，独立）
3. **GitLab 重启自动化**（依赖 #1 #2 完成后的端到端验证）
