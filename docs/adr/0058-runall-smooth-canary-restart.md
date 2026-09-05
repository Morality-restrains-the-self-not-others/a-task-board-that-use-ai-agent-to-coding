# ADR-0058: runAll 金丝雀平滑重启（重叠监听 + 排空旧进程）

- **Status:** accepted
- **Date:** 2026-09-03
- **Author:** cursor
- **Deciders:** `/goal` 零交互采用（goal-mode 覆盖 Step 1 USER GATE）
- **Supersedes (partial):** ADR-0027 的 swap 半段「stop + forceFreshStart」——编译成功后改为 canary overlap；**编译与重启分离条款不变**

---

## Context

「全部重启」=`StopAll`+`StartAll`，整栈空窗。精准编译重启在原子编译后仍先杀旧进程再拉新进程。业务 Go 服务普遍 `http.ListenAndServe` 且不处理 SIGTERM，5s 后 SIGKILL 会切断进行中请求。用户要求两个 9999 按钮支持平滑重启，且每个服务支持金丝雀。

单机多进程（非 K8s）无法靠改 APISIX 上游端口实现内部直连（`127.0.0.1:<port>`）的零空窗。Linux `SO_REUSEPORT` 允许新旧进程同端口并存，内核把新连接分到两者，再 SIGTERM 旧进程并用 `http.Server.Shutdown` 排空。

## Decision

We will implement **per-service canary restart** as the swap mechanism for 全部重启、精准编译重启、以及非热替换的单服务重启：

1. **编排**：按 `PlanStartAll` 同拓扑顺序滚动；**禁止**先 StopAll。已停止的服务只 start；健康服务 canary。
2. **重叠**：新进程 `SO_REUSEPORT` 绑定同一端口；健康检查确认新 PID（`X-RunAll-Pid` 或 listener PID 集合）后再向 **旧 PGID** 发 SIGTERM；drain 默认 25s，超时 SIGKILL 仅旧组。
3. **回退**：重叠 bind 失败（旧进程无 REUSEPORT）→ SIGTERM 排空旧进程 → 再 start（ADR-0027 的 last-good 语义保留）。
4. **进程**：`tracelog.ListenAndServe` 统一 SO_REUSEPORT + SIGTERM `Shutdown`（[net/http Server.Shutdown](https://pkg.go.dev/net/http#Server.Shutdown)）。
5. **热替换**（`skip_stop_on_restart`）保持现有 Docker/nginx 路径，不走重叠。
6. **基础设施**（无 HTTP/无法 REUSEPORT 的 compose）：回退路径，不假装双实例。

## Alternatives Considered

### Alternative 1: 仅加长 SIGTERM 等待（5s→30s）

- **Pros:** 改动小
- **Cons:** 仍先停后起，启动耗时内空窗
- **Why rejected:** 不满足金丝雀/平滑

### Alternative 2: 临时第二端口 + APISIX 切流量

- **Pros:** 不依赖 SO_REUSEPORT
- **Cons:** 内部服务直连端口，APISIX 切了也救不了；端口配置爆炸
- **Why rejected:** 与现网寻址模型冲突

### Alternative 3: 全部重启仍 StopAll，只把精准重启改 canary

- **Pros:** 行为变化面小
- **Cons:** 用户点名的两个按钮都要平滑；全部重启是最大空窗
- **Why rejected:** 不满足「这两个功能」

## Consequences

### Positive

- 滚动期间栈不完全掉线；编译失败仍保留 last-good（ADR-0027）
- 进行中 HTTP 有机会在 Shutdown 窗口完成

### Negative / Trade-offs

- 旧二进制未升级前第一次 canary 常走回退路径（一次空窗）
- Kafka 消费者重叠期会发生两次 rebalance
- 全部重启耗时变长（逐个健康检查）

### Mitigations

- 回退路径写结构化日志 `canary_overlap_fallback`
- 进度 SSE 按服务报告 overlap/drain/start
- UI 确认文案说明滚动金丝雀而非整栈先停

## References

- 设计: `docs/superpowers/specs/2026-09-03-runall-smooth-canary-restart-design.md`
- 意图: `docs/intents/platform/runall_smooth_canary_restart.intent.md`
- [ADR-0027](0027-restart-compile-separation.md)
- Go: [Server.Shutdown](https://pkg.go.dev/net/http#Server.Shutdown)、[ListenConfig](https://pkg.go.dev/net#ListenConfig)
