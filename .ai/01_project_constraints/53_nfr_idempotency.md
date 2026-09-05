# NFR 幂等性审视（元规则）

## 基本信息

- 版本：1.1.0
- 创建日期：2026-08-18
- 维护者：Trae AI 团队
- 约束索引：`00_project_constraints.md` 第 48 条
- Cursor：`.cursor/rules/nfr-idempotency.mdc`（alwaysApply）
- 技能落地：`.claude/skills/5-nfr/SKILL.md` + `references/idempotency.md`

## 背景（为何是元规则）

Kafka at-least-once、HTTP 超时重试、Webhook 重投、用户双击都会把同一笔业务送进系统两次。若 NFR 阶段不定义**业务重复边界**和**与之同粒度的幂等键**，DDD 会把错误去重边界写进聚合与消费者——过粗则吞掉后续合法操作（本仓库失败经验：`company_id`/`user_id` 误作事件幂等键），过细则重复扣费/重复建资源。

NFR 澄清（`/5-nfr`）是进入 DDD 前最后一次系统性质检。必须在此步强制审视副作用路径的幂等性。

## 与既有规则的关系

| 规则 | 管什么 | 与本条关系 |
|---|---|---|
| API 幂等性（`.ai/03_technical_implementation/02_api_specifications.md`） | 接口设计时考虑防重复 | 本条把该要求前移到 `/5-nfr` **硬门禁**，并覆盖事件/Webhook/timer |
| 事件消费者幂等消费（元规则 49 / ADR-0015） | 消费实现必须走共享幂等 runner | 本条管设计文档里的键；49 管落地与 CI |
| 前端按钮防重放（元规则 52 / ADR-0020） | 点击入口同步锁 + `Idempotency-Key` | 「双击」触发源的前端落地；本条仍要求服务端兑现该键 |
| 死信 Topic（`.cursor/rules/dead-letter-topic.mdc`） | 消费失败投 DLT | 本条管成功路径的重复投递；二者互补 |
| `/5-nfr` 技能 | NFR 等级与质量场景 | 本条是该技能的**硬门禁**；技能附录为执行细则 |
| `/6-ddd` | 聚合与仓储建模 | 消费本条产出的「幂等性审视」与领域模型影响 |

## 核心规则

### 1. 执行时机

凡执行 `/5-nfr`（或等价 NFR 澄清、产出 `*-nfr-clarification.md`）时，**必须**在筛选 NFR 类别之前完成副作用路径的幂等性审视（可与路径分片键审视并行，但不得省略）。

### 2. 只读 / 无副作用路径

对每条确认无副作用的路径：

- 可标幂等性 L0，**须写理由**
- 禁止空话跳过；禁止把带隐式写（发事件、打点写库）的路径标成只读

### 3. 副作用路径

对每条有副作用的路径：

- **必须**写明：重复触发源、业务重复边界、幂等键、重放语义、键持久化方式
- **必须**判定幂等键是否与业务重复边界同粒度
- **禁止**默认使用 `company_id` / `tenant_id` / `user_id` 作为事件消费幂等键（过粗，会吞掉后续独立意图）
- 不适配时：记录正确键与纠正动作，并写入领域模型影响表
- 资金、配额、云资源、支付/KYC 路径默认幂等性 **≥ L3**（除非书面证明无资金/资源副作用）
- 若重复触发源含用户点击/双击：NFR 表须写明前端防重放（同步锁 + 键如何产生），并引用元规则 52 / `.ai/01_project_constraints/57_frontend_button_anti_replay.md`

### 4. 文档与门禁

- NFR 澄清文档**必须**含「幂等性审视」表（增量内每条相关路径一行）
- 存在副作用路径且未书面证明 L0 时，**不得**把「数据一致性」「容错机制」标为跳过
- Hard Gate 未通过 → **禁止**进入 `/6-ddd`
- 细则与决策树：`.claude/skills/5-nfr/references/idempotency.md`

## 验收

```bash
# 技能含硬门禁步骤与附录
test -f .claude/skills/5-nfr/references/idempotency.md
rg -n '幂等性强制审视|幂等性审视' .claude/skills/5-nfr/SKILL.md

# 门禁脚本自测
python3 db/scripts/ci/test_check_nfr_idempotency_table.py

# 产出的 NFR 文档应含审视表（人工/评审；新文档 --strict）
# python3 db/scripts/ci/check_nfr_idempotency_table.py --strict --files 'docs/superpowers/plans/<file>.md'
```

## 变更日志

- 2026-08-19：版本 1.1.0 - 与元规则 52 交叉：用户双击须规划前端防重放
- 2026-08-18：版本 1.0.0 - 初版；由「/5-nfr 须考虑幂等性」目标固化为元规则
