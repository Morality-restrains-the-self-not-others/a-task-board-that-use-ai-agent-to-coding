# NFR 澄清：daydaymoney.yaml 元信息全链路

**日期**: 2026-07-18  
**默认等级**: L2 Standard

| 类别 | 等级 | 说明 |
|------|------|------|
| 安全 | L2 | Resolve 租户隔离；YAML 无密钥；禁止硬编码归属 id |
| 性能 | L2 | Resolve 在租户项目规模内全表扫 tags（JSON）可接受；后续可加索引 OPT |
| 可用性 | L2 | YAML/meta 缺失时降级：插件保留上次选择；Grafana 回退 regex |
| 可观测 | L3 | 本迭代直接扩展日志字段与 Grafana 匹配 |
| 可维护 | L2 | 共享解析库；CI 防漂移 |
| 一致性 | L2 | tags 规范化与现有 `normalizeProjectTags` 对齐 |

质量场景（喂给 DDD）：

1. 同一 service_id 映射 3 个项目 / 2 个 WS → resolve 返回全部  
2. meta 缺失 → Chrome 不覆盖用户选择  
3. 日志无 daydaymoney 字段 → Grafana regex 仍可用  
