# NFR：管理端租户下拉联系方式

总体档位 **L2**；本增量只读查询，不改资金路径。

## 路径分片键强制审视

| 路径 | 分片 ID | 可伸缩性 | 判定 |
|------|---------|----------|------|
| GET `/api/system-admin/accounts/admin/tenant-options/?search=` | 无 | **L0** | 平台管理全局目录，LIMIT 80；升级触发：租户数 >1 万且需分页/按区域分片时再加 cursor + 可选 region |
| 前端 `/system-admin/grant-points` | 无 | **L0** | 系统管理壳，非租户工作台 |
| POST `/api/internal/users/batch/details/` | user_id 列表 | **L1** | 内部批量，max 200；user_id 适合作 auth 库主键查找，非租户分片键。管理端一次最多 80 个 creator_id |
| GET `/api/internal/users/?q=` | 无 | **L0** | 已有内部搜索，LIMIT 50；与本增量共用上限 |

## 幂等性强制审视

| 路径 | 副作用 | 幂等 | 理由 |
|------|--------|------|------|
| GET tenant-options | 无 | **L0** | 纯查询；重复请求返回当前公司+联系方式快照 |
| POST batch/details | 无 | **L0** | 纯查询 |
| GET internal users ?q= | 无 | **L0** | 纯查询 |

无写接口、无 Kafka、无 timer。
