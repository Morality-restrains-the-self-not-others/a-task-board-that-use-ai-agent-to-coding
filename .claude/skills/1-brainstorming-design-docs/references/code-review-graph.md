# Runbook：code-review-graph（CRG）— SSOT

面向本机开发与 CI。本文件是技能横切 SSOT；`runbooks/code-review-graph.md` 为同步副本。
CRG = [code-review-graph](https://github.com/tirth8205/code-review-graph)（Python 包，MIT），
持久化增量知识图谱（nodes/edges/flows/communities/embeddings），SQLite 存于仓库根 `.code-review-graph/graph.db`。

**硬度：软依赖** — 图缺失、MCP/CLI 不可用、更新失败时，**记录一行后继续**，不得阻断 `/goal` / `/0-auto-flow` 交付。

## 1. 安装与升级

```bash
# 本机安装（2.3.7 起建议 uv tool，免虚拟环境污染）
uv tool install code-review-graph            # 或 pipx install code-review-graph
code-review-graph --version                  # 应 ≥ 2.3.7

# 升级
uv tool upgrade code-review-graph
```

**网络注意**：大陆直连 pypi.org 可用但 IPv6 下载会卡死（`files.pythonhosted.org` IPv6 不通）。
安装/升级遇到「Downloading 卡住 5 分钟+」时，用清华镜像：

```bash
UV_DEFAULT_INDEX=https://pypi.tuna.tsinghua.edu.cn/simple uv tool install/upgrade code-review-graph
```

不要给业务进程（服务、hook、MCP 常驻）设置全局代理；开发机装包可用上述镜像一次性完成。

### 1.1 MCP 注册（Claude Code）

`.mcp.json`（仓库根）加入：

```json
"code-review-graph": {
  "type": "stdio",
  "command": "code-review-graph",
  "args": ["serve"]
}
```

- 修改 `.mcp.json` 后**重启 Claude Code 会话**才生效
- 生效后可查看 MCP 工具（query/impact/search/flows/communities 等）
- 验证：`code-review-graph serve` 手动可起；`code-review-graph status` 输出正常

## 2. 图更新（/goal 与 /0-auto-flow 自动执行，亦手动可跑）

**入口编排**：`goal-mode` 与 `0-auto-flow` 在流水线**开始前**自动执行一次「图同步」，此后 Step 1/9 等按需读取：

```bash
cd /tmp/ram-work
code-review-graph update --brief    # 增量：只重新解析变更文件（推荐默认）
# 分支/基线漂移或图损坏时：
code-review-graph build             # 全量重建（根 monorepo 较慢，慎用；见 §3）
code-review-graph detect-changes --brief   # 只读变更影响分析
```

**何时 `build` 而非 `update`**：`status` 提示 "Graph was built on <branch> but you are now on <branch>"、
或 `update` 结果节点数异常偏少（图索引面与当前工作面严重不符）时，改跑一次 `build`。

### 2.1 更新时机约定（技能行为）

| 入口 | 时机 | 动作 |
|------|------|------|
| `/goal`（goal-mode） | 目标定义完成、进入流水线前 | `update --brief` 一次（fail-open） |
| `/0-auto-flow`（单独调用） | Step 1 之前 | `update --brief` 一次（fail-open） |
| Step 1 brainstorming | Archimate 读取后、方案定稿前 | 读图（impact/search/communities），写 `🕸️ Code Review Graph 分析` 节 |
| Step 9 review | 审查开始 | 用图核对爆炸半径/死代码（`dead-code` / `impact`） |
| 手动 | 随时 | `code-review-graph update` |

同一会话内多次进入流水线：仅首次执行 `update`；后续步骤直接读图，不重复更新（图快照语义）。

## 3. 子仓 register（避免根仓一次全量 build）

根 monorepo 含大量嵌套仓与第三方树；**不要**默认对仓库根做全量 `build`。按工作重点 register 活跃子仓：

```bash
crg-daemon add /tmp/ram-work/task2app --alias task2app
crg-daemon add /tmp/ram-work/taskAuth --alias taskAuth
crg-daemon add /tmp/ram-work/taskTaskService --alias taskTaskService
crg-daemon add /tmp/ram-work/taskCloudService --alias taskCloudService

crg-daemon start
crg-daemon status
```

等价：`code-review-graph register <path>` / `code-review-graph daemon …`。
配置文件：`~/.code-review-graph/watch.toml`。

根仓图（`.code-review-graph/graph.db`）保留作全仓级查询；频繁变更的服务优先走子仓图 + daemon 增量。

图数据目录 `.code-review-graph/` 须 gitignore（根仓已忽略；各子仓若独立提交也须忽略）。

## 4. CI `fail-on-risk` 策略（OPT-20260721-002 结论）

| 模式 | 触发 | `fail-on-risk` | 用途 |
|------|------|----------------|------|
| **默认（软）** | 任意 PR | `none` | sticky 风险评论，不挡合并 |
| **可选门禁** | PR 带 label **`crg-gate`** | `high`（≥ 0.70） | 高风险变更自愿升为合并闸 |

**不**在默认路径启用 `high`/`critical` 的理由：

1. 根仓 / 嵌套仓检出面与误报率未充分标定（CRG 故意偏 recall）
2. 与流水线 CRG「软依赖」一致：缺图或噪声不得阻断日常 `/goal` 交付
3. label 门禁保留「关键 PR 可升硬闸」而不强迫全仓

Workflow：`.github/workflows/code-review-graph.yml`。

## 5. 设计文档强制章节

Step 1 产出的设计文档**必须**包含：

```markdown
## 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | `code-review-graph status` 输出要点（节点/文件数、branch、last_updated） |
| 关键发现 | impact / search / communities 结论（爆炸半径、依赖、社区归属） |
| 决策影响 | 图证据对本次设计的支撑或警示 |
| skip 理由（如跳过） | `skipped_non_code` / `unavailable`（缺包/缺图） + 一行原因 |
```

## 6. 卸载

```bash
code-review-graph uninstall --dry-run
code-review-graph uninstall --yes   # 确认后
```

## 变更记录

- 2026-08-06：从 2026-07-22 的 git stash 归档恢复本 SSOT；适配 Claude Code 平台（`.mcp.json` serve 条目）、uv tool 安装、IPv6 镜像绕行、/goal 与 /0-auto-flow 自动更新时机；2.3.7 实机冒烟通过
- 2026-07-21：初版（Cursor 平台，stash 归档）
