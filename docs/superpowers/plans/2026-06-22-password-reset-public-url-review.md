# 代码审查: 密码重置邮件域名可配置化

**日期:** 2026-06-22
**审查结果:** ✅ 通过 — 0 个问题

## 变更

| 文件 | 变更 | 行数 |
|------|------|------|
| `conf/frontend/vue/config.yaml` | 新增 `publicBaseUrl: http://183.250.1.132:4000` | +1 |
| `core/config/settings_manager.py` | `get_frontend_domain()` — 优先 `publicBaseUrl`，回退至 `host:port` | +4 |

## 验证

| 检查 | 结果 |
|------|------|
| publicBaseUrl 已配置 → 返回外部 URL | ✅ |
| 回退 — publicBaseUrl 缺失 → host:port | ✅ |
| 无尾部斜杠 | ✅ |
| 向后兼容 (默认未配置行为不变) | ✅ |
