# 容器→SaaS 接口版本（厂商镜像必选）

- **日期**: 2026-08-20 16:50
- **状态**: accepted（/goal 自动采用）
- **作者**: cursor
- **页面**: https://provider.daydaymoney.com/saas-machine-container
- **ADR**: [ADR-0024](../../adr/0024-saas-inbound-skill-version.md)

## 问题

「容器→SaaS 接口」（`saas-machine-container.md`）是容器 `onlineServiceJS` 出站调用 SaaS 的契约 SSOT。契约已发生过破坏性变更（如 path 必须含 `/comment/{cid}/`），但：

1. 文档没有正式版本号，厂商无法声明「镜像实现的是哪一版接口」。
2. 厂商添加镜像版本时只有 **镜像自身版本**（如 `1.0.0`），与接口契约版本混为一谈。
3. 平台无法按镜像过滤不兼容契约，也无法在 skill 页展示当前/历史契约。

## 成功标准

1. 契约以整数版本发布；当前契约为 **v1**（2026-08-16 comment_id path 强制）。
2. skill 页标题旁展示当前版本；可用版本选择器查看已发布契约原文。
3. 厂商 **创建/编辑** 容器镜像时必须从已发布列表选择「容器→SaaS 接口版本」，禁止自由输入。
4. `ai_provider_vendorcontainerimage.saas_inbound_skill_version` 持久化；存量行回填 `1`。
5. POST/PUT 未知或已 sunset 的版本 → 400。
6. 厂商列表、平台审核、公开目录返回并展示该字段。
7. 回归测试覆盖校验、回填、UI 必选。

## 决策（已锁定）

**采用：契约标签 + 单活 HTTP 面（不 URL 分叉）。**

- SaaS inbound HTTP 路径保持一条活契约（One-Version Rule：不立刻拆 `/v1/` `/v2/` URL）。
- 每个容器镜像声明它实现的契约修订 `saas_inbound_skill_version`（字符串数字 `"1"`、`"2"`）。
- 已发布版本目录人类 SSOT：`docs/skills/saas-container/versions.yaml`（与 skill markdown 同目录，厂商可读）。运行时由 `taskAiProvider` `go:embed` 同内容副本（`infrastructure/saascontainer/`），避免 clone-run 无 docs 子仓时 GET catalog 500。
- 破坏性变更时：bump 整数版本，旧稿归档为 `saas-machine-container.vN.md`，主文件指向 current。
- 一期只发布 v1；sunset 策略留给后续版本（v1 状态 `current`）。

拒绝：

| 方案 | 原因 |
|------|------|
| 与镜像 `version` 共用字段 | 语义冲突（镜像标签 ≠ 契约修订） |
| SemVer 自由输入 | 厂商会填镜像版本；无法枚举已发布契约 |
| 立即 URL 分叉 `/api/.../v2/` | 现网只有一活面；Expand/Contract 尚未需要网关双路由 |
| 仅文档写版本、不落库 | 无法在审核/启动链路使用 |

## API 契约

### GET `/api/ai-provider/saas-inbound-skill-versions/`

- 鉴权：公开（与 skill markdown 一致）
- 响应：

```json
{
  "current": "1",
  "versions": [
    {
      "version": "1",
      "status": "current",
      "released": "2026-08-16",
      "summary": "comment_id 必须出现在 path；禁止无 cid 的旧 TaskApiEndPoint",
      "doc_href": "/saas-machine-container.md?version=1"
    }
  ]
}
```

### GET `/saas-machine-container.md?version={n}`

- 缺省 `version` → current
- 未知 version → 404

### POST/PUT `/api/vendor/container-images/`

新增必填 JSON 字段 `saas_inbound_skill_version`（string，须为 published 且非 `sunset`）。

错误：`400` `{"detail":"请选择已发布的容器→SaaS 接口版本"}`

列表/详情/公开目录/管理端镜像 JSON 均含该字段。

落点：扩展现有 Go `taskAiProvider`，不新增 Python 接口。

## 领域概念

| 概念 | 类型 | 说明 |
|------|------|------|
| SaasInboundSkillVersion | VO | 已发布整数版本字符串 |
| ContainerImage | Entity | 新增 `SaasInboundSkillVersion` |
| SkillVersionCatalog | Domain service 输入 | 从 YAML 加载 published 集 |

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 厂商创建/更新镜像并选定接口版本 | `ContainerImageSaasInboundSkillVersionAssigned` | `handleVendorContainerImages` 成功写库后 `EventBus.Publish` | 一期 LogEventBus（与 VendorDocumentUploaded 同模式） | — |
| 查看 skill 文档/版本目录 | — | — | — | 纯查询 |
| 平台审核查看版本 | — | — | — | 纯查询 |

## 价值流影响

影响既有「厂商登记容器镜像 → 审核上架 → 租户选用」流：在登记步骤插入契约版本选择。不新增独立价值流域。

## 🕸️ Code Review Graph 分析

- `code-review-graph update --brief`：增量 9 files / 0 nodes（本会话开始时工作区几乎干净）。
- 代码基线（CLI/源码）：`taskAiProvider` 已有 `GET /saas-machine-container.md`、`CreateContainerImage(vendor, group, version, imageURL, arch)`、厂商门户 `uploadForm.version` 仅为镜像版本。
- 爆炸半径：`store_marketplace.go` 全部 SELECT/INSERT 镜像行、`store_public_catalog.go` 扫描列、`VendorPortal.vue` 添加/编辑模态、`AdminImageReviewTab.vue`、skill 页。
- `codegraph status`：索引可用（50127 nodes）。MCP `codegraph_explore` 本会话不可用，已用 ripgrep 对齐调用链。

## 🏛️ 架构变更影响

- **迭代版本**: v90 🎯 target
- **迭代名称**: saas-inbound-skill-version
- **作者**: cursor
- **设计日期**: 2026-08-20 16:50
- **新增文件**（每个视图四类伴生格式）:
  - `docs/architecture/v90-application-integration-20260820-1650-cursor.puml`
  - `docs/architecture/v90-enterprise-landscape-20260820-1650-cursor.puml`
  - 伴生 `.diff.archimate` / `.full.archimate` / `.mermaid.md`
- **已有文件（未修改）**: v88 current；v89 ztree target 正交
- **变更明细**: 🟢 版本目录 API + YAML；🟡 ContainerImage 字段 + 厂商门户必选；无废弃组件

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| `.diff.archimate` | Plateau v88 → Gap（契约无版本）→ WP → Plateau v90；厂商门户 → taskAiProvider → 镜像表 |
| `.full.archimate` | 同上拓扑 + 公开目录/审核只读该字段 |

## 权限（摘要，详见 permission-analysis）

- 版本目录 GET：公开只读
- 写 `saas_inbound_skill_version`：仅镜像所属厂商，且镜像可编辑（draft/rejected）
- 审核/目录：只读展示，不可改字段（审核通过后冻结，与镜像 URL 一致）

## 运行时一期范围

一期 **不** 改 Cloud/Credential 路由，不按版本做双栈兼容层。字段供审核与后续启动注入（`SAAS_INBOUND_SKILL_VERSION` env）预留。注入 userdata 记 OPT。
