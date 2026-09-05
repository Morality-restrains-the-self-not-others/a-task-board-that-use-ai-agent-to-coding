# taskEvents `bin/{event}/` 目录语义修正

> 日期：2026-06-01  
> 状态：**已实施（方案 A）**  
> 触发：用户指出 `bin/{event}/` 应仅含**运行配置 + 可执行文件**，不应出现 `.go` 源码  
> 关联：`2026-06-01-taskevents-handlers-relocation-design.md` v3.1 决策 #1、§2.2

---

## 1. 问题陈述

**用户期望（决策 #1 + §2.2 运维目录）**

```
taskEvents/bin/user_created/
  config.yaml                      # 可选本地覆盖
  task-events-user-created         # 构建产物
```

**当前实现**

```
taskEvents/bin/user_created/
  main.go                          # ← 不应在此
  config.yaml
  task-events-user-created
```

15 个事件目录均含 `main.go`；`run.sh` 以 `./bin/${slug}` 为 `go build` 包路径。

---

## 2. 根因分析

| 因素 | 说明 |
|------|------|
| **设计文档自相矛盾** | §2.2 运维目录**未列** `main.go`；§2 实现布局示例却写 `bin/{event}/main.go`，并注释「不与运维目录混放」——正文与注释冲突 |
| **构建路径便利** | `go build -o bin/foo/foo ./bin/foo` 把源码与产物放在同一目录，少一层 `cmd/` |
| **脚手架复制** | `bin/_template/main.go` 被复制到各事件目录，强化「bin 即入口」习惯 |
| **域级先例** | 遗留 `cmd/accounts` 等域级 main 在 `cmd/`，事件级未沿用同一模式 |

**结论**：不是业务需求要求源码在 `bin/`，而是**实现阶段误把 Go 构建入口当成了运维目录的一部分**，与已确认的决策 #1 不一致。

---

## 3. 目标布局（修正后）

```
taskEvents/
  cmd/event/                         # 事件级 main 源码（G5 后保留；域级 cmd/* 删除）
    _template/main.go
    user_created/main.go
    company_created/main.go
    …（15 个薄入口）
  bin/                               # 纯运维/部署面
    user_created/
      config.yaml                    # 可选
      task-events-user-created       # go build -o 输出
    …
  internal/handlers/{event}/         # 业务逻辑 + 单测（不变）
  eventbin/ consumer/ config/ …      # 共享库（不变）
```

**`bin/{event}/` 允许的文件**

| 文件 | 用途 |
|------|------|
| `config.yaml` / `config.json` | 端口、groupId 等本地覆盖 |
| `task-events-*` | 可执行文件（建议 `.gitignore`） |
| `README.md` | 运维说明（可选，非源码） |

**禁止**：`*.go`、`*_test.go`、任何需 `go build` 的源包。

**`cmd/event/{slug}/main.go` 职责**（与现 main 相同，约 15–30 行）：

```go
// cmd/event/user_created/main.go
func main() {
    eventbin.Run("user_created", handler, consumer.IdempotencyKeyFromEnvelope)
}
```

---

## 4. 方案对比

### 方案 A — 源码迁至 `cmd/event/{slug}/`（推荐）

- 与 Go 社区惯例一致（`cmd/` = 入口，`bin/` = 产物）
- 与决策 #1、§2.2 完全一致
- 改动面：移动 15 个 `main.go` + 更新 `run.sh` / 设计文档 / `_template`
- **风险低**，不改变运行时行为

### 方案 B — 单一 `cmd/event/main.go` + `bin/{slug}/config.yaml` 声明 slug

- `bin/` 更「纯」，main 仅一份
- 需在 `eventbin` 或 registry 中按 slug 构造 Handler（反射或大型 switch）
- 与「每事件独立 wiring（DB/STS/SMTP 依赖不同）」冲突，**复杂度高**
- 不推荐，除非未来强需求「单二进制多配置部署」

### 方案 C — 维持现状，文档改口称 bin 含 main

- 最小代码改动
- **直接违背**用户已确认决策 #1，运维/CI 易把 `bin/` 当 artifact 目录打包，误含源码

**推荐：方案 A**

---

## 5. 迁移步骤（方案 A，预估 1 个 PR）

1. 创建 `cmd/event/_template/main.go`（自 `bin/_template` 迁出）
2. 将 15 个 `bin/{slug}/main.go` → `cmd/event/{slug}/main.go`（import 路径不变）
3. 更新 `run.sh`：
   ```bash
   go build -o "bin/${slug}/$(binary_name "$slug")" "./cmd/event/${slug}"
   ```
4. 删除 `bin/_template/main.go`；`bin/_template/` 仅保留 `config.yaml` 样例
5. 更新 `2026-06-01-taskevents-handlers-relocation-design.md`：删除 §2 中 `bin/{event}/main.go` 行；构建示例改为 `cmd/event/…`
6. 确认 `.gitignore` 忽略 `bin/**/task-events-*`（若尚未）
7. `go test ./...` + `./run.sh integration-test` 回归

**不在此变更内**：G4.5 E2E、G5 删 Python handler（行为不变）。

---

## 6. 领域概念清单（供后续 DDD / 计划引用）

| 类型 | 候选 |
|------|------|
| **Bounded Context** | 领域事件消费（taskEvents）、各业务域（accounts/projects/cloud/…） |
| **Entity** | EventConsumer（按 event_type 单实例）、EventDefinition（slug/port/groupId） |
| **Aggregate** | PerEventDeployment = slug + config overlay + binary artifact |
| **Domain Event** | 15 种 `event_type`（不变）；目录修正不改变事件语义 |

---

## 7. 价值流影响（value-stream.yaml）

**受影响流**：`domain-events-consumer-split`（系统管理与策略）

| 维度 | 影响 |
|------|------|
| 行为 / 字段 | **无** — 仅目录与构建路径；`task-events-*.health.status`、USER_CREATED 链不变 |
| 测试 | **无** — `go test ./...`、integration tag；pytest 仍针对 dispatch/Redis |
| 新步骤 | 不需要新 value-stream step；可选在 runbook 注明「构建入口 cmd/event」 |
| 交叉依赖 | runAll `domain-events-bin` 的 `build_command` 仍调用 `run.sh build {slug}`，脚本内部改路径即可 |

---

## 8. 待确认

1. **是否采用方案 A**（`cmd/event/{slug}/main.go` + 纯运维 `bin/{slug}/`）？
2. **`bin/{slug}/README.md`** 是否保留（运维文档，非源码）？
3. **是否在本 PR 一并 `.gitignore` 构建产物**，避免 `task-events-*` 进库？

确认后可进入 `/6-plans` 或直接小步实施（移动 + run.sh，约 30 分钟）。
