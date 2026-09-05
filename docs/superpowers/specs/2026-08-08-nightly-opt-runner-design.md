# 夜间 OPT 自动执行器设计（nightly-opt-runner）

- **迭代**: nightly-opt-runner
- **日期**: 2026-08-08
- **作者**: claude
- **状态**: ✅ 已实现

---

## 1. 背景与目标

### 1.1 问题

`.learnings/OPTIMIZATION_TODOS.md` 持续累积优化待办（2026-08-07 当日 ~45 项 pending），
其中大部分是本地可直接完成的代码/配置/测试/部署验证项，但依赖人工在场执行：
- 优化项处理需要完整 agent 上下文（理解、实现、测试、提交、归档），日间人工会话时间成本高
- 部署/复验类项需要在低流量窗口执行（夜间 00:00–08:00）
- 无自动机制时条目会持续堆积，完成即迁移的归档纪律依赖自觉

### 1.2 目标

> 每天晚上 00:00–08:00 窗口，自动启动无头 Claude Code 会话（claude-agent run，goal-mode
> 风格），逐项处理 OPTIMIZATION_TODOS.md 中的 pending 优化项：理解 → 实现（TDD）→
> 测试全绿 → 子仓库原子提交推送 → 状态迁移归档 → meta 指针同步；会话中途死亡时由
> 下一触发点接续（OPT 文件即持久任务队列，自愈不丢项）。

### 1.3 非目标

- ❌ 不处理 `.learnings/UNIT_TEST_DEBT.md`（夜间单测巡检 nightly-test-sweep 的职责）
- ❌ 不在日间运行（窗口守卫硬门禁）
- ❌ 不替代人工对产品决策项（PRODUCT_DECISIONS.md）做决策
- ❌ 不修改既有工具链（claude-agent / session hub / pre-commit 门禁）

## 2. 架构

```
crontab */30 0-7 * * *
        │
        ▼
scripts/nightly-opt-runner.sh
   ├─ 窗口守卫 (00:00 ≤ hour < 08:00; SIMULATE_HOUR/NIGHTLY_OPT_FORCE 可测试)
   ├─ flock 互斥 (logs/.nightly-opt-runner.lock — 运行中静默退出)
   ├─ ANTHROPIC_API_KEY 补齐 (cron 不加载 ~/.bashrc → 403 防护)
   └─ claude-agent run --config-file claude_config.yaml
              --trajectory-file logs/trajectories/opt-runner-*.jsonl
              --file scripts/prompts/nightly-opt-task.md
        │
        ▼
无头 Claude Code 会话（permission_mode: skip, deepseek-v4-flash, max_steps 400）
   ├─ SessionStart/PreToolUse hook 自动注册会话 + Edit/Write 取锁（session hub 互知）
   ├─ 队列分拣：不可执行项快速 skip（防 Reached max turns 无 JSON）
   ├─ 逐项处理可执行 OPT（TDD → 测试全绿 → 子仓原子提交推送 → 迁移归档）
   ├─ 会话锁冲突 → session list/pause/resume 协调；不可协调则跳过记录
   ├─ CLAUDE_CODE_DISABLE_UNKNOWN_MODEL_WINDOW_ENFORCEMENT=1（第三方模型窗口告警）
   └─ 收尾 meta 指针同步提交 → 末行 JSON 摘要（硬性要求，触顶前必须输出）
```


### 2.1 自愈语义（关键设计）

- **持久队列**: OPT 文件 `Status` 字段是唯一队列状态；会话死亡后任何未完成项仍为
  `pending`，下一触发点（30 分钟后）自动接续 —— 无需检查点/恢复协议
- **幂等**: 每项以「子仓提交 + Status 迁移」为完成原子，不重复不丢失
- **锁互斥**: flock 防止双会话并发写同一 OPT 文件；session hub 锁防止与交互会话
  同日间残留锁冲突（prompt 内协调指令）
- **日志**: 全量输出 → `logs/nightly-opt-runner-cron.log`（已有 truncate-ram-work-logs
  每小时轮转）；trajectory → `logs/trajectories/opt-runner-*.jsonl`（审查闭环）

### 2.2 与既有组件关系

| 组件 | 关系 |
|---|---|
| nightly-test-sweep (0-7 每 20min) | 并存；不同锁文件、不同队列（UNIT_TEST_DEBT vs OPT）；session hub 互知 |
| sessionctl / session hub | 无头会话经 hook 自动注册取锁；冲突按 prompt 指令协调 |
| pre-commit 门禁（规则 28/32/41） | 禁止 --no-verify；提交走完整门禁（含随机抽测） |
| claude-agent run | 执行载体（--file 读任务、--trajectory-file 留痕、config 指定模型） |

## 3. 实施

1. `scripts/nightly-opt-runner.sh` — 窗口守卫 + flock + API key 补齐 + claude-agent run 封装
2. `scripts/prompts/nightly-opt-task.md` — 无头执行纲领（队列读取/逐项处理/提交纪律/冲突协调/JSON 输出）
3. crontab 追加: `*/30 0-7 * * * bash /tmp/ram-work/scripts/nightly-opt-runner.sh >> /tmp/ram-work/logs/nightly-opt-runner-cron.log 2>&1`
4. 验证: bash -n / 窗口守卫模拟 / --dry-run / 端到端小任务冒烟（tiny prompt 验证 API key + 二进制 + trajectory + JSON 链路）

## 4. 验收标准

- [x] crontab 已安装且注释分组清晰
- [x] 窗口外触发静默退出（模拟 hour=12 验证）
- [x] 持锁时第二触发点静默退出（flock 验证）
- [x] --dry-run 打印完整命令
- [x] 端到端冒烟：真实 claude-agent run 小任务成功返回 JSON、trajectory 落盘
- [ ] 首夜运行后：OPTIMIZATION_TODOS.md 出现 completed 迁移、日志/摘要可读（运行时验收）

## 5. 风险与注意事项

- **会话锁冲突**: 用户 VSCode 会话夜间仍开启时可能持锁阻塞子仓提交 → prompt 内
  pause/resume 协调或跳过记录；次日人工可见 skipped 原因
- **API key**: cron 环境 403 防护已在脚本内置（从 ~/.bashrc 补齐）
- **模型质量**: deepseek-v4-flash 为低成本模型，复杂项可能需多夜迭代；每天检查
  cron 日志摘要即可
- **部署风险**: 复验类项涉及本机部署（低流量窗口内安全），prompt 限定仅在本窗口操作

## 6. 故障修复记录（2026-08-08）

**现象**: 安装当夜（00:00–07:30）cron 触发多次全部失败，日志报错（后被每小时
truncate 清空）。

**根因链**:
1. 01:05 `@anthropic-ai/claude-code` npm 包被重装为 2.1.224 但安装损坏 —
   `bin/claude.exe` 仅为错误提示 shim（"claude native binary not installed"），
   optional native 依赖（`-linux-x64`）未下载，`.npm-global/bin/claude` 入口消失
2. cron 环境不加载 `~/.bashrc` → PATH 缺 `.npm-global/bin`（bashrc 164 行才 export）
   → claude-agent FindBinary 解析不到可用 claude（交互环境靠 bashrc 才正常）

**修复**:
1. 清理 npm 原子替换残留（`.claude-code-*` 备份目录 ENOTEMPTY 冲突）后
   `npm install -g @anthropic-ai/claude-code@2.1.224 --include=optional` 完整重装
2. runner 脚本加固：
   - PATH 补齐 `$HOME/.npm-global/bin:$HOME/bin`（对齐 bashrc）
   - claude 可用性预检：不可用 → `install.cjs` 自修复 → 仍失败明确报错退出
   - `DISABLE_AUTOUPDATER=1`：防夜间会话触发自动更新再次损坏包
3. 验证：cron 等价环境（env -i + crontab PATH）端到端冒烟通过，命中 2.1.224
