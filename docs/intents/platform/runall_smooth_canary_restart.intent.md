# Intent: 全部重启 / 精准编译重启支持金丝雀平滑重启

## 背景与目标

runAll Status（`:9999`）的「全部重启」当前是 `StopAll` 再 `StartAll`：先把整栈 SIGTERM/SIGKILL 掉，再按依赖拉起。期间 APISIX 上游 Connection refused、Kafka 消费者空窗打 DLT、进行中的 HTTP 请求被掐断。

「精准编译重启」在 compile-then-swap 成功后仍是 **先停旧进程、等端口释放、再启新进程**（ADR-0027 的 swap 半段），编译成功后仍有秒级空窗。

目标：两个按钮改为 **按依赖序的金丝雀平滑重启**——新进程先在同端口就绪（SO_REUSEPORT 重叠），再 SIGTERM 排空旧进程；旧进程用 `http.Server.Shutdown` 完成进行中的请求后再退出。

## 范围与边界

- **范围内**：`POST /api/restart-all`、`POST /api/precise-restart`、单服务 `restartService` 的 swap 半段；Go HTTP 服务统一 `tracelog.ListenAndServe`；taskEvents 健康端口 SO_REUSEPORT + SIGTERM 取消消费。
- **范围外**：全部停止 / 全部启动语义不变；MySQL/Redis/Kafka 等无法同端口双实例的基础设施走排空后启动（不重叠）；runAll 编排器自身热替换不在本意图。

## 约束与风险

- ADR-0027 仍成立：全部重启不编译；精准路径先原子编译再 canary swap；编译失败保留旧进程。
- 首次滚动时旧进程尚未 SO_REUSEPORT：重叠 bind 失败则回退为「SIGTERM 排空 → 启动」（仍优于整栈 stop-all）。
- 金丝雀不得误杀新进程组：只向 **旧 PID/PGID** 发 SIGTERM/SIGKILL。

## 验收标准

1. 「全部重启」不再先 StopAll 再 StartAll；健康服务在新进程就绪前持续响应健康检查。
2. 精准编译重启：编译失败旧进程仍健康；成功则 PID 切到新进程，旧进程在 drain 超时内退出。
3. Go HTTP 服务收到 SIGTERM 后 `Shutdown`，进行中请求不被立刻掐断（测例覆盖）。
4. 确认文案标明金丝雀平滑重启（非整栈先停）。

## 实施计划

见 `docs/superpowers/specs/2026-09-03-runall-smooth-canary-restart-design.md`。

## 业务意图 → 事件对照

> 运维编排器进程生命周期，不产生业务领域事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| Intent: 金丝雀平滑重启 | — | — | — | — | 运维进程生命周期，不产生业务领域事件 |

## 变更记录

- 2026-09-03：新增。来源：Status UI `button#restart-all-btn` / `button#precise-restart-btn` 要求支持平滑重启与金丝雀。
