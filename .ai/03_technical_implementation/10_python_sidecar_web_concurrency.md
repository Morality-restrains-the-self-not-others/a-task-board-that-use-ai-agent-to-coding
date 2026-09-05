# Python 辅助 Web 服务并发与锁规范

## 基本信息

- 版本：1.1.0
- 创建日期：2026-05-17
- 最后修改：2026-05-17
- 维护者：Trae AI 团队

## 背景与问题摘要

本 monorepo 中除 Django 主站外，还存在若干 **本机/容器侧 Python 辅助 Web 服务**（Flask + `threading` + 子进程），例如：

| 服务 | 路径 | 职责 |
|------|------|------|
| go_relayToTrae | `go_relayToTrae/src/`（Go，已替代原 Python `relayToTrae/`） | 拉起 onlineServiceJS、登记任务、向 task2app 推送状态 |
| mock_run_container | `mock_run_container/server.py` | 本地 mock 拉镜像/启停容器、异步 job 日志 |

这些服务在 **多线程共享内存状态** 下运行（`app.run(..., threaded=True)` 或等价 WSGI 多线程），曾多次出现 **进程未崩溃但 HTTP 永久挂起**，经 nginx/Vite 代理后表现为 **500 或前端按钮无响应**。

### 已发生故障模式（须避免复现）

1. **不可重入锁重入（主因）**  
   使用 `threading.Lock()` 时，同一线程在已持锁情况下再次 `with _lock:`，线程永久阻塞。  
   - **案例 A（历史）**：`stop_service()` 持锁调用 `_stop_running_locked()`，其内再调 `_append()`（内部 `with _lock`）→ `/v1/stop` 永久挂起。  
   - **案例 B（历史）**：`_push_status_to_backend()` 持锁调 `_collect_status_snapshot()` → 推送线程死锁；随后 `/v1/start` 也要 `_lock` → 整服务假死。  
   - **案例 C（历史）**：先 `POST /v1/register` 再 `POST /v1/start`，在 B 未修复时必现。  
   - **relayToTrae 已修复（2026-05-17）**：持锁路径改用 `_append_locked`；快照用 `_collect_status_snapshot_locked(..., port_listening=…)`；停止子进程用 `_detach_running_proc_locked` + 锁外 `_terminate_proc` / `_stop_running()`。

2. **持锁期间做阻塞 I/O**  
   在 `with _lock:` 内执行 `subprocess`、网络请求、`join()`、长时间 `sleep` 等，会阻塞 **所有** 其他请求线程与后台线程，放大为「服务锁死」观感。

3. **`*_locked` 与对外 API 混用**  
   带 `_locked` 后缀的函数约定「调用方已持锁」，若其实现或下游仍调用会再次加锁的包装函数，极易在重构时引入重入。

4. **与死锁易混淆的生命周期问题**（辅助说明）  
   短生命周期进程未 `flush` Kafka、Consumer 被 `pkill` 过早终止等，表现为「无响应/无日志」，排查时需与线程锁区分；但新建 Web 服务仍应遵守「请求路径短、后台线程可退出」原则。

## 适用范围

- **强制**：新建或大幅修改的 Python 辅助 Web 服务（Flask/FastAPI + 进程内 `threading.Lock` / 共享 `dict` 状态）。
- **强制**：`mock_run_container/` 及未来同模式的 `*/server.py` 侧车。（`relayToTrae` 已迁移至 Go 实现 `go_relayToTrae/`，不再适用本 Python 锁规范。）
- **参考**：Django 主站内若手写 `threading` + 全局锁管理子进程/SSE，同样遵守「持锁禁区」；主站优先用 Django/Channels/Celery 既有并发模型，避免复制侧车锁模式。

## 核心规则（必须遵守）

### 1. 锁分层命名（禁止模糊）

| 后缀/前缀 | 语义 | 实现要求 |
|-----------|------|----------|
| `*_locked(...)` | **调用方已持有**模块级互斥锁 | 函数体内 **禁止** `with _lock:` / `acquire()`；**禁止**调会再次加锁的 `_append` 等包装函数 |
| 无后缀的对外函数 | 自行管理锁边界 | 仅在必要的最小代码块 `with _lock:`，内部只调 `*_locked` |
| `_append_locked`（relayToTrae） | 持锁写日志 | 与 `_append` 成对；后台注销、推送写回等持锁路径必须用 `_append_locked` |

**禁止**：在已持锁路径上调用会再次获取 **同一把** `threading.Lock` 的包装函数（例如持锁时调 `_collect_status_snapshot()` 而非 `_collect_status_snapshot_locked()`）。

**不推荐**用 `threading.RLock()`「省事」掩盖分层错误；应通过 `*_locked` 拆分修复调用链。

### 2. 持锁临界区禁区

在 `with _lock:`（或等价）内 **禁止**：

- HTTP 客户端请求（`urllib`、`requests` 等）
- 子进程 `Popen.wait` / `communicate` / `terminate` 等等待
- `thread.join`、长 `time.sleep`
- 读可能阻塞的管道（应放到锁外线程，锁内只更新结果字段）
- `socket.connect` / `_port_listening()` 等端口探测（即使 timeout 仅数百毫秒）

允许在锁内：读写内存结构、拷贝 list/dict 快照、布尔标志、短字符串拼接。

### 3. 后台线程与 HTTP 线程共享状态时

- 全局仅 **一把** 业务锁（或文档化后的固定锁顺序）；禁止隐式多锁交叉。
- 后台循环：**先**在锁内复制任务列表或快照，**释放锁后**再 I/O；写回结果时再短持锁。
- 子进程 stdout 读取线程：通过 `_append()`（内部短持锁）写日志，不在读循环里调会二次加锁的 API。
- 子进程 **停止**：锁内 `_detach_*_locked()` 仅取走 `Popen` 引用；`terminate` / `wait` 在锁外（见 relayToTrae `_stop_running()`）。

### 4. Flask 运行方式

- 开发/本机侧车：`threaded=True`，`use_reloader=False`（避免双进程与锁状态分裂）。
- 生产若用 gunicorn：明确 `workers` 与 **每 worker 独立状态**；不要用多 worker 共享内存 dict（应 Redis/文件/DB）。
- 提供 `GET /health` 且 **不依赖** 可能死锁的业务锁路径（或 health 使用独立轻量检查）。

### 5. 交付与回归测试（强制）

每个侧车至少包含 **死锁回归** 单测（`unittest` + 超时），覆盖：

- 典型用户序列（如 `register` → `start`、`start` 当已 running、`stop` 当已 running）；
- 在子线程调用 HTTP 客户端，主线程 `Event.wait(timeout=3)`，超时即失败。

参考实现：

- `relayToTrae/test_server_register_start.py`（register 后 start 不得挂起）
- `relayToTrae/test_server_stop.py`（stop：先 reset、再 `_stop_running`、再 kill 端口）
- `relayToTrae/test_server_deadlock.py`（子进程 wait 期间 `/health` 仍可响应；`running=True` 时 start/stop 不死锁）

mock_run_container：job 在独立线程执行，HTTP 路由持 `_jobs_lock` 仅做表项读写；新增路由时须审查持锁段。当前无 `_append_locked` 分层（`_append` 自管锁），但禁止在已持 `_jobs_lock` 时调 `_append`。

**运维**：修改 Python 侧车 `server.py` 后须 **重启该 Python 进程**（或重启对应 `run.sh` / 容器），否则仍运行旧代码。Go 侧车 `go_relayToTrae` 修改后须重新 `go build` 并重启二进制。

### 6. 代码审查清单（PR 必查）

- [ ] 是否存在 `with _lock:` 嵌套或持锁调用 `with _lock:`？
- [ ] 持锁路径是否调用无 `_locked` 后缀、但内部会加锁的函数？
- [ ] `*_locked` 函数体内是否零次 acquire？
- [ ] 持锁段是否含网络/子进程/join？
- [ ] 是否新增/更新了死锁回归单测？
- [ ] 文档是否说明「重启侧车进程后行为才生效」（本机辅助服务常见运维步骤）？

## 推荐结构（新服务模板）

```python
_lock = threading.Lock()
_state: dict[str, Any] = {}

def _append_locked(line: str) -> None:
    """调用方必须已持有 _lock。"""
    ...

def _append(line: str) -> None:
    with _lock:
        _append_locked(line)

def _mutate_locked() -> None:
    """调用方必须已持有 _lock。"""
    _state["key"] = value  # 仅内存

def _mutate() -> None:
    with _lock:
        _mutate_locked()

def _read_snapshot_locked(*, port_listening: bool) -> dict[str, Any]:
    """调用方必须已持有 _lock。阻塞探测在锁外完成后传入。"""
    return {**_state, "port_listening": port_listening}

def _read_snapshot() -> dict[str, Any]:
    with _lock:
        port = int(_state.get("port") or 0)
    port_listening = _port_listening(port)  # 锁外
    with _lock:
        return _read_snapshot_locked(port_listening=port_listening)

def _detach_proc_locked() -> subprocess.Popen | None:
    """锁内取走子进程引用并重置标志。"""
    ...

def _stop_running() -> None:
    with _lock:
        proc = _detach_proc_locked()
    if proc is not None:
        proc.terminate()
        proc.wait(timeout=12)  # 锁外

def _background_work(item) -> None:
    result = do_network_or_subprocess(item)  # 锁外 I/O
    with _lock:
        _apply_result_locked(item.id, result)
```

## 反例（禁止）

```python
def stop_service():
    with _lock:
        _stop_running_locked()  # 错误：内部 _append() 再次 with _lock

def _push():
    with _lock:
        snap = _collect_status_snapshot()  # 错误：内部再次 with _lock

def stop_service():
    with _lock:
        proc.wait(timeout=12)  # 错误：持锁等待子进程
```

## relayToTrae 实现对照（历史，已迁移至 go_relayToTrae）

> **2026-05-29**：原 Python `relayToTrae/` 侧车已删除，运行时由 `go_relayToTrae/` 替代。以下内容为历史参考，锁规范仍适用于 `mock_run_container` 等 Python 侧车。

| 能力 | 函数 |
|------|------|
| 持锁写日志 | `_append_locked` / `_append` |
| 锁外停子进程 | `_stop_running` → `_detach_running_proc_locked` + `_terminate_proc` |
| 锁外端口探测 | `_collect_status_snapshot`、`_push_status_to_backend` |
| 无锁探活 | `GET /health` |

单请求仍可能长时间占用 **一个** Flask 工作线程（如 reset、推送 HTTP），但不会因全局 `_lock` 导致全站假死。

### 可选环境变量（relayToTrae）

| 变量 | 默认 | 上限 | 用途 |
|------|------|------|------|
| `RELAY_TO_TRAE_RESET_TIMEOUT_SEC` | 45 | 120 | onlineServiceJS `/api/jobs/reset` |
| `RELAY_TO_TRAE_PUSH_TIMEOUT_SEC` | 20 | 60 | 向 task2app 状态推送 |
| `RELAY_TO_TRAE_PROC_TERM_TIMEOUT_SEC` | 12 | 60 | 子进程 SIGTERM 后 wait |
| `RELAY_TO_TRAE_PROC_KILL_TIMEOUT_SEC` | 5 | 30 | SIGKILL 后 wait |
| `RELAY_TO_TRAE_PORT_PROBE_TIMEOUT_SEC` | 0.35 | 5 | 本机端口探测 |

本地调试若 Django 较慢，可适当增大 `PUSH`；若 stop 经常卡在 reset，可减小 `RESET` 或先修 onlineServiceJS 响应时间。

`task2app/conf/port_config.json` → `relayToTrae.timeouts` 由 `go_relayToTrae/src/config.go` 在启动时读取（**不覆盖**进程里已显式 export 的值）。

## 相关仓库文件

- `go_relayToTrae/src/` — 当前 relay 侧车实现（Go）
- `task2app/conf/port_config.json` — `relayToTrae.timeouts`
- `mock_run_container/server.py` — job 表与 `_jobs_lock`

## 变更日志

- 2026-05-17：版本 1.1.0 - 记录 relayToTrae 并发修复落地（`_append_locked`、`_stop_running`、锁外 `_port_listening`、死锁单测）；可配置 HTTP/子进程超时环境变量；推送 `seq` 持锁递增；补充运维重启说明
- 2026-05-17：版本 1.0.0 - 初版；归纳 relayToTrae 锁重入死锁与侧车并发规范
