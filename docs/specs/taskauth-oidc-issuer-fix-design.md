# taskAuth OIDC SSO — GitLab 连接被拒修复设计

**状态**: 设计中  
**日期**: 2026-06-24  
**范围**: OIDC issuer URL 跨 Docker 网络可达性  
**影响**: gitService (docker-compose), taskAuth (OIDC 配置), conf/auth/task-auth

---

## 1. 问题摘要

用户访问 `http://183.250.1.132:8012/users/sign_in`，点击 **taskAuth SSO** 登录时 GitLab 报错：

> Could not authenticate you from OpenIDConnect because "Failed to open tcp connection to 127.0.0.1:8003 (connection refused - connect(2) for "127.0.0.1" port 8003)"

taskAuth 进程已在主机监听 `0.0.0.0:8003`（端口正常），从主机 `curl http://127.0.0.1:8003` 可达，
但从 GitLab Docker 容器内部 `127.0.0.1:8003` 不可达——容器内部 `127.0.0.1` 指向容器自身而非宿主机。

## 2. 根因分析

### 2.1 架构拓扑

```
用户浏览器 ──(HTTP)──> 183.250.1.132:8012 ──> Docker 端口映射 ──> GitLab 容器 (172.21.0.2:8012)
                                    │
                                    │  OIDC 发现 & Token 交换
                                    ▼
                          taskAuth (宿主机 0.0.0.0:8003, 进程 PID 2077327)
```

- GitLab 容器网关: `172.21.0.1`
- taskAuth 可达性验证: `docker exec gitlab curl http://172.21.0.1:8003` → 200 ✅
- taskAuth 可达性验证: `docker exec gitlab curl http://183.250.1.132:8003` → 200 ✅

### 2.2 三层根因

| 层 | 组件 | 当前值 | 问题 |
|----|------|--------|------|
| **L1** | `gitService/docker-compose.yml:16` | `GITLAB_OIDC_ISSUER` 默认 `http://127.0.0.1:8003` | 容器内 `127.0.0.1` 指向容器自身，taskAuth 不在容器内 |
| **L2** | `gitService/docker-compose.yml:59` | `redirect_uri` 硬编码 `http://127.0.0.1:8012/...` | 用户从外部 IP `183.250.1.132` 访问，回调 URL 与浏览器不匹配 |
| **L3** | `conf/auth/task-auth/config.yaml` | `oidc.issuer: ""`（空，fallback 到 `cfg.Host`） | 发现文档返回 `"issuer": "http://0.0.0.0:8003"`，`0.0.0.0` 浏览器和容器均不可路由 |

### 2.3 issuerURL() 解析链

```
issuerURL() [taskAuth/src/oidc_handlers.go:56-64]:
  1. cfg.OidcIssuer         → ""（config.yaml 中 issuer: ""）
  2. cfg.GatewayPublicBase  → "http://183.250.1.132:18081"（task-gateway publicBase）
     ↑ 当前运行进程未读到（可能启动早于 gateway 配置加载）
  3. fallback               → "http://0.0.0.0:8003" ← 当前返回
```

**OIDC 协议约束**: issuer 必须被 OIDC Client (GitLab) 和最终用户浏览器同时可达。
浏览器从 `183.250.1.132` 访问，因此 issuer 必须使用此外部 IP/域名。

## 3. 修复设计

### 3.1 变更矩阵

| # | 文件 | 变更类型 | 描述 |
|---|------|---------|------|
| ① | `conf/auth/task-auth/config.yaml` | 配置修改 | 显式设置 `oidc.issuer` 避免 fallback 到 `0.0.0.0` |
| ② | `gitService/docker-compose.yml` | 配置修改 | `GITLAB_OIDC_ISSUER` 默认值从 `127.0.0.1` 改为 Docker 网关可达地址；`redirect_uri` 参数化 |
| ③ | `gitService/scripts/sync_omniauth_oidc.sh` | 脚本增强 | 从 taskAuth 配置读取 issuer 并注入容器 env，OIDC issuer 不匹配时自动 reconfigure |
| ④ | `gitService/run.sh` | 脚本增强 | 在 compose up 前导出 `GITLAB_OIDC_ISSUER` env |

### 3.2 方案选择：issuer 值

| 候选 | GitLab(容器内) | 浏览器(外部) | 可移植性 |
|------|:---:|:---:|------|
| `http://127.0.0.1:8003` | ❌ | ❌ (非本机) | — |
| `http://host.docker.internal:8003` | ✅ (需 extra_hosts) | ❌ | 中 (Linux 需 Docker 20.10+) |
| `http://183.250.1.132:8003` | ✅ | ✅ | 低 (硬编码 IP) |
| `http://172.21.0.1:8003` | ✅ | ❌ | 低 (网关 IP 不固定) |

**选择**: 使用 **`GITLAB_EXTERNAL_HOST`**（即 `183.250.1.132`）动态构造 issuer URL。
`gitService/run.sh` 已从 `conf/git-service/config.yaml` 读取 `GITLAB_EXTERNAL_HOST` 并导出环境变量。

对于 taskAuth 侧的 issuer，采用**显式配置**方式：在 `conf/auth/task-auth/config.yaml` 中设置 issuer 为空字符串时的行为改为从 task-gateway publicBase 派生（使用 `http://<GITLAB_EXTERNAL_HOST>:8003`）。

**最终方案**: taskAuth 的 OIDC issuer 显式设为 `${INFRA_HOST}:8003` 格式。具体实现：

- `conf/auth/task-auth/config.yaml` 的 `oidc.issuer` 使用 `http://${INFRA_HOST}:8003` 占位符
- `conf/auth/task-auth/sync.sh` 在同步时解析 `${INFRA_HOST}` 为实际的 `GITLAB_EXTERNAL_HOST`
- `gitService/docker-compose.yml` 的 `GITLAB_OIDC_ISSUER` 默认使用 `http://${GITLAB_EXTERNAL_HOST}:8003`

### 3.3 详细变更

#### ① `conf/auth/task-auth/config.yaml` — 显式 issuer

```yaml
# 修改前:
oidc:
  issuer: ""                      # 留空自动使用 GatewayPublicBase

# 修改后:
oidc:
  issuer: "http://${INFRA_HOST}:8003"  # OIDC issuer 需对浏览器和容器内 GitLab 均可达
```

需要配套修改: `conf/auth/task-auth/sync.sh` 或 taskAuth 配置加载逻辑，支持 `${INFRA_HOST}` 占位符替换。
更简单的做法：在 taskAuth 的 `config.go` 中，当 `cfg.GatewayPublicBase` 为 `http://<host>:18081` 时，
提取 host 并拼上 taskAuth 自己的端口 8003，作为 issuer fallback。

#### ② `gitService/docker-compose.yml` — issuer 和 redirect_uri

```yaml
# 修改前 (L16):
GITLAB_OIDC_ISSUER: ${GITLAB_OIDC_ISSUER:-http://127.0.0.1:8003}

# 修改后:
GITLAB_OIDC_ISSUER: ${GITLAB_OIDC_ISSUER:-http://${GITLAB_EXTERNAL_HOST}:8003}
```

```yaml
# 修改前 (L59):
redirect_uri: 'http://127.0.0.1:8012/users/auth/openid_connect/callback'

# 修改后:
redirect_uri: "http://#{ENV['GITLAB_EXTERNAL_HOST']}:#{ENV['GITLAB_HTTP_PORT']}/users/auth/openid_connect/callback"
```

#### ③ `conf/auth/task-auth/config.yaml` — bootstrapRedirectUri

```yaml
# 修改前:
bootstrapRedirectUri: "http://127.0.0.1:8012/users/auth/openid_connect/callback"

# 修改后:
bootstrapRedirectUri: "http://${INFRA_HOST}:8012/users/auth/openid_connect/callback"
```

taskAuth 使用此 URI 校验 GitLab 发来的 redirect_uri。两者不匹配会直接报错。

#### ④ `gitService/run.sh` — 注入 OIDC issuer 环境变量

在 `load_gitservice_config` 之后、`compose up` 之前，导出 `GITLAB_OIDC_ISSUER`:

```bash
# 新增: 同步 OIDC issuer 到 GitLab 容器环境变量
# 优先使用 taskAuth 配置的 issuer，fallback 到 GITLAB_DISPLAY_HOST
export GITLAB_OIDC_ISSUER="${GITLAB_OIDC_ISSUER:-http://${GITLAB_DISPLAY_HOST}:8003}"
```

### 3.4 运行时修复顺序

1. 修改 `conf/auth/task-auth/config.yaml` → 重启 taskAuth
2. 修改 `gitService/docker-compose.yml` → 重建 GitLab 容器 (env 变更需 `docker compose up -d --force-recreate`)
3. 验证 OIDC 发现文档 issuer 正确
4. 验证 SSO 登录完整流程

### 3.5 taskAuth issuer fallback 增强（config.go）

当前 `issuerURL()` fallback 到 `cfg.Host` 时使用 `0.0.0.0`，这是不可路由的地址。
增加一层 fallback：当 `GatewayPublicBase` 可用时，提取其 host 并用 taskAuth 端口拼接。

```go
// oidc_handlers.go — issuerURL() 修改
func issuerURL() string {
    if cfg.OidcIssuer != "" {
        return cfg.OidcIssuer
    }
    if cfg.GatewayPublicBase != "" {
        // GatewayPublicBase 如 "http://183.250.1.132:18081"
        // 提取 host 并拼接 taskAuth 自身端口
        if u, err := url.Parse(cfg.GatewayPublicBase); err == nil {
            return fmt.Sprintf("http://%s:%d", u.Hostname(), cfg.Port)
        }
        return cfg.GatewayPublicBase
    }
    // 最终 fallback — 用 0.0.0.0 在明确配置了 host 时替换
    host := cfg.Host
    if host == "0.0.0.0" || host == "" {
        host = "127.0.0.1"
    }
    return fmt.Sprintf("http://%s:%d", host, cfg.Port)
}
```

## 4. 值流影响分析

### 4.1 受影响的值流

查阅 `conf/value-stream.yaml`，本修复涉及以下值流和步骤：

| 值流 | 步骤 | 影响 |
|------|------|------|
| `user-auth` | `runall-task-auth-orchestration` | taskAuth OIDC issuer 配置变更影响健康检查 |
| `gitlab-oauth-scope-failfast-governance` | `gitlab-oauth-app-bootstrap` | `sync_local_oauth_app_scopes.sh` 已自愈创建 OAuth Application；OIDC issuer 变化不影响 scope 同步 |
| `gitlab-oauth-scope-failfast-governance` | `gitlab-scope-startup-failfast` | issuer 变更不影响 scope 校验逻辑 |

### 4.2 字段影响

| 字段 | 变更 |
|------|------|
| `task-auth.runtime.oidc_issuer` | 新增：OIDC issuer URL 需可路由（非 `0.0.0.0`） |
| `git-service.runtime.oidc_issuer_env` | 新增：`GITLAB_OIDC_ISSUER` 环境变量注入值 |

### 4.3 测试影响

- `gitService/scripts/test_sync_oauth_app.sh` — OIDC issuer 变更后需验证 SSO 流程
- 不需要新增 Django/Python 单元测试（纯配置和脚本变更）

## 5. 领域概念清单

| 概念 | 类型 | 边界 | 说明 |
|------|------|------|------|
| OIDC Provider | Entity | taskAuth | OpenID Connect 身份提供者，签发 ID Token |
| OIDC Client (GitLab) | Entity | gitService | 信赖方，消费 OIDC ID Token 完成 SSO |
| OIDC Bootstrap Client | Aggregate | taskAuth | 启动时自愈创建的 OIDC 客户端（gitlab-git-service） |
| OIDC Discovery Document | Value Object | taskAuth | `.well-known/openid-configuration` 端点元数据 |
| `issuer` URL | Value Object | taskAuth/gitService | 标识 OIDC Provider 的可路由 URL，须对容器和浏览器均可达 |
| `redirect_uri` | Value Object | gitService/taskAuth | OAuth 回调地址，taskAuth 须校验其与注册值匹配 |
| GitLab Docker Network | Infrastructure | gitService | Docker bridge 网络（172.21.0.0/16），容器↔宿主机通信路径 |

## 6. 风险与回滚

- **风险**: `${INFRA_HOST}` 占位符替换逻辑出错 → taskAuth 启动失败
  - **缓解**: taskAuth `config.go` 增加 issuer fallback 增强，占位符失败时 fallback 到 GatewayPublicBase host 提取
- **回滚**: 还原 `docker-compose.yml` 和 `config.yaml`，重启 taskAuth 和 GitLab 容器
- **向后兼容**: 若 `GITLAB_OIDC_ISSUER` 已通过 env 显式设置（非空），行为不变
