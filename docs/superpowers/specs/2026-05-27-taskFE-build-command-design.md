# taskFE 编译按钮 build_command 配置

Date: 2026-05-27  
Status: approved

## 问题

在 `http://localhost:9999/` runAll 控制台对 `taskFE` 点击「编译」，返回：

```
Build failed: service "taskFE" has no build command configured
```

## 根因

`runAll` 的 `BuildService`（`runAll/src/runner.go:1058`）要求服务配置中存在非空 `build_command` 字段。UI「编译」按钮调用该 API。

当前 `taskFE` 仅配置了运行命令：

```yaml
- name: taskFE
  command: "npm run dev"
  working_dir: task2app/front_project/app
```

同文件中 `go-run-container`、`go-relay`、`value-stream` 等已配置 `build_command`，`taskFE` 遗漏。

## 方案

在 **两处** 编排配置中为 `taskFE` 增加 `build_command`：

| 文件 | 用途 |
|------|------|
| `runAll/config.yaml` | runAll 默认配置（`--config` 默认值） |
| `runAll.yaml` | 项目根目录编排副本（与 config 保持同步） |

```yaml
- name: taskFE
  build_command: "npm run build"
  command: "npm run dev"
  working_dir: task2app/front_project/app  # 或 ../task2app/front_project/app（config.yaml 相对路径）
```

`npm run build` 对应 `package.json` 中的 `vite build`，与 dev 模式解耦；「编译」仅执行构建，不重启 dev 进程（符合 `BuildService` 语义）。重启时也会先 build 再换进程（若用户点「重启」）。

## 价值流影响

影响 `value-stream.yaml` 中 `runall-cascade-lifecycle` 域下的 `taskFE.runtime.lifecycle_status` 相关步骤；本次为配置补齐，不改变生命周期状态机逻辑。无需新增 value stream 条目。

## 域概念（轻量）

| 概念 | 说明 |
|------|------|
| **Service Build Config** | runAll 服务编排中的 `build_command` 可选字段 |
| **BuildService** | 仅编译、不启停进程的领域操作 |

## 不在范围内

- 修改 `runner.go` 逻辑（已有能力完备）
- 为无构建步骤的服务（如纯 Python dev server）添加 build
- 变更 dev 启动命令

## 验证

1. 重启或 reload runAll 使配置生效
2. 在 UI 对 running/stopped 的 `taskFE` 点击「编译」→ 成功，无上述错误
3. `cd runAll && go test ./...` 无回归

## 测试计划

- 手动：UI 编译按钮
- 可选：`npm run build` 在 `task2app/front_project/app` 目录可独立成功
