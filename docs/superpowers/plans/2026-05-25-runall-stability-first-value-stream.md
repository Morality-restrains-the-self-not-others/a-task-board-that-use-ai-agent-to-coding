# runAll 稳定性优先改造 - Value Stream

## 目标

通过 runAll 作为唯一启停入口，实现“可预测失败、可诊断恢复、无长时间 pending”的开发环境编排体验。

## 价值流增量

### Increment M1: 单一配置源 + 所有权冲突拦截

- 输入：
  - `runAll/config.yaml`
  - 运行中的服务进程与端口占用状态
- 处理：
  - 解析唯一配置并生成配置指纹
  - 校验服务所有权（owner session）
  - 端口冲突 fail-fast，拒绝隐式接管
- 输出：
  - `PRECHECK_PORT_CONFLICT` 等结构化失败
  - ownership 可查询状态

### Increment M2: 前置检查闸门 + 三阶段生命周期

- 输入：
  - M1 的配置与 ownership 状态
  - 关键前置 probe（如 migration）
- 处理：
  - Preflight -> Launch -> Readiness 分阶段执行
  - 将失败映射到稳定 failure_code
- 输出：
  - `PRECHECK_RUNTIME_PREREQ_FAILED`
  - `LAUNCH_PROCESS_EXITED`
  - `READINESS_TIMEOUT` / `READINESS_BAD_STATUS`

### Increment M3: 可诊断状态页 + doctor 命令

- 输入：
  - 启动会话与失败事件
- 处理：
  - 汇总最近失败摘要与修复 hint
  - 支持 `runAll doctor` 全量预检
- 输出：
  - 状态页结构化诊断
  - 可自动化消费的 exit code

## 验收路径

1. 外来进程占用端口：3 秒内在 Preflight 阶段失败，不进入 Launch。
2. migration 缺失：在 Preflight 阶段返回修复命令，不启动依赖服务。
3. 修复后重试：服务进入 Readiness 并标记 Healthy。

## 领域输入（供 `/5-ddd-领域设计驱动`）

- Bounded Context:
  - `orchestration-lifecycle`
  - `service-runtime-observability`
- 关键不变量：
  - 单服务单 owner session
  - Launch 成功不代表 Readiness 成功
  - 所有失败必须具备 `phase + failure_code + hint + session_id`
