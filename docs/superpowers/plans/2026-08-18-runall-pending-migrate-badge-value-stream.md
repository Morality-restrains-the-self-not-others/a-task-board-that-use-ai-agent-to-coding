# 价值流 — 9999 未 migrate 标注

- **日期**: 2026-08-18
- **设计**: `docs/superpowers/specs/2026-08-18-runall-pending-migrate-badge-design.md`

Mapping the approved design into a value stream.

## 既有相关流

- 编排 · 精准编译重启热加载 YAML / 全部重新编译消费精准登记：同页 Dev 操作，但对象是服务登记而非 schema。
- `conf/value-stream.yaml` 为 Django pytest 流，**不**新增条目（本增量可执行测试是 Go）。

## 增量（唯一，立即交付）

```
打开 9999 → GET migrate-status → 比对 dataMigrate vs data_migrate_log
  → 有 missing：按钮琥珀 + 「N 未 migrate」+ tooltip
  → 点初始化 → SSE 完成 → 再 GET → 徽章清除或更新
```

角色：系统管理员（runAll 运维页）。

## 测试点（写入 `docs/flows/value-stream-test-integration.wsd`）

| ID | 步骤 | 测试 |
|----|------|------|
| TP-DIFF-MISS | diff | missing 计 pending |
| TP-DIFF-STALE | diff | stale 不计 pending |
| TP-UNREACH | reader error | unreachable |
| TP-GET | HTTP | JSON 契约 |
| TP-UI | 装配页 | 徽章 id + 非 2s refresh |

无 MQ 环节。
