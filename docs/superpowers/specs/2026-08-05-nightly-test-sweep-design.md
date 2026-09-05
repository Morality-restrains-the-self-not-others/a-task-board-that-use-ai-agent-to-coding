# 夜间随机单测巡检与自愈机制设计（nightly-test-sweep）

- **迭代**: nightly-test-sweep
- **日期**: 2026-08-05 23:47
- **作者**: claude
- **架构版本**: v65 🎯 target（基于 v64 target）
- **状态**: 🎯 已设计，待审批

---

## 1. 背景与目标

### 1.1 问题

现有随机单测门禁（规则 28，`scripts/hooks/templates/pre-commit.*`）只在 **git commit 时**触发：
- 提交频率决定抽测机会 — 低频仓库（如 `shareLib`、`taskCredentialService`、基建仓）的遗留单测失败可能长期潜伏
- 抽测命中失败时只强制修复 10% 配额（≥1 个），余债写入 `.learnings/UNIT_TEST_DEBT.md` 后靠**下一次提交**继续压迫 — 无提交即无压迫
- 修复依赖人工/agent 当时在场，深夜发现的失败无法自动闭环

### 1.2 目标

建立一个**夜间批处理机制**：

> 每天晚上 00:00–08:00 窗口，对每个仓库（~40 子模块）**随机抽测**单元测试；发现失败 → 用 `claude-agent`（无头 Claude Code）**自动修复** → 复跑验证 → **直推 main**；未解决失败写入债台账。

### 1.3 非目标

- ❌ 不跑 Playwright / E2E / 集成全栈（与规则 28 一致，抽测池仅单元测试）
- ❌ 不新增任何 HTTP 接口 / 服务端点（纯宿主机 cron 工具链）
- ❌ 不改动业务服务代码路径、不发布事件到 Kafka
- ❌ 不触碰有未提交改动的脏工作区仓库

---

## 2. 现状盘点（复用资产）

| 资产 | 位置 | 复用方式 |
|------|------|---------|
| 随机抽测逻辑（Go 目录收集 / JS 文件收集 / 30% 比例 / 至少 1 个） | `scripts/hooks/templates/pre-commit.{go,go_js,go_python,node,python}` | **抽取为共享库**（决策 A：SSOT） |
| 钩子分发脚本 | `scripts/deploy_repo_random_precommit.sh` | 扩展为同时分发共享库 |
| 无头修复代理 | `claude-agent/`（Go CLI 包装 `claude -p`） | 失败修复执行器 |
| 债台账 | `.learnings/UNIT_TEST_DEBT.md` | 未解失败写入（决策 D） |
| cron 基建模式 | `runAll/scripts/ensure_services_healthy.py` + crontab | 同风格：bash 入口 + python 编排 |
| 硬门禁 | 规则 6（禁 `--no-verify`）/ 规则 41（fix 提交必须带回归单测） | 修复提交必须遵守 |
| CI 校验 | `db/scripts/ci/check_subrepo_random_precommit_hooks.py` | 扩展校验共享库分发 |

**测试资产规模**（巡检目标池）：

| 类型 | 规模 | 仓库 |
|------|------|------|
| Go `*_test.go` | ~460 文件 / 16 仓 | taskAuth 144, taskCloud 92, taskEvents 59, taskProject 27, taskBill 22, taskTask 20, taskGitOauth 17, taskContainerGateway 16, go_relayToTrae 15, shareLib 14, taskCredential 12, taskAIComment 8, taskTenant 7, taskAiProvider 7, go_run_container 6, taskAgentSupport 2 |
| JS `*.test.js/ts` | ~520 文件 / 3 仓 | taskFE 490, taskChromePlugin 22, taskAiProvider frontend 9 |
| Python `test_*.py` | 少量 | db, trae-agent, AiMonitor |

---

## 3. 方案设计

### 3.1 组件总览

```
crontab: 20 0 * * *  bash scripts/nightly-test-sweep.sh        (00:20 启动, 避开整点)
                        │
                        ├─ 窗口守卫: 非 00:00–08:00 → 立即退出
                        ├─ flock 互斥锁 (logs/.nightly-sweep.lock)
                        └─ python3 scripts/nightly_test_sweep.py
                              │
                              ├─ 枚举仓 (.gitmodules 为 SSOT, 排除表见 §3.3)
                              ├─ 每仓: source scripts/lib/random_test_runner.sh
                              │        随机抽测 (默认 30%, 至少 1 包)
                              │         ├─ 全绿 → pass
                              │         └─ 有失败 → claude-agent run (限时 25min)
                              │               ├─ 修复 → 复跑验证 → commit → push
                              │               └─ 失败/超时 → 写 UNIT_TEST_DEBT.md
                              └─ 生成 logs/nightly-test-sweep-report-<date>.md
```

### 3.2 新增/修改文件

| 文件 | 动作 | 说明 |
|------|------|------|
| `scripts/lib/random_test_runner.sh` | 🆕 | **共享抽测库（SSOT）**：函数化抽取自 pre-commit 模板 — `rt_has_tests` / `rt_collect_go_dirs` / `rt_collect_js_files` / `rt_collect_py_files` / `rt_pick_random(files ratio)` / `rt_run_go` / `rt_run_js` / `rt_run_py` / `rt_repo_type`；兼容 `PRECOMMIT_TEST_RATIO` 环境变量；内置 `--self-test` |
| `scripts/hooks/templates/pre-commit.*` | 🟡 重构 | 6 个模板改为 `source lib/random_test_runner.sh` 调用（行为不变，供钩子分发的副本将 lib 一并复制） |
| `scripts/hooks/templates/lib/` | 🆕 | 模板随附的 lib 副本（部署时复制进子仓 `scripts/hooks/lib/`） |
| `scripts/deploy_repo_random_precommit.sh` | 🟡 扩展 | 部署时同时复制 lib；保留 SKIP/KEEP 特殊分支 |
| `scripts/nightly_test_sweep.py` | 🆕 | 巡检编排器（Python，风格同 `ensure_services_healthy.py`） |
| `scripts/nightly-test-sweep.sh` | 🆕 | bash 入口（窗口守卫 + flock + 调 python） |
| `db/scripts/ci/check_subrepo_random_precommit_hooks.py` | 🟡 扩展 | 校验子仓 `scripts/hooks/lib/random_test_runner.sh` 已分发且为最新 |
| `.learnings/UNIT_TEST_DEBT.md` | 🟡 联动 | 未解失败自动追加条目（路径 + 时间戳 + 简述） |
| `docs/superpowers/specs/2026-08-05-nightly-test-sweep-design.md` | 🆕 | 本设计文档 |

### 3.3 巡检编排（nightly_test_sweep.py）

**仓库枚举与排除**：
- SSOT 来源：`.gitmodules`（40+ 子模块）→ 全部候选
- 自动跳过：无单测资产仓（`rt_repo_type` 判定为 none）、不存在/未检出、无 remote
- 运行期跳过：worktree 有未提交改动 / 处于 rebase/merge 冲突态 / 当前分支无 upstream
- 可配置排除（`--exclude`）：`DaydaymoneyGrafana AiMonitor docs db valueStream taskChromePlugin`（基建/文档/插件仓默认不巡检，可用 flag 放开）

**每仓流程**：

```
1. 前置: git -C <repo> status --porcelain；仅含未跟踪的 LICENSE/COMMERCIAL.md/README(.md| copy.md) 视为干净；其余改动 → skip(dirty，note 含样例路径)

2. 抽测: source lib → rt_repo_type → rt_pick_random(ratio=30%, 至少 1 个测例单元)
   - Go: 随机 30% 测试目录 + `go test -count=1 ./dir/...`（≤25min 超时）
   - JS: 随机 30% `*.unit.test.js` + vitest/jest 运行
   - Py: 随机 30% `test_*.py` + pytest
3. 全绿 → 记录 pass
4. 失败 → 记录 fail 明细到 tmp/nightly/<repo>-<ts>/fail.log
   → 调起修复（见 §3.4）
   → 修复成功 → 复跑抽测（同参数）→ 记录 fixed + commit hash
   → 修复失败 → 写债台账（§3.5）
```

**预算与硬停**：
- 每仓总预算 40 min（抽测 15 + 修复 25）；全局 deadline **07:30** — 到点停止接新仓，已开始的仓允许收尾
- 每日每仓最多 1 次修复尝试（防死循环）；串行执行（不并发 — 避免 ~40 仓同时跑 agent 的资源冲击，且均为本地仓操作）

**dry-run / 单仓模式**：
- `bash scripts/nightly-test-sweep.sh --dry-run` → 只列出将巡检的仓与抽测池规模
- `python3 scripts/nightly_test_sweep.py --repo taskAuth --ratio 100 --no-fix` → 单仓全量抽测不修复（验证执行器）

### 3.4 自动修复（claude-agent）

失败时在**该子仓目录内**调用：

```bash
claude-agent run "「夜间单测巡检」修复任务 — 仓库: <repo>，失败明细见 <fail.log>。
1. 重跑失败测例复现失败
2. 修复：允许修改产品代码（若测例揭示真实回归）；测试断言过时则只改测试
   （遵守规则 41：fix 提交必须携带与被修源码对应的回归单测）
3. 复跑该测例 + 相关测例直到通过
4. commit（禁止 --no-verify，遵守规则 6/28；commit message 前缀 fix:/test: 如实反映）
5. git push 到当前分支 upstream
完成或失败原因都输出一行 JSON 结果。"
```

**安全约束**：
- 前置检查：仓必须 clean、可写、有 upstream；否则跳过修复仅记录
- 超时 25 min，超时 kill 进程组，标记 unfixed（不产生孤儿提交）
- 修复提交**必须**复跑验证通过才允许 commit；`claude-agent` 输出解析 `JSON` 结果行（`{"ok": true, "commit": "<hash>"}` / `{"ok": false, "reason": "..."}`）
- push 失败不回滚本地 commit — 报告标注 `pending-push`
- 敏感信息（token/password）在日志/报告中脱敏为 `***`
- 巡检脚本自身不携带任何凭据；依赖本机 git 凭据

### 3.5 债台账联动（规则 28 一致）

未解失败追加到 `.learnings/UNIT_TEST_DEBT.md`：

```markdown
## <UTC 时间戳> — nightly sweep — <repo>

- 抽测失败文件（未修复）: `<路径 1>` `<路径 2>` …
- 失败摘要: <首行错误>
- 修复尝试: 失败原因（agent 超时 / agent 放弃 / push 失败）
```

去重：同路径 7 天内已登记且无新 commit 变化则不重复追加（仅更新时间戳）。

### 3.6 报告与日志

| 文件 | 内容 |
|------|------|
| `logs/nightly-test-sweep-cron.log` | cron 调用流水（重定向） |
| `logs/nightly-test-sweep-<date>.log` | 每仓明细（抽测/修复/提交） |
| `logs/nightly-test-sweep-report-<date>.md` | 汇总报告：skip/pass/fail/fixed/unfixed 计数、fixed commits、unfixed 明细、债台账引用 |
| `tmp/nightly/<repo>-<ts>/fail.log` | 失败明细（供 agent 与人工复现） |

---

## 4. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 夜间随机单测巡检 | — | — | — | **无对应事件**：工程自动化批处理（DevOps 工具链），不改变任何业务事实、无跨聚合/跨服务副作用；产出仅为日志/报告/债台账（文档文件）。符合例外条款「纯工程工具、无服务端业务状态变更」 |

---

## 5. 🐍 Python 新增接口清单与 Go 替代评估

**未触发**（阶段 A 不适用）：

| 触发条件 | 判定 |
|---------|------|
| Django 公网/internal/子进程 API | ❌ 无任何 Django 服务（已退役） |
| Python 侧车 API（Flask/FastAPI route） | ❌ 无 |
| 网关后新落 Django handler | ❌ 无 |

巡检脚本（bash + python）是 **cron 批处理工具，非 HTTP 服务** — 不监听端口、无路由、无请求处理，不属于「新增接口」范畴。编排器选 Python 与既有 cron 工具（`ensure_services_healthy.py`、`delete_merged_feat_branches.py`）保持一致；Go-first 元规则针对服务接口，不适用于无接口的宿主脚本。

---

## 6. 价值流影响

- **无用户可见价值流影响**：user-auth / task-management / cloud-resource / billing 等既有价值流均不受影响（巡检只读测试、不改业务代码路径 — 修复动作受门禁约束且复跑验证）
- `conf/value-stream.yaml` 仍为遗留 Django 配置（`../task2app/Saas_project` 已删除，Django 已退役）— 本次不涉及，遗留迁移另行处理
- 本机制属于工程自动化横切能力，类似已有的 service-watchdog / cron 维护族

---

## 7. 🏛️ 架构变更影响

- **迭代版本**: v65 🎯 target（基于 **v64 target**；⚠️ v64 微信身份仍为 target 待交付 — 两个迭代领域独立（DevOps 工具链 vs 微信登录），无文件/组件冲突，可叠放；交付时按序切换 current）
- **迭代名称**: nightly-test-sweep（夜间随机单测巡检与自愈）
- **作者**: claude | **设计日期**: 2026-08-05 23:47
- **新增文件**（application-integration 视图，三类伴生格式）:
  - 🆕 `docs/architecture/v65-application-integration-20260805-2347-claude.puml`
  - 🆕 `docs/architecture/v65-application-integration-20260805-2347-claude.archimate`（含 Plateau v64→Gap→WP→Plateau v65 架构变迁视图 + 目标拓扑视图 sourceConnection 完整连线）
  - 🆕 `docs/architecture/v65-application-integration-20260805-2347-claude.mermaid.md`
- **已有文件（未修改）**:
  - `v63-application-integration-*.puml/.archimate/.mermaid.md` (current)
  - `v64-application-integration-*.puml/.archimate/.mermaid.md` (target, 待交付)
  - `v1-enterprise-landscape-*.puml` (current, 未涉及)
- **变更明细**:
  - 🟢 [NEW] `NightlyTestSweep`（Technology_Process，cron 00:20 触发，窗口 00:00–08:00，串行巡检）
  - 🟢 [NEW] `claude-agent`（Application_Component，无头修复执行器）
  - 🟢 [NEW] `40+ 子仓池`（Application_Component，巡检目标聚合）
  - 🟢 [NEW] `UNIT_TEST_DEBT.md`（Application_DataObject，债台账）
  - 🟢 [NEW] `SweepReport`（Application_DataObject，每日报告）
  - 🟢 [NEW] `random_test_runner.sh`（Technology_Artifact，共享抽测库 SSOT）
  - 🟡 [MODIFIED] 子仓 pre-commit hooks — 重构为 source 共享库（行为不变）
- **新增接口**: 0 个（无 HTTP API，🐍 门禁不触发）

### .archimate 架构变迁要点

| 元素类型 | 内容 |
|----------|------|
| **Plateau v64** | Target — 微信身份统一设计（v64 领域） |
| **Plateau v65** | Target — 夜间单测巡检与自愈（本迭代） |
| **Gap** | 提交时随机门禁无夜间压迫 — 低频仓失败长期潜伏、深夜失败无自动闭环 |
| **WorkPackage** | WP1 共享库抽取与钩子重构 → WP2 巡检编排+修复闭环 → WP3 cron/台账/报告 |
| **视图** | `架构变迁 v64→v65 — nightly-test-sweep`（Plateau→Gap→WP→Plateau 全连线）+ `v65 Target — 巡检与自愈数据流`（Sweep→子仓池/claude-agent/台账/报告 全连线） |

---

## 8. 测试计划

| 层级 | 内容 |
|------|------|
| 共享库自测 | `bash scripts/lib/random_test_runner.sh --self-test`（随机选择确定性、至少 1 单元、比例边界 0/100、go/js/py 类型判定） |
| CI 门禁 | `repo-quality-gates.yml` 追加：lib self-test + `check_subrepo_random_precommit_hooks.py` 扩展（校验 lib 已分发） |
| 巡检 dry-run | `--dry-run` 列出仓清单与抽测池；单仓试运行 `--repo taskAuth --ratio 100 --no-fix` 全量抽测验证 |
| 修复闭环 E2E（一次人工） | 在某仓制造一个确定性测试失败 → 手动触发巡检（`--repo X --ratio 100`）→ 验证：agent 修复 → 复跑绿 → commit 门禁通过 → push 成功 → 报告 fixed；再验证失败场景（制造 agent 无法修复的失败）→ 债台账追加 |
| 窗口守卫 | 手动改系统时间或 `--simulate-hour 9` 参数验证 08:00 后拒绝运行 |

---

## 9. 实施周期建议

| 阶段 | 内容 | 时长 |
|------|------|------|
| D1 | 抽取 `random_test_runner.sh`（函数化 + self-test）→ 重构 6 个 pre-commit 模板 → 扩展 deploy 脚本 → 全仓重部署 → CI 校验扩展 | 1 天 |
| D2 | `nightly_test_sweep.py` 编排（枚举/抽测/预算/报告）+ bash 入口 + dry-run | 1 天 |
| D3 | 修复闭环（claude-agent 调用 + JSON 解析 + 台账联动）+ cron 安装 | 1 天 |
| D4 | E2E 验证（成功/失败/超时/push 失败场景）+ 报告格式收尾 | 1 天 |

---

## 10. 窗口内循环（多轮）— 2026-08-06 增补

**用户决策**（2026-08-06 头脑风暴）：一轮执行完毕若未到 8 点 → **自动开始下一轮**（窗口内循环）；**每轮每仓 1 次修复尝试**。

### 行为定义

| 参数 | 默认 | 说明 |
|------|------|------|
| `--max-rounds` | 3 | 窗口内最大巡检轮数（有界，防失控） |
| `--round-cutoff` | 07:00 | 不在此时间后开始新轮（真实时钟判定；07:30 全局硬停仍生效，用于轮内收尾） |

- 每轮**重新随机抽样**（`rt_pick_random` 每轮独立 `$RANDOM`）→ 单晚覆盖 ≈ 30% × 轮数（3 轮 ≈ 60-90% 全池）
- 每轮重新做脏工作区/remote/会话互知检查（被并行会话占用的仓每轮自动跳过/恢复）
- 修复尝试：每轮每仓 1 次（用户决策；N 轮 = 至多 N 次尝试，台账 7 天去重跨轮生效，不重复登记）
- 报告按轮分区（`| R1 | repo | result | note |`）+ 汇总行（`rounds: N/M (cutoff HH:MM)`）；明细日志带 `=== Round N/M ===` 头
- dry-run 强制单轮；`--simulate-hour` 仅影响窗口/截止判定（真实时钟用于轮次推进）

### 架构影响判定

**无需新架构版本**：循环是既有 `NightlyTestSweep`（Technology_Process，v65）的**行为增强**（调度语义变化），组件、关系、数据流拓扑均不变 —— 符合「纯配置/行为微调不更新架构」条款；本增补仅修订设计文档与编排器参数。

---

## 11. 触发策略：夜间每 20 分钟轮询（2026-08-06 增补）

**用户决策**：夜间（00:00–08:00）**每 20 分钟**检查一次——若没有巡检在运行（flock 空闲），则触发一轮新巡检；运行中则静默跳过。

- **crontab**: `*/20 0-7 * * * bash /tmp/ram-work/scripts/nightly-test-sweep.sh >> .../nightly-test-sweep-cron.log 2>&1`
- **语义**：入口脚本的 flock 互斥即「无运行才启动」判据；持锁时静默退出（不刷日志）。一轮 3 轮循环提前完成（如 03:00）后，下一个 20 分钟 tick 自动启动新的一轮——夜间覆盖连续化（一晚可达 9 轮抽测）
- **边界**：07:00 后触发的 tick 被编排器 round-cutoff 拦下（不接新轮）；07:30 全局硬停负责轮内收尾；窗口守卫（hour<8）兜底
- **架构影响**：同 §10 — 触发源变化（一次性 cron → 轮询 cron），组件/拓扑不变，**无需新架构版本**
