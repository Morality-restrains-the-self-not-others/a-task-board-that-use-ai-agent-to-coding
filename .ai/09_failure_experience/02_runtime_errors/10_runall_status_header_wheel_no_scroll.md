# runAll 状态页工具条占高且无法滚走

## 症状

打开 `http://host:9999/`（runAll Status），可观测 / 日志收集 / Trace / smoke / 开发工具等条带占满大半视口，服务列表只剩底部一条。在该区域滚轮时：

1. 旧版：条带在 `.services-pane` **之外** 且 `flex-shrink: 0`，滚轮无法把条带滚走，主观为「无法向上滚动、可视区域变小」。
2. 更早：`body { overflow: hidden }` 时页头滚轮完全无响应（仅 pane 内可滚）。

## 根因

- 布局：`body` 纵向 flex + `overflow: hidden`；仅 `.services-pane` 为 `overflow: auto`。
- 工具条曾放在 `workspace` **之前**，固定占高；可观测条长路径 + `margin-left: auto` 的清空按钮在 `flex-wrap` 下再撑多行，进一步挤占服务列表。

此前「滚轮转发到 `.services-pane`」只能滚服务列表，**不能**把固定工具条移出视口，故「可视区域变小」仍在。

## 修复

1. 将 observability / 日志收集 / Trace / smoke / 开发工具及详情面板 **移入** `.services-pane`，滚动时整块上移，服务列表获得完整工作区高度。
2. 可观测条路径用 `.obs-path` 单行 ellipsis（`title` 保留全文）；去掉清空按钮的 `margin-left: auto`，减少强制换行。
3. 保留 `document` 非 passive `wheel`：目标不在其他独立滚动容器时，将 `deltaY` 写入 `.services-pane`（覆盖仍在 pane 外的页头 / 进度条）。
4. `.services-pane` 显式 scrollbar 样式（`scrollbar-width: thin` + webkit）。

实现：`runAll/src/status.html`（go:embed，需 `./build.sh` 后 hot-replace）。

## 验证

- Go：`TestUIHomePage_LayoutFlexColumnPresent`（含 pane 内嵌 observability-bar、`findVerticalScrollableAncestor`）
- Playwright：`页头区域滚轮可驱动服务列表滚动`、`工具条区域滚轮可将工具条滚出视口并露出更多服务`
- 手工：工具条上滚轮 → 条带上移离开视口；服务列表可视高度明显增大

## 部署注意

`status.html` 经 `//go:embed` 打进二进制；改 UI 后必须重建并启动新 `bin/runAll`（`/api/shutdown-self` hot-replace 可保留托管进程）。hot-replace 后状态仓为空，需再 `POST /api/start-all`（带 `session_id`）恢复监控。

## 参考

- `docs/superpowers/specs/2026-06-02-runall-services-list-scroll-clipping-design.md`
