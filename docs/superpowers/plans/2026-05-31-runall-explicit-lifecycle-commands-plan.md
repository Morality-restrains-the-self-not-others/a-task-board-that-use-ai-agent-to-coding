# runAll 显式生命周期命令 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Web UI 启停/重启/编译与 YAML 中 `start_command`、`stop_command`、`build_command` 一一对应；重启固定为 runner 先 stop 再 start；strict 缺命令则拒绝启动 runAll。

**Architecture:** 扩展 `Service` 配置与 `validate()`；`stopService`/`restartService` 主路径调用 `runLifecycleCommand`；`launch_mode: detach` 替代 command 字符串启发式；同 PR 补齐 `runAll.yaml` 全服务命令与各服务 stop 脚本。

**Tech Stack:** Go (`runAll/src`), YAML, bash 服务脚本, Docker Compose

> **Design:** `docs/superpowers/specs/2026-05-31-runall-explicit-lifecycle-commands-design.md`

---

## Task 1: 配置模型与 strict 校验

**Files:**
- Modify: `runAll/src/config.go`
- Modify: `runAll/src/config_test.go`

- [ ] **Step 1:** `Service` 增加 `StartCommand`, `StopCommand`, `LaunchMode`；`command` → `start_command` 别名（warn）
- [ ] **Step 2:** `validate()` — strict 要求 `start_command`、`stop_command` 非空；`restart_command` 若存在则 warn
- [ ] **Step 3:** `config_test.go` — 缺 `stop_command` 时 `LoadConfig` 失败
- [ ] **Step 4:** `go test ./src/ -run Config -count=1`

## Task 2: 生命周期执行器

**Files:**
- Create: `runAll/src/lifecycle_exec.go`
- Create: `runAll/src/lifecycle_exec_test.go`

- [ ] **Step 1:** 失败测试 — `runLifecycleCommand` 在 `working_dir` 执行并返回退出码
- [ ] **Step 2:** 实现 `runLifecycleCommand(ctx, svc, command string)`（日志 tee）
- [ ] **Step 3:** `go test ./src/ -run LifecycleExec -count=1`

## Task 3: stopService 主路径

**Files:**
- Modify: `runAll/src/runner.go`
- Modify: `runAll/src/runner_test.go`

- [ ] **Step 1:** 失败测试 — stop 调用 `stop_command`（temp `run.sh stop` 留 marker）
- [ ] **Step 2:** `stopService`：`stop_command` → `stopProcess` 兜底 → `ensureServiceNotReachable`
- [ ] **Step 3:** 删除/旁路 `composeStopShellCommand` 主路径（保留至 P4 删除文件）
- [ ] **Step 4:** `go test ./src/ -run StopService -count=1`

## Task 3b: stopGroup best-effort

**Files:**
- Modify: `runAll/src/runner.go` — `stopGroup` 循环
- Modify: `runAll/src/runner_test.go`
- Modify: `runAll/src/ui_test.go`（如覆盖 stop-group API）

- [ ] **Step 1:** 失败测试 — 组内 A stop 失败、B 仍被调用且变为 `stopped`
- [ ] **Step 2:** `stopGroup` 收集错误、`continue`，最终 `errors.Join` 或格式化聚合返回
- [ ] **Step 3:** `go test ./src/ -run StopGroup -count=1`

## Task 4: restartService（先停后启）

**Files:**
- Modify: `runAll/src/runner.go`
- Modify: `runAll/src/runner_test.go`

- [ ] **Step 1:** 失败测试 — restart 顺序：先执行 stop marker 再 start marker
- [ ] **Step 2:** `restartService`：可选 `build_command`（失败则 return）→ `stop_command` 路径 → `startAndCheck(start_command)`
- [ ] **Step 3:** `startAndCheck` 使用 `start_command`（非 `command`）
- [ ] **Step 4:** `go test ./src/ -run RestartService -count=1`

## Task 5: launch_mode detach

**Files:**
- Modify: `runAll/src/runner.go`, `runAll/src/compose_lifecycle.go`
- Modify: `runAll/src/compose_lifecycle_test.go`

- [ ] **Step 1:** `waitHealthyWithLaunchCheck` 使用 `svc.LaunchMode == "detach"` 而非 `isDetachLaunchCommand(command)`
- [ ] **Step 2:** 更新 detach 相关测试
- [ ] **Step 3:** `go test ./src/ -run Detach -count=1`

## Task 6: infrastructure 组 YAML + 脚本

**Files:**
- Modify: `runAll.yaml`, `runAll/config.yaml`

- [ ] **Step 1:** `docker-redis` / `docker-kafka` / `ai-monitor` 写入 `start_command`、`stop_command`、`launch_mode: detach`
- [ ] **Step 2:** 确认 `dockerInfra/*/run.sh stop` 已存在且可执行
- [ ] **Step 3:** 手工 AC1 — UI 关 docker-kafka 后 `docker ps` 无 kafka 容器

## Task 7: platform 组脚本与 YAML

**Files:**
- Modify/Create: `taskAuth/run.sh`, `taskBill/run.sh`, `gitOauth/run.sh`, `taskSSE/run.sh`, …
- Create: `task2app/scripts/runall-saas-backend.sh`（或等价）
- Create: `task2app/front_project/app/scripts/runall-lifecycle.sh`（vue）
- Modify: `runAll.yaml`, `runAll/config.yaml`

- [ ] **Step 1:** 各服务实现 `stop`（释放端口/进程/venv 子进程）
- [ ] **Step 2:** YAML 填写 `start_command` / `stop_command` / `build_command`（Go 服务保留 `build.sh`）
- [ ] **Step 3:** `saas-backend`、`taskFE` 优先验收

## Task 8: domain-events 组

**Files:**
- Modify: `taskEvents/run.sh`
- Modify: `runAll.yaml`

- [ ] **Step 1:** `run.sh` 增加 `stop <domain>` / 与 `start` 对称
- [ ] **Step 2:** 五个 `task-events-*` 配置 `start_command` / `stop_command` / `build_command`
- [ ] **Step 3:** UI 重启任一 consumer — 先停后启，端口恢复

## Task 9: container-stack + value-stream

**Files:**
- Modify: `go_run_container/`, `go_relayToTrae/`, `valueStream/`
- Modify: `runAll.yaml`

- [ ] **Step 1:** 补 `stop` 脚本或 `run.sh` 子命令
- [ ] **Step 2:** YAML 四条（start/stop/build）补齐

## Task 10: build 与 UI

**Files:**
- Modify: `runAll/src/service_build.go`, `runAll/src/status.go`（或序列化处）
- Modify: `runAll/src/runner.go` — `BuildService` 用显式 `build_command` only

- [ ] **Step 1:** `serviceBuildable` 仅看 YAML `build_command`（移除推断 fallback 在 P4）
- [ ] **Step 2:** API status 返回 `start_command` 展示（可选，替换 `command` 字段）

## Task 11: 清理与文档

**Files:**
- Delete/trim: `runAll/src/compose_lifecycle.go` 推断（P4）
- Modify: `runAll.yaml.ai.md`, `runAll/README.md`, `dockerInfra/README.md`
- Modify: `value-stream.yaml`（runall-cascade-lifecycle 描述）

- [ ] **Step 1:** 删除 `command` 字段与 `resolveBuildCommand` 推断
- [ ] **Step 2:** 更新 companion 与 README
- [ ] **Step 3:** `cd runAll && go test ./...`

## Task 12: 验收

- [ ] **Step 1:** `go test ./...` 全绿
- [ ] **Step 2:** AC1–AC7（设计 §9）手工清单记录在 PR 描述
- [ ] **Step 3:** strict — 故意删掉某服务 `stop_command`，确认 runAll 启动失败

---

## 依赖顺序

```text
Task 1 → Task 2 → Task 3 → Task 3b → Task 4 → Task 5
                ↘ Task 6–9（YAML+脚本，可与 3–5 并行，但合并前须全部完成）
Task 10 → Task 11 → Task 12
```

**合并门槛：** Task 6–9 完成前不得合并 Task 1 strict 到 main（否则 runAll 无法启动）。
