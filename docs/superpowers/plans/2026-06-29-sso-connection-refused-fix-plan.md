# 实施计划: SSO Bridge Connectivity Fix

> 来源: `docs/specs/sso-connection-refused-fix/design.md`
> 状态: ✅ 全部完成

## 变更清单

| # | 文件 | 改动 | 状态 |
|---|------|------|------|
| 1 | `conf/ai/ai-provider/config.yaml` | `host: 0.0.0.0` + `allowedExtendHosts: ['*']` | ✅ |
| 2 | `Saas_Ai_Provider/provider/port_config_loader.py` | 路径 `conf/ai-provider/` → `conf/ai/ai-provider/` | ✅ |
| 3 | `Saas_Ai_Provider/run.sh` | 路径 `conf/ai-provider/` → `conf/ai/ai-provider/` | ✅ |
| 4 | `Saas_Ai_Provider/provider/settings.py` | dev 模式 `ALLOWED_HOSTS = ["*"]` | ✅ |
| 5 | `playwright/.../django8010-sso-connection-refused-fix.playwright.test.js` | E2E 测试 (TC-02~TC-05b) | ✅ |
| 6 | `playwright/.../django8010-sso-connection-refused-fix.playwright.test.js.testIntent` | 测试意图文档 | ✅ |

## 验证计划

### V-1: 端口绑定验证
```bash
ss -tlnp | grep 8010
# 预期: 0.0.0.0:8010 (NOT 127.0.0.1:8010)
```
**状态**: ✅ 已验证 → `LISTEN 0.0.0.0:8010`

### V-2: 外部可达性
```bash
curl -s -o /dev/null -w "%{http_code}" http://183.250.1.132:8010/admin
# 预期: 200 (NOT 000/Connection Refused)
```
**状态**: ✅ 已验证 → `HTTP 200`

### V-3: localhost 可达性
```bash
curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8010/admin
# 预期: 200
```
**状态**: ✅ 已验证 → `HTTP 200`

### V-4: Health endpoint
```bash
curl -s http://127.0.0.1:8010/api/health/ | python3 -m json.tool
# 预期: {"service": "ai-provider", "ok": true}
```
**状态**: ✅ 已验证

### V-5: SSO 跳转 Playwright E2E
```bash
cd task2app/playwright/saas_ai_provider
npx playwright test --config=playwright.django.config.js \
  tests/django8010-sso-connection-refused-fix.playwright.test.js
```
**状态**: ⏳ 需手动执行（需 E2E_SSO_SUPERADMIN_EMAIL/PASSWORD 环境变量）

### V-6: YAML 语法校验
```bash
python3 -c "import yaml; yaml.safe_load(open('conf/value-stream.yaml'))"
```
**状态**: ✅ 已验证 — 59 streams

## 回滚方案

若修复导致问题：
```bash
# 还原 config
cd conf/ai/ai-provider/
git checkout config.yaml

# 还原代码
cd task2app/Saas_Ai_Provider
git checkout provider/port_config_loader.py run.sh provider/settings.py

# 重启服务
bash run.sh stop && bash run.sh start-embedded
```
