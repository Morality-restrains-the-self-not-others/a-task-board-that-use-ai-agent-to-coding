# NFR 澄清：闲置复用启动保护 + 孤儿交叉校验

- **日期**: 2026-07-22
- **默认等级**: L2 Standard（goal-mode）

| 类别 | 等级 | 说明 |
|------|------|------|
| 正确性 | L3 | 误复用/误删云实例属高成本；门闩必须单测覆盖 |
| 性能 | L2 | orphan 预加载 workspace instance_id 集合，O(n) CSC 行，可接受 |
| 安全 | L2 | 无新公网面；删除仍限本 workspace |
| 可观测性 | L2 | `idle_reuse_skip_boot_guard` / `orphan_reconcile_skip_owned` 日志 |
| 可用性 | L2 | boot 中 fork→冷启动，延迟可接受 |

### 质量场景 → DDD

- QS1: 启动保护拒绝 reuse → TrueIdleMachine 不变式
- QS2: cross-CSC 持有跳过删除 → CrossCSCOwnership 不变式
