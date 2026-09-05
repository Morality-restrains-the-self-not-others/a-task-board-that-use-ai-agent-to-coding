# Value Stream: conf 硬化 Phase 4.5

> 设计：`docs/superpowers/specs/2026-06-02-conf-hardening-phase45-design.md`

## Related Value Streams

- **修改型**：`2026-06-02-port-config-split-monorepo-conf`（Phase 1–3 已完成）
- 重叠：`domain-events-consumer-split`、`message-queue-kafka-to-redis` — 仅 **字段描述** 从 `port_config.*` → `conf/<app>/`

## Increments

| Inc | 价值 | 验证 |
|-----|------|------|
| 1 | CI 保证 `conf/` 与 sync 一致 | `check_conf_sync.sh` |
| 2 | Saas/工具直读 YAML，无 JSON 聚合 | pytest + `rg assemble_legacy` |
| 3 | 前端/E2E 直读 conf | Playwright helper + vite 无 `conf_emit_json` |
| 4 | wt-relay-stop 与主树一致 | `rg port_config.json task2app-wt-relay-stop` |
| 5 | 密钥分层文档（不迁明文） | `config.example.yaml` + README |

## value-stream.yaml 变更

更新 `value-stream.yaml` 中 `port_config` 文案为 `conf/<app>/config.yaml`（4 处 description，无新 step）。
