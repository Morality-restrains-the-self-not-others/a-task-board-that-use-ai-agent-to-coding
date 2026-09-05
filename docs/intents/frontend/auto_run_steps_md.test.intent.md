# 测试意图：autoRunStep.md

## 用例

### T1 — 容器 API
给定镜像内有 `/app/autoRunStep.md`，当 `GET /api/auto-run-steps`，则返回 markdown。

### T2 — 注册抽取（含 DRAFT）
给定厂商保存 DRAFT 且 `image_url` 可访问，当抽取完成，则 `VendorContainerImage.auto_run_steps_md` 非空且 status=ok。实现：`POST/PATCH /api/vendor/container-images/` 触发 `scheduleAutoRunExtract`（生产异步；单测 ExtractSync）。

### T3 — 开发中 catalog
给定绑定厂商的 DRAFT 镜像已抽取，当 `GET …/installed-images/dev-catalog/`，则条目含 `auto_run_steps_md`。

### T4 — image-market 开发中预览
给定 T3 数据，当打开 `/tenant/…/image-market` 开发中区块，则可展开看到说明；卡片同时展示 `updated_at`（更新时间）。

### T5 — 创建任务
给定已安装（含 is_dev_mode）镜像且 auto_run 勾选，当选中该镜像，则展示缓存说明。

### T6 — 抽取失败 / 空说明
给定仓库中无该文件或尚未抽取，当抽取结束 status=not_found 或 markdown 为空：镜像市场 / 创建任务不渲染自动运行说明区块（不出现「暂无自动运行说明」）；任务详情在 auto_run=true 时仍可展示暂无说明。

### T7 — 任务详情 live + 回退
给定 auto_run=true：容器 `server_url` 就绪时优先 `container-auto-run-steps` live；否则展示已安装镜像缓存字段。

### T8 — catalog 空缓存回填
给定已上架镜像 `auto_run_steps_extract_status` 为空且 `image_url` 可抽取，当 `GET /api/public/catalog/`，则触发抽取并写入 status=ok（或 not_found/failed）。首次响应可将空 status 标为 pending，刷新后展示缓存说明。
