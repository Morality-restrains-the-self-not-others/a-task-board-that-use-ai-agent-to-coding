# 实施计划: 密码重置邮件域名可配置化

> 输入:
> - 设计: `docs/superpowers/specs/2026-06-22-password-reset-public-url-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-22-password-reset-public-url-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-22-password-reset-public-url-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-22-password-reset-public-url-ddd.md`

**规模:** 小型配置修复 — 2 文件，约 5 行

## 任务清单

- [ ] **Task 1** — 新增 `publicBaseUrl` 至 vue 配置
  - **文件:** `conf/frontend/vue/config.yaml`
  - **变更:** 新增一行 `publicBaseUrl: http://183.250.1.132:4000`
  - **依赖:** 无

- [ ] **Task 2** — 修改 `get_frontend_domain()` 优先 `publicBaseUrl`
  - **文件:** `task2app/Saas_project/core/config/settings_manager.py` (第 211-216 行)
  - **变更:** 在 `host:port` 逻辑之前增加 `publicBaseUrl` 优先判断，回退保持兼容
  - **依赖:** Task 1

- [ ] **Task 3** — 验证回退行为
  - **命令:** `python -c "from core.config.settings_manager import SettingsManager; ..."`
  - **预期:** `publicBaseUrl` 存在时返回外部 URL；缺失时回退至 `http://127.0.0.1:4000`
  - **依赖:** Task 2

## 文件变更

| 文件 | 操作 | 行数 |
|------|------|------|
| `conf/frontend/vue/config.yaml` | 修改 | +1 行 |
| `core/config/settings_manager.py` | 修改 | ~5 行 |

## 依赖图

```
Task 1 (YAML 配置)
  → Task 2 (Python 逻辑)
    → Task 3 (验证)
```
