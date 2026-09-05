# 测试意图：评论级容器绑定调度

## 用例

### T1 — independent 可并行且各挂独立 CSC

- **给定** 两评论均为 independent 且已 create binding，任务已有 cloud_server_config（任务级 comment_id=''）
- **当** advance
- **则** 二者均 `running`，各有独立 `mock_container_name` 与**不同** `csc_id`（可并行跑在不同机器/容器上）

### T2 — wait_previous 阻塞

- **给定** c2 depends_on c1 且均为 wait_previous
- **当** 仅 advance
- **则** c1 running、c2 waiting_previous；c1 complete 后再 advance → c2 running

### T3 — workspace compute query task_id

- **给定** 无 X-Task-Id
- **当** POST/GET `.../workspace/.../cloud/compute/comment-container-bindings/?task_id=`
- **则** 使用 query task_id 成功

### T4 — ensure 同步 execution_mode（防 409 漂移）

- **给定** 已存在 wait_previous 且 status=waiting_previous 的 binding
- **当** 再次 POST 同 comment_id 且 execution_mode=independent
- **则** 返回 200，binding.execution_mode=independent，status 重置为 pending（可立即 advance）

### T5 — 容器命名 task_{taskId}_{commentId}

- **给定** create binding comment_id=cNew task_id=task1
- **当** 返回 binding
- **则** `container_name` / `mock_container_name` = `task_task1_cNew`（创建时即写入，无需 advance）

### T5b — 任务 ID 已含 `task_` 时不二次加前缀

- **给定** create binding task_id=`task_15666874162351520866` comment_id=`cmt_15666877397783348868`
- **当** 返回 binding / 前端展示
- **则** 容器名为 `task_15666874162351520866_cmt_15666877397783348868`（不得出现 `task_task_…`）

### T6 — 启动 TraceId 按评论独立持久化

见 `comment_container_binding_start_trace.test.intent.md`。

### T7 — 启动日志 created_at 为 UTC RFC3339

见 `comment_container_binding_log_utc.test.intent.md`。

### T8 — @镜像 wait_previous 有前序不得 StartVM（2026-08-16）

- **给定** 任务已有未完成评论，新评论 `execution_mode=wait_previous` 且 @镜像
- **当** 发布 `TASK_COMMENT_IMAGE_MENTIONED`（含 `has_unfinished_predecessors=true`）
- **则** taskEvents 消费成功且 **不** 调用 StartVM；SSE 提示等待前序
- **并且** 直接 POST start-vm 且 binding 为 `waiting_previous` → 409
- **并且** `independent` 即使有前序仍 StartVM

### T9 — 评论级 ECS InstanceName 与容器名一致（2026-08-16）

- **给定** 同任务两条评论 cmt_java / cmt_lisp 均 RunInstances
- **当** 构建阿里云请求
- **则** InstanceName 分别为 `task_{taskId}_cmt_java` 与 `task_{taskId}_cmt_lisp`，互不相同，且等于各自 `container_name`
- **并且** Describe/heal **只**按评论名查找，不得回退任务级/`task-{taskId}`
- **并且** 缺少 `comment_id` 时不设置 InstanceName（不回退任务 id）

### T10 — 终止等待前序（2026-08-17）

- **给定** binding status=`waiting_previous`
- **当** POST `.../comment-container-bindings/{commentId}/cancel/`
- **则** status=`cancelled`，阶段日志「用户已终止等待」；再 advance 不再调度该 comment
- **并且** status=`running` 时 cancel → 400
- **并且** 任务详情执行细节摘要在 `waiting_previous` 时展示「终止」按钮

### T11 — 公网 IP ≠ 服务可用（2026-08-18）

- **给定** 评论 CSC 已有 `instance_id`，binding=`starting`
- **当** `persistCloudServerPublicIPByInstanceID`（含调用方传入推测性 `http://ip:8080`）
- **则** 仅回填 `public_ip` + VM `last_runtime_status=Running`，`server_url` 仍为空；`commentCSCHasRuntime` 为假；advance **不**升 `running`
- **并且** 仅在 register-reachability 写入真实 `server_url` 后 advance 才升 `running`

### T12 — independent @镜像 mention+bootstrap 只建一次机（2026-08-18）

- **给定** independent 评论 `@镜像`，mention 与 `ccbStartBinding` bootstrap 几乎同时 POST start-vm
- **当** 同进程处理两条请求
- **则** 只一次 `RunInstances`；后到者 HTTP 200 且 `duplicate_skipped=true`，`start_trace_id` 保持先到者
- **并且** mention 路径未持锁且 CSC 无真实实例时 bootstrap 仍触发 start-vm（Kafka 挂掉可恢复）
- **并且** 仅 sibling 评论的 start 事件不得让本评论 bootstrap 跳过

## 自动化落点

- `taskCloudService/src/comment_container_bindings_test.go`
- `taskCloudService/src/comment_container_bindings_cancel_test.go`
- `taskCloudService/src/compute_start_vm_trace_test.go`
- `taskCloudService/src/comment_container_binding_start_trace_test.go`
- `taskCloudService/src/cloud_utc_datetime_test.go`
- `taskCloudService/src/comment_container_bindings_log_test.go`
- `taskCloudService/src/compute_server_content_test.go`（T11：公网 IP 不写 server_url）
- `taskCloudService/src/comment_csc_instance_bind_test.go`（T9 + T11）
- `taskEvents/internal/handlers/taskcommentimagementioned/mention_start_gate_test.go`
- `taskEvents/internal/handlers/taskcommentimagementioned/handler_test.go`（T8：wait_previous 跳过 StartVM）
- `taskTaskService/src/comment_image_mention_event_test.go`
- `taskCloudService/src/comment_binding_start_vm_gate_test.go`
- `taskCloudService/src/comment_start_vm_inflight_test.go`（T12：mention+bootstrap 互斥 / duplicate_skipped）
- `taskCloudService/src/aliyun_run_instances_test.go`（T9）
- `taskCloudService/src/orphan_instance_reconcile_test.go`（T9：评论级 InstanceName 对账）
- `taskEvents/internal/cloud/aliyun/instance_name_test.go`
- `taskEvents/internal/cloud/aliyun/start_vm_test.go`（T9：CLOUD_SERVER_STARTED RunInstances）
- `taskEvents/internal/handlers/cloudserverstarted/handler_test.go`（CommentID 透传到 StartVM）
