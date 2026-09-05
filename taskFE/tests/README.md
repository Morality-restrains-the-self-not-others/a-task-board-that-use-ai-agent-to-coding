# Playwright 端到端测试

本目录存放所有 Playwright 测试脚本，规范见 `.ai/05_testing_quality/02_test_management_rules.md`。

## 运行测试

```bash
# 在 taskFE 根或 ./app 下执行（根 package.json 委托到 app）
npm run test:e2e          # 默认有头（未设 CI），60 秒超时；使用根 playwright.config.js（testDir ./tests）
npm run test:e2e:headed   # 显式 `--headed`（与默认等价，便于脚本引用）
```

配置与测试发现：`playwright.config.js` 位于 taskFE 根，`testDir: './tests'` 即本目录；CI 环境（`CI=1`）自动无头。

## 命名规范

- 组件级：`{组件名}.{测试意图}.playwright.test.js`
- 流程级：`{流程简述}.playwright.test.js`

## 前置条件

- 后端服务运行于 localhost:8000
- 前端服务运行于 localhost:3000
- 或使用默认 `playwright.config.js`（taskFE 根）的 webServer 自动启动（CI 模式）

## 云平台授权 — 启动测试与 SSH 核验（可选，会产生阿里云费用）

需登录账号，并设置：

- `E2E_EMAIL` / `E2E_PASSWORD`
- `E2E_VERIFY_CLOUD_PROBE=1`：才会真实调用「启动测试」创建 ECS；未设置则该用例会 skip
- `E2E_PROBE_INSTALLED_IMAGE_ID=<已安装镜像ID>`：系统管理员探针接口要启动的镜像
- `E2E_SYSTEM_ADMIN_UID=<uid>`：系统管理员 URL 段（默认 `1`）
- `E2E_SKIP_INSTANCE_SSH=1`：仅测 UI 与弹窗，**不**通过本机 `ssh` 连实例检查 `/root/init_from_task2app.sh.log`
- `E2E_REQUIRE_SSHD_DROPIN=1`：额外要求实例上存在 `/etc/ssh/sshd_config.d/99-task2app.conf`（镜像 UserData 须含 SSH 模板）
- `E2E_EXPECT_CONTAINER_IMAGE_REF=<registry/image:tag>`：可选；额外校验远端 `/root/init_from_task2app.sh` 包含该镜像引用（用于核验 tag）

**本机需安装 OpenSSH 客户端**（`ssh` 在 PATH 中），且出网能访问 ECS 公网 IP:22，安全组放行 SSH。

完整核验（含 SSH 读日志）：

```bash
cd ../  # taskFE 根
E2E_EMAIL=... E2E_PASSWORD='...' E2E_VERIFY_CLOUD_PROBE=1 \\
  npx playwright test -c playwright.config.js \\
  tests/CloudAuthorizations.probe-start-ssh-modal.playwright.test.js --project=chromium
```

若镜像里保存的 UserData 仍是旧版 sshd 片段（含错误的 `Match all` + `PubkeyAuthentication no`），请在管理台重新保存为当前默认模板后再测。
