# 镜像容器技能列表（imageSkills.yaml）

- **日期:** 2026-08-21
- **状态:** accepted（/goal 自动采用）
- **作者:** cursor
- **ADR:** [ADR-0026](../../adr/0026-image-container-skill-list.md)

## 目标

约定镜像内技能列表文件；厂商在 `https://provider.daydaymoney.com/` 登记/更新镜像时自动抽取并绑定；首项为默认技能；镜像市场可查看；创建任务描述可引用并高亮 `/技能名`；评论 `@镜像` 后可继续 `/{技能}`；规范写入 `saas-machine-container.md`（门户 `/saas-machine-container` 渲染）。

与 ADR-0024 `saas_inbound_skill_version`（容器→SaaS HTTP 契约版本）正交：本决策管**镜像内 Agent 技能目录**，不是 inbound API 版本。

## 🕸️ Code Review Graph 分析

`CRG unavailable: codegraph MCP 未接入本会话；按既有 autoRunStep 抽取调用链设计。`

既有链：`POST /api/vendor/container-images/` → `scheduleAutoRunExtract` → OCI layer 抽 `/app/autoRunStep.md` → 落 `ai_provider_vendorcontainerimage` → 公开 catalog → Cloud 安装快照 `cloud_tenant_installed_images` → taskFE ImageMarket / CreateTask / 评论 @mention。

## 当前架构理解

- 应用层：taskAiProvider（登记/抽取）、taskCloudService（租户安装快照）、taskTaskService（评论 mentions）、taskFE（市场/创建任务/评论）、trae-agent（示例镜像）
- 既有 sidecar 文件：`/app/autoRunStep.md`
- 厂商门户 skill 页：`GET /saas-machine-container.md`

## 决策（锁定）

1. **路径 SSOT**：`/app/imageSkills.yaml`（与 autoRunStep 同层，Dockerfile `COPY`）。
2. **格式**：YAML `version` + 有序列表 `skills[]`；`name` = `[a-z0-9][a-z0-9-]{0,62}`；**列表第一项即默认技能**（不另写 `default:` 字段，避免双源）。
3. **抽取**：厂商保存 `image_url`/`version` 时与 autoRun 并行抽层；catalog 空缓存回填；Cloud 安装时若快照为空则 best-effort 再抽。
4. **落库 JSON**：`image_skills_json` + `image_skills_extract_status` + `image_skills_digest`。缺文件 = `not_found`（镜像仍可上架）；非法 YAML/重名 = `failed`。
5. **API 透传**：公开 catalog / 已安装镜像 JSON 增加 `image_skills`（对象：`version, default_skill, skills[]`）与抽取状态。
6. **创建任务**：选中镜像后，描述中匹配该镜像 `skills[].name` 的 `/name` token 高亮；芯片可插入。
7. **评论**：`@镜像` 确认后可输入 `/` 弹出技能列表；未指定则绑定 `default_skill`。`mentions[].skill` 写入 comments。
8. **容器运行**：UserData/环境注入 `IMAGE_SKILL=<name>`（缺省=默认技能）。容器按该名加载 `/app/skills/<name>/SKILL.md`（可选，列表文件不强制正文路径）。
9. **文档**：`docs/skills/saas-container/saas-machine-container.md` 新增「镜像内技能列表」节。Demo 镜像 `COPY onlineServiceJS/imageSkills.yaml /app/imageSkills.yaml`。

## 示例

```yaml
version: 1
skills:
  - name: general-coding
    description: 本镜像默认软件开发流程
  - name: k8s-debug
    description: Kubernetes 排障
```

评论正文示例：`@trae-agent /k8s-debug 请查 CrashLoop`

## 不做

- 不把技能列表与 `saas_inbound_skill_version` 混用
- 不要求镜像必须带技能文件
- 不在本期做技能正文全文入库（只绑列表元数据）
- 不 URL 分叉 inbound API

## Python 新增接口清单

无。全部落现有 Go 服务字段与前端。

## 架构交付物

- `docs/architecture/v93-application-integration-20260821-1635-cursor.{puml,diff.archimate,full.archimate,mermaid.md}`
- `docs/architecture/v93-enterprise-landscape-20260821-1635-cursor.{puml,diff.archimate,full.archimate,mermaid.md}`
