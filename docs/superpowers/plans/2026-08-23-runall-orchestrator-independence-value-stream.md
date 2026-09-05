# 价值流：runAll 挂掉后业务面仍可用

- **Date:** 2026-08-23
- **Design:** `docs/superpowers/specs/2026-08-23-runall-orchestrator-independence-design.md`

## Related Value Streams

Greenfield for this ops increment — 无既有 `*runall*orchestrator*` value-stream。与「精准编译重启 / shutdown-self 热替换」相邻，但是 **故障域隔离** 而非编译路径。

## 价值主张

操作员或意外信号弄掉 `:9999` 时，租户仍能使用 APISIX/taskAuth/taskFE 等已启动服务；拉起新 runAll 后 UI 重新显示健康状态。

## 增量

| # | 增量 | 用户可感知结果 | 验证 |
|---|------|----------------|------|
| 1 | 退出策略默认保留服务 | kill/SIGTERM runAll 后面板服务端口仍监听 | T1 T4 |
| 2 | 子进程独立 session | SIGKILL runAll 后服务不因 SIGHUP 退出 | T5 + 属性测试 |
| 3 | 新实例 adopt | `./run.sh` 空窗拉起 UI 不为空、不误杀 | 既有 adopt/orphan 测 |

## 步骤（端到端）

1. 托管服务已 Healthy（start-all 完成）。
2. runAll 收到 SIGTERM 或进程消失（9999 无监听）。
3. 托管端口仍在 LISTEN。
4. 操作员 `runAll/run.sh` 再拉起。
5. Status UI adopt 后显示 healthy，无需 start-all。

## 非增量

- 把每个服务改成 systemd。
- 改变 stop-all 语义。
