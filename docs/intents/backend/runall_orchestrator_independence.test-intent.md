# 测试意图：runAll 编排器退出不拆栈

## 测试目标

证明编排器退出与托管进程生命周期解耦。

## 测试分层

| 层 | 内容 |
|----|------|
| 领域单测 | `KeepManagedServicesOnOrchestratorExit` 默认 true；env=1 且 trigger=signal 为 false |
| 进程单测 | UI `Run` cancel 后 sleep 子进程仍存活；`managedServiceSysProcAttr` 含 Setsid |
| 回归 | shutdown-self / uiListenerLost / HadPrevious=false skip orphan 仍绿 |

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 默认策略 + signal | keep=true |
| T2 | env=1 + signal | keep=false |
| T3 | env=1 + shutdown-self | keep=true |
| T4 | UI Run 启动 sleep，cancel | PID 仍存活，再 stopProcess 清理 |
| T5 | SysProcAttr helper | Setsid 与 Setpgid 均为 true |

## 数据与环境

Linux；不依赖真实 :9999。

## 通过标准

上述用例全绿；无残留 sleep（测试 Cleanup 杀组）。
