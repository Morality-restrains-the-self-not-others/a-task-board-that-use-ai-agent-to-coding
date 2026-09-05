# host-network relay 容器占用端口导致 EADDRINUSE（lsof 盲区）

## 现象

selected_image 启动日志：

- token-exchange / exchange-refresh OK
- `[onlineServiceJS] port 8765 already in use (listen EADDRINUSE …)`
- 容器随后退出

## 根因

1. `go_relayToTrae` 使用 `docker run --network host`，业务端口（默认 8765）绑在宿主机命名空间。
2. 残留的 `relay_taskId_*` 容器继续监听该端口。
3. `ensureOnlineServicePortFree` → `killPortListeners` 依赖非 root `lsof -tiTCP:PORT -sTCP:LISTEN`；对 root/host-network 容器监听常返回空，清理无效。

## 解决

1. `docker run` 前 `docker ps -aq --filter name=relay_taskId_` + `docker rm -f` 清理孤儿。
2. 再探测端口；仍占用则拒绝启动并写明错误。
3. 停止路径在 PID 杀不掉时同样走孤儿容器清理。
4. `onlineServiceJS` **先 `listen` 再换票**，避免换票 await 窗口内端口被抢走；本地 relay 可将 monorepo `server.mjs` overlay 进容器。

## 预防

- 勿假设 `lsof` 能看见所有 host-network 监听。
- 单机单活跃 relay 任务时，启动前清理全部 `relay_taskId_*` 是安全默认。
- host-network 服务勿在占用端口前执行长时间 await。
