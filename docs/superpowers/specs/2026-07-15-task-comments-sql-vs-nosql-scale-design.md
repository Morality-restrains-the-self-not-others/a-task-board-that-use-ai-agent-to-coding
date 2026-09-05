# 设计：任务评论存储 — SQL 留存 vs 迁 NoSQL（规模/性能驱动）

- **日期**: 2026-07-15 14:40
- **作者**: claude
- **状态**: approved / implemented（P0–P2，2026-07-15）
- **迭代名**: `task-comments-sql-vs-nosql-scale`
- **相关页面**: `https://www.daydaymoney.com/tenant/.../task-detail/{task_id}/`（统一评论区：人类 / AI / 容器 Agent）
- **驱动因素（用户确认）**: **规模/性能** — 单任务评论量很大、列表/流式更新变慢
- **追问（本次）**: 若准备上线，当前模式有何风险
- **python_api_approval**: n/a（零新增 Python 接口；若落地优化均在既有 Go 服务）
- **架构变更**: **本期不引入 NoSQL / 不新增微服务**；若后续采纳「流式 chunk 表」或「SQLite→Postgres」再开独立架构 target（见 §7）

---

## 1. 问题 / 意图

评估：任务详情页评论（含 AI 与非 AI）是否应从 **SQL（当前为嵌入式 SQLite）** 转为 **NoSQL（如 Mongo）** 以改善大规模下列表与流式更新性能。

补充评估：在**不迁 NoSQL、维持当前双 Go/SQLite 模式**前提下，正式上线的风险面。

### 已确认决策（用户）

| 项 | 决策 |
|----|------|
| 驱动因素 | 规模/性能（非结构灵活、非运维统一） |

---

## 2. 现状（架构与实现）

### 2.1 存储真源（current）

| 类型 | Owner | DB | 表 |
|------|-------|----|----|
| 人类评论 | `taskTaskService` | SQLite `data/task_task.db` | `comments` |
| AI 指令评论 | `taskAIComment` | SQLite `db/task-ai-comment/task_ai_comment.db` | `ai_task_comments` |
| 容器 Agent | `taskAIComment` | 同上 | `container_agent_comments` |

历史路径：Django/PostgreSQL → Go/SQLite（v8/v9 cutover）。**不是**「MySQL 中心库」。

### 2.2 访问模式

| 路径 | 行为 | 规模敏感点 |
|------|------|------------|
| 列表 | `WHERE task_id=? ORDER BY created_at`，**无 cursor/LIMIT**；序列化含完整 `assistant_response` | 评论条数 × 单条回复长度 → 响应体膨胀 |
| AI instruct 流式 | 内存合并 + SSE 批推（`sse_batch.go`）；**结束时**再写 `assistant_response` | DB 写尚可；列表仍拉全量正文 |
| Agent `/stream` | **每 chunk**：读出全文 → `prev+chunk` → **整列 UPDATE** `assistant_response` | **O(流长度²) 写放大**；SQLite 写锁争用 |
| ContextPack | 聚合线程进 JSON（设计上可截断） | 大任务冷启动 I/O |

本地样例库量级极小（个位数行）；线上「变慢」更可能来自：**长 `assistant_response` + 全量列表 + Agent 流式整文重写**，而非「SQL 引擎选型错误」。

### 2.3 架构上下文

- **Current**: v27；评论统一区相关 **v26 仍为 target**
- 原则：单库/单表单 owner（`.ai/01_project_constraints/19_single_service_data_ownership.md`）；Go 服务优先嵌入式 SQLite 分库

---

## 3. 根因判定（相对「该不该上 NoSQL」）

| 症状 | 更可能根因 | NoSQL 能否天然消除 |
|------|------------|-------------------|
| 列表变慢 | 无分页；payload 含全文；前端合并多源 | **否** — 文档库同样要分页/投影，否则更重 |
| 流式变慢 | Agent 路径每 chunk 整文 `UPDATE`；写放大 + 锁 | **否** — 若仍 `$set` 整文档，Mongo 同样 O(n²)；需 **append 模型**，与是否 NoSQL 无关 |
| 单任务「很多条」 | 索引已按 `task_id`；缺分页与摘要字段 | **否** — B-Tree / 索引在 SQL 已够用 |
| 多实例 HA / 文件锁 | 嵌入式 SQLite 单写者局限 | **部分** — 但下一跳更合理是 **共享 SQL（Postgres）**，而非为此引入文档库 |

**结论（推荐）**：**不宜为规模/性能把评论迁 NoSQL。**  
应先改 **访问模式与写路径**；若日后挤出单机 SQLite，优先 **同模型迁 Postgres**，而不是 Mongo。

---

## 4. 方案对比

| 方案 | 描述 | 优点 | 缺点 | 对「列表/流式变慢」 |
|------|------|------|------|-------------------|
| **A. 维持 SQLite + 访问模式优化（推荐）** | 分页、列表投影、流式批写/append | 改动面小；贴合现有 owner；无新中间件 | 不解决多副本共享写 | **直接命中根因** |
| B. 迁 Mongo/文档库 | 评论作 document；可能 `$push` chunks | 灵活嵌套；水平扩展叙事 | 新运维面；与分库 SQLite 模式冲突；迁移成本高；列表/投影仍要自己做 | **不自动更快**；若模型做错仍慢 |
| C. 迁共享 Postgres（仍 SQL） | 同 schema，换引擎 | 多实例、备份、监控成熟；迁移路径清晰 | 运维比 SQLite 重 | 列表仍要分页；流式仍要改写模型 |
| D. 混合：元数据 SQL + 大正文对象存储 | 超阈值 transcript 外置 | 控库体积；列表轻 | 双写一致性；实现复杂 | 适合「单条超长」；非首选第一步 |

### 选型结论

| 项 | 结论 |
|----|------|
| **是否迁 NoSQL** | **否** |
| **是否立刻迁 Postgres** | **否**（无多实例写冲突/备份硬需求前不引入） |
| **本期方向** | **方案 A**；阈值触发时再评估 C；单条 transcript 极端大时可选 D |

---

## 5. 推荐落地（方案 A）— 分阶段

> 以下为实现建议；**审批本设计 = 批准「不迁 NoSQL」+ 批准按阶段做 SQL 侧优化方向**。具体切片由 `/3-value-stream` / `/7-plans` 拆解。

### Phase 0 — 可观测与门槛（先证据后改）

- 列表 API：记录 `count`、`payload_bytes`、`p95_ms`（按 task）
- Agent stream：记录 `chunk_count`、`assistant_len`、每 chunk DB 耗时
- 门槛草案（可调）：单任务列表 `>200` 条或 `assistant_response` 合计 `>2MB` 或 stream 单评论 `>500KB` → 进入 Phase 1

### Phase 1 — 列表：分页 + 投影（人类 / AI / Agent 三 API 对齐）

| 改动 | 说明 |
|------|------|
| Cursor 分页 | `?cursor=&limit=`（按 `created_at,id`）；默认 limit 如 50 |
| 列表投影 | 默认不返回完整 `assistant_response`；返回 `assistant_preview`（如前 2KB）+ `assistant_len`；详情/展开再拉全文 |
| 前端时间线 | `buildDisplayComments` 改为增量合并 + 懒加载长回复 |

**预期**：列表延迟与带宽随「可见窗口」线性，而非随全历史。

### Phase 2 — 流式写：消除 O(n²) 整文 UPDATE

优先顺序（由易到难）：

1. **批写 debounce**（对齐已有 SSE `aiChunkBatchWindow`/`MaxLen`）：内存合并，每 80ms 或 2KB 刷一次 `assistant_response`；`complete` 再最终落库  
2. **Append-only chunk 表**（若批写仍不够）：如 `container_agent_comment_chunks(comment_id, seq, body)`；列表/完成时拼装或只存最终快照  
3. **禁止**读-改-写整列于每个 HTTP chunk（当前 `appendContainerAgentAssistantResponse`）

AI instruct 路径已是「SSE 实时 + 结束写库」，可保持；重点修 **container agent `/stream`**。

### Phase 3 — 仅当门槛再次触发

| 触发 | 动作 |
|------|------|
| 单机 SQLite 写锁/多副本需求 | 同 schema **Postgres**（方案 C），非 Mongo |
| 单条 transcript 持续 > 数 MB | 对象存储外置（方案 D）+ DB 存指针 |

---

## 6. 明确不采纳

- 为「看起来像文档」而把三表并成 Mongo collection
- 在未改访问模式前做存储引擎迁移（成本高、根因未解）
- 把 Agent 流式正文塞回 `taskTaskService.comments`（与 v26 边界冲突）

---

## 7. 🏛️ 架构变更影响

| 项 | 内容 |
|----|------|
| **本期** | **不新增** `docs/architecture/vN-*` target（无组件/数据流/基础设施增删） |
| **决策记录** | 本设计文档即为「评论存储保持 SQL/SQLite；拒 NoSQL」的书面结论 |
| **若落地 Phase 2 chunk 表** | 另开迭代：🟡 `taskAIComment` 数据对象；更新 `db/table_ownership.yaml`；再写 architecture target（三类伴生） |
| **若落地 Phase 3 Postgres** | 另开迭代：Technology 层 DB 节点变更；须完整 `.puml` + `.archimate` + `.mermaid.md` |

并行 target 提醒：v26（统一评论区）、v28（嵌套 Git）等仍积压；本决策不增加新 target 文件，避免堆叠。

---

## 8. Domain Concept Inventory（轻量）

| 概念 | 说明 |
|------|------|
| Bounded Context | 任务协作（人类评论）/ AI 评论编排（AI + Agent） |
| Entities | `Comment`、`AITaskComment`、`ContainerAgentComment` |
| Candidate Aggregate | 按 `task_id` 的 Comment Feed（读侧合成，非单库聚合） |
| 可选新 VO | `CommentCursor`、`AssistantPreview`、`StreamChunk` |

### 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|--------|--------------|---------|
| （本期决策）拒迁 NoSQL / 仅读路径优化 | — | — | — | **纯查询与存储策略**；无新业务事实 |
| 人类发表评论（既有） | 既有 `TASK_COMMENT_*` / mention 事件 | taskTaskService | taskEvents / Agent 编排 | 不改语义 |
| AI 回复完成（既有） | `AI_ASSISTANT_REPLY_COMPLETED` | taskAIComment | 既有消费者 | 不改语义 |
| Agent 流式 chunk 落库（若 Phase 2 改写路径） | **无新业务事件**（或保持现有 SSE 侧信道） | taskAIComment | 前端 SSE | 内部持久化优化；若引入「回复完成」已有 complete 路径 |

---

## 9. 价值流影响（输入给 step 3）

- **受影响 stream**: `task-comments-api`（`conf/value-stream.yaml`）；task-detail 评论时间线相关步骤
- **字段**: 可能新增投影字段（`assistant_preview` 等）——落地时以三元组登记；**无** NoSQL collection
- **测试**: `tests/test_task_comments_api.py`、`tests/test_ai_task_comment.py`；需补分页与 stream 批写回归
- **新 stream**: 不需要；属既有任务协作流的性能硬化

完整切片交给 `/3-value-stream-价值流`。

---

## 10. 成功标准（若批准方案 A）

| # | 标准 | 验证 |
|---|------|------|
| S1 | 书面结论：**不迁 NoSQL** 作为规模驱动的默认选型 | 本设计 approved |
| S2 | Phase 1 后：单任务 1k 条评论列表 p95 与 payload 受 limit 约束 | 压测/集成 |
| S3 | Phase 2 后：Agent 流式 DB 写次数 ≪ chunk 次数（批写或 append） | 指标 + 单测 |
| S4 | 零新增 Python HTTP；owner 不变 | 路由 / ownership CI |
| S5 | 未达 Phase 3 门槛前不引入 Mongo/Postgres | 架构无对应 target |

---

## 11. 风险与待验证

| 风险 | 缓解 |
|------|------|
| 线上真实分布未知（本地库极小） | Phase 0 指标；用生产只读统计复核门槛 |
| 分页改变前端假设（一次拉全量） | 与 v26 统一评论区前端一并改；向后兼容 `limit` 缺省行为需约定 |
| 批写崩溃丢尾块 | `complete` 强制 flush；崩溃恢复可接受「尾部缺失」或 complete 重放 |

---

## 12. 审批请求

请确认：

1. **批准「不迁 NoSQL」**，规模问题走 **方案 A（SQLite + 访问模式优化）**  
2. 是否立即进入实现（Phase 0→1），或仅归档本决策、待有生产指标再开 Phase 1  
3. **上线前**：是否将 §13 中 **P0（详情评论空洞）** 列为必须先修的阻断项  

（本设计**不**触发 Python 接口专项审批；**不**写入 architecture target，直至某 Phase 改变表/基础设施。）

---

## 13. 上线风险：当前评论模式（不迁 NoSQL）

> 范围默认指 **评论子系统 + 任务详情评论时间线**。整站其它风险见既有 `2026-07-15-high-traffic-dev-hardening-design.md` PRE-PROD。

### 13.1 风险总览

| 级别 | ID | 风险 | 为何上线会痛 | 与 NoSQL 关系 |
|------|-----|------|--------------|---------------|
| **P0 阻断** | R1 | 详情聚合空洞 | 公网 todos 走 Go，`comments`/`ai_comments` **恒为空**；前端只补拉 Agent | **无关** — 产品正确性 |
| **P0 阻断** | R2 | 刷新丢人类/AI 评论展示 | `fetchTaskDetail` 用空洞 payload 覆盖 `localTask` | 无关 |
| **P1 高** | R3 | 无分页全量列表 | 热门任务一次拉全文 → 延迟/带宽/浏览器卡顿 | 迁 NoSQL **不解**；要分页 |
| **P1 高** | R4 | Agent 流式整文 UPDATE | 长回复 O(n²) 写 + SQLite 写锁 | 迁 NoSQL **不解**；要批写/append |
| **P1 高** | R5 | 嵌入式 SQLite 单写者 | 多副本/滚动发布时文件锁或只读副本漂移 | 下一跳应是 **Postgres**，非 Mongo |
| **P1 高** | R6 | 备份/灾难/跨机一致性 | 两库文件（`task_task.db` + `task_ai_comment.db`）无统一备份契约 | 引擎无关；运维债 |
| **P2 中** | R7 | 双服务时间线合成 | 时钟/排序/部分失败 → 顺序错乱或半边空白 | 无关 |
| **P2 中** | R8 | ContextPack 随线程膨胀 | `@镜像` 冷启动变慢或截断丢上下文 | 无关 |
| **P2 中** | R9 | ownership 登记漂移 | `ai_task_comments` 同时出现在 `task-task` 与 `task-ai-comment` 表清单；task 库可能残留空壳表 | 无关 |
| **P2 中** | R10 | SSE 与落库双路径 | 流式靠 SSE，落库滞后/失败时刷新后内容回退 | 无关 |
| **P3 低** | R11 | 限流/连接护栏已部分落地 | 超限 503；需生产调参 | 见 high-traffic 设计 |

### 13.2 P0 详解：详情评论空洞（上线阻断）

**证据链**

1. 网关 `task-task-service` 接管 `/api/tenant/*/workspace/*/todos/*` → **Go**，不再经 Django。
2. `taskTaskService` `taskToJSON` **写死**：
   - `"comments": []`
   - `"ai_comments": []`
3. 前端 `fetchTaskDetail` 仅在缺失时补拉 `container_agent_comments`；**不**拉 `/comments/` 与 `/ai-comments/`。
4. `displayComments = buildDisplayComments(localTask)` → 人类/AI 历史在刷新后消失。
5. Django `todo_functions.get_todo_detail` 仍有「调 Go 列表再填入」的残留逻辑，但 **公网路径已被网关绕过**，对线上详情无效。

**上线表现**

- 打开任务详情：统一评论区可能 **只见 Agent 节点**，或几乎为空。
- 用户发评论后 `fetchTaskDetail()`：刚写入的人类/AI 评论也可能被空数组冲掉（直至某条路径本地乐观更新 — 当前主路径是整页重拉）。

**上线前必须处理（三选一，推荐 A）**

| 方案 | 做法 | 备注 |
|------|------|------|
| **A. Go 详情聚合（推荐）** | `GET todos/{id}` 内填人类评论；AI/Agent 经 HTTP 调 taskAIComment 或约定 BFF | 与「单 owner」一致：Task 只读自己的 `comments`，AI 侧转发 |
| **B. 前端三源拉取** | 对齐 Agent：并行 GET comments + ai-comments + container-agent | 快；多一次 RTT；须处理失败降级 |
| C. 恢复 Django 聚合 | 网关把 todos 详情打回 Django | **不推荐**：扩大 Python 公网面，与 Go 真源方向冲突 |

> 此缺口 **不是**「该不该上 NoSQL」的理由；修好后仍建议保持 SQLite + 方案 A 性能项。

### 13.3 P1：规模与可用性（与先前规模驱动一致）

| 风险 | 上线门槛建议 | 缓解（仍属方案 A） |
|------|--------------|-------------------|
| R3 全量列表 | 单任务 >200 条或 payload >2MB 不可接受 | Phase 1 分页 + preview |
| R4 流式写放大 | 单次 Agent 回复 >500KB 时 DB/锁抖动 | Phase 2 批写/chunk 表 |
| R5/R6 SQLite 运维 | 多实例写同一文件、无定期备份 | 单实例粘滞 + 文件备份；规模后再 Postgres（high-traffic P1） |

已有护栏（降低但不消除）：WAL、`busy_timeout=30s`、`MaxOpenConns≤4`、AI instruct SSE 批推、网关全局限流、SSE 连接上限（见 high-traffic NOW）。

### 13.4 P2：正确性与一致性

| 风险 | 说明 | 缓解 |
|------|------|------|
| R7 合成 | 三源独立失败策略不一致（Agent 失败→`[]` 静默） | 统一错误态；部分失败标 banner |
| R8 ContextPack | 大线程截断策略已有设计上限，需生产验证 | 监控 pack 大小；截断告警 |
| R9 ownership | 双清单/空壳表易误导迁库与 CI | 清理 `task-task.db` 空壳 `ai_task_comments`；YAML 只保留真源 |
| R10 SSE vs DB | 用户依赖刷新看到最终稿 | complete 路径强一致；失败可重试 PATCH |

### 13.5 「当前模式」上线结论

| 问题 | 结论 |
|------|------|
| 能否带着 **现状** 直接上线？ | **不能放心上线**：存在 **P0 详情评论空洞**（功能正确性），与引擎选型无关 |
| 修好 P0 后，SQLite 双库能否上线？ | **可以作为首发形态**，须接受单写者/备份约束，并带上 Phase 0 指标；热门任务前落地 Phase 1 |
| 是否必须先上 NoSQL？ | **否**；上线风险清单里 **没有任何一项以迁 NoSQL 为最优解** |
| 与整站 PRE-PROD | 评论侧 P0/P1 应并入上线 checklist；Postgres 等仍见 high-traffic P1–P6 |

### 13.6 建议的上线 checklist（评论相关）

- [x] **P0**：详情可见人类 + AI + Agent 三类评论（刷新后仍在）
- [x] **P0**：发评/AI 回复后重拉不会清空时间线
- [x] **P1**：列表有 limit/cursor 或生产确认量级远低于门槛
- [x] **P1**：Agent 长流式有批写或等价护栏；SQLite 备份/恢复演练通过（runbook：`docs/runbooks/sqlite-comment-dbs-backup.md`）
- [x] **P2**：ownership YAML 与空壳表清理；ContextPack/SSE 失败可观测

---
