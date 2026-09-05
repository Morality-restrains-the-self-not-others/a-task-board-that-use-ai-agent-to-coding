# Implementation Plan: taskAuth OIDC Issuer Docker 网络可达性修复

> 输入:
> - 设计文档: `docs/specs/taskauth-oidc-issuer-fix-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-24-taskauth-oidc-issuer-fix-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-24-taskauth-oidc-issuer-fix-nfr-clarification.md`
> - DDD: 跳过 (配置修复)

## 任务清单

### Task 1: 增强 issuerURL() fallback — GatewayPublicBase host 提取

**文件**: `taskAuth/src/oidc_handlers.go`
**类型**: Go 代码修改
**依赖**: 无
**验证**: `go build ./src`

当前 `issuerURL()` fallback 链:
1. `cfg.OidcIssuer` → 空
2. `cfg.GatewayPublicBase` → 返回 `http://183.250.1.132:18081`（网关端口，非 taskAuth 端口）
3. `http://cfg.Host:cfg.Port` → `http://0.0.0.0:8003`（不可路由）

**修改**: 在步骤 2 中，不直接返回 `GatewayPublicBase`，而是提取其 hostname 并拼接 taskAuth 自身端口。

```go
func issuerURL() string {
    if cfg.OidcIssuer != "" {
        return cfg.OidcIssuer
    }
    if cfg.GatewayPublicBase != "" {
        if u, err := url.Parse(cfg.GatewayPublicBase); err == nil {
            return fmt.Sprintf("http://%s:%d", u.Hostname(), cfg.Port)
        }
        return cfg.GatewayPublicBase
    }
    host := cfg.Host
    if host == "0.0.0.0" || host == "" {
        host = "127.0.0.1"
    }
    return fmt.Sprintf("http://%s:%d", host, cfg.Port)
}
```

### Task 2: 修正 GITLAB_OIDC_ISSUER 默认值 + redirect_uri 参数化

**文件**: `gitService/docker-compose.yml`
**类型**: Docker Compose 配置修改
**依赖**: Task 1 (issuer 正确后 GitLab 侧 issuer 才能对齐)

**变更 A** (L16): `GITLAB_OIDC_ISSUER` 默认值
```yaml
# 旧:
GITLAB_OIDC_ISSUER: ${GITLAB_OIDC_ISSUER:-http://127.0.0.1:8003}
# 新:
GITLAB_OIDC_ISSUER: ${GITLAB_OIDC_ISSUER:-http://${GITLAB_EXTERNAL_HOST}:8003}
```

**变更 B** (L59): `redirect_uri` 参数化
```ruby
# 旧:
redirect_uri: 'http://127.0.0.1:8012/users/auth/openid_connect/callback'
# 新:
redirect_uri: "http://#{ENV['GITLAB_EXTERNAL_HOST']}:#{ENV['GITLAB_HTTP_PORT']}/users/auth/openid_connect/callback"
```

### Task 3: gitService/run.sh 导出 GITLAB_OIDC_ISSUER

**文件**: `gitService/run.sh`
**类型**: Shell 脚本增强
**依赖**: Task 2 (docker-compose.yml 需要此 env var)

在 `load_gitservice_config` 之后、`compose up` 之前，增加:

```bash
# 同步 OIDC issuer 到 GitLab 容器环境变量
# 优先使用已设置的 GITLAB_OIDC_ISSUER，fallback 到 GITLAB_DISPLAY_HOST
export GITLAB_OIDC_ISSUER="${GITLAB_OIDC_ISSUER:-http://${GITLAB_DISPLAY_HOST}:8003}"
```

### Task 4: 修正 bootstrapRedirectUri 以匹配外部可达地址

**文件**: `conf/auth/task-auth/config.yaml`
**类型**: YAML 配置修改
**依赖**: Task 1 (taskAuth 重启后生效)

```yaml
# 旧:
bootstrapRedirectUri: "http://127.0.0.1:8012/users/auth/openid_connect/callback"
# 新 (与 GITLAB_EXTERNAL_HOST 对齐):
bootstrapRedirectUri: "http://183.250.1.132:8012/users/auth/openid_connect/callback"
```

> 注: bootstrapRedirectUri 在 taskAuth 启动时用于 seed OIDC client 到 auth.db。
> 如果 auth.db 中已有该 client 记录，需手动更新或删除后重启 taskAuth 触发重新 seed。

### Task 5: 验证 — 重启服务并测试

**类型**: 手动验证
**依赖**: Task 1-4 全部完成

```bash
# 1. 重新编译并重启 taskAuth
cd /tmp/ram-work/taskAuth && ./run.sh stop && ./run.sh start

# 2. 验证 OIDC 发现文档 issuer 正确
curl -s http://127.0.0.1:8003/.well-known/openid-configuration | jq .issuer
# 预期: "http://183.250.1.132:8003"

# 3. 重建 GitLab 容器（env 变更需要 force-recreate）
cd /tmp/ram-work/gitService && bash run.sh stop && bash run.sh start

# 4. 验证容器内可达性
docker exec gitlab curl -sS -o /dev/null -w '%{http_code}' http://183.250.1.132:8003/.well-known/openid-configuration
# 预期: 200

# 5. 验证 SSO 登录
# 浏览器访问 http://183.250.1.132:8012/users/sign_in → 点击 taskAuth SSO → 完成登录
```

## 执行顺序

```
Task 1 (Go fallback) ──┐
                       ├──> Task 5 (验证)
Task 2 (docker-compose)─┤
                       │
Task 3 (run.sh) ───────┤
                       │
Task 4 (config.yaml) ──┘
```

Tasks 1-4 可并行执行，Task 5 在所有 task 完成后串行验证。

## 回滚计划

如有问题，逐文件回滚:
```bash
git checkout -- taskAuth/src/oidc_handlers.go
git checkout -- gitService/docker-compose.yml
git checkout -- gitService/run.sh
git checkout -- conf/auth/task-auth/config.yaml
# 重启 taskAuth + GitLab 容器
```
