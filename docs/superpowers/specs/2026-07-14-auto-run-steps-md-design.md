# autoRunStep.md：镜像内真源 + 注册抽取 + 创建/详情/镜像市场展示

- **日期**: 2026-07-14 13:27
- **作者**: claude / brainstorming
- **状态**: approved（goal-mode 跳过确认门，2026-07-14 14:25）
- **python_api_approval**: scoped-down（goal-mode 确认）
- **相关页面**:
  - 任务详情：`…/task-detail/…/?relayToTrae=true`
  - 创建任务：WorkPanel / CreateTaskModal
  - 镜像市场：`…/tenant/…/image-market`（含 **开发中镜像**）
  - 厂商门户：`provider.daydaymoney.com` VendorPortal
- **python_api_approval**: **scoped-down / n/a** — **不新增** Python HTTP path；仅扩展既有 vendor/public catalog 响应字段。镜像文件抽取落 **Go**。
- **前置设计**: `2026-07-13-auto-run-first-instruction-and-delivery-design.md`（auto_run 运行时步骤）
- **相关意图草案**: `docs/intents/frontend/exec_log_per_step_push.intent.md`（执行日志逐步推送，正交）

---

## 1. 问题 / 意图

用户需要在勾选「自动运行」或查看任务时，**按所选容器镜像**看到「自动运行会执行哪些操作」的说明。

约束：

1. **真源只在容器镜像内**（`autoRunStep.md`），避免厂商双写漂移。
2. **容器未启动**（创建任务、镜像市场浏览）时仍须可读说明。
3. 须覆盖镜像市场 **开发中镜像**（`DRAFT` / `PENDING_REVIEW` / `REJECTED`，经 `dev-catalog` 展示、可 `is_dev_mode` 安装）。

## 2. 成功标准（SMART）

| # | 标准 | 可验证方式 |
|---|------|------------|
| S1 | Trae 镜像内固定路径 `/app/autoRunStep.md`；容器 `GET /api/auto-run-steps` 返回 markdown | onlineServiceJS 单测 |
| S2 | 厂商保存/更新 `image_url`（含 **草稿/开发中**）时，Go 抽取器从 OCI 仓库抽出该文件并写回 `VendorContainerImage.auto_run_steps_md` | Go 单测（mock registry） |
| S3 | 公开 catalog / `vendor-development-catalog` / `unsubmitted-image` 响应含 `auto_run_steps_md` + `auto_run_steps_extract_status` | ai-provider 既有 API 字段断言 |
| S4 | 租户安装（市场上架 **或** 开发中）时，`tenant_installed_images` 拷贝该字段 | taskCloudService 单测 |
| S5 | `image-market` 的「开发中镜像」与「可用镜像」均可展开预览自动运行说明 | Vue/组件测或 Playwright |
| S6 | 创建任务勾选 auto_run 且已选镜像 → 展示说明；容器未起时仅依赖 installed/catalog 缓存 | Vue 测 |
| S7 | 任务详情 `auto_run=true`：容器就绪优先 live API，否则回退 installed 缓存 | Vue 测 |
| S8 | 抽取失败：字段空 + status=`failed`；UI 明确「暂无自动运行说明 / 抽取失败」 | 单测 + UI |
| S9 | 意图 + 价值流测试点更新；Swagger（Go 抽取 API）可见 | 文档 / OpenAPI |

## 3. 方案对比与采纳

| 方案 | 描述 | 结论 |
|------|------|------|
| A. 仅容器内文件 | 未起容器无法展示 | 拒（不满足创建任务） |
| B. 仅市场手填 Markdown | 易与镜像漂移 | 拒 |
| C. 镜像真源 + 注册时 OCI 抽取缓存 | 未起可读；与镜像一致 | **采纳** |
| D. 注册时 docker run 读文件 | 重、需 Docker daemon | 拒；用 Registry HTTP / crane 抽 blob |

**开发中镜像**：与已上架 **同一抽取管道**；状态集合仍为 `_VENDOR_DEV_CATALOG_STATUSES`（DRAFT / PENDING_REVIEW / REJECTED）。开发迭代频繁改 `image_url` → **每次 URL/digest 变更触发重抽**。

## 4. 详细设计

### 4.1 镜像内约定

| 项 | 值 |
|----|-----|
| 路径 | **`/app/autoRunStep.md`**（固定；本期不可配置） |
| 格式 | UTF-8 Markdown |
| 内容建议 | 有序步骤：bootstrap → 首指令 → 身份/commit/push/PR；可含镜像特有步骤 |
| 缺失 | 抽取结果空；运行时 API `404` 或 `markdown: ""` + `note` |

默认内容（Trae online 镜像仓库内提交一份模板，非 SaaS 动态生成）。

### 4.2 容器运行时 API（onlineServiceJS）

```
GET /api/auto-run-steps
→ 200 { "markdown": "...", "path": "/app/autoRunStep.md", "source": "image_filesystem" }
→ 404 { "detail": "autoRunStep.md not found" }
```

经 taskContainerGateway 转发时可挂到既有 L0 代理模式（与其它 `/api/*` 一致）；**不新增 Django 路由**。

### 4.3 Go 抽取服务（推荐落点）

**落点顺序**：扩展 **taskCloudService**（已代理 ai-provider、持有 `tenant_installed_images`）或新建小模块 `taskImageInspect`；本期推荐 **taskCloudService internal API**：

```
POST /api/internal/extract-auto-run-steps/
Body: { "image_url": "registry…/repo:tag", "path": "/app/autoRunStep.md" }
→ 200 { "markdown": "...", "digest": "sha256:…", "status": "ok" }
→ 422 { "status": "not_found" | "auth_failed" | "timeout", "detail": "…" }
```

实现要点：

- 复用 / 对齐 ai-provider `registry_manifest` 的参考解析与 Bearer 拉 token；
- 拉 manifest → config/layers → 在 tar layer 中定位文件（可用 `crane`/`regctl` 子进程或纯 Go OCI 库）；
- **超时**（建议 30–60s）、限并发；禁止阻塞 Django 请求线程。

### 4.4 注册触发（含开发中）

| 触发 | 行为 |
|------|------|
| VendorPortal 创建/更新版本（`image_url` 或 version 变更） | 异步或短同步调用 Go 抽取 → 写 `auto_run_steps_md` / `auto_run_steps_extract_status` / `auto_run_steps_digest` |
| 提交审核 `submit` | 若 status ≠ ok，可警告但仍允许提交（产品可选：强制 ok 才可审核） |
| 主站安装开发中镜像 `POST unsubmitted-image` 路径 | 若市场字段仍空，安装时可 **再触发一次抽取**（兜底） |

**字段（VendorContainerImage + 公开 payload）**：

| 字段 | 说明 |
|------|------|
| `auto_run_steps_md` | 抽取到的 Markdown 文本 |
| `auto_run_steps_extract_status` | `pending` / `ok` / `not_found` / `failed` |
| `auto_run_steps_digest` | 抽取时镜像 digest（变更则重抽） |

**既有 API 扩展（无新 path）**：

- `_public_container_image_payload` / `VendorDevelopmentCatalogView` / `UnsubmittedImageView`
- Vendor 序列化器读写仅服务端写入上述字段（厂商 UI **只读预览**，不手填）

### 4.5 租户侧

`tenant_installed_images` 增加同名字段；安装时从 catalog / unsubmitted 响应拷贝。

`GET …/installed-images/`、`…/catalog/`、`…/dev-catalog/`、`…/{id}/` 均返回这些字段。

### 4.6 前端展示

| 页面 | 行为 |
|------|------|
| **image-market · 可用镜像** | 卡片可展开「自动运行说明」预览（markdown 渲染） |
| **image-market · 开发中镜像** | **同等**预览；status 非 ok 时显示抽取状态文案；安装后进入已安装列表仍可看 |
| **image-market · 已安装** | 展示缓存说明；`is_dev_mode` 标签旁可提示「开发模式安装，说明随厂商更新镜像后需重装或刷新」 |
| **CreateTaskModal** | 勾选 auto_run + 已选 `container_image_id` → 拉取 installed 详情并展示 |
| **TaskDetail 身份区** | `auto_run=true`：优先容器 live API；失败回退 installed 缓存 |

### 4.7 开发中镜像专项（本轮补充）

```text
厂商 VendorPortal 保存草稿 (DRAFT)
  → Go 抽取 /app/autoRunStep.md
  → VendorContainerImage 缓存

主站用户绑定该厂商
  → GET installed-images/dev-catalog/
  → 代理 AI Provider vendor-development-catalog
  → 列表含 auto_run_steps_md（**未安装也可预览**）

用户点击安装开发中镜像
  → unsubmitted-image + 写入 tenant_installed_images（含 md 快照、is_dev_mode=1）

创建任务选该已安装开发镜像 + auto_run
  → 读快照展示（容器仍未起）
```

注意：

- 开发中镜像 **未上架** 也必须抽取；不能只在 APPROVED 时抽。
- 厂商改 URL 后，已安装租户侧快照 **不会自动变**；设计选项（本期采纳 **B**）：
  - **A**：安装时定时刷新 — 过重
  - **B**：快照语义 + UI 注明「以安装时为准」；可选「重新同步说明」按钮调一次 unsubmitted 刷新字段
  - **C**：每次打开详情重抽 — 成本高

本期：**B**（快照 + 可选手动刷新，可放 P1）。

## 5. Domain Concept Inventory（供后续 /6-ddd）

| 概念 | 上下文 | 说明 |
|------|--------|------|
| AutoRunStepsDocument | Container / Image | 镜像内 Markdown 真源 |
| AutoRunStepsSnapshot | Marketplace / TenantInstalledImage | 注册/安装时缓存 |
| ExtractJob | taskCloudService | OCI 抽取过程 |
| VendorContainerImage (dev) | ai-provider | DRAFT 等状态亦有 Snapshot |

## 6. 价值流影响（输入给 /3-value-stream）

| 现有 stream / step | 影响 |
|--------------------|------|
| `installed-image-api` / catalog | 字段扩展；dev-catalog 同等 |
| `create-task-auto-run-backend-start` | 创建任务 UI 展示说明 |
| auto_run 006 交付流 | 正交；说明文档描述运行时步骤 |
| **新增量建议** | `auto-run-steps-doc`：抽取 → 市场/开发中预览 → 创建/详情展示 |

Fields（三要素示例）：

- `ai-provider.marketplace_vendorcontainerimage.auto_run_steps_md`
- `task-cloud.tenant_installed_images.auto_run_steps_md`

## 7. 🐍 Python 新增接口清单与 Go 替代评估

### 拟新增接口

**无。** 不新增 Django/ai-provider path。

### 既有接口字段扩展（非新 path）

| 方法 | 路径 | 归属 | 变更 |
|------|------|------|------|
| GET | `/api/public/catalog/` | ai-provider | 响应增字段 |
| GET | `/api/public/vendor-development-catalog/` | ai-provider | 响应增字段（**开发中**） |
| GET/POST | `/api/public/unsubmitted-image/` | ai-provider | 响应/安装源增字段 |
| CRUD | `/api/vendor/container-images/` | ai-provider | 存字段；触发抽取调用 Go |

### Go 替代（抽取能力）

| # | 方案 | 推荐 |
|---|------|------|
| 1 | taskCloudService internal extract | ⭐ |
| 2 | 新建 taskImageInspect | 备选 |
| 3 | 在 Django registry_manifest 内拉 layer | 拒（单线程 + 长耗时） |

**选型结论**：抽取 **Go**；Python 仅字段扩展 → **`python_api_approval: scoped-down`**。

## 8. 🏛️ 架构变更影响（批准后写入）

批准后创建 **v23** target（当前应用集成基线 v21；v22 微信支付 target 积压中，本迭代 **另开 v23**，基于 v21 current，并在 VERSION_HISTORY 注明与 v22 并行）：

| 制品 | 路径模式 |
|------|----------|
| PlantUML | `docs/architecture/v23-application-integration-<ts>-claude.puml` |
| ArchiMate | 同名 `.archimate`（含 Plateau/Gap/WP + `sourceConnection`） |
| Mermaid | 同名 `.mermaid.md` |

变更预告：

- 🟢 `[NEW]` AutoRunSteps 抽取（taskCloudService internal）
- 🟡 `[MODIFIED]` onlineServiceJS — `GET /api/auto-run-steps`
- 🟡 `[MODIFIED]` ai-provider catalog / **vendor-development-catalog** 字段
- 🟡 `[MODIFIED]` Vue ImageMarket（含开发中）、CreateTask、TaskDetail

> ⚠️ **批准前不写入** `docs/architecture/` target 文件。

## 9. 非目标 / 风险

- 不实现厂商手填 Markdown 作为真源。
- 不保证私有仓库无凭证时抽取成功（失败可预期）。
- 多架构 manifest list：按平台默认 amd64/arm64 策略抽一层（设计实现时写死优先级）。
- 超大镜像抽 layer 成本：仅扫到文件即停；超时记 failed。

## 10. 测试意图摘要

见将同步的 `docs/intents/frontend/auto_run_steps_md.intent.md` / `.test.intent.md`（批准后落盘完善）。

开发中相关用例：

- **T-DEV-1**：DRAFT 镜像保存后 `vendor-development-catalog` 含 `auto_run_steps_md`
- **T-DEV-2**：image-market 开发中区块可预览
- **T-DEV-3**：开发模式安装后 CreateTask 可选该镜像并展示说明

---

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-07-14 | 初版：镜像真源 + 注册抽取；明确覆盖 image-market **开发中镜像** |
