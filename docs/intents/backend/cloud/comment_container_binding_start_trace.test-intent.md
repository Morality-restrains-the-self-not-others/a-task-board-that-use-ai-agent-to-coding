# 测试意图：评论容器启动 TraceId 按评论持久化

## 测试目标

验证启动链路 TraceId 独立于 `task_id`、按评论持久化，且 list / HTTP 回传该值。

## 测试分层

- 单元：`taskCloudService/src/compute_start_vm_trace_test.go`
- 集成（MySQL）：`taskCloudService/src/comment_container_binding_start_trace_test.go`
- 复用 HTTP：`taskCloudService/src/compute_start_vm_idle_reuse_test.go`（403 / 不跨任务 reuse / inflight 附着 trace_id）

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 无入站 X-Trace-Id，task_id 合法 | 连续两次 `bindStartVmTraceContext` | 两个非空 ID，互不相同，均 ≠ task_id |
| T2 | 入站 X-Trace-Id=`web-start-cmt-aaa` | bind | 保留该入站 ID |
| T3 | 入站 X-Trace-Id 等于 task_id | bind | 生成新 ID，≠ task_id |
| T4 | 同任务两评论 binding | persist 两个不同 ID 后 list | JSON 各自 `start_trace_id` 不同 |
| T5 | 已有独立 ID | persist(task_id) | 列不变；空 comment_id 不扇出 |
| T6 | 无 comment_id 的 publishTaskSSE | 扇出调度日志 | 正文不含同一 `trace_id=` |
| T7 | 同评论 inflight 附着 start-vm-auto 200 | 读 body | `trace_id` 非空且 ≠ task_id；`inflight_attach=true` |
| T8 | persist 时 binding 行尚不存在 | 随后 POST create binding | list 该评论 `start_trace_id` 为暂存值 |
| T9 | 仅有评论 A 的 start 事件 | `loadStartEventPayload` 评论 B | 返回 A 的镜像/网络参数 |
| T10 | binding 已有、CSC 行存在、`start_trace_id` 空 | `persistStartVmInstanceBinding` 带 `trace_id` | list 该评论列为该 ID |
| T11 | 已有独立 `start_trace_id` | `ensureCommentBindingStartTraceID` 另一 preferred | 列保持原值 |
| T12 | binding `csc_id` 非空且列空 | GET list | 补写独立 ID，≠ task_id |

## 自动化落点

- `taskCloudService/src/compute_start_vm_trace_test.go`
- `taskCloudService/src/comment_container_binding_start_trace_test.go`
- `taskCloudService/src/comment_container_binding_server_log_test.go`
- `taskCloudService/src/compute_start_vm_idle_reuse_test.go`
- `taskCloudService/src/comment_csc_bootstrap_test.go`
