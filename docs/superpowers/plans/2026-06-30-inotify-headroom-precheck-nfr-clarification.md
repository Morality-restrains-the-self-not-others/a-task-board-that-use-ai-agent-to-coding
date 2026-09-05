# NFR 澄清: inotify 余量不足预检与系统性修复

> 输入:
> - 设计文档: `docs/specs/inotify-headroom-precheck-设计文档.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-30-inotify-headroom-precheck-value-stream.md`
>
> 输出使用者: `/6-ddd-领域设计驱动`, `/7-plans-实施计划`, `/8-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 容错机制 | L2 | Vite 启动后 100% 存活确认；inotify 余量不足时预检告警 |
| 可观测性 | L2 | ENOSPC 失败信息写入 vite.log，预检警告输出到终端 |
| 可维护性 | L1 | 脚本改动 ≤10 行/文件，回滚直接 revert |
| 性能 | L0 | 不适用 — 启动流程，无运行时延迟要求 |
| 可伸缩性 | L0 | 不适用 — 单机开发环境 |
| 可用性 | L0 | 不适用 — 开发工具，非生产服务 |
| 安全性 | L0 | 不适用 — 无认证/授权/加密变更 |
| 数据一致性 | L0 | 不适用 — 无数据写入 |
| 合规与隐私 | L0 | 不适用 — 无数据处理 |

## 逐增量 NFR 分析

### Increment 1: 释放 inotify 配额（ramsync-daemon 改用轮询）

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: ramsync-daemon inotify watch 消耗从 ~34,000 降至 0
- **质量场景**: QS-01（见下方）

#### NFR 类别: 可维护性
- **等级**: L1 - 基础
- **量化目标**: 改动 ≤4 行，10 秒轮询间隔可环境变量覆盖

### Increment 2: 提高系统上限（sysctl）

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: max_user_watches 从 65,536 → 524,288（8× 扩容）
- **质量场景**: QS-02（见下方）

### Increment 3: 启动可靠性增强（run.sh + vite.config.js）

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: Vite 启动失败 100% 检测；inotify <20% 余量时预检告警
- **质量场景**: QS-03, QS-04（见下方）

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**: 终端可见告警消息 + 修复建议；vite.log 含完整错误栈

## 质量场景

### QS-01: ramsync-daemon 零 inotify 消耗
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | 系统启动 ramsync-daemon |
| 刺激 | ramsync-daemon 开始监控 /tmp/ram-work |
| 制品 | ramsync-daemon 进程 |
| 环境 | 正常运行 |
| 响应 | 不调用 inotify_add_watch，改用 sleep + rsync 轮询 |
| 响应度量 | `find /proc/$(pgrep -f ramsync)/fd -lname 'inotify*' \| wc -l` 输出 0 |

### QS-02: inotify 上限持久化
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | 系统重启 |
| 刺激 | 内核加载 sysctl 配置 |
| 制品 | /etc/sysctl.d/99-inotify.conf |
| 环境 | 系统启动 |
| 响应 | max_user_watches 自动设置为 524288 |
| 响应度量 | `sysctl fs.inotify.max_user_watches` 输出 524288 |

### QS-03: Vite 启动后 ENOSPC 崩溃被捕获
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | Vite 进程在 ready 后因 ENOSPC 退出 |
| 刺激 | inotify 监听数达到内核上限 |
| 制品 | `start_vite_dev()` 存活确认逻辑 |
| 环境 | 多服务并发运行，inotify 接近上限 |
| 响应 | `start_vite_dev()` 检测到 PID 消失 → 输出 RED 错误消息 → 返回 1 |
| 响应度量 | `start_vite_dev` 返回值非零；vite.log 含 ENOSPC 栈；终端输出 "Vite 进程在启动后退出" |

### QS-04: inotify 余量预警
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L2 |
| 刺激源 | run.sh 启动 Vite |
| 刺激 | inotify 已用量 > 80% max_user_watches |
| 制品 | `start_vite_dev()` 预检逻辑 |
| 环境 | 正常启动流程 |
| 响应 | 黄色警告输出到终端，建议提高 max_user_watches 或检查 ramsync |
| 响应度量 | 终端可见 `⚠ inotify 监听余量不足（已用 X / Y），建议：1. 提高系统上限 ... 2. 检查 ramsync-daemon` |

## 领域模型影响

无。纯基础设施/运维修复，不涉及业务领域概念、实体、聚合或领域事件。

## 权衡与边界

### 取舍
- ramsync-daemon 从实时（inotify 事件驱动）改为 10 秒轮询 —— 接受 ≤10 秒同步延迟，换取 ~34,000 inotify watches 释放
- sysctl 扩容到 524288 约消耗 512MB 内核内存 —— 开发机 16GB+ RAM 可接受

### 明确不做什么
- 不在此次变更中引入 per-service inotify namespace 隔离（过于复杂，开发环境不需要）
- 不修改 front_project 的 `usePolling: true` 配置（已是 inotify-free）
- 不在 Vite 启动脚本中自动执行 `sudo sysctl`（权限分离原则）

### 升级触发条件
- 如果 524288 仍不够（再次出现 ENOSPC），升级至 1048576 或考虑 cgroup inotify 隔离
- 如果 10 秒轮询延迟导致数据丢失场景，可降低 POLL_INTERVAL 至 5 秒或增加显式 sync 钩子

## 跳过声明
- **性能**: 跳过。启动流程无运行时延迟要求，不涉及请求处理路径。
- **可伸缩性**: 跳过。单机开发环境，无水平扩展需求。
- **可用性**: 跳过。开发工具非生产服务，无 SLA 要求。
- **安全性**: 跳过。无认证/授权/加密变更，sysctl 写操作需 sudo（已有系统权限控制）。
- **数据一致性**: 跳过。无数据写入，rsync 自身保证文件级原子性。
- **合规与隐私**: 跳过。无数据处理。
