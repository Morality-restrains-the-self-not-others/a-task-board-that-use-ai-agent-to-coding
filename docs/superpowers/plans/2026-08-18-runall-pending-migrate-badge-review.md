# Review — 9999 未 migrate 标注

- **日期**: 2026-08-18
- **计划**: `docs/superpowers/plans/2026-08-18-runall-pending-migrate-badge-plan.md`

## CRG

`code-review-graph query --brief` 不支持；`callers_of`/`impact` 对新建符号可能尚未入图（update 时 0 nodes）。Domain 无基础设施 import。

## 五轴

| 轴 | 结论 |
|---|---|
| Correctness | missing 计 pending；stale 不计；表缺失=全部 pending；unreachable 不计 pending；GET 契约测过；`refresh()` 不含巡检 |
| Readability | 比对在 domain；mysql CLI 在 infrastructure |
| Architecture | 扩现有 runAll 编排器；只读 `data_migrate_log`；无新 BC / 无 MQ |
| Security | 库名来自 registry；不返回 DSN；写路径仍 confirm；GET 与 `/api/status` 同为 9999 内网 |
| Performance | 15s 缓存；禁止 2s 轮询；首屏串行查库（OPT 并行） |

## 安全清单

- 无密钥入日志
- SQL 固定 `SELECT step_key FROM data_migrate_log`
- 无用户指定 DB query
- mysql `-p` 在 argv（与 apply_datamigrate 相同，Nit）

## Intent→Event

纯查询例外已写意图表。无 publish。

## Simplify

无死代码。未改 04.js 行数（wrap 在 16.js）。

## 分级

- Critical / Required：无
- Nit：串行 mysql；argv 含密码（既有模式）

## 测试

`go test ./src/domain/ ./src/infrastructure/ ./src/` 全绿（src 107s）
