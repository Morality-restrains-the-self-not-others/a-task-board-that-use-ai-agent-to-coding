# DDD 领域建模: gitOauth DisallowedHost 修复 — 跳过声明

> 输入:
> - 设计文档: `docs/designs/disallowed-host-gateway-fix.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-26-disallowed-host-gateway-fix-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-26-disallowed-host-gateway-fix-nfr-clarification.md`

## 跳过理由

本次变更为纯基础设施配置修正，满足 DDD 技能指引的跳过条件：

> **跳过条件：** 配置变更、或价值流增量不涉及新的业务概念。

具体判断：

1. **无新增实体或值对象** — 不引入新的业务对象
2. **无新增聚合或一致性边界** — ALLOWED_HOSTS 是 Django 安全配置，不属于领域模型
3. **无新增仓储接口** — 不操作持久化数据
4. **无新增领域事件** — 不产生跨上下文通信
5. **无新增领域服务** — `load_gateway_public_host()` 是配置读取工具函数，非业务逻辑
6. **不改变现有领域模型** — 现有 OAuth 流程的领域模型（Provider、AccessToken、BindState）无变化

## 微小变更说明

变更仅涉及两个基础设施层文件：

| 文件 | 性质 | 领域影响 |
|------|------|---------|
| `gitOauth/config/port_config.py` | 配置加载器 | 零 — 读取 YAML 的工具函数 |
| `gitOauth/config/settings.py` | Django settings | 零 — 白名单扩展 |

这些文件属于基础设施/配置层，不在 `domain/` 目录范围内，不涉及 DDD 建模。
