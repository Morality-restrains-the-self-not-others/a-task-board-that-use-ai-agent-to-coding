# 实施计划: taskAuth SSO 登录 404 修复

> 基于: 设计文档 `docs/specs/oidc-sso-404-fix-design.md`, 价值流 `...-value-stream.md`, NFR `...-nfr-clarification.md`, DDD `...-ddd.md`

## 计划概览

**3 个增量，8 个任务，预计 1-2 小时完成。**

## Increment 1: 修复 OIDC Authorize 重定向 404 (Thin Slice)

### Task 1.1: 修正 taskAuth OIDC authorize 登录重定向 URL
- **文件**: `taskAuth/src/oidc_handlers.go`
- **变更**: 第 171 行 `loginURL` 从 `/login/` 改为 `/auth/login/`
- **验证**: `cd taskAuth && go build ./...` 编译通过
- **NFR**: QS-01 (redirect_uri 白名单不变), QS-02 (未认证 → 登录页)

```diff
- loginURL := cfg.GatewayPublicBase + "/login/?next=" + url.QueryEscape(r.URL.String())
+ loginURL := cfg.GatewayPublicBase + "/auth/login/?next=" + url.QueryEscape(r.URL.String())
```

### Task 1.2: 网关新增 `/auth/login` 路由
- **文件**: `taskGateway/routes/routes.yaml`
- **变更**: 在 catch-all `django-default` 之前添加 `/auth/login` 路由
- **配置**:
  ```yaml
  - id: auth-login-page
    priority: 801
    uri: /auth/login
    methods: [GET]
    upstream: django
    auth_mode: none
  ```
- **验证**: `cd taskGateway && bash scripts/ci/check_routes.sh --check` 通过

### Task 1.3: 重新生成 APISIX 配置
- **文件**: `taskGateway/apisix/apisix.yaml`
- **动作**: 运行 `cd taskGateway && bash scripts/ci/check_routes.sh` 生成 apisix.yaml
- **验证**: apisix.yaml 包含 `auth-login-page` 路由

## Increment 2: 统一 OIDC Issuer 为网关地址

### Task 2.1: 更新 taskAuth OIDC Issuer 配置
- **文件**: `conf/auth/task-auth/config.yaml`
- **变更**: `oidc.issuer` 从 `http://183.250.1.132:8003` 改为 `http://183.250.1.132:18081`
- **验证**: 重启 taskAuth 后 `curl http://183.250.1.132:18081/.well-known/openid-configuration` 返回 `issuer: http://183.250.1.132:18081`
- **NFR**: QS-04 (token 端点不受影响), QS-05 (容器内可达)

```diff
-   issuer: "http://183.250.1.132:8003"
+   issuer: "http://183.250.1.132:18081"
```

### Task 2.2: 同步 gitService run.sh 的 OIDC Issuer 推导
- **文件**: `gitService/run.sh`
- **变更**: `GITLAB_OIDC_ISSUER` 的默认推导从 `:8003` 更新为使用 gateway 端口
- **当前逻辑**: `export GITLAB_OIDC_ISSUER="${GITLAB_OIDC_ISSUER:-http://${GITLAB_HOSTNAME}:8003}"`
- **新逻辑**: 从 gateway config 读取 `publicBase` 并拼接，或保持为 `http://${GITLAB_HOSTNAME}:18081`
- **验证**: `bash gitService/run.sh` 输出 `OIDC issuer: http://183.250.1.132:18081`

## Increment 3: SSL Fix 持久化 + Playwright E2E

### Task 3.1: SSL Fix 持久化到 GitLab entrypoint
- **文件**: `gitService/docker-compose.yml` (或者新增 entrypoint 脚本)
- **变更**: 在 GitLab 容器启动时自动应用 `SWD.url_builder = URI::HTTP`
  - 方案 A: 在 docker-compose 的 `command` 或 `entrypoint` 中注入 initializer 文件
  - 方案 B: 挂载一个包含 initializer 的 volume
- **推荐**: 在 `docker-compose.yml` 中添加 volume 挂载:
  ```yaml
  volumes:
    - './initializers/zzz_fix_oidc_http.rb:/opt/gitlab/embedded/service/gitlab-rails/config/initializers/zzz_fix_oidc_http.rb:ro'
  ```
- **新文件**: `gitService/initializers/zzz_fix_oidc_http.rb`
  ```ruby
  require "swd"
  SWD.url_builder = URI::HTTP
  ```
- **验证**: 容器重建后 `docker exec gitlab gitlab-rails runner "require 'swd'; puts SWD.url_builder"` 输出 `URI::HTTP`
- **NFR**: QS-06 (容器重建后自动恢复)

### Task 3.2: Playwright 诊断测试 — 验证 404 已修复
- **文件**: `gitService/playwright/tests/oidc-sso-404-verify.playwright.test.js` (新增)
- **测试内容**:
  1. 导航到 GitLab 登录页 → 点击「taskAuth SSO」
  2. 等待重定向完成
  3. **断言**: 页面 URL 不含 `404`，页面 body 不含 `404 page not found`
  4. 如果重定向到登录页 → 验证登录页正常显示（200）
  5. 如果已登录 → 验证到达 GitLab Dashboard
- **运行**: `cd gitService/playwright && npx playwright test oidc-sso-404-verify.playwright.test.js`

### Task 3.3: Playwright 端到端测试 — 完整 SSO 流程
- **文件**: `gitService/playwright/tests/oidc-sso-e2e-full-flow.playwright.test.js` (新增)
- **测试内容**:
  1. 主站登录 (`http://183.250.1.132:4000/auth/login/`)
  2. 点击「代码仓库」→ 进入 GitLab
  3. 点击「taskAuth SSO」
  4. 验证 OIDC 回调成功（无 404、无 SSL 错误）
  5. 验证最终在 GitLab Dashboard 或项目页
- **环境变量**: `LOGIN_URL`, `PW_EMAIL`, `PW_PASSWORD`, `GITLAB_URL`
- **运行**: `cd gitService/playwright && npx playwright test oidc-sso-e2e-full-flow.playwright.test.js`
- **NFR**: QS-01 (redirect_uri 安全), QS-02 (无 404)

### Task 3.4: 更新现有测试的断言
- **文件**: `gitService/playwright/tests/oidc-sso-login.playwright.test.js`
- **变更**: 在 Step 6 之后增加 404 检测断言
  ```js
  // 新增: 验证不出现 404
  expect(bodyText).not.toMatch(/404 page not found/i);
  expect(currentUrl).not.toMatch(/\/404/);
  ```
- **验证**: 运行现有测试套件全部通过

## 任务依赖图

```
Task 1.1 (redirect fix)  ──┐
                            ├──> Increment 1 完成
Task 1.2 (gateway route) ──┤
                            │
Task 1.3 (apisix gen)    ──┘
         │
         ▼
Task 2.1 (issuer config) ──┐
                            ├──> Increment 2 完成
Task 2.2 (run.sh sync)   ──┘
         │
         ▼
Task 3.1 (ssl persistence) ──┐
Task 3.2 (404 verify test) ──┤
Task 3.3 (E2E test)       ──┼──> Increment 3 完成
Task 3.4 (update existing) ──┘
```

## 验证命令汇总

```bash
# Increment 1
cd taskAuth && go build ./...
cd taskGateway && bash scripts/ci/check_routes.sh --check

# Increment 2
curl http://183.250.1.132:18081/.well-known/openid-configuration | jq .issuer

# Increment 3
docker exec gitlab gitlab-rails runner "require 'swd'; puts SWD.url_builder"
cd gitService/playwright && npx playwright test
```

## 回滚计划

如果 issuer 变更导致问题：
1. 回滚 `conf/auth/task-auth/config.yaml` issuer 为 `http://183.250.1.132:8003`
2. 回滚 `gitService/run.sh` GITLAB_OIDC_ISSUER 端口
3. 重启 taskAuth + GitLab reconfigure

如果 `/auth/login` 路由有问题：
1. 从 `routes.yaml` 删除 `auth-login-page` 路由
2. 重新生成 apisix.yaml
3. 重启 taskGateway
