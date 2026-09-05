# NFR 澄清：重启与编译分离

- **日期:** 2026-08-22
- **级别默认:** L2；可用性在编译失败路径升至 L3（不得因脏树编译失败而停机）

## 路径分片键强制审视

| 路径 | 分片 ID | 判定 | 动作 |
|------|---------|------|------|
| POST /api/restart-all | 无（本机编排） | L0 | 单机 runAll，无水平分片需求；升级触发：多节点编排器 |
| POST /api/precise-restart | 无 | L0 | 同上 |
| taskEvents run.sh start/build | 无 | L0 | 单机二进制生命周期 |

## 幂等性强制审视

| 路径 | 副作用 | 重复触发 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|----------|--------------|--------|----------|
| RestartAll | 停/启进程 | 按钮连点 | 一次 bulk 互斥 | TryBeginRestartAll / bulk mutex | 进行中 409；完成后可再点（再拉 last-good） |
| PreciseRestart | 编译+切进程 | 按钮连点 | 一次 bulk + 登记文件 | TryBeginPreciseRestart | 失败项保留登记，成功项清除 |
| run.sh start | exec 二进制 | 脚本重入 | 已有 pidfile 则 skip | pidfile | 已运行则 return 0 |
| run.sh build | 写 ELF | 并发 build | 输出路径 | 文件 rename | 后写覆盖；失败保留 last-good |

无 Kafka 新消费者。资金/云资源路径不适用。

## 质量场景

- **可用性:** 编译失败时 last-good 进程继续监听；健康检查仍通过
- **可观测性:** 编译失败日志含服务名与 build stderr；不记录源码全文
