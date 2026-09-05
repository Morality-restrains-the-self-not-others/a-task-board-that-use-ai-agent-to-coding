# Value Stream: port_config → monorepo conf/

> 设计：`docs/superpowers/specs/2026-06-02-port-config-split-monorepo-conf-design.md`

## Related Value Streams

- **修改型**：`git-site-oauth-*`、`domain-events-transport`、`runall-*` 相关步骤的字段描述从 `port_config.json` 改为 `conf/<app>/` 路径。
- **无新 stream**：平台配置治理，不新增用户可见功能。

## Increments（按价值排序）

| # | 增量 | 价值 | 验证 |
|---|------|------|------|
| 1 | `conf-sync` + YAML 迁移骨架 | 可生成碎片、目录可审阅 | `scripts/conf-sync-all.sh` exit 0 |
| 2 | Django / 测试号码 `config.test.yaml` | 主站与短信测试规则继续成立 | `test_phone_config.py` |
| 3 | taskEvents `conf/domain-events/<event>/` | intent 端口可配置、runAll 一致 | `taskEvents/config` tests |
| 4 | 删除 JSON + 文档 | 单一配置源 | grep 无 `port_config.json` 运行时依赖 |

## value-stream.yaml 更新（描述级）

- `git-oauth.api_githubappusercredential.provider` → 来源 `conf/git-oauth/providers/*.yaml`
- `saas-backend.runtime.*` domainEvents → `conf/domain-events/config.yaml` + `<event>/config.yaml`
- 测试短信 → `conf/core/django/config.test.yaml` → `django.test_phone_numbers`
