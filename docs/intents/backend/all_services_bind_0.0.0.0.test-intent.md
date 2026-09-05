# Test Intent: 全量应用服务监听 0.0.0.0

| ID | 用例 | 期望 |
|---|---|---|
| T1 | `ss -tlnp` 对 8001–8019 / 8796–8798 / 4000 / 18020–18037 | 无应用进程独占 `127.0.0.1:PORT` |
| T2 | runAll `stop-all` → `start-all` | 应用服务 healthy |
| T3 | task-bill / vue / events intents conf `host` | `0.0.0.0` |
| T4 | redis/kafka/`docker-infra` / `djangoInternalApi*` | 仍为 `127.0.0.1`（连接目标，非监听） |
| T5 | value-stream `:9998` | 若失败属 `value-stream.yaml` 字段校验既有问题，与绑定无关 |
