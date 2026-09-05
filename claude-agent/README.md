# Claude Agent

CLI wrapper for [Claude Code](https://github.com/anthropics/claude-code) — provides config management, trajectory recording, session coordination, and Docker isolation, following the architecture pattern of [trae-agent](https://github.com/bytedance/trae-agent).

## 会话互知与冲突防护（Session Hub, v14/v66）

多智能体会话同时执行时互相知晓、避免编辑冲突。设计：`docs/superpowers/specs/2026-08-06-agent-session-coordination-design.md`。

```
claude-agent session register   注册会话（SessionStart hook 自动调用）
claude-agent session list       列出活跃会话（--active / --repo R / --json）
claude-agent session acquire    获取仓库锁（多仓字典序防死锁；--wait 等待）
claude-agent session release    释放仓库锁（--all 释放全部）
claude-agent session heartbeat  刷新心跳（UserPromptSubmit hook 自动调用）
claude-agent session check      检查锁状态（FREE / HELD_BY / HELD_BY_SELF）
claude-agent session precheck   PreToolUse hook: 编辑前警告其他活跃会话
claude-agent session pause/resume  暂停/恢复会话（SIGSTOP/SIGCONT）
claude-agent session shadow-begin/apply/abort  影子编辑（临时目录编辑→原子拷回）
```

- **仓库级锁**：租约 TTL 10min + 心跳 30s + 僵尸/孤儿窃取；锁有效性以「心跳新鲜 + pid 存活」双重判定
- **死锁**：多仓字典序加锁预防；等待超时后沿持有者等待集 DFS 检测依赖环
- **Hub 目录**：meta root（含 `.gitmodules`）下的 `logs/sessions/{registry,locks,shadow,audit}`；`SESSION_HUB_DIR` 可覆盖 meta root
- **headless 接入**：`claude-agent run --session-repo <repo,...>` 自动注册会话 + 加锁，冲突等待 3min 后跳过
- **hooks 接线**：`<meta-root>/.claude/settings.json`（SessionStart/End/Stop/UserPromptSubmit/PreCompact/PreToolUse），经 `scripts/lib/sessionctl.sh` 调用

> **License note**: Claude Code is used under the [Apache 2.0 license](https://github.com/anthropics/claude-code/blob/main/LICENSE). Claude Agent does **not** modify Claude Code source — it calls the `claude` CLI externally via subprocess.

## Architecture

```
claude-agent CLI
    │
    ├── run "task"          →  claude -p "task" (subprocess)
    ├── interactive         →  claude (foreground, passthrough)
    ├── show-config         →  display resolved config
    └── tools               →  list Claude Code built-in tools
```

## Installation

### 一键安装（推荐）

```bash
./setup.sh
```

此脚本会依次检查 Node.js / Python 版本，通过 npm 安装 `@anthropic-ai/claude-code`，通过 pip 安装 `claude-agent`，并验证两个命令行是否可用。

### 手动安装

```bash
# 1. 安装 Claude Code（Node.js >= 18）
npm install -g @anthropic-ai/claude-code

# 2. 安装 claude-agent（Python >= 3.12）
pip install -e .

# 验证
claude-agent --version
```

### Docker 构建

```bash
# 构建镜像（内含 claude-agent + Claude Code，无需外部依赖）
docker build -t claude-agent .

# 验证
docker run --rm claude-agent --version
```

---

## 使用方式

### 命令总览

```
claude-agent [COMMAND]

Commands:
  run           单次任务（封装 `claude -p`）
  interactive   交互式会话（透传 stdin/stdout 到 `claude`）
  show-config   显示当前配置
  tools         列出 Claude Code 内置工具
```

---

### `run` — 单次任务

最常用的模式，传入任务描述，等待 Claude Code 完成并返回结果。

```bash
# 基础用法：直接传入任务
claude-agent run "Explain the architecture of this project"

# 从文件读取任务
claude-agent run -f task.md

# 指定工作目录
claude-agent run -w /path/to/project "Add unit tests for the utils module"

# 指定模型
claude-agent run -m claude-opus-4-20250514 "Review this code for security vulnerabilities"

# 指定最大轮次
claude-agent run --max-steps 200 "Refactor the entire auth module"

# 使用自定义配置文件
claude-agent run --config-file production_config.yaml "Deploy checklist"

# 保存轨迹文件
claude-agent run -t ./trajectories/debug.json "Debug the CI pipeline failure"

# 直接传入 API key（适用于 CI 环境）
claude-agent run -k "sk-ant-api03-xxx" "Generate API documentation"
```

#### `run` 完整选项

| Option | 说明 |
|---|---|
| `TASK` (位置参数) | 任务描述或问题 |
| `--file, -f PATH` | 从文件读取任务内容 |
| `--provider, -p NAME` | LLM provider（如 anthropic、openrouter） |
| `--model, -m NAME` | 模型名称（如 claude-sonnet-4-20250514） |
| `--api-key, -k KEY` | API key（优先级高于环境变量和配置文件） |
| `--model-base-url URL` | 自定义 API base URL |
| `--max-steps N` | 最大执行轮次 |
| `--working-dir, -w PATH` | 工作目录 |
| `--config-file PATH` | 配置文件路径（默认 claude_config.yaml） |
| `--trajectory-file, -t PATH` | 轨迹输出路径 |
| `--agent-type, -at NAME` | Agent 类型（当前仅 claude_agent） |
| `--docker-image IMAGE` | 在指定 Docker 镜像中运行 |
| `--docker-container-id ID` | 接入已有容器运行 |
| `--docker-keep BOOL` | 任务结束后保留容器（默认 true） |

---

### `interactive` — 交互式会话

透传终端到 `claude` 命令，等同于直接使用 Claude Code 的交互模式，但享受 claude-agent 的统一配置管理。

```bash
# 基础用法
claude-agent interactive

# 指定工作目录
claude-agent interactive -w /path/to/project

# 指定模型
claude-agent interactive -m claude-opus-4-20250514

# 指定配置文件
claude-agent interactive --config-file production_config.yaml
```

输入 `/exit` 或 `Ctrl+C` 退出会话。

---

### `show-config` — 查看配置

```bash
# 显示默认配置文件的内容
claude-agent show-config

# 显示指定配置文件
claude-agent show-config --config-file production_config.yaml

# 覆盖特定值后查看
claude-agent show-config -m claude-haiku-4-5-20251001 --max-steps 50
```

输出示例:

```
                  General Settings
┌──────────────────────┬─────────────────────────────────┐
│ Setting              │ Value                           │
├──────────────────────┼─────────────────────────────────┤
│ Provider             │ anthropic                       │
│ Model                │ claude-sonnet-4-20250514        │
│ Max Steps            │ 100                             │
│ Working Dir          │ /home/user/project              │
│ Trajectory Enabled   │ True                            │
│ Docker Enabled       │ False                           │
└──────────────────────┴─────────────────────────────────┘
```

---

### `tools` — 列出内置工具

```bash
claude-agent tools
```

显示 Claude Code 提供的所有内置工具及其说明（Read、Write、Edit、Bash、Glob、Grep、WebSearch、WebFetch 等）。

---

### Docker 使用场景

#### 构建镜像

```bash
docker build -t claude-agent .
```

镜像基于 `node:20-slim`，内含 Python 3.12 + Claude Code + claude-agent，无需外部依赖。

#### 单次任务

```bash
# 基础
docker run --rm -e ANTHROPIC_API_KEY claude-agent run "Explain this codebase"

# 挂载项目目录
docker run --rm \
  -e ANTHROPIC_API_KEY \
  -v $(pwd):/workspace \
  claude-agent run "Add type hints to all functions"

# 第三方 API endpoint
docker run --rm \
  -e ANTHROPIC_API_KEY=sk-xxx \
  -e ANTHROPIC_BASE_URL=https://api.openrouter.ai/v1 \
  claude-agent run -m anthropic/claude-sonnet-4 "Review this code"
```

#### 交互式会话

```bash
docker run --rm -it \
  -e ANTHROPIC_API_KEY \
  -v $(pwd):/workspace \
  claude-agent interactive
```

#### CI/CD 管道

```bash
docker run --rm \
  -e ANTHROPIC_API_KEY \
  -e CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=true \
  -v $(pwd):/workspace \
  claude-agent run --max-steps 50 "Generate release notes from git log"
```

---

### 轨迹 (Trajectory) 文件

每次 `run` 执行都会自动生成轨迹文件，保存在 `.trajectories/` 目录下（JSON 格式）。

```bash
# 指定轨迹文件路径
claude-agent run -t ./my_trajectory.json "Optimize database queries"

# 查看轨迹内容
cat .trajectories/trajectory_20260701_120000.json
```

轨迹文件包含:
- 任务描述、provider、model、开始/结束时间
- 每次 session 的 stdout/stderr、exit code、耗时
- 执行步骤记录（state、response 摘要、错误信息）

### 完整使用流程示例

```bash
# 1. 设置 API key
export ANTHROPIC_API_KEY="sk-ant-api03-xxx"

# 2. 一键安装
./setup.sh

# 3. 创建配置
cp claude_config.yaml.example claude_config.yaml
# 编辑 claude_config.yaml（可选，不编辑则使用环境变量）

# 4. 执行任务
claude-agent run -w /path/to/project "Write unit tests for all public functions"

# 5. 查看结果
cat .trajectories/trajectory_*.json | python3 -m json.tool | head -30
```

---

## CLI Reference

```
claude-agent [COMMAND]

Commands:
  run            Run a task (wraps `claude -p`)
  interactive    Start interactive session (foreground passthrough)
  show-config    Display current configuration
  tools          List available Claude Code tools
```

### `run` options

| Option | Description |
|---|---|
| `--file, -f` | Read task from file |
| `--provider, -p` | LLM provider |
| `--model, -m` | Model name |
| `--api-key, -k` | API key |
| `--max-steps` | Max execution turns |
| `--working-dir, -w` | Working directory |
| `--config-file` | Config file path |
| `--trajectory-file, -t` | Trajectory output path |
| `--docker-image` | Docker image for isolation |
| `--docker-container-id` | Attach to existing container |
| `--docker-keep` | Keep container after task |

## Configuration

See `claude_config.yaml.example` for a complete example with comments.

```yaml
model_providers:
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    provider: anthropic

models:
  default:
    model_provider: anthropic
    model: claude-sonnet-4-20250514
    max_tokens: 4096

agents:
  claude_agent:
    model: default
    max_steps: 100
    enable_trajectory: true
```

## Project Structure

```
claude-agent/
├── claude_agent/
│   ├── cli.py               # CLI entry point (click)
│   ├── agent/
│   │   ├── agent.py         # Agent abstraction
│   │   ├── agent_basics.py  # State/execution types
│   │   └── claude_agent.py  # Claude Code subprocess wrapper
│   ├── utils/
│   │   ├── config.py        # YAML config system
│   │   ├── trajectory_recorder.py
│   │   ├── claude_process.py  # Subprocess manager
│   │   └── cli/             # Console UI
│   └── prompt/
├── tests/
├── claude_config.yaml.example
├── pyproject.toml
├── Makefile
└── README.md
```

## Environment Variables

### claude-agent (wrapper layer)

| Variable | Description | Default |
|---|---|---|
| `CLAUDE_CONFIG_FILE` | Path to the YAML configuration file | `claude_config.yaml` |
| `CLAUDE_MAX_STEPS` | Maximum execution turns (overrides config) | `100` (from config) |
| `CLAUDE_MODEL_PROVIDER` | Override the model provider name | (from config) |

The YAML configuration file supports `${ENV_VAR}` and `${ENV_VAR:default}` placeholders in **any** string value. For example:

```yaml
model_providers:
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"              # required — error if unset
    provider: anthropic
  openrouter:
    api_key: "${OPENROUTER_API_KEY:sk-or-placeholder}"  # with fallback default
    provider: openrouter
    base_url: "${OPENROUTER_BASE_URL:https://openrouter.ai/api/v1}"
```

The config system resolves values with priority: **CLI flag → Environment Variable → Config File → Default**.

### Provider credentials

Credentials are resolved via the naming convention `{PROVIDER}_API_KEY` and `{PROVIDER}_BASE_URL` (upper-cased provider name). For the built-in `anthropic` provider:

| Variable | Description | Required |
|---|---|---|
| `ANTHROPIC_API_KEY` | Anthropic API key (format: `sk-ant-api03-...`) | **Yes** |
| `ANTHROPIC_BASE_URL` | Custom API base URL (third-party / compatible endpoints) | No |
| `ANTHROPIC_API_VERSION` | API version string (for Azure / Foundry) | No |

Additional providers (OpenRouter, Google, OpenAI, etc.) can be configured in `claude_config.yaml` and follow the same `{PROVIDER}_API_KEY` / `{PROVIDER}_BASE_URL` convention.

### Claude Code (subprocess — passed through)

These variables are **inherited by the subprocess** when `claude-agent` spawns the `claude` CLI. They are configured **outside** claude-agent (in your shell profile, Docker env, or `~/.claude/settings.json`).

#### Authentication

| Variable | Description |
|---|---|
| `ANTHROPIC_API_KEY` | Primary API key (also read by claude-agent's config layer) |
| `ANTHROPIC_AUTH_TOKEN` | Alternative auth token (used by some third-party providers, e.g. SiliconFlow, Alibaba Bailian) |

#### API Routing

| Variable | Description |
|---|---|
| `ANTHROPIC_BASE_URL` | Override the default API base URL |
| `ANTHROPIC_MODEL` | Set the default model |

#### Model Overrides

| Variable | Description |
|---|---|
| `ANTHROPIC_DEFAULT_HAIKU_MODEL` | Override the default Haiku model |
| `ANTHROPIC_DEFAULT_SONNET_MODEL` | Override the default Sonnet model |
| `ANTHROPIC_DEFAULT_OPUS_MODEL` | Override the default Opus model |
| `ANTHROPIC_SMALL_FAST_MODEL` | Model for lightweight operations |
| `CLAUDE_CODE_SUBAGENT_MODEL` | Model used by internal sub-agents |

#### Timeouts & Reliability

| Variable | Description |
|---|---|
| `ANTHROPIC_TIMEOUT` | Custom timeout for API requests (e.g. `30s`) |
| `ANTHROPIC_RETRY_COUNT` | Retry count for failed requests (e.g. `3`) |
| `BASH_DEFAULT_TIMEOUT_MS` | Timeout for bash commands executed by Claude Code |
| `MCP_TIMEOUT` | Timeout for MCP server connections |
| `CLAUDE_CODE_API_KEY_HELPER_TTL_MS` | Refresh frequency for API key helper scripts |

#### Debugging & Telemetry

| Variable | Description |
|---|---|
| `ANTHROPIC_LOG` | Set to `debug` to enable debug-level logging |
| `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC` | Set to `true` to disable telemetry, auto-updater, bug reporting, and error reporting |

#### Microsoft Foundry (Azure)

| Variable | Description |
|---|---|
| `CLAUDE_CODE_USE_FOUNDRY` | Set to `1` to enable Foundry backend |
| `ANTHROPIC_FOUNDRY_RESOURCE` | Azure resource name |
| `ANTHROPIC_FOUNDRY_BASE_URL` | Full base URL (`https://{resource}.services.ai.azure.com`) |
| `ANTHROPIC_FOUNDRY_API_KEY` | API key for Foundry |

### Docker quick reference

```bash
# Minimal — API key only
docker run --rm -e ANTHROPIC_API_KEY claude-agent run "Explain this project"

# Third-party endpoint
docker run --rm \
  -e ANTHROPIC_API_KEY=sk-xxx \
  -e ANTHROPIC_BASE_URL=https://api.daydaymoney.com/v1 \
  -e ANTHROPIC_MODEL=claude-sonnet-4-20250514 \
  claude-agent run "Review the code"

# Disable telemetry in CI
docker run --rm \
  -e ANTHROPIC_API_KEY \
  -e CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=true \
  claude-agent run "Run the test suite"
```

### CI / non-interactive setup

When running in CI pipelines or Docker containers where the OAuth login flow is unavailable, use the API key helper pattern:

```bash
echo 'echo ${ANTHROPIC_API_KEY}' > ~/.claude/anthropic_key_helper.sh
chmod +x ~/.claude/anthropic_key_helper.sh
claude config set --global apiKeyHelper ~/.claude/anthropic_key_helper.sh
```

## License

MIT — see LICENSE file.
