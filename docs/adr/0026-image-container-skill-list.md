# ADR-0026: 镜像容器技能列表文件与上传时绑定

- **Status:** accepted
- **Date:** 2026-08-21
- **Author:** cursor
- **Deciders:** 工程团队（/goal 自动采用）

---

## Context

镜像市场已能抽取 `/app/autoRunStep.md` 作为自动运行说明，但镜像内 Agent 技能（用户在任务描述/评论中用 `/技能名` 选用）没有约定文件，也无法在上传时绑定到镜像。厂商门户 skill 文档只覆盖容器→SaaS HTTP inbound。

跨服务数据格式约定属于必须记录的架构决策。

## Decision

We will treat `/app/imageSkills.yaml` as the well-known skill catalog inside a container image.

1. Ordered YAML list; **the first skill is the default**.
2. taskAiProvider extracts the file from OCI layers on vendor create/update (same registry walk as autoRunStep.md) and stores JSON + extract status on `ai_provider_vendorcontainerimage`.
3. Public catalog and tenant installed-image snapshots expose `image_skills`.
4. taskFE highlights `/skill` tokens against the selected/mentioned image; comments persist `mentions[].skill`.
5. Runtime env `IMAGE_SKILL` carries the chosen (or default) skill name into the container.
6. The convention is documented in `saas-machine-container.md` (rendered at `/saas-machine-container`).

This is independent of ADR-0024 inbound HTTP contract versioning.

## Alternatives Considered

### Alternative 1: Scan `/.claude/skills/*/SKILL.md` with no list file

- **Pros:** 对齐 Agent Skills 目录惯例
- **Cons:** 无稳定顺序，无法定义「第一个即默认」；OCI 抽多层目录成本高
- **Why rejected:** 用户明确要求列表文件且第一项为默认

### Alternative 2: 厂商表单手填技能

- **Pros:** 不拉镜像层
- **Cons:** 与镜像内容漂移；用户要求上传时自动解析
- **Why rejected:** 违反「自动解析镜像内文件」

### Alternative 3: 复用 autoRunStep.md 正文当技能列表

- **Pros:** 无新文件
- **Cons:** 语义冲突（运行步骤 vs 技能目录）
- **Why rejected:** 两种产物面向不同 UI

## Consequences

### Positive

- 镜像、市场、任务、评论共用同一技能目录
- 默认技能无需额外配置字段

### Negative / Trade-offs

- 再抽一层 OCI（可与 autoRun 合并为一次 walk，本期先并行，记 OPT）
- 缺文件的镜像技能列表为空，UI 不展示 `/` 补全

### Mitigations

- 抽取失败不阻断上架/安装（与 autoRun 一致）
- name 正则校验，拒绝重名与非法 token

## References

- 设计: `docs/superpowers/specs/2026-08-21-image-skills-list-design.md`
- 相关: [ADR-0024](0024-saas-inbound-skill-version.md)
