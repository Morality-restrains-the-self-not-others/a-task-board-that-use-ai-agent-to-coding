# Runbook：容器实例卡在「启动中」的补救说明

- **适用**：云端容器实例（start-vm / UserData `init_from_task2app.sh` 初始化）状态长期停在「启动中」，无法靠前端刷新自愈
- **相关代码**：`taskAiProvider/frontend/src/utils/userDataScriptLinux.js`、`userDataTemplate.js`、`taskCloudService/src/container_inbound_actions.go`（`handleBootProgress`）、`taskCloudService/src/compute_handlers.go`（`server-userdata-verify`）
- **关联 OPT**：OPT-20260811-001（重生 UserData）、OPT-20260811-002（本文）

## 症状

1. 任务详情页服务器状态一直显示「启动中」，不进入「运行中」。
2. 实例上 `/root/init_from_task2app.sh` 是**旧模板**，脚本内容带以下三类历史缺陷（已修复但存量脚本不会自动更新）：

   | 缺陷 | 旧脚本表现 | 后果 |
   |------|-----------|------|
   | `COMMENT_ID='-'` | 字面量 `-`（RunInstances 前未替换 `__TASK2APP_COMMENT_ID__`） | 旧逻辑：评论启动日志路由丢失。**2026-08-12 起** `resolveBootProgressCSC` 在任务级 CSC 已有 instance 时仍切评论 CSC，SSE 可补 `comment_id`；前端 filter 亦兼容 `task2app-container` 标签 |
   | `docker inspect -` | 健康检查用了**未定义的 camelCase** `${containerName}`，展开为空/`-` | `docker inspect` 失败 → `set -e` exit 1 → 脚本中断 |
   | `report_progress` 缺闭合引号 | JSON 载荷破损 | boot-progress 上报失败 → 状态卡「启动中」、verify 回调不触发 |

3. 该脚本已写入磁盘，**不会**随服务重启/前端刷新自愈。

## 根因

模板源码 `userDataScriptLinux.js` / `userDataTemplate.js` 中 health 检查与 `report_progress` 的缺陷在**新生成**的模板里已修复：

- `export COMMENT_ID='__TASK2APP_COMMENT_ID__'`（RunInstances 时替换）
- 健康检查一律用已 export 的 `"${CONTAINER_NAME}"`（**禁止** camelCase `${containerName}`）
- `report_progress` 输出闭合 JSON 引号

但镜像市场/租户**已安装**的旧 UserData 模板内容不会自动重生，厂商需在门户「重新生成容器脚本」并保存后，新开实例才拿到修复版脚本。

## 补救步骤（二选一）

### 方案 A（推荐）：释放实例，按新模板重跑 start-vm

1. 在厂商门户/镜像市场找到该模板，点击「重新生成容器脚本」并**保存**（确保内容含 `export COMMENT_ID='__TASK2APP_COMMENT_ID__'` 与 `"${CONTAINER_NAME}"` 健康检查）。
2. 对卡住实例执行**释放**（terminate），确认已无残留资源。
3. 用任务「重新启动」/ start-vm 重新拉起一台新机，观察启动日志出现「安装基础依赖 / 拉取容器镜像」等 userdata_boot 步骤，状态离开「启动中」。

### 方案 B：SSH 手工修复存量脚本（不换机）

> 适用：不愿释放实例、机器可 SSH、仅需让当前脚本跑通。

1. SSH 到实例，编辑 `/root/init_from_task2app.sh`：
   - 把 `COMMENT_ID='-'` 改成真实评论 ID：`COMMENT_ID='cmt_<实际评论ID>'`（从 @镜像 评论链接中取）。
   - 把健康检查里 `${containerName}` 改为 `"${CONTAINER_NAME}"`（脚本头部已 export）。
   - 核对 `report_progress` 行的 JSON 载荷闭合引号完整。
2. `bash /root/init_from_task2app.sh` 重跑，观察逐步进度上报。
3. 若 `server-userdata-verify` 回调已触发但状态仍旧，等一次轮询刷新；仍卡则走方案 A。

### 只解除「启动中」状态的兜底（慎用）

- 若仅需让状态离开「启动中」、不关心进度归位：确认 `server-userdata-verify` 回调（`/api/container/.../server-userdata-verify/`）是否可达；或**人工**将 CSC（Cloud Server Config）标为 running。
- ⚠️ 人工标 running 会跳过真实健康校验，可能导致「运行中」但容器实际未就绪，仅限临时解除阻塞。

## 预防

- 厂商修改模板后，须在门户**保存**新模板再开新实例；存量模板内容不会自动重生（OPT-20260810-037 盘点确认空表，无迁移必要，但未来新增模板需养成保存习惯）。
- 新实例启动日志中应看到 `report_progress` 各步骤与「拉取容器镜像」等字样；若只看到容器调度三行：先查 Loki 是否已有 `sse_redis_published` 且 message 含 UserData 步骤——有则多为评论扇出问题（见 failure `90_boot_progress_task_level_csc_skips_comment_route`）；无则脚本未跑通或旧模板缺陷（本 runbook 方案 A/B）。
