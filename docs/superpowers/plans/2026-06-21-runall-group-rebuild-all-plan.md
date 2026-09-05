# 实施计划: runAll 分组批量重新编译

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-21-runall-group-rebuild-all-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-21-runall-group-rebuild-all-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-21-runall-group-rebuild-all-nfr-clarification.md`
> - 领域模型: `runAll/src/domain/build_group_result_value_object.go`

## 概览

| 属性 | 值 |
|------|-----|
| 增量 | 1 (Thin Slice) |
| 涉及文件 | 5 个（含测试） |
| 预估代码量 | ~120 行（含测试） |
| 关键依赖 | 复用已有 `BuildService`、`findService`、`resolveBuildCommand` |

## 任务清单

### Phase 1: 领域层 ✅ (已完成于 DDD 步骤)

- [x] `runAll/src/domain/build_group_result_value_object.go` — `BuildGroupResult` 值对象 + `ComputeBuildGroupStatus` + `IsTerminalBuildStatus`
- [x] `runAll/src/domain/build_group_result_value_object_test.go` — 10 个测试用例，全部通过

### Phase 2: Runner 层 — 分组编译核心逻辑

- [ ] **T2.1** 在 `runAll/src/runner_test.go` 中编写 `TestBuildGroup` 系列测试
  - `TestBuildGroup_AllSucceed` — 组内全部编译成功
  - `TestBuildGroup_PartialFailure` — 部分失败仍继续
  - `TestBuildGroup_GroupNotFound` — 不存在的组返回错误
  - `TestBuildGroup_SkipsNonTerminalStatus` — 跳过 building/starting 中的服务
  - `TestBuildGroup_NoBuildableServices` — 组内无可编译服务返回 none
  - **验证**: `go test ./src/ -run TestBuildGroup -v` → FAIL (RED)

- [ ] **T2.2** 在 `runAll/src/runner.go` 中实现 `BuildGroup` 方法
  - 查找组内所有服务 (`findServicesByGroup`)
  - 按依赖深度排序
  - 遍历 buildable 服务，仅编译 terminal 状态的服务
  - 调用已有 `BuildService` 执行编译
  - 返回 `domain.BuildGroupResult`
  - **验证**: `go test ./src/ -run TestBuildGroup -v` → PASS (GREEN)

- [ ] **T2.3** 在 `runAll/src/runner.go` 中实现辅助方法
  - `findServicesByGroup(groupName string) []*Service`
  - `orderByDependencyDepth(svcs []*Service) []*Service`
  - **验证**: `go test ./src/ -run TestBuildGroup -v` → PASS

### Phase 3: API 层 — HTTP 端点

- [ ] **T3.1** 在 `runAll/src/ui_test.go` 中编写 `POST /api/build-group` 端点测试
  - 成功场景（组内全部可编译）
  - 部分失败场景
  - 组不存在场景
  - 缺少 group 参数场景
  - 方法不允许（GET 请求）
  - **验证**: `go test ./src/ -run TestBuildGroupEndpoint -v` → FAIL (RED)

- [ ] **T3.2** 在 `runAll/src/ui.go` 中注册 `POST /api/build-group` 路由
  - 解析 body `{"group": "..."}`
  - 调用 `runner.BuildGroup(ctx, group)`
  - 返回 JSON `BuildGroupResult`
  - **验证**: `go test ./src/ -run TestBuildGroupEndpoint -v` → PASS (GREEN)

### Phase 4: 前端 — UI 按钮与交互

- [ ] **T4.1** 在 `runAll/src/status.html` 中修改 `renderStatus` 函数
  - 在 groupActions 中增加「全部重新编译」按钮
  - 有 buildable 服务 → 可点击按钮 (`data-action="build-group"`)
  - 无 buildable 服务 → disabled 按钮 + tooltip
  - 参考设计文档中的 JavaScript 代码

- [ ] **T4.2** 在 `runAll/src/status.html` 中新增 `buildGroup` 函数
  - `POST /api/build-group` 请求
  - 解析响应，alert 汇总结果
  - 异常处理（网络错误、HTTP 错误）

- [ ] **T4.3** 在 `runAll/src/status.html` 的事件委托中增加 `build-group` action
  - 在现有的 click 委托 switch/if 中增加 `case "build-group"`
  - 调用 `buildGroup(group)` 并应用 `pulseClickFeedback`

- [ ] **T4.4** 在 `runAll/src/status.html` CSS 中增加 `.rebuild-all-btn` 样式
  - 参考设计文档中的 CSS（蓝色系，与 build-btn 协调）

### Phase 5: 端到端验证

- [ ] **T5.1** 编译并启动 runAll
  ```bash
  cd runAll && go build -o bin/runAll ./src/
  ```
- [ ] **T5.2** 手动验证完整流程
  - 打开 `http://localhost:9999`
  - 确认每个分组 header 有「全部重新编译」按钮
  - 点击 platform 组按钮 → 观察编译进度
  - 确认 alert 显示汇总结果
  - 确认无 buildable 服务的组按钮为 disabled
- [ ] **T5.3** 运行完整测试套件确保无回归
  ```bash
  cd runAll && go test ./src/... -v
  ```

## 任务依赖图

```
T2.1 (RED test) → T2.2 + T2.3 (GREEN impl)
T3.1 (RED test) → T3.2 (GREEN impl)
T2.2 + T2.3 → T3.2 (API 依赖 Runner)
T3.2 → T4.1 + T4.2 + T4.3 + T4.4 (前端依赖 API)
T4.* → T5.* (E2E 验证)
```

## 验证命令速查

| 阶段 | 命令 |
|------|------|
| Domain 测试 | `go test ./src/domain/ -run "TestNewBuildGroupResult\|TestComputeBuildGroupStatus\|TestIsTerminalBuildStatus" -v` |
| Runner 测试 | `go test ./src/ -run TestBuildGroup$ -v` |
| API 测试 | `go test ./src/ -run TestBuildGroupEndpoint -v` |
| 全量回归 | `go test ./src/... -v` |
| 编译 | `cd runAll && go build -o bin/runAll ./src/` |
