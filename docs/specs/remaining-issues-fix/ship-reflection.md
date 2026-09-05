# Ship Reflection: 剩余问题修复

## 交付日期
2026-06-29

## 已解决

| 问题 | 修复 | 文件 |
|------|------|------|
| Vendor DB 缺失 | INSERT Vendor `contact@daydaymoney.com` | `db/ai-provider/ai-provider.sqlite3` |
| taskAuth 单 client 限制 | `OidcBootstrapClients[]` + backward compat | `config.go`, `oidc_bootstrap.go`, `main.go`, `config.yaml` |
| OIDC View 测试缺失 | 13 tests (authorize + callback) | `test_oidc_views.py` |

## 测试

- ✅ 40/40 tests pass (0.39s)
- ✅ Go syntax OK (`go fmt`)

## 待重启服务

以下服务需要重启以加载新配置：
- **taskAuth** (port 8003): 重新编译 + 重启，加载 `bootstrapClients` 列表
- **AI Provider** (port 8010): 无需重启（仅配置变更在 taskAuth 侧）

## 验证步骤

1. Build taskAuth: `cd taskAuth && go build -o bin/taskAuth ./src/`
2. Restart taskAuth (check logs for "oidc bootstrap: client ai-provider ready")
3. Test OIDC login: 访问 `http://183.250.1.132:8010/` → 点击 "通过 OIDC 账号登录"
4. Test SSO bridge: 主站登录后点击 "厂商门户（SSO）"（Vendor 已创建，应成功）
