# valueStream — 价值流单元测试编排服务

**Date:** 2026-05-19  
**Status:** approved (amended 2026-05-19 — fields, API concurrency, cancel, failed_steps order)  
**Language:** Go (single binary)  
**Scope:** `valueStream/` directory in mono-repo root  

## Summary

独立 Go 服务 `valueStream`：通过**命令行指定**的 YAML 配置，注册多条跨服务业务价值流及其环节；每个环节对应一个 **pytest 单元测试文件**。提供 Web UI 与 API，支持单环节测试、整条价值流顺序测试，并展示各环节与价值流的通过/失败状态。

**不依赖 runAll 或线上服务**：测试在 `Saas_project` 内以 `settings_test` + pytest 自洽运行（与现有 `pytest.ini` 一致）。

## Motivation

- `runAll` 解决**进程/HTTP 健康**；无法回答「用户注册 → 登录 → 建项目」等业务链路是否正常。
- `task2app` 文档（`docs/flows/价值流/`、`unit-test-cases.md`）已把价值流环节映射到 `*_test.py`，但缺少**可执行、可可视化的编排入口**。
- 研发撰写价值流测试时，只需在 YAML 中**注册环节对应的测试文件路径**，即可在本地一键验证整条流。

## User-Confirmed Decisions

| 项 | 决定 |
|----|------|
| 架构 | 独立 `valueStream/`，不并入 runAll |
| 环节测试类型 | 执行 YAML 注册的**单元测试文件**（pytest） |
| 运行前提 | **不需要** runAll / 线上服务（选项 A） |
| 整条流失败策略 | **否**（不停止）：某环节失败后**继续**执行后续环节；流级标红，并明确列出**失败环节** |
| 配置文件 | **仅**通过命令行 `--config` 指定路径，无隐式默认根目录文件 |
| 环节数据字段 | **YAML 显式注册**；命名 `应用服务名.数据库表名.字段名` |
| 应用服务 | **runAll `config.yaml` 中的 `services[].name`**（首段须为合法服务名） |

## YAML Configuration

路径由 `--config` 传入；`runner.working_dir` 与 `test_file` 的相对路径均相对于**配置文件所在目录**解析（与 runAll 一致）。

```yaml
version: "1"

# 用于校验字段名首段（应用服务名）；相对本 YAML 文件目录
# 若 YAML 放在仓库根目录，通常为 config.yaml / task2app/Saas_project
runall_config: config.yaml

runner:
  working_dir: task2app/Saas_project
  pytest_bin: pytest
  env:
    DJANGO_SETTINGS_MODULE: saas_project.settings_test
  pytest_args: []

value_streams:
  - name: user-auth
    description: 用户注册与认证
    steps:
      - name: email-register
        test_file: accounts/view_test/UserViewSet_email_register_test.py
        fields:
          - name: saas-backend.accounts_user.email
          - name: saas-backend.accounts_user.password
      - name: activate
        test_file: accounts/view_test/UserViewSet_activate_test.py
        fields:
          - name: saas-backend.accounts_user.is_active
          - name: saas-backend.accounts_user.activation_token
      - name: login
        test_file: accounts/view_test/UserViewSet_login_test.py
        fields:
          - name: saas-backend.accounts_user.email
          - name: saas-backend.accounts_user.password
          - name: saas-backend.authtoken_token.key
            description: 登录返回的认证令牌

  - name: company-management
    description: 公司与成员管理
    steps:
      - name: company-crud
        test_file: accounts/view_test/CompanyViewSet_test.py
        fields:
          - name: saas-backend.accounts_company.name
```

### 数据字段与应用服务（显式注册）

**建模方式：** 在 `steps[].fields` 中声明该环节涉及的数据字段；**不在 MVP 中**从测试代码或数据库自动抽取。

**字段命名规范（必填）：**

```text
<应用服务名>.<数据库表名>.<字段名>
```

| 段 | 含义 | 示例 |
|----|------|------|
| 应用服务名 | runAll 根目录 `config.yaml` 里 `groups[].services[].name` | `saas-backend`、`git-oauth`、`go-relay` |
| 数据库表名 | 持久化表名（Django 默认表名，含 app 前缀时用下划线） | `accounts_user`、`accounts_company` |
| 字段名 | 列/属性名 | `email`、`password`、`key` |

示例：`saas-backend.accounts_user.email` → 提供方应用服务为 **`saas-backend`**，表 **`accounts_user`**，字段 **`email`**。

**`fields[]` 项：**

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | 完整三段式字段名 |
| `description` | no | UI 展示用说明 |

**提供方应用服务：** 不单独配置；由 `name` **首段**（第一个 `.` 之前）解析，必须与 runAll 服务名一致。

**runAll 对照：**

| 配置项 | 说明 |
|--------|------|
| `runall_config` | 可选但**强烈推荐**；相对价值流 YAML 目录的 runAll 配置文件路径 |
| 加载方式 | 解析 runAll YAML 的 `groups[].services[].name` 集合，用于校验每个 `fields[].name` 的首段 |
| 未配置 `runall_config` | 仅校验三段式格式，**不**校验首段是否在 runAll 中（启动时 `log` 警告） |

**格式校验（启动时）：**

- `name` 恰好分为 **3 段**（按 `.` 分割，每段非空）
- 段内字符：`[a-z0-9_-]+`（兼容 `git-oauth`、`accounts_user`）
- 同一 `step` 内 `fields[].name` 不重复
- 若配置了 `runall_config`：首段 ∈ runAll 服务名集合，否则启动失败并列出非法字段

**与 runAll 的关系：** valueStream **不**启动 runAll；`runall_config` 仅作**命名白名单**，保证「提供方」与运维编排使用同一服务 id。

### Field Reference

#### `runner`

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `working_dir` | yes | — | pytest 的工作目录；相对路径相对 YAML 文件目录 |
| `pytest_bin` | no | `pytest` | pytest 可执行文件（PATH 或绝对路径） |
| `env` | no | `{}` | 追加到子进程环境（覆盖同名变量） |
| `pytest_args` | no | `[]` | 追加在 `test_file` 之前的全局 pytest 参数 |

#### `value_streams[]`

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | 价值流唯一标识（API/UI 使用） |
| `description` | no | 展示用说明 |
| `steps` | yes | 有序环节列表 |

#### 根级

| Field | Required | Description |
|-------|----------|-------------|
| `runall_config` | no | runAll `config.yaml` 路径（相对本文件目录），用于校验字段首段 |

#### `steps[]`

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `name` | yes | — | 环节唯一标识（流内唯一） |
| `status` | no | `active` | `active`：启动时校验 `test_file` 存在，参与整条流与单环节执行；`planned`：目标用例，跳过文件校验与执行，UI/API 不可单跑 |
| `test_file` | yes | — | 相对 `runner.working_dir` 的测试文件路径 |
| `fields` | no | — | 本环节涉及的数据字段列表（见上节） |

#### `steps[].fields[]`

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | `<runAll服务>.<表>.<列>` |
| `description` | no | 可选说明 |

### Validation Rules

- `version` 必须为 `"1"`
- 全局 `value_streams[].name` 唯一
- 每个流内 `steps[].name` 唯一
- `test_file` 非空；启动时若文件不存在 → **启动失败**并打印明确路径
- `runner.working_dir` 必须存在且为目录
- 至少一条 `value_streams` 且每条至少一个 `step`
- 每个 `fields[].name` 满足三段式命名；若配置 `runall_config`，首段为合法 runAll 服务名

## Pass / Fail Semantics

### 环节（Step）

- 执行：`{pytest_bin} {pytest_args...} {test_file}`，`Dir = working_dir`
- **通过**：子进程退出码 `0`（该文件内所有用例通过）
- **失败**：非零退出码；记录 `exit_code`、耗时、`stderr`/`stdout` 尾部片段（例如最后 80 行）

### 价值流（Value Stream）

- **整条流测试**：按 `steps` 顺序**串行**执行；**跳过 `status: planned` 环节**；**任一步失败不中断**，继续后续 `active` 环节
- **流级通过**：最近一次「整条流运行」中，**全部 `active` 环节**均为 `passed`
- **流级失败**：最近一次「整条流运行」中，**至少一个 `active` 环节**为 `failed`；UI/API 必须列出 `failed_steps: ["step-a", ...]`（仅 `active`，顺序与 YAML 一致）
- **`failed_steps` 顺序**：与 YAML 中 `steps` 列表顺序一致（非 map 随机序）
- 单环节测试：只更新该环节状态；若该流此前有整条流运行结果，流级状态按规则重算：
  - 若存在「整条流运行」快照：流级 = 该快照中各步最新结果合并（单步重跑覆盖对应步）
  - MVP 简化：**流级状态仅由「最近一次整条流运行」决定**；单环节测试只更新该步的 `last_run`，不单独改变流级灯（避免语义混乱）。流级灯灰表示「尚未跑过整条流」。

> **MVP 流级灯规则（明确）**
> - 从未 `POST /api/test/stream` → 流级 `unknown`（灰）
> - 最近一次整条流运行后 → 全绿 `passed` / 有红 `failed` + 失败环节列表
> - 单环节测试 → 仅更新该环节行；不改变流级灯（除非用户再次跑整条流）

## CLI

```bash
cd valueStream
./build.sh

# 必须指定配置（产物在 bin/valueStream）
./bin/valueStream --config /path/to/value-streams.yaml

# Web UI 端口（默认 :9998，避免与 runAll :9999 冲突）
./bin/valueStream --config ./my-streams.yaml --ui-port :9998
```

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--config` | **yes** | — | YAML 配置文件路径 |
| `--ui-port` | no | `:9998` | Web UI 监听地址 |

缺少 `--config` → 打印用法并以非零退出。

## HTTP API

| Method | Path | Body | Description |
|--------|------|------|-------------|
| GET | `/api/streams` | — | 全部价值流、环节、最近运行状态 |
| POST | `/api/test/step` | `{"stream":"user-auth","step":"login"}` | 测试单个环节 |
| POST | `/api/test/stream` | `{"stream":"user-auth"}` | 顺序测试整条流（失败继续） |

### GET `/api/streams` 响应形状（示例）

```json
{
  "streams": [
    {
      "name": "user-auth",
      "description": "用户注册与认证",
      "stream_status": "failed",
      "failed_steps": ["activate"],
      "last_stream_run_at": "2026-05-19T10:00:00Z",
      "steps": [
        {
          "name": "email-register",
          "test_file": "accounts/view_test/UserViewSet_email_register_test.py",
          "fields": [
            {
              "name": "saas-backend.accounts_user.email",
              "provider_service": "saas-backend",
              "table": "accounts_user",
              "column": "email",
              "description": ""
            }
          ],
          "status": "passed",
          "last_run": {
            "started_at": "...",
            "duration_ms": 1200,
            "exit_code": 0,
            "passed": true,
            "log_tail": ""
          }
        }
      ]
    }
  ]
}
```

`stream_status` 枚举：`unknown` | `running` | `passed` | `failed`

环节 `status` 枚举：`pending` | `running` | `passed` | `failed`

- 同一时刻全局仅一个测试任务：`POST` 时 **同步** `TryAcquire()`，成功后再异步执行 pytest；重复请求 → `409` + `{"error":"a test is already running"}`
- 未知 `stream` / `step` → **400** + `{"error":"..."}`（在返回 `started` 之前校验）
- 合法请求 → **200** + `{"status":"started"}`
- 运行中 UI 轮询 GET；按钮禁用

## Web UI

- 风格对齐 `runAll/status.html`（深色仪表盘、状态圆点、2s 自动刷新）
- 每个价值流：流名、描述、**流级状态灯**、失败环节摘要（红字列表）、「测试整条流」按钮
- 展开环节：环节名、`test_file` basename、「测试本环节」、环节状态灯
- **数据字段表**（有 `fields` 时展示）：

  | 数据字段 | 数据库表 | 列 | 应用服务（提供方） |
  |----------|----------|-----|-------------------|
  | `saas-backend.accounts_user.email` | `accounts_user` | `email` | `saas-backend` |

  - 完整 `name` 显示在第一列；后三列由服务端解析 `name` 填入 JSON，便于扫读
  - 有 `description` 时在行下或 tooltip 展示
- 失败环节：可折叠 `log_tail`
- API 错误（400/409 等）：顶部 toast 提示，约 5s 自动消失
- 页眉说明：不依赖 runAll；需本机已安装 pytest 及 `Saas_project` 测试依赖

## Architecture

### File Structure

```
valueStream/
├── src/              # Go 源码 + index.html（embed）
├── bin/              # 编译产物 valueStream（build.sh 输出）
├── build.sh
├── go.mod
├── testdata/
└── README.md

# 示例配置（仓库内参考，实际路径由用户 --config 指定）
docs/examples/value-streams.example.yaml
```

### Components

| 模块 | 职责 |
|------|------|
| `config` | 读 YAML、校验、将 `working_dir` / `test_file` 转为绝对路径 |
| `runner` | `exec` pytest；整条流顺序执行；失败不 break |
| `status` | 线程安全存储；记录 `last_stream_run` 与各步 `last_run` |
| `ui` | REST + 静态页 |

### Concurrency

- 全局 mutex：同一时刻仅一个测试任务（任意流/步）
- 整条流内步骤串行

### Error Handling

- pytest 未找到 → 启动时不检测；首次运行返回明确错误写入 `log_tail`
- 配置文件解析/校验失败 → 进程启动失败
- UI 服务器错误 → 打日志，不崩溃主进程
- **SIGINT / SIGTERM**：`main` 将 signal context 注入 Runner；`exec.CommandContext` 终止 pytest；整条流被取消时当前环节标 `failed` 并 `EndStreamRun`（流级通常为 `failed`，未跑环节可能仍为 `pending`）
- Web UI：对 `400`/`409` 显示错误提示（toast），不假装已启动

## Domain Model（服务内部）

| 概念 | 角色 |
|------|------|
| **ValueStream** | 聚合：name、steps、stream_status、failed_steps、last_stream_run_at |
| **Step** | 实体：name、test_file、fields[]、status、last_run |
| **StepField** | 值对象：name（三段式）、可选 description；解析出 provider_service / table / column |
| **TestRun** | 值对象：started_at、duration_ms、exit_code、passed、log_tail |

与 `Saas_project` Django 领域层无耦合；本工具为本地运维/质量门禁。

## Relationship to Existing Artifacts

| 现有物 | 关系 |
|--------|------|
| `runAll/` | 独立；可选并行使用，无硬依赖 |
| `task2app/Saas_project/pytest.ini` | runner 继承其 `DJANGO_SETTINGS_MODULE` 等（通过 `runner.env` 显式设置） |
| `docs/flows/价值流/`、`unit-test-cases.md` | 撰写 YAML 时的**来源对照**；新价值流测试需同步注册到 YAML |
| 根目录 `config.yaml` | runAll 专用；由 valueStream 的 `runall_config` **引用**以校验应用服务名 |

## Testing Strategy（valueStream 自身）

- `config_test.go`：校验规则、路径解析、重复 name
- `runner_test.go`：用 `pytest --version` 或假脚本 mock（`#!/bin/sh` 退出码 0/1）
- `ui_test.go`：HTTP handler 状态码与 JSON 形状

## Non-Goals (MVP)

- 不启动/不检测 runAll 服务
- 不支持 Playwright、go test、多仓库 runner
- 不做测试历史持久化（仅内存；进程退出即清空）
- 不做环节间 `depends_on`（顺序仅由 YAML 列表顺序决定）
- 不自动从 PlantUML / ORM 生成 `fields`
- 不从 pytest 或数据库 introspect 字段

## Example Workflow

1. 本地安装 `task2app` 虚拟环境与 `Saas_project` 测试依赖
2. 编写/维护 `my-value-streams.yaml`，注册价值流、`test_file` 与各环节 `fields`（三段式字段名 + `runall_config`）
3. `./valueStream --config ./my-value-streams.yaml`
4. 浏览器打开 `http://localhost:9998`，对某流点击「测试整条流」
5. 查看哪些环节失败，修复后单步重跑或再跑整条流

## Open Items (Post-MVP)

- 可选 `runner.setup_command`（如 `source activate_env.sh`）
- 流级灯在单步重跑后的增量更新策略
- 历史运行记录导出 JSON
- 与 CI 集成：无 UI 的 `--run-stream name` 一次性退出码
