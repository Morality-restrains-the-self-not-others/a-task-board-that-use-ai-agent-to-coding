# 设计文档：清理项目根目录零散文件并修复来源配置

- **日期**：2026-06-23
- **状态**：待审批
- **类型**：清理 + 配置修复

---

## 1. 问题陈述

项目根目录 `/tmp/ram-work/` 有 31+ 个零散文件，应该只有目录和少量必要配置文件。

## 2. 根因分析 — 三个系统性来源

### 来源 A：`scripts/ramsync.sh` — 无过滤的全量同步

**当前行为**：rsync 将 `/tmp/ram-work` 下所有内容（除了显式排除的几个目录）同步到磁盘。

**问题**：`EXCLUDE_ARGS` 只排除了 `.git/`、`node_modules/` 等目录，没有排除根目录的临时文件模式。开发者在根目录随手创建的任何文件都会被永久保留。

**当前排除列表**：
```bash
EXCLUDE_ARGS=(
    --exclude='.git/'
    --exclude='.fseventsd/'
    --exclude='__pycache__/'
    --exclude='node_modules/'
    --exclude='.pytest_cache/'
    --exclude='*.pyc'
    --exclude='.DS_Store'
    --exclude='tmp/chrome-profile-*/'
)
```

### 来源 B：`.gitignore` — 覆盖不足

**当前行为**：只排除了嵌套 git 仓库、SDK、node_modules、构建产物和 secrets。

**问题**：缺少对常见临时文件/调试产物的忽略规则（`*.log`、`*_test.*`、`*_response` 等）。

### 来源 C：无项目约定 — 开发者习惯

开发者习惯在项目根目录随手创建调试脚本、测试文件、日志。没有文档或钩子引导文件放到正确的位置。

---

## 3. 修改方案

### 修改 1：`scripts/ramsync.sh` — 增强排除规则

在第 23 行后（`.DS_Store` 和 `tmp/chrome-profile-*/` 之后）添加根目录临时文件排除模式：

```bash
# 根目录临时/调试文件（防止同步到磁盘）
--exclude='test-sync.txt'
--exclude='.*_test'
--exclude='*.log'
--exclude='cookies.txt'
--exclude='curl_response'
--exclude='*.tgz'
--exclude='*.mjs'
--exclude='*.py'
--exclude='test_*/'
```

### 修改 2：`.gitignore` — 增强忽略规则

在现有规则后添加：

```gitignore
# 调试和临时文件
*.log
*_response
*.tgz
diagnose_*.mjs
login_test.*
playwright_login.*

# 根目录测试残留
test_*/
test-*.txt
.*_test
```

### 修改 3：清理当前 31 个零散文件

| 操作 | 文件 |
|------|------|
| 删除 | `0`, `=`, `Claude`, `.w-dropdown-list` |
| 删除 | `.autosync_test`, `.direct_test`, `.final_test`, `.manual_test`, `.rsync_test`, `.safety_test`, `.sync_test`, `.systemd_test`, `.test_fix`, `.unique_test_file`, `.verify_test`, `test-sync.txt` |
| 删除 | `gitServiceRoot.md`（含硬编码密码） |
| 删除 | `vite.log`, `cookies.txt`, `curl_response`, `ai_配置目录.tgz` |
| 移入 `playwright_test/` | `diagnose_cors.mjs`, `login_test.mjs`, `playwright_login.mjs`, `test_login.py` |
| 移入 `scripts/` | `runDebugChrome.sh` |
| 移入 `conf/` | `com.user.ramsync.plist` |
| 保留 | `.ai.md`, `.gitignore`, `value-stream.yaml`, `value-stream.yaml.ai.md`, `runAll.yaml` |

### 修改 4：清理 runAll 实验目录

8 个测试目录（`test_bash/`, `test_chdir/`, `test_chdir_abs/`, `test_exec/`, `test_forkexec/`, `test_pipe/`, `test_runall_sim/`, `test_runall_actual/`）全部是 Go 语言实验，移入 `.runall/tests/` 归档。

---

## 4. 修改后的根目录预期结构

```
/tmp/ram-work/
├── .ai.md                    # AI 指令规则
├── .gitignore                # Git 忽略规则
├── value-stream.yaml         # 价值流配置
├── value-stream.yaml.ai.md   # 价值流 AI 文档
├── runAll.yaml -> conf/runAll.yaml
├── .claude/                  # Claude Code 配置
├── .cursor/                  # Cursor 配置
├── .runall/                  # runAll 运行时状态
├── .superpowers/             # Superpowers 状态
├── DaydaymoneyGrafana/             # Grafana 配置
├── AiMonitor/                # AI 监控
├── conf/                     # 全局配置
├── db/                       # 数据库
├── dockerInfra/              # Docker 基础设施
├── docs/                     # 文档
├── gitOauth/                 # Git OAuth 服务
├── gitService/               # Git 服务
├── go_relayToTrae/           # Go relay
├── go_run_container/         # Go 运行容器
├── mock_run_container/       # Mock 容器
├── playwright_test/          # Playwright 测试
├── runAll/                   # 服务编排器
├── scripts/                  # 工具脚本
├── sdk/                      # 第三方 SDK（已忽略）
├── task2app/                 # 任务→应用
├── taskAIEndPoint/           # AI 端点
├── taskAgentSupport/         # Agent 支持
├── taskAuth/                 # 认证服务
├── taskBill/                 # 计费服务
├── taskContainerGateway/     # 容器网关
├── taskEvents/               # 事件服务
├── taskGateway/              # API 网关
├── taskSSE/                  # SSE 服务
├── tmp/                      # 临时文件
├── trae-agent/               # Trae Agent
└── valueStream/              # 价值流引擎
```

---

## 5. 价值流影响

- 不涉及现有价值流修改
- 纯清理操作，无业务逻辑变更

## 6. 领域概念

- 无新领域概念引入
