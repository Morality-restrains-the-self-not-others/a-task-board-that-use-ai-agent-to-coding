# relay zTree「提交」无限 pending 与 trace 日志断层

**现状（2026-07-09）：** Python `start_container_job_stream_sse` daemon 已删除；job-stream 轮询仅 Go Gateway；下文根因描述为故障当时沿革。

**日期：** 2026-05-31  
**状态：** 已审批（用户确认症状 A：无限挂起）  
**页面：** `task-detail/.../?relayToTrae=true` — zTree「提交」  
**任务示例：** `848546827193511936`  
**trace 示例：** `ca2fec3a-c48b-445d-a8e5-61a228a4de76`

---

## 现象

1. 浏览器 `POST …/cloud/compute/container-layer-git-commit/` **一直 pending**，直至手动刷新/关页（**非** 120s 超时）。
2. Grafana Trace Log Explore 按 `trace_id` 过滤时 **仅 1 条**日志（onlineServiceJS `http_request`）。
3. 容器运行日志有 `outbound POST (zTree layer git commit)`，但 saas-backend **24h 内无任何** `container-layer-git-commit` 完成日志。

---

## 运行时证据（2026-05-30 现场）

| 时间 (UTC) | 组件 | 事件 |
|------------|------|------|
| 16:47:37 | Django | `POST /api/jobs` 发指令 → 启动 `container_job_stream_sse` 后台轮询 |
| 16:48:44.479 | Django | `append_container_run_log`: outbound git commit |
| 16:48:44.582 | onlineServiceJS | `POST …/git/commit` **200，100ms**（Loki 唯一 trace 行） |
| 之后 | saas-backend | **无** `http_request` 完成记录 |

**用户确认：** 症状 **A** — 无限挂起，非 2 分钟超时。排除「仅 READ timeout 过短」；更符合 **连接池/Session 层死锁**，urllib3 timeout 无法触发。

---

## 根因

### 主因（高置信）：`container_http_session()` 跨线程共享

- `cloud/services/container_forward_requests.py` 使用 **进程级单例** `requests.Session`。
- `container_job_stream_sse.start_container_job_stream_sse` 在 **daemon 线程** 中每秒 `GET …/jobs/{id}/events`，与主请求线程的 `POST …/git/commit` **并发**使用同一 Session。
- **`requests.Session` / urllib3 连接池非线程安全**；并发可导致 `post()` 在容器已返回 200 后仍永久阻塞，`HttpAccessLogMiddleware` 永不写日志 → Grafana 只见下游 1 条。

### 次因：可观测性缺口

- saas-backend 访问日志 **仅在请求完成时**写入；pending 期间 Loki 无 inbound/outbound 阶段。
- `append_container_run_log` 写文件、不进 Loki；成功路径无 outbound_done 记录。

---

## 目标

1. **功能：** zTree「提交」在 Agent job 轮询活跃时也能 **≤5s** 返回（200/4xx/502），不再无限 pending。
2. **可观测：** 同一 `trace_id` 在 Loki 可见 saas-backend **inbound → outbound_start → outbound_done → http_request** 与 onlineServiceJS `http_request`。
3. **回归：** 现有 container forward pytest 全绿。

## 非目标

- 迁移 `cloud/compute/*` 出 saas-backend（见 2026-05-28 taskAgentSupport 设计）。
- 修改 onlineServiceJS git/commit 语义。
- Grafana dashboard 结构大改（沿用 trace-log-explore）。

---

## 价值流影响

| 流 | 步骤 | 影响 |
|----|------|------|
| `task-detail-runtime-relay` | container-runtime-context | git commit 转发可靠性 |
| `platform-centralized-logging` | increment2-grafana-trace-dashboard | 长 pending 期间可见 forward 阶段日志 |
| `task-detail-relay-debug-agent-observability` | outbound visibility | 补 Django 出站 JSON 日志 |

**字段：** `saas-backend.cloud_cloudserverconfig.business_api_endpoint`（不变）

---

## 领域概念清单（供 /5-ddd）

| 概念 | 边界 | 说明 |
|------|------|------|
| `ContainerHttpClientPool` | 云平台出站 | 线程隔离的 Session 工厂 |
| `ContainerForwardRequest` | 任务协作 | 浏览器 → Django → onlineServiceJS 同步转发 |
| `ContainerJobStreamPoller` | 任务协作 | 后台 job events 轮询（须独立 Session） |
| 领域事件 | — | `LayerGitCommitForwarded`（可选审计，本迭代不持久化） |

---

## 方案对比

| 方案 | 摘要 | 推荐 |
|------|------|------|
| **A — thread-local Session + 阶段日志** | `_local.session`；job-stream 可选独立 Session；forward 打 `forward_stage` | **是** |
| B — httpx 全量替换 | 新依赖、全模块替换 | 否，范围过大 |
| C — 仅加日志 | 不改 Session | 否，不解决 pending |

---

## 详细设计（方案 A）

### 1. `container_forward_requests.py`

```python
import threading
import requests

_local = threading.local()

def container_http_session() -> requests.Session:
    s = getattr(_local, "session", None)
    if s is None:
        s = requests.Session()
        s.trust_env = False
        _local.session = s
    return s
```

### 2. `container_job_stream_sse.py`（加固）

后台轮询使用 **独立 Session**（不经过 thread-local 亦可），避免与任意主线程 forward 共享连接：

```python
_poll_session = requests.Session()
_poll_session.trust_env = False
# 在 _run 内用 _poll_session.get(...)
```

### 3. 结构化 forward 日志

新增 `cloud/services/container_forward_trace_log.py`：

```python
logger.info("container_forward", extra={
    "trace_id": ...,
    "forward_stage": "inbound|outbound_start|outbound_done|error",
    "forward_action": "layer_git_commit",
    "upstream_url": url,  # 不含 token
    "upstream_status": r.status_code,  # outbound_done 时
})
```

在 `forward_container_layer_git_commit` 入口、post 前、post 后各打一条（本迭代至少 git-commit；push/create 可 follow-up）。

### 4. 测试

| 层 | 文件 | 覆盖 |
|----|------|------|
| 单元 | `tests/test_container_forward_requests.py`（新建） | 两线程各得不同 Session 对象 |
| 单元 | `tests/test_container_job_stream_sse.py`（新建或扩） | 轮询不使用主 thread-local session |
| 回归 | `tests/test_ai_task_comment.py` | mock session 路径仍绿 |

---

## 验收标准

- [ ] 任务 `848546827193511936`（或等价）：发指令后 zTree「提交」**不再无限 pending**。
- [ ] Loki 同一 trace_id ≥3 条（saas-backend forward 阶段 + onlineServiceJS + saas-backend http_request）。
- [ ] `pytest` 相关用例全绿。
- [ ] 24h 内 saas-backend `path=~".*git-commit.*"` 完成日志 > 0。

---

## 风险与回滚

| 风险 | 缓解 |
|------|------|
| thread-local 略增连接数 | 仅 localhost 8765，可接受 |
| 回滚 | 还原 `container_forward_requests.py` 单例 |

---

**审批后下一步：** `/0-auto-flow` 或 `/6-plans` 生成实施计划。
