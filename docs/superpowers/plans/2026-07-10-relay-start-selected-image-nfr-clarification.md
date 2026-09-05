# NFR：relay 启动所选镜像

默认 L2（Standard）。

| 属性 | 级别 | 说明 |
|------|------|------|
| 正确性 | L3 | 必须跑所选镜像，禁止静默回退到 run.sh（当已传 image） |
| 可用性 | L2 | pull 失败明确错误；不阻塞其它任务详情功能 |
| 安全 | L2 | 仅租户已安装镜像；内部 resolve；secret 头不变 |
| 性能 | L2 | pull 可能较慢；异步 accepted + 日志流，与 mock-run 一致 |
| 可观测 | L2 | 日志含 image ref、pull/run 阶段；禁止打印完整 ACCESS_TOKEN |
| 兼容 | L2 | 无 image 的旧调用保留 run.sh |

## 领域影响

- Runtime 启动命令从「固定 host 脚本」扩展为「可选容器镜像引用」值对象。
- 不新增聚合根；复用 TenantInstalledImage 与既有 resolve。
