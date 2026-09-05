# NFR 路径分片键与可伸缩性审视（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-12
- 维护者：Trae AI 团队
- 约束索引：`00_project_constraints.md` 第 43 条
- Cursor：`.cursor/rules/nfr-path-shard-id-scalability.mdc`（alwaysApply）
- 技能落地：`.claude/skills/5-nfr/SKILL.md` + `references/path-shard-id-scalability.md`

## 背景（为何是元规则）

水平扩展依赖稳定的分片/分区键。若 API、前端路由或消息路由键**不携带可分片 ID**，后续按租户或实体分库分表时只能全表扫描或二次查找；若路径**携带了错误的分片 ID**（低基数、与访问模式不对齐、破坏租户隔离），扩容成本更高。

NFR 澄清（`/5-nfr`）是进入 DDD 前最后一次系统性质检。必须在此步强制审视路径与分片键，避免领域模型把错误边界固化进聚合与仓储。

## 与既有规则的关系

| 规则 | 管什么 | 与本条关系 |
|---|---|---|
| 冷热分离与分库分表（`.cursor/rules/database-cold-hot-separation.mdc`） | Schema 伸缩要素、分片策略 | 本条管**路径/契约是否暴露正确伸缩键**；表级分片仍遵守该元规则 |
| `/5-nfr` 技能 | NFR 等级与质量场景 | 本条是该技能的**硬门禁**；技能附录为执行细则 |
| `/6-ddd` | 聚合与仓储建模 | 消费本条产出的「路径分片键审视」与领域模型影响 |

## 核心规则

### 1. 执行时机

凡执行 `/5-nfr`（或等价 NFR 澄清、产出 `*-nfr-clarification.md`）时，**必须**在筛选 NFR 类别之前完成路径分片键审视。

### 2. 无分片 ID 的路径

对每条未携带可分片 ID 的路径（HTTP/API、前端路由、Kafka/消息路由键、作为扩展边界的缓存/对象键等）：

- **必须**评估是否存在可伸缩性需求（支撑等级 L0–L4）
- 若 ≥ L1：须给出补救动作（补路径参数、强制带 shard key 的查询、或接受单分片并写明容量上限）
- 若 L0：须写明理由与**升级触发条件**；禁止空话跳过

### 3. 有分片 ID 的路径

对每条已携带分片 ID 的路径：

- **必须**评估该 ID 是否为合适的分片键（基数与分布、与读写模式对齐、租户隔离、稳定性、热点风险、与库表分片策略一致）
- 不适配时：记录正确分片键、路径/事件键纠正动作，并写入领域模型影响表

### 4. 文档与门禁

- NFR 澄清文档**必须**含「路径分片键审视」表（增量内每条相关路径一行）
- Hard Gate 未通过 → **禁止**进入 `/6-ddd`
- 细则与决策树：`.claude/skills/5-nfr/references/path-shard-id-scalability.md`

## 验收

```bash
# 技能含硬门禁步骤与附录
test -f .claude/skills/5-nfr/references/path-shard-id-scalability.md
rg -n '路径分片键强制审视|路径分片键审视' .claude/skills/5-nfr/SKILL.md

# 产出的 NFR 文档应含审视表（人工/评审检查）
# rg -n '## 路径分片键审视' docs/superpowers/plans/*-nfr-clarification.md
```

## 变更日志

- 2026-08-12：版本 1.0.0 - 初版；由「/5-nfr 须审视路径可分片 ID」目标固化为元规则
