# Claude Agent Go 重写 — 设计文档

- **状态**: 🎯 设计阶段（待审批）
- **作者**: claude
- **日期**: 2026-07-02
- **类型**: 重构（Python → Go 重写）
- **关联迭代**: claude-agent sidecar Go 化

---

## 1. 问题陈述

### 1.1 现状

`claude-agent/` 当前是 **Python 3.12** 实现，约 440 行核心代码，功能包括：

| 模块 | 文件 | 行数 | 职责 |
|------|------|------|------|
| CLI | `cli.py` | 430 | click 命令行入口，4 个子命令 |
| Agent 抽象 | `agent.py` + `claude_agent.py` + `agent_basics.py` | ~250 | Agent 类型系统 + 执行器 |
| 子进程管理 | `claude_process.py` | 220 | `claude` CLI 二进制发现 + subprocess 调用 |
| 配置系统 | `config.py` | 300 | YAML 解析 + `${ENV_VAR}` 展开 + CLI/ENV/Config 优先级 |
| 轨迹录制 | `trajectory_recorder.py` | 140 | JSON 轨迹文件写入 |
| 控制台 UI | `cli_console.py` | ~50 | Rich 库终端输出 |

**依赖链**: Python 3.12 + click + docker + pydantic + python-dotenv + rich + pyyaml + pexpect

### 1.2 问题

1. **语言不统一** — 项目中 8 个 Go 服务 + 2 个 Django 服务 + 1 个 Node.js 服务，Python 是边缘语言（仅 claude-agent 使用）
2. **部署重量级** — Python 依赖管理（venv + pip + hatchling）导致 Docker 镜像需要 `node:20-slim` + Python 3.12 双运行时，镜像体积大
3. **启动性能** — Python 解释器冷启动 + 模块导入延迟明显
4. **分发困难** — Python 无法编译为单文件静态二进制，需要 `pip install -e .` 或虚拟环境
5. **类型安全缺失** — 没有编译时类型检查，运行时 `AttributeError` 风险

### 1.3 用户约束

> **必须用 Go 实现。Claude Code CLI 的 sidecar 项目中不使用 Python、JS 等动态语言。**

---

## 2. 设计目标

| 目标 | 说明 | 优先级 |
|------|------|--------|
| **Go 单二进制** | 一个静态编译的 `claude-agent` 二进制，零运行时依赖 | P0 |
| **功能对等** | 保持现有 4 个子命令 (`run`/`interactive`/`show-config`/`tools`) 的行为兼容 | P0 |
| **配置兼容** | 保持 `claude_config.yaml` 格式兼容，CLI 参数兼容 | P0 |
| **轨迹兼容** | 保持 `.trajectories/*.json` 输出格式兼容 | P0 |
| **Docker 精简** | 镜像从 `node:20-slim + Python 3.12` → `node:20-slim` 单运行时 | P1 |
| **项目模式一致** | 采用 monorepo 内现有 Go 服务的代码组织模式（`src/`、`go.mod`、`confload`、`tracelog`） | P1 |
| **可观测性** | 接入 `tracelog`（OpenTelemetry tracing + slog 结构化日志） | P2 |
| **测试覆盖** | 单元测试覆盖核心路径（子进程 mock、配置解析、CLI 参数） | P1 |

---

## 3. 技术方案

### 3.1 语言与标准库选型

| 维度 | 选择 | 理由 |
|------|------|------|
| **CLI 框架** | `flag` (标准库) + 手写子命令路由 | 零外部依赖，Go 服务惯例（`go_relayToTrae`、`taskAIEndPoint` 均无 CLI 框架依赖） |
| **配置解析** | `confload` (项目内) + `gopkg.in/yaml.v3` | 跟其他 Go 服务统一，复用 `${ENV_VAR}` 展开逻辑 |
| **子进程管理** | `os/exec` (标准库) | Go 标准库的子进程管理足够健壮 |
| **JSON 序列化** | `encoding/json` (标准库) | 轨迹文件写入 |
| **日志** | `tracelog` (项目内) + `log/slog` | 结构化日志 + OpenTelemetry tracing |
| **测试** | `testing` (标准库) + `go test` | Go 惯例 |

**关键决策：不使用 cobra/viper 等第三方 CLI/配置库。** 理由：
- 项目内所有 Go 服务（go_relayToTrae、taskAIEndPoint、taskAuth 等）均使用标准库 + confload
- claude-agent 作为 sidecar 工具，CLI 参数简单（4 个子命令），不需要框架复杂度
- 零外部依赖意味着更小的二进制、更快的编译、更少的供应链风险

### 3.2 项目结构

```
claude-agent/
├── go.mod                    # module claude-agent
├── go.sum
├── main.go                   # 入口：解析 CLI + 路由到子命令
├── build.sh                  # 编译脚本（Go 交叉编译）
├── run.sh                    # 本地运行
├── test.sh                   # 测试脚本
├── Dockerfile                # 精简版 Docker 镜像（单 Go 二进制）
├── claude_config.yaml.example
├── Makefile                  # 保留开发便利命令
├── README.md
│
├── src/
│   ├── cli/
│   │   ├── run.go            # `run` 子命令实现
│   │   ├── interactive.go    # `interactive` 子命令实现
│   │   ├── show_config.go    # `show-config` 子命令实现
│   │   ├── tools.go          # `tools` 子命令实现
│   │   └── cli_test.go
│   │
│   ├── agent/
│   │   ├── agent.go          # Agent 接口 + AgentType
│   │   ├── claude_agent.go   # ClaudeAgent: claude CLI subprocess 封装
│   │   ├── execution.go      # AgentExecution / AgentStep 类型
│   │   └── agent_test.go
│   │
│   ├── process/
│   │   ├── claude_process.go # claude 二进制发现 + subprocess 管理
│   │   └── process_test.go
│   │
│   ├── config/
│   │   ├── config.go         # YAML 配置解析（复用 confload 模式）
│   │   ├── resolve.go        # CLI > ENV > Config 优先级解析
│   │   └── config_test.go
│   │
│   ├── trajectory/
│   │   ├── recorder.go       # JSON 轨迹文件写入
│   │   └── recorder_test.go
│   │
│   └── console/
│       ├── console.go        # 终端输出格式化
│       └── console_test.go
│
├── bin/                       # 编译产物目录
│   └── claude-agent
│
└── tests/                     # 集成测试
    └── integration_test.go
```

#### 与现有 Go 服务对齐的结构对比

| 现有模式 | claude-agent (Go) |
|----------|-------------------|
| `go_relayToTrae/src/main.go` | `claude-agent/main.go` + `src/cli/*.go` |
| `go_relayToTrae/src/handlers.go` | `src/cli/run.go` 等子命令 |
| `go_relayToTrae/build.sh` | `build.sh` |
| `go_relayToTrae/bin/` | `bin/claude-agent` |
| confload + tracelog | confload + tracelog |

### 3.3 核心类型设计

```go
// src/agent/execution.go

// AgentState 表示 Agent 整体执行状态
type AgentState string
const (
    AgentStateIdle      AgentState = "idle"
    AgentStateRunning   AgentState = "running"
    AgentStateCompleted AgentState = "completed"
    AgentStateError     AgentState = "error"
)

// AgentStepState 表示单个步骤状态
type AgentStepState string
const (
    StepThinking    AgentStepState = "thinking"
    StepCallingTool AgentStepState = "calling_tool"
    StepReflecting  AgentStepState = "reflecting"
    StepCompleted   AgentStepState = "completed"
    StepError       AgentStepState = "error"
)

// AgentExecution 追踪一次完整的 Agent 任务执行
type AgentExecution struct {
    Task          string       `json:"task"`
    Steps         []AgentStep  `json:"steps"`
    State         AgentState   `json:"agent_state"`
    Success       bool         `json:"success"`
    FinalResult   string       `json:"final_result,omitempty"`
    ExecutionTime float64      `json:"execution_time"`
    ErrorMessage  string       `json:"error_message,omitempty"`
}

// AgentStep 表示单个执行步骤
type AgentStep struct {
    StepNumber  int            `json:"step_number"`
    State       AgentStepState `json:"state"`
    LLMResponse string         `json:"llm_response,omitempty"`
    Error       string         `json:"error,omitempty"`
}
```

```go
// src/config/config.go

// Config 顶层配置 — 与现有 claude_config.yaml 格式兼容
type Config struct {
    ModelProviders map[string]ModelProvider  `yaml:"model_providers"`
    Models         map[string]ModelConfig    `yaml:"models"`
    Agents         map[string]AgentConfig    `yaml:"agents"`
    Docker         DockerConfig              `yaml:"docker"`
    MCPServers     map[string]MCPServerConfig `yaml:"mcp_servers"`
}

type AgentConfig struct {
    Model            string `yaml:"model"`
    MaxSteps         int    `yaml:"max_steps"`
    WorkingDir       string `yaml:"working_dir"`
    EnableTrajectory bool   `yaml:"enable_trajectory"`
    EnableDocker     bool   `yaml:"enable_docker"`
}
```

### 3.4 CLI 设计

保持与 Python 版本完全兼容的 CLI 接口：

```
claude-agent [全局选项] <command> [命令选项]

全局选项:
  --version              显示版本号

命令:
  run                    单次任务 (封装 `claude -p`)
  interactive            交互式会话 (透传 stdin/stdout)
  show-config            显示当前配置
  tools                  列出 Claude Code 内置工具

run 选项:
  TASK                   任务描述（位置参数）
  --file, -f             从文件读取任务
  --provider, -p         LLM provider
  --model, -m            模型名称
  --api-key, -k          API key
  --model-base-url       自定义 API base URL
  --max-steps            最大执行轮次
  --working-dir, -w      工作目录
  --config-file          配置文件路径 (默认: claude_config.yaml)
  --trajectory-file, -t  轨迹输出路径
  --docker-image         在指定 Docker 镜像中运行
  --docker-container-id  接入已有容器
  --docker-keep          保留容器 (默认: true)
```

实现方式：使用 `flag` 标准库，手写子命令路由（类似 `go tool` 模式）。

```go
// main.go 核心路由逻辑示意
func main() {
    if len(os.Args) < 2 {
        printUsage()
        os.Exit(1)
    }
    switch os.Args[1] {
    case "run":
        cmdRun(os.Args[2:])
    case "interactive":
        cmdInteractive(os.Args[2:])
    case "show-config":
        cmdShowConfig(os.Args[2:])
    case "tools":
        cmdTools()
    default:
        printUsage()
        os.Exit(1)
    }
}
```

### 3.5 子进程管理

Go 的 `os/exec` 比 Python `subprocess` 更简洁：

```go
// src/process/claude_process.go

type ClaudeProcess struct {
    Binary     string   // claude CLI 路径（自动检测）
    WorkingDir string   // 工作目录
    Env        []string // 额外环境变量
}

// FindClaudeBinary 自动检测 claude 二进制路径
// 搜索顺序: ./node_modules/.bin/claude → PATH → ~/.npm-global/bin/claude
func FindClaudeBinary() (string, error) { ... }

// RunTask 执行 `claude -p <task>` 并返回输出
func (p *ClaudeProcess) RunTask(ctx context.Context, task string, opts RunOptions) (*RunResult, error) {
    args := []string{"-p", task}
    if opts.Model != "" {
        args = append(args, "--model", opts.Model)
    }
    if opts.MaxSteps > 0 {
        args = append(args, "--max-turns", strconv.Itoa(opts.MaxSteps))
    }
    cmd := exec.CommandContext(ctx, p.Binary, args...)
    cmd.Dir = p.WorkingDir
    cmd.Env = append(os.Environ(), p.Env...)
    
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    
    start := time.Now()
    err := cmd.Run()
    duration := time.Since(start)
    
    return &RunResult{
        Stdout:     stdout.String(),
        Stderr:     stderr.String(),
        ExitCode:   cmd.ProcessState.ExitCode(),
        DurationMs: duration.Milliseconds(),
    }, err
}

// RunInteractive 交互式模式 — 透传 stdin/stdout/stderr
func (p *ClaudeProcess) RunInteractive(opts RunOptions) (int, error) {
    cmd := exec.Command(p.Binary, args...)
    cmd.Dir = p.WorkingDir
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    err := cmd.Run()
    return cmd.ProcessState.ExitCode(), err
}
```

### 3.6 配置加载策略

配置优先级（与 Python 版本一致）：**CLI flag > 环境变量 > 配置文件中值 > 默认值**

```go
// src/config/resolve.go

// ResolveString 按优先级解析字符串配置值
// 优先级: cliValue > envVar > configValue
func ResolveString(cliValue, configValue string, envVar string) string {
    if cliValue != "" {
        return cliValue
    }
    if envVar != "" {
        if v := os.Getenv(envVar); v != "" {
            return v
        }
    }
    return configValue
}

// ExpandEnvVars 展开 ${VAR} 和 ${VAR:default} 占位符
func ExpandEnvVars(s string) string {
    re := regexp.MustCompile(`\$\{(\w+)(?::([^}]*))?\}`)
    return re.ReplaceAllStringFunc(s, func(match string) string {
        // 提取 varName 和 defaultVal
        // os.Getenv(varName) 或返回 defaultVal
    })
}
```

配置加载集成 `confload`：

```go
import "confload"

func LoadConfig(configFile string) (*Config, error) {
    cfg := &Config{}
    // confload 内置 YAML 加载 + overlay 合并能力
    loader := confload.NewLoader(configFile)
    if err := loader.Load(cfg); err != nil {
        return nil, fmt.Errorf("load config: %w", err)
    }
    // 展开所有 ${ENV_VAR} 占位符
    expandConfigEnvVars(cfg)
    return cfg, nil
}
```

### 3.7 Docker 镜像精简

```
之前 (Python):
  node:20-slim → +Python 3.12 + venv + pip + claude-agent (Python)
  ≈ 800MB+

之后 (Go):
  node:20-slim → +npm install claude-code + claude-agent (Go 静态二进制)
  ≈ 400MB

Dockerfile:
  FROM node:20-slim
  COPY bin/claude-agent /usr/local/bin/claude-agent
  COPY package.json /opt/claude-agent/
  RUN cd /opt/claude-agent && npm install --omit=dev
  ENTRYPOINT ["claude-agent"]
```

### 3.8 测试策略

| 层级 | 工具 | 范围 |
|------|------|------|
| 单元测试 | `go test` + mock | 配置解析、CLI 参数、轨迹序列化、进程管理（mock exec） |
| 集成测试 | `go test` + 真实 `claude` 二进制 | 端到端 `run` 执行、轨迹文件验证 |
| E2E 测试 | `tests/` + shell | Docker 镜像构建 + 运行验证 |

Go 内置的 `testing` 包 + 表格驱动测试（table-driven tests）模式与项目其他 Go 服务一致。

---

## 4. 迁移策略

### 4.1 兼容性保证

| 维度 | 兼容方式 |
|------|---------|
| CLI 接口 | 完全一致的 flag 名称、位置参数、环境变量 |
| 配置文件 | `claude_config.yaml` 格式不变 |
| 轨迹输出 | `.trajectories/*.json` JSON schema 不变 |
| 环境变量 | `CLAUDE_CONFIG_FILE`、`ANTHROPIC_API_KEY` 等全部保持 |
| 退出码 | 与 Python 版本一致 |

### 4.2 渐进迁移路径

```
Phase 1: Go 二进制独立编译验证（不替换 Python）
  └── 产出: bin/claude-agent (Go) 可通过绝对路径调用

Phase 2: 并行对比测试
  └── 用同样的 claude_config.yaml + task，对比 Python/Go 输出是否一致

Phase 3: 替换
  └── 将 pip install -e . 替换为 Go 二进制路径
  └── 更新 Dockerfile 移除 Python 运行时
  └── 更新 setup.sh 移除 pip + venv 步骤

Phase 4: 清理
  └── 删除 claude_agent/*.py、pyproject.toml、.venv/
  └── 保留 README.md、claude_config.yaml.example 等通用文件
```

---

## 5. 领域概念清单

> 供 `/6-ddd-领域设计驱动` 使用

| Bounded Context | 关键实体 | 说明 |
|-----------------|---------|------|
| **Agent Execution** | AgentExecution, AgentStep, AgentState | Agent 执行的生命周期追踪 |
| **Claude Process** | ClaudeProcess, RunResult | Claude Code CLI 子进程抽象 |
| **Configuration** | Config, ModelProvider, ModelConfig | 多层级配置解析 |
| **Trajectory** | TrajectoryRecorder, Session | 执行轨迹录制与回放 |

这是一个 lightweight 工具型组件，不需要复杂的 DDD 建模。Agent Execution 和 Configuration 是两个最核心的领域概念。

---

## 6. 架构变更影响

### 6.1 对当前架构的影响

- **新增 Application Component**: `claude-agent` (Go Sidecar CLI) — 属于 Dev Tools 分组
- **技术栈变更**: Python 3.12 Runtime → Go Runtime（与现有 Go 服务统一）
- **Docker 依赖**: 移除 Python 3.12 + venv + pip → 仅保留 Node.js + Go 二进制
- **现有架构文件**: `application-integration` 视图中 Dev Tools 组需更新

### 6.2 架构版本

- **当前**: v1 (current) + v2 (target, 待交付)
- **本次设计**: v3 🎯 target — claude-agent Python → Go 重写
- **已有文件（不修改）**:
  - `enterprise-landscape-v1-20260701-1630-claude.puml` (current)
  - `application-integration-v1-20260701-1630-claude.puml` (current)
  - `enterprise-landscape-v2-20260701-1745-claude.puml` (target)
  - `application-integration-v2-20260701-1745-claude.puml` (target)

---

## 7. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| Go subprocess 行为差异 | `claude -p` 输出格式不同 | 并行对比测试 Phase 2 |
| confload 配置加载差异 | `${VAR}` 展开行为与 Python 不一致 | 单元测试覆盖所有展开场景 |
| YAML 解析精度差异 | Pydantic → struct 类型转换差异 | 使用严格类型 tag + 测试 |
| CLI flag 解析行为差异 | `--model-base-url` 等复合 flag 命名 | 保留完全相同的 flag 名称 |

---

## 8. 下一步

设计文档审批后：
1. 创建 Git Worktree 隔离工作空间
2. 按 Phase 1-4 渐进实现
3. Phase 2 完成时运行并行对比测试确认兼容性

---

## 9. 附录：与 Python 版本的差异摘要

| 方面 | Python | Go |
|------|--------|-----|
| CLI 框架 | click（第三方） | flag（标准库） |
| 配置解析 | pyyaml + 手写 | confload + yaml.v3 |
| 子进程 | subprocess.run | os/exec |
| 异步 | asyncio | goroutine + context |
| 日志 | rich / print | tracelog + slog |
| 测试 | pytest | go test |
| 分发 | pip install -e . | 单文件静态二进制 |
| Docker 镜像 | node + Python 3.12 + venv | node + Go 二进制 |
| 启动时间 | ~300ms (解释器启动) | ~5ms (静态二进制) |
