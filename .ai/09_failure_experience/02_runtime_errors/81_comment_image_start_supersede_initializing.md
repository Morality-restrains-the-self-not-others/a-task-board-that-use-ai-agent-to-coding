# [运行时] 第二个 @镜像评论冷启动尝试释放仍在 Initializing 的实例

## 失败现象

同任务连续两个 `@镜像` 评论触发启动时，任务详情服务器日志交错出现：

1. 第一个评论已 `aliyun服务器启动成功`，UserData 开始「容器初始化」
2. 第二个评论出现「正在根据 @镜像 启动运行环境…」「已提交启动请求（含闲置复用策略）」
3. 随后报错：`释放旧实例失败，取消本次启动: 停止虚拟机失败: … IncorrectInstanceStatus.Initializing`
4. 日志未标明具体镜像/评论，两路启动与 UserData 输出混在同一 `statusLogs` 流中，难以区分归属

## 根因

1. **同任务 CSC 已有 instance_id**：首评冷启动写入 `cloud_server_configs.instance_id`，状态多为 `Starting`/`Initializing`。
2. **闲置复用不会命中启动中机器**：`tryReuseIdleMachine` 要求 `idle_since` 且非 starting，因此第二评走冷启动。
3. **`CLOUD_SERVER_STARTED` 的 `supersedeExistingInstance`**：为防同 `InstanceName` 孤儿实例，启动前对已绑定实例执行 `DeleteInstance`；阿里云在 `Initializing` 时返回 403，handler 将整次启动取消。
4. **用户观感「被释放」**：文案含「释放旧实例」；若状态已 Running 则 supersede 可能真删首实例。本次日志显示释放失败，首机 UserData 仍继续。
5. **日志无镜像维度**：SSE `message` 未带 `installed_image_id` / `comment_id` / `log_label`。

## 解决方案

1. **同任务启动中附着**：`tryAttachSameTaskInFlightMachine` — 本任务已有 Starting/Running 实例时直接 reuse，跳过二次冷启动。
2. **supersede 禁止误杀存活节点**：`shouldPreserveBoundInstance` — Running/Starting/Pending/Initializing 一律附着不删；`runtime_source=cloud_vm_comment_mention` 对空/未知状态也附着（防 mid-boot 误删）；仅 Stopped 等终态才允许 DeleteInstance 后冷启动。DeleteInstance 若仍返回 `IncorrectInstanceStatus.*` 则附着。
3. **SSE/UserData 统一前缀**：`[实例ID、容器名]`（缺省位 `-`；容器名优先显式 `container_name`/`mock_container_name`，否则 `task_{taskID}_{commentID}`）；`log_label` / `instance_id` / `container_name` 透传；UserData 经 `__TASK2APP_CONTAINER_NAME__` + 元数据 instance-id 拼装同格式。

## 关键路径

- `taskCloudService/src/workspace_machine_inflight_attach.go`
- `taskEvents/internal/handlers/cloudserverstarted/supersede.go`
- `taskEvents/internal/handlers/cloudcommon/startup_log_scope.go`
- `taskEvents/internal/handlers/taskcommentimagementioned/handler.go`
- `taskFE/app/src/utils/serverStatusLogLines.js`

## 预防

- 并发 `@镜像` / 二次 start-vm 须先判定本任务是否已有 in-flight ECS。
- 任务级 SSE 多启动源必须带 `log_label`（实例ID + 容器名）。
- 禁止在 `IncorrectInstanceStatus.*` 过渡态硬失败整条启动链。
