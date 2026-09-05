# DDD 领域建模: Fix Port Occupancy Race

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-22-port-occupancy-race-analysis.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-22-port-occupancy-race-value-stream.md`
> - NFR 澄清文档: `docs/superpowers/plans/2026-06-22-port-occupancy-race-nfr-clarification.md`
>
> 输出使用者: `/6-plans-实施计划`

## 跳过声明

纯基础设施修复，不涉及领域模型变更：

1. **无新业务概念** — `startAndCheck` 端口等待循环和 `SO_REUSEADDR` 都是基础设施层变更
2. **现有模型不变** — `ManagedService`、`ServiceLifecycle` 实体和行为保持不变
3. **NFR 确认** — NFR 文档的「领域模型影响」表格标注为「无」

## 现有限界上下文参考

```
平台与本地开发 (platform-dev)
├── 聚合根: ManagedService
│   ├── 状态: pending → starting → healthy → stopping → stopped
│   ├── 行为: start(), stop(), healthCheck()
│   └── 值对象: PortProbe (端口检测)
```

修复仅在基础设施层改变 `start()` 行为的端口等待逻辑，领域模型接口契约不变。
