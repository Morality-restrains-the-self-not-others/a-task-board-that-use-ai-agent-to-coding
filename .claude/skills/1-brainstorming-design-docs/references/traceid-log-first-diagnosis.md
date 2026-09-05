# data-traceId → 日志优先排障（横切规范）

> 被排障类技能引用的短规范。完整 Loki 查询、空结果诊断、Grafana 链接见父目录 [`SKILL.md`](../SKILL.md)「TraceId 驱动的 Grafana 日志分析」。

## 强制优先级（硬门禁）

当用户报告、页面快照、Playwright/DevTools 摘录或错误 UI 中出现 **`data-traceId`**（及大小写变体 `data-traceid` 等）或可提取的 **traceId** 时：

| 顺序 | 动作 | 禁止 |
|------|------|------|
| 1 | **提取** traceId 值（标注名大小写不敏感；**值**保持原样） | 漏检 `traceid` / `TraceId` / `trace_id` 等变体 |
| 2 | **优先** 用 Loki/Grafana 检索该 trace 的全量关联日志（全 job：`{job=~".+"}`，禁止锁单一服务） | 未查日志就凭错误文案猜根因 |
| 3 | **按时间线重建** 请求路径：涉及服务 → 关键事件 → 首个 ERROR/WARN → 上下游调用 | 只读一条 ERROR、忽略前后 INFO |
| 4 | 基于日志路径再形成可证伪假设，再改代码 / 写设计 | 在日志检索完成（或已确认 Loki 不可达并记录）前直接改代码 |

**一句话**：有 `data-traceId` = 已有全链路钥匙；**先日志、后源码**。

## 完成判据（排障语境）

在提出根因或修复前，须能简要说明：

1. 提取到的 traceId
2. Loki/替代途径是否命中、时间范围
3. 时间线：哪些服务参与、何处首次失败
4. 根因假设如何被日志证据支持（或「日志缺失」及已做诊断步骤）

## 适用入口

| 入口 | 要求 |
|------|------|
| `/1-brainstorming-design-docs` | 主流程**之前**强制执行完整 TraceId 节 |
| `pua` / diagnose / `ce-debug` | 硬 bug / 卡壳排障时叠加本规范 |
| `webapp-testing` | 捕获带 `data-traceId` 的错误节点后回传 ID 并触发检索 |
| `logging-audit` | 实现侧保证 `trace_id` 可检索；排障侧指向本规范 |
| 元规则 24 | DOM 契约 + Agent 读取；见 `.ai/01_project_constraints/24_frontend_error_data_trace_id.md` |
| `.ai/09_failure_experience/` | 运行时失败流程：有 traceId 时先日志再查经验库 |

## 与「先复现」的关系

Matt Pocock **diagnose** 要求确定性反馈环。当已有 `data-traceId` 时：

- **日志时间线** 即为首选反馈环（比盲目改代码再猜更快）
- 仍应在需要时补失败测试 / curl / Playwright；二者不互斥
- **禁止**用「先读源码猜一下」替代日志检索
