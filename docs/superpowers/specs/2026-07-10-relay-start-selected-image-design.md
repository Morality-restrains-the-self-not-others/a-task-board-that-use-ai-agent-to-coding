# 设计：relayToTrae「启动」拉取并运行用户所选镜像

- 日期：2026-07-10
- 状态：已采纳（goal-mode 自动决策，跳过确认门）
- 相关 URL：`?relayToTrae=true` 任务详情「直接启动」

## 1. 问题

任务详情页顶部可选择 `TenantInstalledImage`，但 `?relayToTrae=true` 下点击「启动」时：

- 前端仅用 `selectedImageId` 做 UI 门禁（`hasImage`）
- `POST .../relay-to-trae/start/` **不传** `installed_image_id`
- `go_relayToTrae` 固定执行 `onlineServiceJS/run.sh`，**不** `docker pull` 所选镜像

与云主机 `start-vm` / 本地 `mock-run-container` 的「选哪个镜像就拉哪个、跑哪个」语义不一致。

## 2. 目标与成功标准

1. 用户在镜像下拉中选中镜像后点「启动」，系统解析该镜像的可拉取引用（`image_url` + `version`）。
2. 本机侧车执行 `docker pull <ref>`，成功后 `docker run`（`--network host`，注入既有 relay env：`ACCESS_TOKEN`、`TaskApiEndPoint` 等）。
3. 未选镜像或镜像无 `image_url` → 启动失败并返回明确错误。
4. 「停止」可停止该容器；status/SSE 能反映 running。
5. 既有 OAuth / token-init / 凭证预检流程保留。

## 3. 方案对比（自动采纳 A）

| 方案 | 描述 | 取舍 |
|------|------|------|
| **A（采纳）** | relay 编排不变；Gateway/Django 解析镜像；`go_relayToTrae` 在有 `image` 时走 docker pull/run | 保留 relay 令牌与 SSE；与云 UserData 语义对齐 |
| B | 「启动」改调 mock-run-container | 丢失 relay token-init/UI/SSE 一体体验 |
| C | 仅前端改文案，不改运行时 | 不满足「拉取并运行」 |

## 4. 设计要点

### 4.1 前端

- `startRelayToTrae` 的 `start/` body 增加 `installed_image_id: selectedImageId`。
- 未选镜像时提前返回「请先选择镜像」（与 mock-run 一致）。

### 4.2 Gateway 热路径（主路径）

`handleRelayStart`：

1. 读 body `installed_image_id`；若空则查任务快照（可选，本期以前端必传为主）。
2. 调用 Cloud `GET /api/internal/mock-run/resolve-image?tenant_id=&installed_image_id=`。
3. 将 `image`、`installed_image_id` 写入转发 `go_relayToTrae` 的 payload。
4. 解析失败 → 4xx，不派发侧车。

### 4.3 Django 薄路径（兜底）

`relay_to_trae_start` 同样解析镜像并写入 payload（与 Gateway 对齐），避免仅走 Django 时行为分叉。

### 4.4 go_relayToTrae

- `POST /v1/start` 接受可选字段 `image`、`installed_image_id`。
- **有 `image`**：`startSelectedImageContainer` — `docker pull` → `docker run -d --rm --network host --name relay_task_<taskId> -e ... <image>`；后台 `docker logs -f`；**不**再跑 host `run.sh`；**不**等待宿主机 `container_refresh_token.json`（容器内自行换票，与云对齐）。
- **无 `image`**：保留既有 `startOnlineService`（run.sh）以兼容旧调用方/测试。
- state 增加 `ContainerID` / `ContainerName` / `Image`；stop 时若有容器则 `docker stop`。
- status：`running` 以容器存活或既有进程为准；`online_service_up` 在容器模式下以容器 running 为准（host 端口探测作辅助）。

### 4.5 权限

不新增公开 API；复用既有 `relay-to-trae/start` 会话鉴权 + 内部 resolve-image。仅租户已安装镜像可解析。

### 4.6 架构影响

属应用集成变更：relay 启动数据流增加「镜像解析 → docker 运行时」。产出 v15 application-integration（`.puml` + `.archimate` + `.mermaid.md`）。

## 5. 非目标

- 不合并 mock-run 与 relay 两个 Tab。
- 不改变云 `start-vm` UserData 逻辑。
- 不强制删除 run.sh 回退路径（无 image 时仍可用）。

## 6. 测试意图摘要

- 单元：resolve 失败拒绝启动；有 image 时侧车调用 docker pull/run（可 mock exec）。
- Gateway：start body 带 `installed_image_id` 时调用 resolve-image 并转发 `image`。
- 前端：start body 含 `installed_image_id`。
- 回归：无 image 的旧 start 仍可走 run.sh（若测试依赖）。
