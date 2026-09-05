# taskAuth 拆分 — 实施计划

> 领域模型: `docs/superpowers/domain/2026-05-28-taskauth-split-domain.md`
> 价值流: `docs/superpowers/plans/2026-05-28-taskauth-split-value-stream.md`

## Increment 1: 登录薄切片

- [x] **Task 1.1** 创建 `taskAuth/` Go 模块、`run.sh`、`/api/health/`
  - 验证: `curl http://127.0.0.1:8003/api/health/`
- [x] **Task 1.2** 实现 login/logout + SQLite token CRUD
  - 验证: `go test ./taskAuth/src/...`
- [x] **Task 1.3** Django `taskauth_bridge` + UserViewSet delegate
  - 验证: `pytest accounts/view_test/UserViewSet_login_test.py`
- [x] **Task 1.4** internal `enrich-login` 回调
  - 验证: POST with secret → 200 + user JSON
- [x] **Task 1.5** `port_config.json` + runAll `task-auth` 服务
  - 验证: `cd valueStream && go test ./...`

## Increment 2: 邮箱注册闭环

- [x] **Task 2.1** email_register + createUserWithEmailLogin
- [x] **Task 2.2** confirm_activation + resend_activation
- [x] **Task 2.3** post-register / post-activate / resend internal 回调
  - 验证: `pytest accounts/view_test/UserViewSet_email_register_test.py accounts/view_test/UserViewSet_activate_test.py accounts/view_test/UserViewSet_resend_activation_email_test.py`

## Increment 3: 配置与无感保障

- [x] **Task 3.1** `settings_test.TASKAUTH_ENABLED=False`（pytest 隔离）
- [x] **Task 3.2** value-stream.yaml 增补 `task-auth.*` 字段
- [x] **Task 3.3** forward-login 委托（phone OTP）
  - 验证: `pytest tests/test_login_phone_code.py`（Django 路径）

## Increment 4: 待办（Future）

- [ ] **Task 4.1** 密码重置端点迁入 taskAuth
- [ ] **Task 4.2** phone_register 原生 Go 实现（去 forward）
- [ ] **Task 4.3** runAll 全栈 E2E（TASKAUTH_ENABLED=true）
- [ ] **Task 4.4** 提取 `taskAuth/domain/` 包（从 handlers  refactor）

## 验证命令汇总

```bash
# Go
cd taskAuth && go test ./src/... && go build -o taskAuth ./src

# Django user-auth
cd task2app/Saas_project && DJANGO_SETTINGS_MODULE=saas_project.settings_test \
  pytest accounts/view_test/UserViewSet_login_test.py \
         accounts/view_test/UserViewSet_email_register_test.py \
         accounts/view_test/UserViewSet_activate_test.py \
         accounts/view_test/UserViewSet_reset_password_test.py -q

# valueStream 配置
cd valueStream && go test ./...
```
