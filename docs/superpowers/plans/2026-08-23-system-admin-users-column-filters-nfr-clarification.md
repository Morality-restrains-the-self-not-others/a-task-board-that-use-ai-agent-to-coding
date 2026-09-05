# NFR 澄清：系统管理用户表列过滤器

支撑等级：L2（标准）。鉴权沿用既有超管门禁。

## 路径分片键审视

| 路径 | 是否携带分片 ID | 该 ID 是否合适 | 可伸缩性 | 动作 |
|------|-----------------|----------------|----------|------|
| GET `/api/system-admin/users/?email=&role=…` | 否 | n/a | L0 | 超管全局用户目录，QPS 低；本库字段走 WHERE + 分页。升级触发：列表 p95>300ms 或用户数>5 万且 enrichment 过滤成为默认路径 |
| FE `/system-admin/users/` | 否 | n/a | L0 | 平台管理页，非租户分片。升级触发：需按租户拆分管理控制台 |

Hard Gate：通过（L0 均有理由与升级触发）。

## 幂等性审视

| 路径 | 副作用 | 等级 | 理由 |
|------|--------|------|------|
| GET `/api/system-admin/users/` 列过滤 | 无 | L0 | 纯查询；重复 GET 不写库、不发事件 |
| 表头过滤输入/下拉 | 无（只读 GET） | L0 | Anti-Replay-OK: 只读列表过滤；防抖减少连打请求，不需要 Idempotency-Key |
| 重置按钮 | 无 | L0 | 只读重新拉取 |

Hard Gate：通过。

## 其他

- 安全：LIKE 去掉 `%`/`_`，避免通配符注入；日志只记过滤键名。
- 性能：本库过滤走 SQL；跨服务字段候选 ID 上限 5000。
- 可观测：`system_admin_users_list` info 含 `filter_keys`、`total`、`duration_ms`。
- 无新 MQ。
